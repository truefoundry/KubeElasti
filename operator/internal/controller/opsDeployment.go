package controller

import (
	"context"
	"fmt"
	"strings"
	"truefoundry/elasti/operator/internal/crddirectory"

	"github.com/truefoundry/elasti/pkg/config"
	"github.com/truefoundry/elasti/pkg/k8shelper"
	"github.com/truefoundry/elasti/pkg/values"
	"go.uber.org/zap"
	appsv1 "k8s.io/api/apps/v1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
)

func (r *ElastiServiceReconciler) handleResolverChanges(ctx context.Context, obj interface{}) error {
	resolverDeployment := &appsv1.Deployment{}
	err := k8shelper.UnstructuredToResource(obj, resolverDeployment)
	if err != nil {
		return fmt.Errorf("failed to convert unstructured to deployment: %w", err)
	}
	if resolverDeployment.Name != config.GetResolverConfig().DeploymentName {
		return nil
	}

	crddirectory.CRDDirectory.Services.Range(func(key, value interface{}) bool {
		crdDetails := value.(*crddirectory.CRDDetails)
		if crdDetails.Status.Mode != values.ProxyMode {
			return true
		}

		// Extract namespace and service name from the key
		keyStr := key.(string)
		parts := strings.Split(keyStr, "/")
		if len(parts) != 2 {
			r.Logger.Error("Invalid key format", zap.String("key", keyStr))
			return true
		}
		namespacedName := types.NamespacedName{
			Namespace: parts[0],
			Name:      parts[1],
		}

		targetService := &v1.Service{}
		if err := r.Get(ctx, namespacedName, targetService); err != nil {
			r.Logger.Warn("Failed to get service to update EndpointSlice", zap.Error(err))
			return true
		}

		if err := r.createOrUpdateEndpointsliceToResolver(ctx, targetService); err != nil {
			r.Logger.Error("Failed to update EndpointSlice",
				zap.String("service", crdDetails.CRDName),
				zap.Error(err))
		}
		return true
	})

	return nil
}
