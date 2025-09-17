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

	"github.com/truefoundry/elasti/pkg/scaling/scalers"
	"github.com/truefoundry/elasti/pkg/values"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/meta"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/discovery"
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

	logger         *zap.Logger
	watchNamespace string
	restMapper     meta.RESTMapper
	cachedDisc     discovery.CachedDiscoveryInterface
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

	return &ScaleHandler{
		logger:         logger.Named("ScaleHandler"),
		kClient:        kClient,
		kDynamicClient: kDynamicClient,
		restMapper:     restMapper,
		cachedDisc:     cachedDisc,
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
		}

		if scaleDirection == NoScale {
			continue
		}

		switch scaleDirection {
		case ScaleDown:
			err := h.handleScaleToZero(ctx, cooldownPeriod, es)
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
			return NoScale, nil
		}
		if !healthy {
			h.logger.Warn("scaler is not healthy, skipping scale to zero", zap.String("namespace", es.Namespace), zap.String("service", es.Spec.Service))
			return "", fmt.Errorf("scaler: %s, cooldownPeriod: %s, is not healthy", trigger.Type, cooldownPeriod)
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

func (h *ScaleHandler) handleScaleToZero(ctx context.Context, cooldownPeriod time.Duration, es *v1alpha1.ElastiService) error {
	serviceNamespacedName := types.NamespacedName{
		Name:      es.Spec.Service,
		Namespace: es.Namespace,
	}

	// If the cooldown period is not met, we skip the scale down
	if es.Status.LastScaledUpTime != nil {
		if time.Since(es.Status.LastScaledUpTime.Time) < cooldownPeriod {
			h.logger.Debug("Skipping scale down as minimum cooldownPeriod not met",
				zap.String("service", serviceNamespacedName.String()),
				zap.Duration("cooldown", cooldownPeriod),
				zap.Time("last scaled up time", es.Status.LastScaledUpTime.Time))
			return nil
		}
	}

	// Pause the KEDA ScaledObject
	if es.Spec.Autoscaler != nil && strings.ToLower(es.Spec.Autoscaler.Type) == "keda" {
		err := h.UpdateKedaScaledObjectPausedState(ctx, es.Spec.Autoscaler.Name, es.Namespace, true)
		if err != nil {
			return fmt.Errorf("failed to update Keda ScaledObject for service %s: %w", serviceNamespacedName.String(), err)
		}
	}

	if err := h.ScaleTargetToZero(ctx, serviceNamespacedName, es.Spec.ScaleTargetRef.APIVersion, es.Spec.ScaleTargetRef.Kind, es.Spec.ScaleTargetRef.Name, es.Name); err != nil {
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
	serviceNamespacedName := types.NamespacedName{
		Name:      es.Spec.Service,
		Namespace: es.Namespace,
	}

	// We update the last scaled up time every time we evaluate that the trigger evaluates to scale-up. This means even if the scale-up is not successful, we update the last scaled up time to avoid the cooldown period increment
	if err := h.UpdateLastScaledUpTime(ctx, es.Name, es.Namespace); err != nil {
		h.logger.Error("Failed to update LastScaledUpTime", zap.Error(err), zap.String("namespacedName", serviceNamespacedName.String()))
	}

	// Unpause the KEDA ScaledObject if it's paused
	if es.Spec.Autoscaler != nil && strings.ToLower(es.Spec.Autoscaler.Type) == "keda" {
		err := h.UpdateKedaScaledObjectPausedState(ctx, es.Spec.Autoscaler.Name, es.Namespace, false)
		if err != nil {
			return fmt.Errorf("failed to update Keda ScaledObject for service %s: %w", serviceNamespacedName.String(), err)
		}
	}

	if err := h.ScaleTargetFromZero(ctx, serviceNamespacedName, es.Spec.ScaleTargetRef.APIVersion, es.Spec.ScaleTargetRef.Kind, es.Spec.ScaleTargetRef.Name, es.Spec.MinTargetReplicas, es.Name); err != nil {
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

// ScaleTargetFromZero scales the TargetRef to the provided replicas when it's at 0
func (h *ScaleHandler) ScaleTargetFromZero(ctx context.Context, serviceNamespacedName types.NamespacedName, targetAPIVersion string, targetKind, targetName string, replicas int32, elastiServiceName string) error {
	mutex := h.getMutexForScale(serviceNamespacedName.String())
	mutex.Lock()
	defer mutex.Unlock()

	h.logger.Info("Scaling up from zero", zap.String("kind", targetKind), zap.String("namespacedName", serviceNamespacedName.String()), zap.Int32("replicas", replicas))

	scaled, err := h.Scale(ctx, serviceNamespacedName.Namespace, targetAPIVersion, targetKind, targetName, replicas)

	if err != nil {
		h.createEvent(serviceNamespacedName.Namespace, elastiServiceName, "Warning", "ScaleFromZeroFailed", fmt.Sprintf("Failed to scale %s from zero to %d replicas: %v", targetKind, replicas, err))
		return fmt.Errorf("ScaleTargetFromZero - %s: %w", targetKind, err)
	}

	if !scaled {
		// Returning nil as the scale target is already scaled
		return nil
	}

	h.createEvent(serviceNamespacedName.Namespace, elastiServiceName, "Normal", "ScaledUpFromZero", fmt.Sprintf("Successfully scaled %s from zero to %d replicas", targetKind, replicas))

	return nil
}

// ScaleTargetToZero scales the target to zero
func (h *ScaleHandler) ScaleTargetToZero(ctx context.Context, serviceNamespacedName types.NamespacedName, targetAPIVersion string, targetKind string, targetName string, elastiServiceName string) error {
	mutex := h.getMutexForScale(serviceNamespacedName.String())
	mutex.Lock()
	defer mutex.Unlock()

	h.logger.Info("Scaling down to zero", zap.String("kind", targetKind), zap.String("namespacedName", serviceNamespacedName.String()))

	scaled, err := h.Scale(ctx, serviceNamespacedName.Namespace, targetAPIVersion, targetKind, targetName, 0)

	if err != nil {
		h.createEvent(serviceNamespacedName.Namespace, elastiServiceName, "Warning", "ScaleToZeroFailed", fmt.Sprintf("Failed to scale %s to zero: %v", targetKind, err))
		return fmt.Errorf("ScaleTargetToZero - %s: %w", targetKind, err)
	}

	if !scaled {
		// Returning nil as the scale target is already scaled down to zero
		return nil
	}

	h.createEvent(serviceNamespacedName.Namespace, elastiServiceName, "Normal", "ScaledDownToZero", fmt.Sprintf("Successfully scaled %s to zero", targetKind))

	return nil
}

func (h *ScaleHandler) Scale(ctx context.Context, namespace string, targetAPIVersion string, targetKind string, targetName string, replicas int32) (bool, error) {
	// Basic validation
	if h.kDynamicClient == nil {
		return false, fmt.Errorf("handler missing clients: kDynamicClient or kClient is nil")
	}

	// NOTE: Required for backwards compatibility, since so far, we have been using "deployments" instead of "Deployment" in exisiting
	// CRD files. Since calse doesn't recognize "deployments" as a valid kind, we need to convert it to "Deployment".
	// We can remove it once we have migrated all the existing CRD files to use "Deployment" instead of "deployments".
	if targetKind == "deployments" {
		targetKind = "Deployment"
	}

	h.logger.Debug("Scaling", zap.String("kind", targetKind), zap.String("name", targetName), zap.Int32("replicas", replicas))

	// Parse apiVersion into group/version
	var gv schema.GroupVersion
	if targetAPIVersion == "" {
		return false, fmt.Errorf("empty apiVersion")
	}
	// schema.ParseGroupVersion doesn't exist in apimachinery? Use manual splitting
	// Accept forms: "group/version" or "version" (core)
	gv, err := schema.ParseGroupVersion(targetAPIVersion)
	if err != nil {
		return false, fmt.Errorf("invalid apiVersion %q: %w", targetAPIVersion, err)
	}

	// Build GroupVersionKind from input
	gvk := schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    targetKind,
	}

	// Find RESTMapping (resources) for the GVK
	mapping, err := h.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		// try a fresh discovery refresh and retry once (deferred mapper caches)
		h.cachedDisc.Invalidate()
		h.restMapper = restmapper.NewDeferredDiscoveryRESTMapper(h.cachedDisc)
		mapping, err = h.restMapper.RESTMapping(gvk.GroupKind(), gvk.Version)
		if err != nil {
			return false, fmt.Errorf("failed to find resource mapping for %s: %w", gvk.String(), err)
		}
	}

	gvr := mapping.Resource // group-version-resource

	// Choose the resource interface: namespaced or not
	var ri dynamic.ResourceInterface
	if mapping.Scope.Name() == meta.RESTScopeNameNamespace {
		ri = h.kDynamicClient.Resource(gvr).Namespace(namespace)
	} else {
		ri = h.kDynamicClient.Resource(gvr)
	}

	// Get the scale subresource via dynamic client: GET .../scale
	scaleObj, err := ri.Get(ctx, targetName, metav1.GetOptions{}, "scale")
	if err != nil {
		return false, fmt.Errorf("failed to get scale subresource for %s/%s (%s): %w", gvr.String(), targetName, namespace, err)
	}

	// Extract current replicas from the scale subresource
	h.logger.Info("Scale object retrieved", zap.Any("scaleObj", scaleObj.Object))
	currentReplicas, found, err := unstructured.NestedInt64(scaleObj.Object, "spec", "replicas")
	h.logger.Info("currentReplicas result", zap.Int64("replicas", currentReplicas), zap.Bool("found", found), zap.Error(err))

	if err != nil {
		return false, fmt.Errorf("failed to get current replicas: %w", err)
	}
	if !found {
		return false, nil
	}

	h.logger.Debug("Target found", zap.String("kind", targetKind), zap.String("name", targetName), zap.Int64("current replicas", currentReplicas), zap.Int32("desired replicas", replicas))

	// Check if already at desired replicas
	if currentReplicas == int64(replicas) {
		h.logger.Info("Target already scaled", zap.String("kind", targetKind), zap.String("name", targetName), zap.Int32("replicas", replicas))
		return false, nil
	}

	// Check if already scaled beyond desired (for scale up operations)
	if replicas > 0 && currentReplicas > int64(replicas) {
		h.logger.Info("Target already scaled beyond desired replicas",
			zap.String("kind", targetKind),
			zap.String("name", targetName),
			zap.Int64("current replicas", currentReplicas),
			zap.Int32("desired replicas", replicas))
		return false, nil
	}

	// Update the replicas in the scale object
	err = unstructured.SetNestedField(scaleObj.Object, int64(replicas), "spec", "replicas")
	if err != nil {
		return false, fmt.Errorf("failed to set replicas: %w", err)
	}

	// Update the scale subresource
	_, err = ri.Update(ctx, scaleObj, metav1.UpdateOptions{}, "scale")
	if err != nil {
		return false, fmt.Errorf("failed to update scale: %w", err)
	}

	h.logger.Info("Target scaled", zap.String("kind", targetKind), zap.String("name", targetName), zap.Int32("replicas", replicas))
	return true, nil
}

// ScaleDeployment scales the deployment to the provided replicas
// TODO: use a generic logic to perform scaling similar to HPA/KEDA
func (h *ScaleHandler) ScaleDeployment(ctx context.Context, namespace, targetName string, replicas int32) (bool, error) {
	deploymentClient := h.kClient.AppsV1().Deployments(namespace)
	deploy, err := deploymentClient.Get(ctx, targetName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("ScaleDeployment - GET: %w", err)
	}

	h.logger.Debug("Deployment found", zap.String("deployment", targetName), zap.Int32("current replicas", *deploy.Spec.Replicas), zap.Int32("desired replicas", replicas))
	if deploy.Spec.Replicas == nil {
		return false, fmt.Errorf("ScaleDeployment - no replicas found for deployment %s", targetName)
	}
	if *deploy.Spec.Replicas == replicas {
		h.logger.Info("Deployment already scaled", zap.String("deployment", targetName), zap.Int32("current replicas", *deploy.Spec.Replicas))
		return false, nil
	}
	if replicas > 0 && *deploy.Spec.Replicas > replicas {
		h.logger.Info(
			"Deployment already scaled beyond desired replicas",
			zap.String("deployment", targetName),
			zap.Int32("current replicas", *deploy.Spec.Replicas),
			zap.Int32("desired replicas", replicas),
		)
		return false, nil
	}

	patchBytes := []byte(fmt.Sprintf(`{"spec":{"replicas":%d}}`, replicas))
	_, err = deploymentClient.Patch(ctx, targetName, types.StrategicMergePatchType, patchBytes, metav1.PatchOptions{})
	if err != nil {
		return false, fmt.Errorf("ScaleDeployment - Patch: %w", err)
	}
	h.logger.Info("Deployment scaled", zap.String("deployment", targetName), zap.Int32("replicas", replicas))
	return true, nil
}

// ScaleArgoRollout scales the rollout to the provided replicas
func (h *ScaleHandler) ScaleArgoRollout(ctx context.Context, namespace, targetName string, replicas int32) (bool, error) {
	rollout, err := h.kDynamicClient.Resource(values.RolloutGVR).Namespace(namespace).Get(ctx, targetName, metav1.GetOptions{})
	if err != nil {
		return false, fmt.Errorf("ScaleArgoRollout - GET: %w", err)
	}

	if rollout.Object["spec"] == nil || rollout.Object["spec"].(map[string]interface{})["replicas"] == nil {
		return false, fmt.Errorf("ScaleArgoRollout - no replicas found for rollout %s", targetName)
	}
	currentReplicas := rollout.Object["spec"].(map[string]interface{})["replicas"].(int64)
	h.logger.Info("Rollout found", zap.String("rollout", targetName), zap.Int64("current replicas", currentReplicas), zap.Int32("desired replicas", replicas))

	if currentReplicas == int64(replicas) {
		h.logger.Info("Rollout already scaled", zap.String("rollout", targetName), zap.Int64("current replicas", currentReplicas))
		return false, nil
	}

	if replicas > 0 && currentReplicas > int64(replicas) {
		h.logger.Info("Rollout already scaled beyond desired replicas", zap.String("rollout", targetName), zap.Int64("current replicas", currentReplicas), zap.Int32("desired replicas", replicas))
		return false, nil
	}

	patchBytes := []byte(fmt.Sprintf(`{"spec":{"replicas":%d}}`, replicas))
	_, err = h.kDynamicClient.Resource(values.RolloutGVR).Namespace(namespace).Patch(
		ctx,
		targetName,
		types.MergePatchType,
		patchBytes,
		metav1.PatchOptions{},
	)
	if err != nil {
		return false, fmt.Errorf("ScaleArgoRollout - Patch: %w", err)
	}
	h.logger.Info("Rollout scaled", zap.String("rollout", targetName), zap.Int32("replicas", replicas))
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
