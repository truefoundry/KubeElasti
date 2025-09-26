package controller

import (
	"context"
	"fmt"
	"sync"

	"github.com/truefoundry/elasti/pkg/config"
	"github.com/truefoundry/elasti/pkg/values"
	"go.uber.org/zap"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"truefoundry/elasti/operator/api/v1alpha1"
	"truefoundry/elasti/operator/internal/informer"
	"truefoundry/elasti/operator/internal/prom"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	runtime "k8s.io/apimachinery/pkg/runtime"
)

const (
	// Prefix is the name of the NamespacedName string for CRD
	lockKeyPostfixForPublicSVC = "public-service"
	lockKeyPostfixForTargetRef = "scale-target-ref"
)

func (r *ElastiServiceReconciler) getMutexForInformerStart(key string) *sync.Once {
	l, _ := r.InformerStartLocks.LoadOrStore(key, &sync.Once{})
	return l.(*sync.Once)
}

func (r *ElastiServiceReconciler) resetMutexForInformer(key string) {
	r.InformerStartLocks.Delete(key)
}

func (r *ElastiServiceReconciler) getMutexKeyForPublicSVC(req ctrl.Request) string {
	return req.String() + lockKeyPostfixForPublicSVC
}

func (r *ElastiServiceReconciler) getMutexKeyForTargetRef(req ctrl.Request) string {
	return req.String() + lockKeyPostfixForTargetRef
}
func (r *ElastiServiceReconciler) getResolverChangeHandler(ctx context.Context) cache.ResourceEventHandlerFuncs {
	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			err := r.handleResolverChanges(ctx, obj)
			if err != nil {
				r.Logger.Error("Failed to handle resolver changes", zap.Error(err))
			}
		},
		UpdateFunc: func(_, newObj interface{}) {
			err := r.handleResolverChanges(ctx, newObj)
			if err != nil {
				r.Logger.Error("Failed to handle resolver changes", zap.Error(err))
			}
		},
		DeleteFunc: func(_ interface{}) {
			// TODO: Handle deletion of resolver deployment
			// We can do two things here
			// 1. We can move to the serve mode
			// 2. We can add a finalizer to the deployent to avoid deletion
			//
			//
			// Another situation is, if the resolver has some issues, and is restarting.
			// In that case, we can wait for the resolver to come back up, and in the meanwhile, we can move to the serve mode
			r.Logger.Warn("Resolver deployment deleted", zap.String("deployment_name", config.GetResolverConfig().DeploymentName))
		},
	}
}

func (r *ElastiServiceReconciler) getPublicServiceChangeHandler(ctx context.Context, es *v1alpha1.ElastiService, req ctrl.Request) cache.ResourceEventHandlerFuncs {
	key := r.InformerManager.GetKey(informer.KeyParams{
		Namespace:    config.GetResolverConfig().Namespace,
		CRDName:      req.Name,
		ResourceName: es.Spec.Service,
		ResourceType: values.ResourceService,
	})

	return cache.ResourceEventHandlerFuncs{
		AddFunc: func(obj interface{}) {
			errStr := values.Success
			err := r.handlePublicServiceChanges(ctx, obj, es.Spec.Service, req.Namespace)
			if err != nil {
				errStr = err.Error()
				r.Logger.Error("Failed to handle public service changes", zap.Error(err))
			} else {
				r.Logger.Info("Public service added", zap.String("service", es.Spec.Service), zap.String("es", req.String()))
			}
			prom.InformerHandlerCounter.WithLabelValues(req.String(), key, errStr).Inc()
		},
		UpdateFunc: func(_, newObj interface{}) {
			errStr := values.Success
			err := r.handlePublicServiceChanges(ctx, newObj, es.Spec.Service, req.Namespace)
			if err != nil {
				errStr = err.Error()
				r.Logger.Error("Failed to handle public service changes", zap.Error(err))
			} else {
				r.Logger.Info("Public service updated", zap.String("service", es.Spec.Service), zap.String("es", req.String()))
			}
			prom.InformerHandlerCounter.WithLabelValues(req.String(), key, errStr).Inc()
		},
		DeleteFunc: func(_ interface{}) {
			r.Logger.Debug("public deployment deleted",
				zap.String("es", req.String()),
				zap.String("service", es.Spec.Service))
		},
	}
}

func (r *ElastiServiceReconciler) getScaleTargetRefChangeHandler(ctx context.Context, es *v1alpha1.ElastiService, req ctrl.Request) cache.ResourceEventHandlerFuncs {
	key := r.InformerManager.GetKey(informer.KeyParams{
		Namespace:    req.Namespace,
		CRDName:      req.Name,
		ResourceName: es.Spec.ScaleTargetRef.Name,
		ResourceType: es.Spec.ScaleTargetRef.Kind,
	})
	return cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, obj interface{}) {
			errStr := values.Success
			err := r.handleScaleTargetRefChanges(ctx, es, req, obj)
			if err != nil {
				errStr = err.Error()
				r.Logger.Error("Failed to handle ScaleTargetRef changes", zap.Error(err))
			} else {
				r.Logger.Info("ScaleTargetRef updated", zap.String("es", req.String()), zap.String("scaleTargetRef", es.Spec.ScaleTargetRef.Name))
			}

			prom.InformerHandlerCounter.WithLabelValues(req.String(), key, errStr).Inc()
		},
	}
}

