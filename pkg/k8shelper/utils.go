package k8shelper

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func UnstructuredToResource(obj interface{}, resource interface{}) error {
	unstructuredObj := obj.(*unstructured.Unstructured)
	err := runtime.DefaultUnstructuredConverter.FromUnstructured(unstructuredObj.UnstructuredContent(), resource)
	if err != nil {
		return fmt.Errorf("UnstructuredToResource: %w", err)
	}
	return nil
}

// Kind is singular and might be in camelCase
// Resource is plural and is in smallcase
// Since CRD accept kind, it needs to be converted
func KindToResource(kind string) string {
	k := strings.ToLower(kind)
	// tolerate legacy already-plural inputs
	if strings.HasSuffix(k, "s") {
		return k
	}
	return k + "s"
}

// APIVersionStrToGVK converts an API version string to a GroupVersionKind
func APIVersionStrToGVK(apiVersion string, kind string) (schema.GroupVersionKind, error) {
	gv, err := schema.ParseGroupVersion(apiVersion)
	if err != nil {
		return schema.GroupVersionKind{}, fmt.Errorf("failed to parse API version: %w", err)
	}
	return schema.GroupVersionKind{
		Group:   gv.Group,
		Version: gv.Version,
		Kind:    kind,
	}, nil
}
