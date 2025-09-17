package controller

import (
	"context"
	"sync"

	"github.com/truefoundry/elasti/pkg/values"
	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"

	"truefoundry/elasti/operator/api/v1alpha1"
	"truefoundry/elasti/operator/internal/informer"
	"truefoundry/elasti/operator/internal/prom"
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
			r.Logger.Warn("Resolver deployment deleted", zap.String("deployment_name", resolverDeploymentName))
		},
	}
}

func (r *ElastiServiceReconciler) getPublicServiceChangeHandler(ctx context.Context, es *v1alpha1.ElastiService, req ctrl.Request) cache.ResourceEventHandlerFuncs {
	key := r.InformerManager.GetKey(informer.KeyParams{
		Namespace:    resolverNamespace,
		CRDName:      req.Name,
		ResourceName: es.Spec.Service,
		Resource:     values.KindService,
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
		ResourceName: es.Spec.ScaleTargetRef.Kind,
		Resource:     es.Spec.ScaleTargetRef.Name,
	})
	return cache.ResourceEventHandlerFuncs{
		UpdateFunc: func(_, newObj interface{}) {
			errStr := values.Success
			err := r.handleScaleTargetRefChanges(ctx, newObj, es, req)
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

func (r *ElastiServiceReconciler) handleScaleTargetRefChanges(ctx context.Context, obj interface{}, es *v1alpha1.ElastiService, req ctrl.Request) error {
	r.Logger.Info("ScaleTargetRef changes detected", zap.String("es", req.String()), zap.Any("scaleTargetRef", es.Spec.ScaleTargetRef))
	// Convert to unstructured to work with any resource type
	unstructuredObj, ok := obj.(*unstructured.Unstructured)
	if !ok {
		r.Logger.Error("Failed to convert object to unstructured", zap.String("es", req.String()))
		return nil // Don't fail the reconciliation, just log and continue
	}

	if es.Spec.ScaleTargetRef.Kind == "deployments" {
		es.Spec.ScaleTargetRef.Kind = "Deployment"
	}

	// Extract replica information from the resource
	replicas, readyReplicas, isHealthy := r.extractReplicaInfo(unstructuredObj, es.Spec.ScaleTargetRef.Kind)

	r.Logger.Debug("Target resource status",
		zap.String("es", req.String()),
		zap.String("kind", es.Spec.ScaleTargetRef.Kind),
		zap.String("name", es.Spec.ScaleTargetRef.Name),
		zap.Int64("replicas", replicas),
		zap.Int64("readyReplicas", readyReplicas),
		zap.Bool("isHealthy", isHealthy))

	// Determine mode based on replica status
	if replicas == 0 || readyReplicas == 0 {
		r.Logger.Info("ScaleTargetRef has no ready replicas, switching to proxy mode",
			zap.String("kind", es.Spec.ScaleTargetRef.Kind),
			zap.String("name", es.Spec.ScaleTargetRef.Name),
			zap.String("es", req.String()))
		if err := r.switchMode(ctx, req, values.ProxyMode); err != nil {
			return err
		}
	} else if readyReplicas > 0 && isHealthy {
		r.Logger.Info("ScaleTargetRef has ready replicas and is healthy, switching to serve mode",
			zap.String("kind", es.Spec.ScaleTargetRef.Kind),
			zap.String("name", es.Spec.ScaleTargetRef.Name),
			zap.String("es", req.String()))
		if err := r.switchMode(ctx, req, values.ServeMode); err != nil {
			return err
		}
	}

	return nil
}

// extractReplicaInfo extracts replica information from any resource type
func (r *ElastiServiceReconciler) extractReplicaInfo(obj *unstructured.Unstructured, kind string) (replicas, readyReplicas int64, isHealthy bool) {
	// Default to healthy unless we have specific health checks
	isHealthy = true

	// Extract status from the unstructured object
	status, found, err := unstructured.NestedMap(obj.Object, "status")
	if !found || err != nil {
		r.Logger.Debug("No status found in target resource", zap.String("kind", kind), zap.Error(err))
		return 0, 0, false
	}

	// Extract replicas - try common field names
	if replicasVal, found, err := unstructured.NestedInt64(status, "replicas"); found && err == nil {
		replicas = replicasVal
	}

	// Extract ready replicas - try common field names
	if readyVal, found, err := unstructured.NestedInt64(status, "readyReplicas"); found && err == nil {
		readyReplicas = readyVal
	}

	// Handle specific resource types for health checks
	switch kind {
	case "Rollout":
		// For Argo Rollouts, check the phase
		if phase, found, err := unstructured.NestedString(status, "phase"); found && err == nil {
			isHealthy = (phase == values.ArgoPhaseHealthy)
		}
	case "Deployment":
		// For Deployments, we consider it healthy if readyReplicas > 0
		isHealthy = readyReplicas > 0
	case "StatefulSet":
		// For StatefulSets, check if readyReplicas matches replicas
		isHealthy = readyReplicas > 0
	default:
		// For other resource types, consider healthy if readyReplicas > 0
		isHealthy = readyReplicas > 0
	}

	return replicas, readyReplicas, isHealthy
}
