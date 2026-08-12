package crdcache

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/kubeelasti/kubeelasti/operator/api/v1alpha1"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

func TestMatchProbeResponseFromSpec_pathPrefix(t *testing.T) {
	spec := v1alpha1.ElastiServiceSpec{
		ProbeResponse: []v1alpha1.ProbeResponseRule{
			{
				Path: &v1alpha1.ProbeResponsePathMatch{Type: "PathPrefix", Value: "/health"},
				Response: v1alpha1.ProbeResponse{
					Status: 200,
					Body:   `{"ok":true}`,
				},
			},
		},
	}
	raw, err := json.Marshal(spec)
	require.NoError(t, err)

	t.Run("match child path", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/health/ready", http.NoBody)
		body, status, ok := MatchProbeResponseFromSpec(raw, req, zap.NewNop())
		require.True(t, ok)
		require.Equal(t, http.StatusOK, status)
		require.Equal(t, `{"ok":true}`, body)
	})
	t.Run("no match foobar", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/healthz", http.NoBody)
		_, _, ok := MatchProbeResponseFromSpec(raw, req, zap.NewNop())
		require.False(t, ok)
	})
}

func TestMatchProbeResponseFromSpec_emptyBody204(t *testing.T) {
	spec := v1alpha1.ElastiServiceSpec{
		ProbeResponse: []v1alpha1.ProbeResponseRule{
			{
				Path: &v1alpha1.ProbeResponsePathMatch{Type: "Exact", Value: "/ready"},
				Response: v1alpha1.ProbeResponse{
					Status: 204,
					Body:   "",
				},
			},
		},
	}
	raw, err := json.Marshal(spec)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/ready", http.NoBody)
	body, status, ok := MatchProbeResponseFromSpec(raw, req, zap.NewNop())
	require.True(t, ok)
	require.Equal(t, 204, status)
	require.Equal(t, "", body)
}

func TestMatchProbeResponseFromSpec_defaultPathAndStatus(t *testing.T) {
	spec := v1alpha1.ElastiServiceSpec{
		ProbeResponse: []v1alpha1.ProbeResponseRule{
			{Response: v1alpha1.ProbeResponse{Status: 0, Body: "pong"}},
		},
	}
	raw, err := json.Marshal(spec)
	require.NoError(t, err)
	req := httptest.NewRequest(http.MethodGet, "/any", http.NoBody)
	body, status, ok := MatchProbeResponseFromSpec(raw, req, zap.NewNop())
	require.True(t, ok)
	require.Equal(t, http.StatusOK, status)
	require.Equal(t, "pong", body)
}

func TestMatchProbeResponseFromSpec_headersAndQueryANDed(t *testing.T) {
	spec := v1alpha1.ElastiServiceSpec{
		ProbeResponse: []v1alpha1.ProbeResponseRule{
			{
				Path: &v1alpha1.ProbeResponsePathMatch{Value: "/probe"},
				Headers: []v1alpha1.ProbeResponseHeaderMatch{
					{Name: "X-Check", Value: "1"},
				},
				QueryParams: []v1alpha1.ProbeResponseQueryParamMatch{
					{Name: "k", Value: "v"},
				},
				Method: ptr("GET"),
				Response: v1alpha1.ProbeResponse{
					Status: 204,
					Body:   "x",
				},
			},
		},
	}
	raw, err := json.Marshal(spec)
	require.NoError(t, err)

	t.Run("match", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/probe?k=v", http.NoBody)
		req.Header.Set("X-Check", "1")
		_, status, ok := MatchProbeResponseFromSpec(raw, req, zap.NewNop())
		require.True(t, ok)
		require.Equal(t, 204, status)
	})
	t.Run("missing header", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/probe?k=v", http.NoBody)
		_, _, ok := MatchProbeResponseFromSpec(raw, req, zap.NewNop())
		require.False(t, ok)
	})
	t.Run("wrong query", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/probe?k=other", http.NoBody)
		req.Header.Set("X-Check", "1")
		_, _, ok := MatchProbeResponseFromSpec(raw, req, zap.NewNop())
		require.False(t, ok)
	})
}

func ptr(s string) *string { return &s }
