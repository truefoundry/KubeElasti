package controller

import "testing"

func TestNamespaceInScope(t *testing.T) {
	tests := []struct {
		name            string
		watchNamespaces []string
		namespace       string
		want            bool
	}{
		{name: "cluster scope allows any namespace", watchNamespaces: nil, namespace: "anything", want: true},
		{name: "watched namespace is in scope", watchNamespaces: []string{"team-a", "team-b"}, namespace: "team-a", want: true},
		{name: "unwatched namespace is ignored", watchNamespaces: []string{"team-a", "team-b"}, namespace: "team-c", want: false},
		{name: "release namespace in scope", watchNamespaces: []string{"team-a", "elasti"}, namespace: "elasti", want: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := namespaceInScope(tt.watchNamespaces, tt.namespace); got != tt.want {
				t.Errorf("namespaceInScope(%#v, %q) = %v, want %v", tt.watchNamespaces, tt.namespace, got, tt.want)
			}
		})
	}
}
