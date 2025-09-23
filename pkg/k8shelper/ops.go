package k8shelper

import (
	"context"
	"fmt"

	"github.com/truefoundry/elasti/pkg/logger"
	"go.uber.org/zap"
	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// Ops help you do various operations in your kubernetes cluster
type Ops struct {
	kClient        *kubernetes.Clientset
	kDynamicClient *dynamic.DynamicClient
	logger         *zap.Logger
}

// NewOps create a new instance for the k8s Operations
func NewOps(logger *zap.Logger, config *rest.Config) *Ops {
	kClient, err := kubernetes.NewForConfig(config)
	if err != nil {
		logger.Fatal("Error connecting with kubernetes", zap.Error(err))
	}
	kDynamicClient, err := dynamic.NewForConfig(config)
	if err != nil {
		logger.Fatal("Error connecting with kubernetes", zap.Error(err))
	}
	return &Ops{
		logger:         logger.Named("k8sOps"),
		kClient:        kClient,
		kDynamicClient: kDynamicClient,
	}
}

// CheckIfServiceEndpointSliceActive returns true if endpoint for a service is active
func (k *Ops) CheckIfServiceEndpointSliceActive(ns, svc string) (bool, error) {
	endpointSlices, err := k.kClient.DiscoveryV1().EndpointSlices(ns).List(context.TODO(), metav1.ListOptions{
		LabelSelector: discoveryv1.LabelServiceName + "=" + svc,
	})
	if err != nil {
		return false, fmt.Errorf("CheckIfServiceEndpointSliceActive - GET: %w", err)
	}

	for _, slice := range endpointSlices.Items {
		for _, endpoint := range slice.Endpoints {
			// According to K8s docs: "ready" should be marked if endpoint is serving and not terminating
			// So checking ready alone should be sufficient for most use cases
			// nil should be interpreted as "true"
			if endpoint.Conditions.Ready == nil || *endpoint.Conditions.Ready {
				k.logger.Debug("Service endpoint is active", zap.String("service", logger.MaskMiddle(svc, 3, 3)), zap.String("namespace", logger.MaskMiddle(ns, 3, 3)))
				return true, nil
			}
		}
	}

	return false, nil
}
