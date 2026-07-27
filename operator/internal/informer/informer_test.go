package informer

import (
	"fmt"
	"testing"
	"time"

	"go.uber.org/zap"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/dynamic"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
	"k8s.io/client-go/tools/cache"
	ctrl "sigs.k8s.io/controller-runtime"
)

var deploymentGVR = schema.GroupVersionResource{Group: "apps", Version: "v1", Resource: "deployments"}

func newTestDeployment(namespace, name string) *unstructured.Unstructured {
	return &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"namespace": namespace,
			"name":      name,
		},
	}}
}

func newTestDynamicClient(objects ...runtime.Object) *dynamicfake.FakeDynamicClient {
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(
		runtime.NewScheme(),
		map[schema.GroupVersionResource]string{deploymentGVR: "DeploymentList"},
		objects...,
	)
}

func newTestManager(dc dynamic.Interface) *Manager {
	return &Manager{
		dynamicClient: dc,
		logger:        zap.NewNop(),
		resyncPeriod:  time.Minute,
		syncTimeout:   2 * time.Second,
	}
}

func newTestRequestWatch(namespace, name string) *RequestWatch {
	return &RequestWatch{
		Req:                  ctrl.Request{NamespacedName: types.NamespacedName{Namespace: namespace, Name: "test-es"}},
		ResourceName:         name,
		ResourceNamespace:    namespace,
		GroupVersionResource: &deploymentGVR,
		Handlers:             cache.ResourceEventHandlerFuncs{},
	}
}

func TestAddStartsAndSyncsInformer(t *testing.T) {
	dc := newTestDynamicClient(newTestDeployment("demo", "target"))
	m := newTestManager(dc)
	req := newTestRequestWatch("demo", "target")
	key := m.getKeyFromRequestWatch(req)
	defer func() { _ = m.StopInformer(key) }()

	if err := m.Add(req); err != nil {
		t.Fatalf("Add failed: %v", err)
	}
	value, ok := m.informers.Load(key)
	if !ok {
		t.Fatal("informer not stored after Add")
	}
	if !value.(info).Informer.HasSynced() {
		t.Fatal("informer not synced after Add returned")
	}
}

func TestAddFailsWhenTargetMissing(t *testing.T) {
	m := newTestManager(newTestDynamicClient())
	req := newTestRequestWatch("demo", "missing")

	if err := m.Add(req); err == nil {
		t.Fatal("expected Add to fail for missing target")
	}
	if _, ok := m.informers.Load(m.getKeyFromRequestWatch(req)); ok {
		t.Fatal("informer stored despite missing target")
	}
}

func TestAddCleansUpOnSyncTimeout(t *testing.T) {
	dc := newTestDynamicClient(newTestDeployment("demo", "target"))
	dc.PrependReactor("list", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, fmt.Errorf("simulated list failure")
	})
	m := newTestManager(dc)
	m.syncTimeout = 300 * time.Millisecond
	req := newTestRequestWatch("demo", "target")
	key := m.getKeyFromRequestWatch(req)

	if err := m.Add(req); err == nil {
		t.Fatal("expected Add to fail on sync timeout")
	}
	if _, ok := m.informers.Load(key); ok {
		t.Fatal("unsynced informer left in map after sync timeout")
	}
}
