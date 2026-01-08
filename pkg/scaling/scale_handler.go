package scaling

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"truefoundry/elasti/operator/api/v1alpha1"

	"k8s.io/client-go/scale"

	"github.com/truefoundry/elasti/pkg/k8shelper"
	"github.com/truefoundry/elasti/pkg/scaling/scalers"
	"github.com/truefoundry/elasti/pkg/values"
	"go.uber.org/zap"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	discocache "k8s.io/client-go/discovery/cached/memory"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/tools/record"
)

const (
	kedaPausedAnnotation         = "autoscaling.keda.sh/paused"
	kedaPausedReplicasAnnotation = "autoscaling.keda.sh/paused-replicas"
)

type ScaleDirection string

const (
	ScaleUp   ScaleDirection = "scaleup"
	ScaleDown ScaleDirection = "scaledown"
	NoScale   ScaleDirection = "noscale"
)

type ScaleHandler struct {
	kClient        *kubernetes.Clientset
	kDynamicClient *dynamic.DynamicClient
	EventRecorder  record.EventRecorder

	scaleLocks sync.Map

	scaleClient scale.ScalesGetter
	restMapper  *restmapper.DeferredDiscoveryRESTMapper

	logger         *zap.Logger
	watchNamespace string
}

// getMutexForScale returns a mutex for scaling based on the input key
func (h *ScaleHandler) getMutexForScale(key string) *sync.Mutex {
	l, _ := h.scaleLocks.LoadOrStore(key, &sync.Mutex{})
	return l.(*sync.Mutex)
}

// NewScaleHandler creates a new instance of the ScaleHandler
func NewScaleHandler(logger *zap.Logger, config *rest.Config, watchNamespace string, eventRecorder record.EventRecorder) *ScaleHandler {
	kClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Fatal("Error connecting with kubernetes", zap.Error(err))
	}

	kDynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		logger.Fatal("Error connecting with kubernetes", zap.Error(err))
	}

	// Setup a cached discovery + deferred RESTMapper
	cachedDisc := discocache.NewMemCacheClient(kClient.Discovery())
	restMapper := restmapper.NewDeferredDiscoveryRESTMapper(cachedDisc)
	kindResolver := scale.NewDiscoveryScaleKindResolver(cachedDisc)

	scaleClient, err := scale.NewForConfig(config, restMapper, dynamic.LegacyAPIPathResolverFunc, kindResolver)
	if err != nil {
		logger.Fatal("Error connecting with kubernetes", zap.Error(err))
	}

	return &ScaleHandler{
		logger:         logger.Named("ScaleHandler"),
		kClient:        kClient,
		kDynamicClient: kDynamicClient,
		scaleClient:    scaleClient,
		restMapper:     restMapper,
		watchNamespace: watchNamespace,
		EventRecorder:  eventRecorder,
	}
}

func (h *ScaleHandler) StartScaleDownWatcher(ctx context.Context) {
	pollingInterval := 30 * time.Second
	if envInterval := os.Getenv("POLLING_VARIABLE"); envInterval != "" {
		duration, err := time.ParseDuration(envInterval)
		if err != nil {
			h.logger.Warn("Invalid POLLING_VARIABLE value, using default 30s", zap.Error(err))
		} else {
			pollingInterval = duration
		}
	}
	ticker := time.NewTicker(pollingInterval)

	go func() {
		for {
			select {
			case <-ctx.Done():
				ticker.Stop()
				return
			case <-ticker.C:
				if err := h.checkAndScale(ctx); err != nil {
					h.logger.Error("failed to run the scale down check", zap.Error(err))
				}
			}
		}
	}()
}

func (h *ScaleHandler) checkAndScale(ctx context.Context) error {
	elastiServiceList, err := h.kDynamicClient.Resource(values.ElastiServiceGVR).Namespace(h.watchNamespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return fmt.Errorf("failed to list ElastiServices: %w", err)
	}

	for _, item := range elastiServiceList.Items {
		es := &v1alpha1.ElastiService{}
		if err := runtime.DefaultUnstructuredConverter.FromUnstructured(item.Object, es); err != nil {
			h.logger.Error("failed to convert unstructured to ElastiService", zap.Error(err))
			continue
		}
		cooldownPeriod := resolveCooldownPeriod(es)

		scaleDirection, err := h.calculateScaleDirection(ctx, cooldownPeriod, es)
		if err != nil {
			h.logger.Error("failed to calculate scale direction", zap.String("service", es.Spec.Service), zap.String("namespace", es.Namespace), zap.Error(err))
			continue
		} else if scaleDirection == NoScale {
			continue
		}

		switch scaleDirection {
		case ScaleDown:
			err := h.handleScaleToZero(ctx, es)
			if err != nil {
				h.logger.Error("failed to scale target to zero", zap.String("service", es.Spec.Service), zap.String("namespace", es.Namespace), zap.Error(err))
				continue
			}
		case ScaleUp:
			err := h.handleScaleFromZero(ctx, es)
			if err != nil {
				h.logger.Error("failed to scale target from zero", zap.String("service", es.Spec.Service), zap.String("namespace", es.Namespace), zap.Error(err))
				continue
			}
		}
	}

	return nil
}

