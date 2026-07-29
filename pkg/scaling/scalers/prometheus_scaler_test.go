package scalers

import (
	"context"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestIsBlockedIP(t *testing.T) {
	tests := []struct {
		name    string
		ip      string
		blocked bool
	}{
		{name: "IPv4 loopback", ip: "127.0.0.1", blocked: true},
		{name: "IPv6 loopback", ip: "::1", blocked: true},
		{name: "unspecified", ip: "0.0.0.0", blocked: true},
		{name: "IPv4 link-local", ip: "169.254.10.20", blocked: true},
		{name: "IPv6 link-local", ip: "fe80::1", blocked: true},
		{name: "multicast", ip: "224.0.0.1", blocked: true},
		// Private ranges are allowed: in-cluster Prometheus lives on a ClusterIP.
		{name: "private 10.x (ClusterIP)", ip: "10.96.0.10", blocked: false},
		{name: "private 172.16.x", ip: "172.16.5.4", blocked: false},
		{name: "private 192.168.x", ip: "192.168.1.1", blocked: false},
		{name: "IPv6 ULA (in-cluster)", ip: "fd12:3456::1", blocked: false},
		{name: "public IPv4", ip: "8.8.8.8", blocked: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip := net.ParseIP(tt.ip)
			if ip == nil {
				t.Fatalf("failed to parse test IP %q", tt.ip)
			}
			if got := isBlockedIP(ip); got != tt.blocked {
				t.Errorf("isBlockedIP(%s) = %v, want %v", tt.ip, got, tt.blocked)
			}
		})
	}
}

func TestNewSafeHTTPClientBlocksLoopback(t *testing.T) {
	// A real listener on loopback: the dial guard must refuse to connect to it.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newHTTPClient(true)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	if _, err := client.Do(req); err == nil {
		t.Fatal("expected connection to loopback to be blocked, got nil error")
	}
}

func TestNewHTTPClientDialGuardDisabledAllowsLoopback(t *testing.T) {
	// When an allowlist is configured (dialGuard=false), the allowlist is the
	// sole gate, so an explicitly-allowed loopback target must be reachable.
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer srv.Close()

	client := newHTTPClient(false)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, srv.URL, nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	resp, err := client.Do(req)
	if err != nil {
		t.Fatalf("expected loopback to be reachable without dial guard, got %v", err)
	}
	defer resp.Body.Close()
}

func TestNewSafeHTTPClientRejectsRedirects(t *testing.T) {
	client := newHTTPClient(true)
	req, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://example.com", nil)
	if err != nil {
		t.Fatalf("failed to build request: %v", err)
	}
	if err := client.CheckRedirect(req, nil); err == nil {
		t.Fatal("expected CheckRedirect to reject redirects, got nil error")
	}
}

func TestGetServerAddress(t *testing.T) {
	tests := []struct {
		name         string
		metadataAddr string
		defaultAddr  string
		want         string
		wantErr      bool
	}{
		{name: "crd overrides default", metadataAddr: "http://prom.monitoring:9090", defaultAddr: "http://d:9090", want: "http://prom.monitoring:9090"},
		{name: "falls back to default", defaultAddr: "http://d:9090", want: "http://d:9090"},
		{name: "nothing configured errors", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &prometheusScaler{
				metadata:             &prometheusMetadata{ServerAddress: tt.metadataAddr},
				defaultServerAddress: tt.defaultAddr,
			}
			got, err := s.getServerAddress()
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got address %q", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("getServerAddress() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestValidateServerAddress(t *testing.T) {
	tests := []struct {
		name         string
		metadataAddr string
		allowed      []string
		wantErr      bool
	}{
		{name: "no crd override is a no-op", allowed: []string{"prom.monitoring"}},
		{name: "empty allowlist accepts any", metadataAddr: "http://anything:9090"},
		{name: "host match", metadataAddr: "http://prom.monitoring:9090", allowed: []string{"prom.monitoring"}},
		{name: "host:port match", metadataAddr: "http://prom.monitoring:9090", allowed: []string{"prom.monitoring:9090"}},
		{name: "rejects unlisted host", metadataAddr: "http://request-catcher.evil:9090", allowed: []string{"prom.monitoring"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &prometheusScaler{
				metadata:               &prometheusMetadata{ServerAddress: tt.metadataAddr},
				allowedServerAddresses: tt.allowed,
			}
			err := s.validateServerAddress()
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestExecutePromQueryCRDHeaderWins(t *testing.T) {
	// A per-trigger (CRD) header must override the operator default.
	gotAuth := make(chan string, 1)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth <- r.Header.Get("Authorization")
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1,"1"]}]}}`))
	}))
	defer srv.Close()

	s := &prometheusScaler{
		// httptest listens on loopback, so use a plain client here (the guard is
		// covered by its own test); this test targets header precedence only.
		httpClient: &http.Client{Timeout: 5 * time.Second},
		metadata: &prometheusMetadata{
			ServerAddress: srv.URL,
			Headers:       map[string]string{"Authorization": "crd-token"},
		},
		defaultHeaders: map[string]string{"Authorization": "operator-token"},
	}

	if _, err := s.executePromQuery(context.Background(), "up"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if auth := <-gotAuth; auth != "crd-token" {
		t.Errorf("Authorization header = %q, want per-trigger CRD header to win", auth)
	}
}