func (r *ElastiServiceReconciler) handleScaleTargetRefChanges(ctx context.Context, es *v1alpha1.ElastiService, req ctrl.Request, obj interface{}) error {
	r.Logger.Info("ScaleTargetRef changes detected", zap.String("es", req.String()), zap.Any("scaleTargetRef", es.Spec.ScaleTargetRef))

	// Gather target object info (replicas, selector) from the updated resource
	updatedObj, err := r.getUpdateObjInfo(ctx, obj, req)
	if err != nil {
		return fmt.Errorf("failed to get scale for target resource: %w", err)
	}

	// Extract replica information from the resource
	ready, err := r.isTargetReady(ctx, updatedObj)
	if err != nil {
		return fmt.Errorf("failed to get target ready status: %w", err)
	}

	// Determine mode based on replica status
	// If NOT ready, switch to proxy mode
	// else, switch to serve mode
	if !ready {
		r.Logger.Info("ScaleTargetRef has 0 replicas or not ready, switching to proxy mode",
			zap.String("kind", es.Spec.ScaleTargetRef.Kind),
			zap.String("name", es.Spec.ScaleTargetRef.Name),
			zap.String("es", req.String()))
		if err := r.switchMode(ctx, req, values.ProxyMode); err != nil {
			return fmt.Errorf("failed to switch to proxy mode: %w", err)
		}
	} else {
		r.Logger.Info("ScaleTargetRef has ready replicas and is healthy, switching to serve mode",
			zap.String("kind", es.Spec.ScaleTargetRef.Kind),
			zap.String("name", es.Spec.ScaleTargetRef.Name),
			zap.String("es", req.String()))
		if err := r.switchMode(ctx, req, values.ServeMode); err != nil {
			return fmt.Errorf("failed to switch to serve mode: %w", err)
		}
	}

	return nil
}

type updateObjInfo struct {
	specReplicas   int64
	statusReplicas int64
	selector       map[string]interface{}
	namespace      string
	name           string
}

func (r *ElastiServiceReconciler) getUpdateObjInfo(_ context.Context, obj interface{}, req ctrl.Request) (*updateObjInfo, error) {
	// Convert to unstructured to work with any resource type
	objInfo, ok := obj.(*unstructured.Unstructured)
	if !ok {
		r.Logger.Error("Failed to convert ScaleTargetRef to unstructured")
		return nil, fmt.Errorf("failed to convert ScaleTargetRef to unstructured")
	}

	scaleObj := &updateObjInfo{
		specReplicas:   0,
		statusReplicas: 0,
		selector:       nil,
		namespace:      req.Namespace,
		name:           req.Name,
	}

	specReplicasVal, found, err := unstructured.NestedInt64(objInfo.Object, "spec", "replicas")
	if err != nil {
		return nil, fmt.Errorf("failed to get replicas from spec, %w", err)
	} else if !found {
		// If replicas are not found and no error, we can assume the resource is not ready
		r.Logger.Warn("spec replicas not found")
		return scaleObj, nil
	}
	scaleObj.specReplicas = specReplicasVal

	statusReplicasVal, found, err := unstructured.NestedInt64(objInfo.Object, "status", "replicas")
	if err != nil {
		return nil, fmt.Errorf("failed to get replicas from spec, %w", err)
	} else if !found {
		// If replicas are not found and no error, we can assume the resource is not ready
		r.Logger.Warn("status replicas not found")
		return scaleObj, nil
	}
	scaleObj.statusReplicas = statusReplicasVal

	selectorVal, found, err := unstructured.NestedMap(objInfo.Object, "spec", "selector")
	if err != nil {
		return nil, fmt.Errorf("failed to get selector from spec, %w", err)
	} else if !found {
		// If selector is not found and no error, we can assume the resource is not ready
		r.Logger.Warn("selector not found")
		return scaleObj, nil
	}
	scaleObj.selector = selectorVal

	return scaleObj, nil
}

// isTargetReady to check if the target resource has 1 running pod
func (r *ElastiServiceReconciler) isTargetReady(ctx context.Context, obj *updateObjInfo) (bool, error) {
	if obj.specReplicas <= 0 || obj.statusReplicas <= 0 {
		return false, nil
	} else if obj.selector == nil {
		r.Logger.Warn("selector is empty", zap.Any("obj", obj))
		return false, nil
	}

	labelSelector := &metav1.LabelSelector{}
	if err := runtime.DefaultUnstructuredConverter.FromUnstructured(obj.selector, labelSelector); err != nil {
		return false, fmt.Errorf("failed to convert selector map to LabelSelector, %w", err)
	}

	// Get ready replicas of a pod using selector and namespace
	selector, err := metav1.LabelSelectorAsSelector(labelSelector)
	if err != nil {
		return false, fmt.Errorf("failed to convert label selector to selector, %w", err)
	}

	podList := &corev1.PodList{}
	listOptions := []client.ListOption{
		client.InNamespace(obj.namespace),
		client.MatchingLabelsSelector{Selector: selector},
	}

	if err := r.List(ctx, podList, listOptions...); err != nil {
		return false, fmt.Errorf("failed to list pods for label selector, %w", err)
	}

	// Default to healthy unless we have specific health checks
	ready := false
	for _, pod := range podList.Items {
		// Skip terminating pods
		if pod.DeletionTimestamp != nil {
			continue
		}

		for _, condition := range pod.Status.Conditions {
			if condition.Type == corev1.PodReady && condition.Status == corev1.ConditionTrue {
				ready = true
				break
			}
		}

		if ready {
			break
		}
	}

	return ready, nil
}
