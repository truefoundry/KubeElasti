package config

import (
	"reflect"
	"testing"
)

func TestGetWatchNamespaces(t *testing.T) {
	tests := []struct {
		name string
		env  string
		set  bool
		want []string
	}{
		{name: "unset returns nil (cluster scope)", set: false, want: nil},
		{name: "empty string returns nil (cluster scope)", env: "", set: true, want: nil},
		{name: "single namespace", env: "team-a", set: true, want: []string{"team-a"}},
		{name: "multiple namespaces", env: "team-a,team-b", set: true, want: []string{"team-a", "team-b"}},
		{name: "trims whitespace", env: " team-a , team-b ", set: true, want: []string{"team-a", "team-b"}},
		{name: "drops empty entries", env: "team-a,,team-b,", set: true, want: []string{"team-a", "team-b"}},
		{name: "only separators returns empty", env: " , , ", set: true, want: []string{}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.set {
				t.Setenv(EnvWatchNamespaces, tt.env)
			} else {
				// t.Setenv guarantees restoration; unset explicitly for the "unset" case.
				t.Setenv(EnvWatchNamespaces, "")
			}

			got := GetWatchNamespaces()
			if len(got) == 0 && len(tt.want) == 0 {
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t.Errorf("GetWatchNamespaces() = %#v, want %#v", got, tt.want)
			}
		})
	}
}
