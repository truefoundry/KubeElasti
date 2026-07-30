package hostmanager

import (
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/truefoundry/elasti/pkg/messages"
	"go.uber.org/zap"
)

// fakeChecker implements ElastiServiceChecker over a fixed set of "namespace/service" keys.
type fakeChecker struct {
	known map[string]bool
}

func (f *fakeChecker) GetElastiService(key string) (*messages.ElastiServiceEntry, bool) {
	if f.known[key] {
		return &messages.ElastiServiceEntry{Name: key}, true
	}
	return nil, false
}

func TestGetHost(t *testing.T) {
	logger, _ := zap.NewDevelopment()

	tests := []struct {
		name          string
		req           *http.Request
		expectedHost  *messages.Host
		expectedError bool
	}{
		{
			name: "Host in header",
			req: &http.Request{
				Host: "target.com",
				Header: http.Header{
					"X-Envoy-Decorator-Operation": []string{"service.namespace.svc.cluster.local:8080/test/*"},
				},
			},
			expectedHost: &messages.Host{
				IncomingHost:   "service.namespace.svc.cluster.local:8080",
				Namespace:      "namespace",
				SourceService:  "service",
				TargetService:  "elasti-service-pvt-9df6b026a8",
				SourceHost:     "http://service.namespace.svc.cluster.local:8080",
				TargetHost:     "http://elasti-service-pvt-9df6b026a8.namespace.svc.cluster.local:8080",
				TrafficAllowed: true,
			},
			expectedError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hm := NewHostManager(logger, 10*time.Second, 15*time.Second, "X-Envoy-Decorator-Operation", nil)
			host, err := hm.GetHost(tt.req)
			if tt.expectedError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
			assert.Equal(t, tt.expectedHost, host)
		})
	}
}

// TestExtractNamespaceAndService asserts only strict k8s service DNS forms are accepted and
// the previous catch-all patterns (arbitrary two-/single-segment hosts) are rejected.
func TestExtractNamespaceAndService(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	hm := NewHostManager(logger, 10*time.Second, 15*time.Second, "Host", nil)

	tests := []struct {
		name      string
		input     string
		wantSvc   string
		wantNs    string
		wantError bool
	}{
		{name: "fqdn with port and path", input: "svc.ns.svc.cluster.local:8080/x/*", wantSvc: "svc", wantNs: "ns"},
		{name: "fqdn no port", input: "svc.ns.svc.cluster.local", wantSvc: "svc", wantNs: "ns"},
		{name: "short svc form", input: "svc.ns.svc", wantSvc: "svc", wantNs: "ns"},
		{name: "with http scheme", input: "http://svc.ns.svc.cluster.local:8080", wantSvc: "svc", wantNs: "ns"},
		{name: "hyphenated labels", input: "my-svc.my-ns.svc.cluster.local", wantSvc: "my-svc", wantNs: "my-ns"},

		// The vulnerable catch-all cases must now be rejected.
		{name: "arbitrary two-segment", input: "malicious.attacker", wantError: true},
		{name: "single segment", input: "evil-service", wantError: true},
		{name: "two segment with port", input: "malicious.attacker:8012", wantError: true},
		{name: "real cluster svc without elastisvc still parses format", input: "kube-dns.kube-system.svc.cluster.local", wantSvc: "kube-dns", wantNs: "kube-system"},
		{name: "missing svc segment", input: "svc.ns.cluster.local", wantError: true},
		{name: "empty", input: "", wantError: true},
		{name: "uppercase rejected", input: "Svc.Ns.svc.cluster.local", wantError: true},
		{name: "trailing dot label", input: "svc-.ns.svc", wantError: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc, ns, err := hm.extractNamespaceAndService(tt.input)
			if tt.wantError {
				assert.Error(t, err)
				return
			}
			assert.NoError(t, err)
			assert.Equal(t, tt.wantSvc, svc)
			assert.Equal(t, tt.wantNs, ns)
		})
	}
}

// TestGetHostRejectsUnknownService asserts that a well-formed host is still rejected when no
// matching ElastiService exists, blocking cross-namespace probing / SSRF via the resolver.
func TestGetHostRejectsUnknownService(t *testing.T) {
	logger, _ := zap.NewDevelopment()
	checker := &fakeChecker{known: map[string]bool{"namespace/service": true}}
	hm := NewHostManager(logger, 10*time.Second, 15*time.Second, "Host", checker)

	t.Run("known service accepted", func(t *testing.T) {
		req := &http.Request{Host: "service.namespace.svc.cluster.local:8080"}
		host, err := hm.GetHost(req)
		assert.NoError(t, err)
		assert.Equal(t, "service", host.SourceService)
		assert.Equal(t, "namespace", host.Namespace)
	})

	t.Run("unknown cross-namespace service rejected", func(t *testing.T) {
		req := &http.Request{Host: "kube-dns.kube-system.svc.cluster.local:8080"}
		host, err := hm.GetHost(req)
		assert.Error(t, err)
		assert.Equal(t, &messages.Host{}, host)
		// Rejected hosts must not be cached.
		_, cached := hm.hosts.Load("kube-dns.kube-system.svc.cluster.local:8080")
		assert.False(t, cached)
	})

	t.Run("malformed host rejected", func(t *testing.T) {
		req := &http.Request{Host: "malicious.attacker"}
		host, err := hm.GetHost(req)
		assert.Error(t, err)
		assert.Equal(t, &messages.Host{}, host)
	})
}