func (h *ScaleHandler) calculateScaleDirection(ctx context.Context, cooldownPeriod time.Duration, es *v1alpha1.ElastiService) (ScaleDirection, error) {
	if len(es.Spec.Triggers) == 0 {
		h.logger.Info("No triggers found, skipping scale to zero", zap.String("namespace", es.Namespace), zap.String("service", es.Spec.Service))
		return "", fmt.Errorf("no triggers found")
	}

	// Check that the ElastiService was created at least cooldownPeriod ago
	if es.CreationTimestamp.Time.Add(cooldownPeriod).After(time.Now()) {
		h.logger.Debug("Skipping scaling decision as ElastiService was created too recently",
			zap.String("service", es.Spec.Service),
			zap.Duration("cooldown", cooldownPeriod),
			zap.Time("creation timestamp", es.CreationTimestamp.Time))
		return NoScale, nil
	}

	for _, trigger := range es.Spec.Triggers {
		scaler, err := h.createScalerForTrigger(&trigger, cooldownPeriod)
		if err != nil {
			h.logger.Warn("failed to create scaler", zap.String("namespace", es.Namespace), zap.String("service", es.Spec.Service), zap.Error(err))
			return "", fmt.Errorf("failed to create scaler: %w", err)
		}
		defer scaler.Close(ctx)

		// TODO: Cache the health of the scaler if the server address has already been checked
		healthy, err := scaler.IsHealthy(ctx)
		if err != nil {
			h.logger.Warn(
				"failed to check scaler health",
				zap.String("namespace", es.Namespace),
				zap.String("service", es.Spec.Service),
				zap.String("scaler", trigger.Type),
				zap.Duration("cooldownPeriod", cooldownPeriod),
				zap.Error(err),
			)
			return "", fmt.Errorf("scaler: %s, cooldownPeriod: %s, is not healthy: %w", trigger.Type, cooldownPeriod, err)
		}
		if !healthy {
			h.logger.Warn("scaler is not healthy, skipping scale to zero", zap.String("namespace", es.Namespace), zap.String("service", es.Spec.Service))
			return NoScale, nil
		}

		scaleToZero, err := scaler.ShouldScaleToZero(ctx)
		if err != nil {
			h.logger.Warn("failed to check scaler", zap.String("namespace", es.Namespace), zap.String("service", es.Spec.Service), zap.Error(err))
			return "", fmt.Errorf("failed to check scaler: %w", err)
		}

		if !scaleToZero {
			return ScaleUp, nil
		}
	}

	return ScaleDown, nil
}

func (h *ScaleHandler) handleScaleToZero(ctx context.Context, es *v1alpha1.ElastiService) error {
	// If the cooldown period is not met, we skip the scale down
	cooldownPeriod := resolveCooldownPeriod(es)
	spec := es.GetSpec()
	if es.Status.LastScaledUpTime != nil {
		if time.Since(es.Status.LastScaledUpTime.Time) < cooldownPeriod {
			h.logger.Debug("Skipping scale down as minimum cooldownPeriod not met",
				zap.String("service", spec.Service),
				zap.Duration("cooldown", cooldownPeriod),
				zap.Time("last scaled up time", es.Status.LastScaledUpTime.Time))
			return nil
		}
	}

	// Pause the KEDA ScaledObject
	if spec.Autoscaler != nil && strings.ToLower(spec.Autoscaler.Type) == "keda" {
		err := h.UpdateKedaScaledObjectPausedState(ctx, spec.Autoscaler.Name, es.Namespace, true)
		if err != nil {
			return fmt.Errorf("failed to update Keda ScaledObject for service %s: %w", spec.Service, err)
		}
	}

	targetGVK, err := k8shelper.APIVersionStrToGVK(spec.ScaleTargetRef.APIVersion, spec.ScaleTargetRef.Kind)
	if err != nil {
		return fmt.Errorf("failed to parse API version: %w", err)
	}
	if _, err := h.Scale(ctx,
		es.Namespace,
		targetGVK,
		spec.ScaleTargetRef.Name,
		0,
	); err != nil {
		return fmt.Errorf("failed to scale target to zero: %w", err)
	}
	return nil
}

