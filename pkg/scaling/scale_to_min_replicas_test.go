package scaling

import (
	"context"
	"fmt"
	"sync"
	"testing"

	"github.com/kubeelasti/kubeelasti/operator/api/v1alpha1"

	"go.uber.org/zap"
	autoscalingv1 "k8s.io/api/autoscaling/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/scale"
	"k8s.io/client-go/tools/record"
)

type fakeScaleClient struct {
	mu     sync.Mutex
	scales map[string]*autoscalingv1.Scale
}

func newFakeScaleClient(initial ...*autoscalingv1.Scale) *fakeScaleClient {
	f := &fakeScaleClient{scales: make(map[string]*autoscalingv1.Scale)}
	for _, s := range initial {
		f.scales[scaleKey(s.Namespace, "deployments", s.Name)] = s.DeepCopy()
	}
	return f
}

func scaleKey(namespace, resource, name string) string {
	return namespace + "/" + resource + "/" + name
}

func (f *fakeScaleClient) Scales(namespace string) scale.ScaleInterface {
	return &fakeNamespacedScale{parent: f, namespace: namespace}
}

type fakeNamespacedScale struct {
	parent    *fakeScaleClient
	namespace string
}

func (f *fakeNamespacedScale) Get(_ context.Context, resource schema.GroupResource, name string, _ metav1.GetOptions) (*autoscalingv1.Scale, error) {
	f.parent.mu.Lock()
	defer f.parent.mu.Unlock()
	s, ok := f.parent.scales[scaleKey(f.namespace, resource.Resource, name)]
	if !ok {
		return nil, apierrors.NewNotFound(resource, name)
	}
	return s.DeepCopy(), nil
}

func (f *fakeNamespacedScale) Update(_ context.Context, resource schema.GroupResource, scaleObj *autoscalingv1.Scale, _ metav1.UpdateOptions) (*autoscalingv1.Scale, error) {
	f.parent.mu.Lock()
	defer f.parent.mu.Unlock()
	key := scaleKey(f.namespace, resource.Resource, scaleObj.Name)
	if _, ok := f.parent.scales[key]; !ok {
		return nil, apierrors.NewNotFound(resource, scaleObj.Name)
	}
	updated := scaleObj.DeepCopy()
	updated.Status.Replicas = scaleObj.Spec.Replicas
	f.parent.scales[key] = updated
	return updated.DeepCopy(), nil
}

func (f *fakeNamespacedScale) Patch(context.Context, schema.GroupVersionResource, string, types.PatchType, []byte, metav1.PatchOptions) (*autoscalingv1.Scale, error) {
	return nil, fmt.Errorf("patch not implemented")
}

func (f *fakeScaleClient) get(namespace, resource, name string) *autoscalingv1.Scale {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := f.scales[scaleKey(namespace, resource, name)]
	if s == nil {
		return nil
	}
	return s.DeepCopy()
}

func newTestScaleHandler(scaleClient scale.ScalesGetter) *ScaleHandler {
	return &ScaleHandler{
		logger:        zap.NewNop(),
		scaleClient:   scaleClient,
		EventRecorder: record.NewFakeRecorder(16),
	}
}

func TestScaleToMinReplicas(t *testing.T) {
	tests := []struct {
		name            string
		currentReplicas int32
		minTarget       int32
		kind            string
		apiVersion      string
		missingTarget   bool
		wantScaled      bool
		wantReplicas    int32
		wantErr         bool
	}{
		{
			name:            "scales from zero to MinTargetReplicas",
			currentReplicas: 0,
			minTarget:       2,
			kind:            "Deployment",
			apiVersion:      "apps/v1",
			wantScaled:      true,
			wantReplicas:    2,
		},
		{
			name:            "defaults MinTargetReplicas below 1 to 1",
			currentReplicas: 0,
			minTarget:       0,
			kind:            "Deployment",
			apiVersion:      "apps/v1",
			wantScaled:      true,
			wantReplicas:    1,
		},
		{
			name:            "normalizes legacy deployments kind",
			currentReplicas: 0,
			minTarget:       1,
			kind:            "deployments",
			apiVersion:      "apps/v1",
			wantScaled:      true,
			wantReplicas:    1,
		},
		{
			name:            "no-op when already at desired replicas",
			currentReplicas: 1,
			minTarget:       1,
			kind:            "Deployment",
			apiVersion:      "apps/v1",
			wantScaled:      false,
			wantReplicas:    1,
		},
		{
			name:            "no-op when already beyond desired replicas",
			currentReplicas: 3,
			minTarget:       1,
			kind:            "Deployment",
			apiVersion:      "apps/v1",
			wantScaled:      false,
			wantReplicas:    3,
		},
		{
			name:       "invalid API version returns error",
			minTarget:  1,
			kind:       "Deployment",
			apiVersion: "a/b/c",
			wantErr:    true,
		},
		{
			name:          "missing target returns error",
			minTarget:     1,
			kind:          "Deployment",
			apiVersion:    "apps/v1",
			missingTarget: true,
			wantErr:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			const (
				namespace = "default"
				name      = "target"
			)

			client := newFakeScaleClient()
			if !tt.missingTarget {
				client = newFakeScaleClient(&autoscalingv1.Scale{
					ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
					Spec:       autoscalingv1.ScaleSpec{Replicas: tt.currentReplicas},
					Status:     autoscalingv1.ScaleStatus{Replicas: tt.currentReplicas},
				})
			}

			h := newTestScaleHandler(client)
			scaled, err := h.ScaleToMinReplicas(context.Background(), namespace, v1alpha1.ElastiServiceSpec{
				MinTargetReplicas: tt.minTarget,
				ScaleTargetRef: v1alpha1.ScaleTargetRef{
					APIVersion: tt.apiVersion,
					Kind:       tt.kind,
					Name:       name,
				},
			})

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil (scaled=%v)", scaled)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if scaled != tt.wantScaled {
				t.Fatalf("scaled = %v, want %v", scaled, tt.wantScaled)
			}

			got := client.get(namespace, "deployments", name)
			if got == nil {
				t.Fatal("expected scale object to exist")
			}
			if got.Spec.Replicas != tt.wantReplicas {
				t.Fatalf("spec.replicas = %d, want %d", got.Spec.Replicas, tt.wantReplicas)
			}
		})
	}
}
