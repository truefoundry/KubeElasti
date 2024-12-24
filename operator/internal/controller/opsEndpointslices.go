package controller

import (
	"context"
	"fmt"

	"github.com/truefoundry/elasti/pkg/utils"
	"go.uber.org/zap"
	v1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/discovery/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func (r *ElastiServiceReconciler) getIPsForResolver(ctx context.Context) ([]string, error) {
	resolverSlices := &networkingv1.EndpointSliceList{}
	if err := r.List(ctx, resolverSlices, client.MatchingLabels{
		"kubernetes.io/service-name": resolverServiceName,
	}); err != nil {
		r.Logger.Error("Failed to get Resolver endpoint slices", zap.Error(err))
		return nil, fmt.Errorf("getIPsForResolver: %w", err)
	}
	var resolverPodIPs []string
	for _, endpointSlice := range resolverSlices.Items {
		for _, endpoint := range endpointSlice.Endpoints {
			resolverPodIPs = append(resolverPodIPs, endpoint.Addresses...)
		}
	}
	if len(resolverPodIPs) == 0 {
		return nil, ErrNoResolverPodFound
	}
	return resolverPodIPs, nil
}

func (r *ElastiServiceReconciler) deleteEndpointsliceToResolver(ctx context.Context, serviceNamespacedName types.NamespacedName) error {
	endpointSlice := &networkingv1.EndpointSlice{}
	serviceNamespacedName.Name = utils.GetEndpointSliceToResolverName(serviceNamespacedName.Name)
	if err := r.Get(ctx, serviceNamespacedName, endpointSlice); err != nil && !errors.IsNotFound(err) {
		r.Logger.Error("Failed to get endpoint slice", zap.String("service", serviceNamespacedName.String()), zap.Error(err))
		return fmt.Errorf("failed to get endpointslice: %w", err)
	} else if errors.IsNotFound(err) {
		return nil
	}

	if err := r.Delete(ctx, endpointSlice); err != nil {
		return fmt.Errorf("failed to delete endpointslice: %w", err)
	}
	return nil
}

func (r *ElastiServiceReconciler) createOrUpdateEndpointsliceToResolver(ctx context.Context, service *v1.Service) error {
	resolverPodIPs, err := r.getIPsForResolver(ctx)
	if err != nil {
		r.Logger.Error("Failed to get IPs for Resolver", zap.String("service", service.Name), zap.Error(err))
		return err
	}

	// NOTE: Suggestion is to give it a random name in end, to avoid any conflicts, which is rare, but possible.
	// In case of random name, we need to store the name in CRD. Right now, we provide a deterministic hashed name.
	newEndpointsliceToResolverName := utils.GetEndpointSliceToResolverName(service.Name)
	EndpointsliceNamespacedName := types.NamespacedName{
		Name:      newEndpointsliceToResolverName,
		Namespace: service.Namespace,
	}

	isResolverSliceFound := false
	sliceToResolver := &networkingv1.EndpointSlice{}
	if err := r.Get(ctx, EndpointsliceNamespacedName, sliceToResolver); err != nil && !errors.IsNotFound(err) {
		r.Logger.Debug("Error getting a endpoint slice to Resolver", zap.String("endpointslice", EndpointsliceNamespacedName.String()), zap.Error(err))
		return fmt.Errorf("createOrUpdateEndpointsliceToResolver: %w", err)
	} else if errors.IsNotFound(err) {
		// TODO: This can be handled better
		// This is a similar case as seen in resolver informer
		// We can handler this with the same logic as that
		isResolverSliceFound = false
		r.Logger.Debug("EndpointSlice not found, will try creating one", zap.String("endpointslice", EndpointsliceNamespacedName.String()))
	} else {
		isResolverSliceFound = true
		r.Logger.Debug("EndpointSlice Found", zap.String("endpointslice", EndpointsliceNamespacedName.String()))
	}

	newEndpointSlice := &networkingv1.EndpointSlice{
		ObjectMeta: metav1.ObjectMeta{
			Name:      newEndpointsliceToResolverName,
			Namespace: service.Namespace,
			Labels: map[string]string{
				"kubernetes.io/service-name": service.Name,
			},
		},
		AddressType: networkingv1.AddressTypeIPv4,
		Ports: []networkingv1.EndpointPort{
			{
				Name:     ptr.To(service.Spec.Ports[0].Name),
				Protocol: ptr.To(v1.ProtocolTCP),
				// Make this dynamic too
				Port: ptr.To(int32(resolverPort)),
			},
		},
	}

	// sliceToResolver.DeepCopy()

	for _, ip := range resolverPodIPs {
		newEndpointSlice.Endpoints = append(newEndpointSlice.Endpoints, networkingv1.Endpoint{
			Addresses: []string{ip},
		})
	}

	if isResolverSliceFound {
		if err := r.Update(ctx, newEndpointSlice); err != nil {
			r.Logger.Error("failed to update sliceToResolver", zap.String("endpointslice", EndpointsliceNamespacedName.String()), zap.Error(err))
			return fmt.Errorf("createOrUpdateEndpointsliceToResolver: %w", err)
		}
		r.Logger.Info("EndpointSlice updated successfully", zap.String("endpointslice", EndpointsliceNamespacedName.String()))
	} else {
		// TODOS: Make sure the private service is owned by the ElastiService
		if err := r.Create(ctx, newEndpointSlice); err != nil {
			r.Logger.Error("failed to create sliceToResolver", zap.String("endpointslice", EndpointsliceNamespacedName.String()), zap.Error(err))
			return fmt.Errorf("createOrUpdateEndpointsliceToResolver: %w", err)
		}
		r.Logger.Info("EndpointSlice created successfully", zap.String("endpointslice", EndpointsliceNamespacedName.String()))
	}

	return nil
}
