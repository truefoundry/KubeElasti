package scaling

import (
	"reflect"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestListNamespaces(t *testing.T) {
	tests := []struct {
		name            string
		watchNamespaces []string
		want            []string
	}{
		{
			name:            "empty watch set queries all namespaces once (cluster scope, unchanged default)",
			watchNamespaces: nil,
			want:            []string{metav1.NamespaceAll},
		},
		{
			name:            "single namespace is queried scoped, never all-namespace",
			watchNamespaces: []string{"team-a"},
			want:            []string{"team-a"},
		},
		{
			name:            "each configured namespace is queried once",
			watchNamespaces: []string{"team-a", "team-b", "elasti"},
			want:            []string{"team-a", "team-b", "elasti"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := listNamespaces(tt.watchNamespaces)
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("listNamespaces(%#v) = %#v, want %#v", tt.watchNamespaces, got, tt.want)
			}
			// In scoped mode the all-namespaces sentinel must never appear.
			if len(tt.watchNamespaces) > 0 {
				for _, ns := range got {
					if ns == metav1.NamespaceAll {
						t.Errorf("scoped mode issued an all-namespace query for %#v", tt.watchNamespaces)
					}
				}
			}
		})
	}
}
