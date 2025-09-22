package k8shelper

import (
	"fmt"
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
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

// GetGroupFromAPIVersion returns the group from the API version
func GetGroupFromAPIVersion(apiVersion string) string {
	if !strings.Contains(apiVersion, "/") {
		// This is a core API group.
		return ""
	}
	return strings.Split(apiVersion, "/")[0]
}