func resolveCooldownPeriod(es *v1alpha1.ElastiService) time.Duration {
	cooldownPeriod := time.Second * time.Duration(es.Spec.CooldownPeriod)
	if cooldownPeriod == 0 {
		cooldownPeriod = values.DefaultCooldownPeriod
	}
	return cooldownPeriod
}

func (h *ScaleHandler) handleScaleFromZero(ctx context.Context, es *v1alpha1.ElastiService) error {
	spec := es.GetSpec()
	// We update the last scaled up time every time we evaluate that the trigger evaluates to scale-up. This means even if the scale-up is not successful, we update the last scaled up time to avoid the cooldown period increment
	if err := h.UpdateLastScaledUpTime(ctx, es.Name, es.Namespace); err != nil {
		h.logger.Error("Failed to update LastScaledUpTime", zap.Error(err), zap.String("service", spec.Service), zap.String("namespace", es.Namespace))
	}

	// Unpause the KEDA ScaledObject if it's paused
	if spec.Autoscaler != nil && strings.ToLower(spec.Autoscaler.Type) == "keda" {
		err := h.UpdateKedaScaledObjectPausedState(ctx, spec.Autoscaler.Name, es.Namespace, false)
		if err != nil {
			return fmt.Errorf("failed to update Keda ScaledObject for service %s in namespace %s: %w", spec.Service, es.Namespace, err)
		}
	}

	targetGVK, err := k8shelper.APIVersionStrToGVK(spec.ScaleTargetRef.APIVersion, spec.ScaleTargetRef.Kind)
	if err != nil {
		return fmt.Errorf("failed to parse API version: %w", err)
	}
	if _, err := h.Scale(ctx,
		es.Namespace,
		targetGVK,
		spec.ScaleTargetRef.Name,
		spec.MinTargetReplicas,
	); err != nil {
		return fmt.Errorf("failed to scale target from zero: %w", err)
	}

	return nil
}

func (h *ScaleHandler) createScalerForTrigger(trigger *v1alpha1.ScaleTrigger, cooldownPeriod time.Duration) (scalers.Scaler, error) {
	var scaler scalers.Scaler
	var err error

	switch trigger.Type {
	case "prometheus":
		scaler, err = scalers.NewPrometheusScaler(trigger.Metadata, cooldownPeriod)
	default:
		return nil, fmt.Errorf("unsupported trigger type: %s", trigger.Type)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create scaler: %w", err)
	}
	return scaler, nil
}

