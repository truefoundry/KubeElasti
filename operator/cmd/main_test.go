package main

import (
	"sort"
	"testing"
)

func TestEffectiveWatchNamespaces(t *testing.T) {
	tests := []struct {
		name          string
		watch         []string
		alwaysInclude []string
		want          []string // order-insensitive
	}{
		{
			name:          "empty watch set is cluster scope",
			watch:         nil,
			alwaysInclude: []string{"elasti", "elasti"},
			want:          nil,
		},
		{
			name:          "folds in operator and resolver namespaces",
			watch:         []string{"team-a"},
			alwaysInclude: []string{"elasti", "elasti"},
			want:          []string{"team-a", "elasti"},
		},
		{
			name:          "de-duplicates and drops empties",
			watch:         []string{"team-a", "team-b", "team-a", ""},
			alwaysInclude: []string{"team-b", "elasti"},
			want:          []string{"team-a", "team-b", "elasti"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := effectiveWatchNamespaces(tt.watch, tt.alwaysInclude...)
			if !equalAsSets(got, tt.want) {
				t.Errorf("effectiveWatchNamespaces(%#v, %#v) = %#v, want set %#v", tt.watch, tt.alwaysInclude, got, tt.want)
			}
		})
	}
}

func TestBuildCacheOptions(t *testing.T) {
	t.Run("empty is zero-value (cluster scope, byte-identical default)", func(t *testing.T) {
		opts := buildCacheOptions(nil)
		if opts.DefaultNamespaces != nil {
			t.Errorf("expected nil DefaultNamespaces for cluster scope, got %#v", opts.DefaultNamespaces)
		}
	})

	t.Run("non-empty scopes to exactly the given namespaces", func(t *testing.T) {
		in := []string{"team-a", "team-b", "elasti"}
		opts := buildCacheOptions(in)
		if len(opts.DefaultNamespaces) != len(in) {
			t.Fatalf("expected %d namespaces, got %d: %#v", len(in), len(opts.DefaultNamespaces), opts.DefaultNamespaces)
		}
		for _, ns := range in {
			if _, ok := opts.DefaultNamespaces[ns]; !ok {
				t.Errorf("expected namespace %q in DefaultNamespaces, got %#v", ns, opts.DefaultNamespaces)
			}
		}
	})
}

func equalAsSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	ac := append([]string{}, a...)
	bc := append([]string{}, b...)
	sort.Strings(ac)
	sort.Strings(bc)
	for i := range ac {
		if ac[i] != bc[i] {
			return false
		}
	}
	return true
}