func (h *ScaleHandler) Scale(ctx context.Context,
	namespace string,
	targetGVK schema.GroupVersionKind,
	targetName string,
	desiredReplicas int32) (bool, error) {
	// Get mutex for the target
	mutex := h.getMutexForScale(namespace + "/" + targetGVK.Kind + "/" + targetName)
	mutex.Lock()
	defer mutex.Unlock()
	h.logger.Debug("Scaling", zap.String("kind", targetGVK.Kind), zap.String("namespace", namespace), zap.String("name", targetName), zap.Int32("desired replicas", desiredReplicas))

	// Get the scale object
	groupResource := schema.GroupResource{
		Group:    targetGVK.Group,
		Resource: k8shelper.KindToResource(targetGVK.Kind),
	}

	var err error
	var currentScale *autoscalingv1.Scale
	for i := 0; i < 2; i++ {
		currentScale, err = h.scaleClient.Scales(namespace).Get(ctx, groupResource, targetName, metav1.GetOptions{})
		if err == nil {
			break
		}
		if meta.IsNoMatchError(err) {
			h.logger.Info("retrying scale operation after resetting RESTMapper cache due to NoMatchError",
				zap.String("kind", targetGVK.Kind),
				zap.String("namespace", namespace),
				zap.String("name", targetName),
				zap.Error(err))
			h.restMapper.Reset()
			continue
		}
		break
	}

	if err != nil {
		h.createEvent(namespace, targetName, "Error", "FailedToScale", fmt.Sprintf("Failed to scale to %d replicas for %s/%s: %v", desiredReplicas, targetGVK.Kind, targetName, err))
		return false, fmt.Errorf("failed to get scale for %s/%s (%s): %w", targetGVK.Kind, targetName, namespace, err)
	}

	// Check if already at desired replicas
	if currentScale.Status.Replicas == desiredReplicas {
		h.logger.Info("No scale required. Target already at desired replicas",
			zap.String("kind", targetGVK.Kind),
			zap.String("namespace", namespace),
			zap.String("name", targetName),
			zap.Int32("replicas", desiredReplicas))
		return false, nil
	}

	// Check if already scaled beyond desired (for scale up operations)
	if desiredReplicas > 0 && currentScale.Status.Replicas > desiredReplicas {
		h.logger.Info("No scale required. Target already scaled beyond desired replicas",
			zap.String("kind", targetGVK.Kind),
			zap.String("namespace", namespace),
			zap.String("name", targetName),
			zap.Int32("current replicas", currentScale.Status.Replicas),
			zap.Int32("desired replicas", desiredReplicas))
		return false, nil
	}

	currentScale.Spec.Replicas = desiredReplicas
	if _, err := h.scaleClient.Scales(namespace).Update(ctx, groupResource, currentScale, metav1.UpdateOptions{}); err != nil {
		h.createEvent(namespace, targetName, "Warning", "FailedToScale", fmt.Sprintf("Failed to scale %d replicas for %s/%s: %v", desiredReplicas, targetGVK.Kind, targetName, err))
		return false, fmt.Errorf("failed to update scale for %s/%s (%s): %w", targetGVK.Kind, targetName, namespace, err)
	}

	h.createEvent(namespace, targetName, "Normal", "SuccessToScale", fmt.Sprintf("Successfully scaled %d replicas for %s/%s", desiredReplicas, targetGVK.Kind, targetName))
	h.logger.Info("Target scaled", zap.String("kind", targetGVK.Kind), zap.String("namespace", namespace), zap.String("name", targetName), zap.Int32("replicas", desiredReplicas))
	return true, nil
}

func (h *ScaleHandler) UpdateKedaScaledObjectPausedState(ctx context.Context, scaledObjectName, namespace string, paused bool) error {
	var patchBytes []byte
	if paused {
		// When pausing, set both annotations: paused=true and paused-replicas="0"
		patchBytes = []byte(fmt.Sprintf(`{"metadata": {"annotations": {"%s": "%s", "%s": "0"}}}`,
			kedaPausedAnnotation,
			strconv.FormatBool(paused),
			kedaPausedReplicasAnnotation))
	} else {
		// When unpausing, set paused=false and remove the paused-replicas annotation
		patchBytes = []byte(fmt.Sprintf(`{"metadata": {"annotations": {"%s": "%s", "%s": null}}}`,
			kedaPausedAnnotation,
			strconv.FormatBool(paused),
			kedaPausedReplicasAnnotation))
	}

	_, err := h.kDynamicClient.Resource(values.ScaledObjectGVR).Namespace(namespace).Patch(
		ctx,
		scaledObjectName,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		return fmt.Errorf("failed to patch ScaledObject: %w", err)
	}
	return nil
}

func (h *ScaleHandler) UpdateLastScaledUpTime(ctx context.Context, crdName, namespace string) error {
	h.logger.Debug("Updating LastScaledUpTime", zap.String("service", crdName), zap.String("namespace", namespace))
	now := metav1.Now()
	patchBytes := []byte(fmt.Sprintf(`{"status": {"lastScaledUpTime": "%s"}}`, now.Format(time.RFC3339Nano)))

	_, err := h.kDynamicClient.Resource(values.ElastiServiceGVR).
		Namespace(namespace).
		Patch(ctx, crdName, types.MergePatchType, patchBytes, metav1.PatchOptions{}, "status")
	if err != nil {
		return fmt.Errorf("failed to patch ElastiService status: %w", err)
	}
	return nil
}

// createEvent creates a new event on scaling up or down
func (h *ScaleHandler) createEvent(namespace, name, eventType, reason, message string) {
	h.logger.Info("createEvent", zap.String("eventType", eventType), zap.String("reason", reason), zap.String("message", message))
	ref := &v1.ObjectReference{
		APIVersion: "elasti.truefoundry.com/v1alpha1",
		Kind:       "ElastiService",
		Name:       name,
		Namespace:  namespace,
	}
	h.EventRecorder.Event(ref, eventType, reason, message)
}
