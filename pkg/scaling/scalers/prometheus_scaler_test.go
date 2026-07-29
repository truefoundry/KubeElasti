package scalers

import (
	"context"
	"encoding/json"
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
		name        string
		addr        string
		defaultAddr string
		allowed     []string
		wantErr     bool
	}{
		{name: "empty allowlist accepts any", addr: "http://anything:9090"},
		{name: "host match", addr: "http://prom.monitoring:9090", allowed: []string{"prom.monitoring"}},
		{name: "host:port match", addr: "http://prom.monitoring:9090", allowed: []string{"prom.monitoring:9090"}},
		{name: "rejects unlisted host", addr: "http://request-catcher.evil:9090", allowed: []string{"prom.monitoring"}, wantErr: true},
		{name: "default is allowed despite non-matching allowlist", addr: "http://d:9090", defaultAddr: "http://d:9090", allowed: []string{"prom.monitoring"}},
		{name: "unparseable address is rejected", addr: "http://\x00", allowed: []string{"prom.monitoring"}, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &prometheusScaler{
				defaultServerAddress:   tt.defaultAddr,
				allowedServerAddresses: tt.allowed,
			}
			err := s.validateServerAddress(tt.addr)
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

func TestGuardDialAddress(t *testing.T) {
	tests := []struct {
		name    string
		addr    string
		wantErr bool
	}{
		{name: "missing port", addr: "127.0.0.1", wantErr: true},
		{name: "non-IP host", addr: "notanip:80", wantErr: true},
		{name: "blocked loopback", addr: "127.0.0.1:80", wantErr: true},
		{name: "allowed private", addr: "10.0.0.1:80", wantErr: false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := guardDialAddress(tt.addr)
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestNewPrometheusScaler(t *testing.T) {
	t.Run("valid metadata", func(t *testing.T) {
		sc, err := NewPrometheusScaler(json.RawMessage(`{"serverAddress":"http://p:9090","query":"up","threshold":"1"}`), time.Minute)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if sc == nil {
			t.Fatal("expected a scaler, got nil")
		}
	})

	t.Run("invalid metadata errors", func(t *testing.T) {
		if _, err := NewPrometheusScaler(json.RawMessage(`{bad`), time.Minute); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("allowlist is parsed from env", func(t *testing.T) {
		t.Setenv("PROMETHEUS_TRIGGER_ALLOWED_SERVER_ADDRESSES", "prom.monitoring")
		sc, err := NewPrometheusScaler(json.RawMessage(`{"query":"up","threshold":"1"}`), time.Minute)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got := sc.(*prometheusScaler).allowedServerAddresses; len(got) != 1 || got[0] != "prom.monitoring" {
			t.Fatalf("allowedServerAddresses = %v, want [prom.monitoring]", got)
		}
	})
}

func TestFetchDefaultHeaders(t *testing.T) {
	t.Run("unset returns empty", func(t *testing.T) {
		t.Setenv("PROMETHEUS_TRIGGER_AUTHORIZATION_HEADER", "")
		if h := fetchDefaultHeaders(); len(h) != 0 {
			t.Fatalf("headers = %v, want empty", h)
		}
	})

	t.Run("set returns Authorization", func(t *testing.T) {
		t.Setenv("PROMETHEUS_TRIGGER_AUTHORIZATION_HEADER", "Bearer x")
		if h := fetchDefaultHeaders(); h["Authorization"] != "Bearer x" {
			t.Fatalf("headers = %v, want Authorization=Bearer x", h)
		}
	})
}

func TestFetchAllowedServerAddresses(t *testing.T) {
	t.Run("unset returns nil", func(t *testing.T) {
		t.Setenv("PROMETHEUS_TRIGGER_ALLOWED_SERVER_ADDRESSES", "")
		if got := fetchAllowedServerAddresses(); got != nil {
			t.Fatalf("got %v, want nil", got)
		}
	})

	t.Run("parsed, trimmed, empties dropped", func(t *testing.T) {
		t.Setenv("PROMETHEUS_TRIGGER_ALLOWED_SERVER_ADDRESSES", " a , ,b ")
		got := fetchAllowedServerAddresses()
		if len(got) != 2 || got[0] != "a" || got[1] != "b" {
			t.Fatalf("got %v, want [a b]", got)
		}
	})
}

// promServer starts an httptest server returning a fixed status and body.
func promServer(t *testing.T, status int, body string) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(status)
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// plainScaler builds a scaler with a plain http.Client (no dial guard), since
// httptest servers listen on loopback which the guard would otherwise block.
func plainScaler(serverURL string) *prometheusScaler {
	return &prometheusScaler{
		httpClient: &http.Client{Timeout: 5 * time.Second},
		metadata:   &prometheusMetadata{ServerAddress: serverURL},
	}
}

// okVal is a well-formed single-vector Prometheus response carrying value v.
func okVal(v string) string {
	return `{"status":"success","data":{"resultType":"vector","result":[{"metric":{},"value":[1,"` + v + `"]}]}}`
}

func TestExecutePromQuery(t *testing.T) {
	tests := []struct {
		name    string
		status  int
		body    string
		want    float64
		wantErr bool
	}{
		{name: "single value", status: 200, body: okVal("1"), want: 1},
		{name: "null value returns -1", status: 200, body: `{"data":{"result":[{"value":[1,null]}]}}`, want: -1},
		{name: "non-200 status", status: 500, body: "", wantErr: true},
		{name: "invalid json", status: 200, body: "not-json", wantErr: true},
		{name: "empty result", status: 200, body: `{"data":{"result":[]}}`, wantErr: true},
		{name: "multiple results", status: 200, body: `{"data":{"result":[{"value":[1,"1"]},{"value":[1,"2"]}]}}`, wantErr: true},
		{name: "empty value list", status: 200, body: `{"data":{"result":[{"value":[]}]}}`, wantErr: true},
		{name: "too few values", status: 200, body: `{"data":{"result":[{"value":[1]}]}}`, wantErr: true},
		{name: "non-numeric value", status: 200, body: `{"data":{"result":[{"value":[1,"abc"]}]}}`, wantErr: true},
		{name: "infinite value", status: 200, body: okVal("+Inf"), wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := promServer(t, tt.status, tt.body)
			got, err := plainScaler(srv.URL).executePromQuery(context.Background(), "up")
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got value %v", got)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("executePromQuery() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestExecutePromQueryRequestErrors(t *testing.T) {
	t.Run("no address configured", func(t *testing.T) {
		s := &prometheusScaler{metadata: &prometheusMetadata{}, httpClient: &http.Client{}}
		if _, err := s.executePromQuery(context.Background(), "up"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("address not in allowlist", func(t *testing.T) {
		s := &prometheusScaler{
			metadata:               &prometheusMetadata{ServerAddress: "http://evil:9090"},
			allowedServerAddresses: []string{"prom.monitoring"},
			httpClient:             &http.Client{},
		}
		if _, err := s.executePromQuery(context.Background(), "up"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("invalid request url", func(t *testing.T) {
		if _, err := plainScaler("http://\x00bad").executePromQuery(context.Background(), "up"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})

	t.Run("connection refused", func(t *testing.T) {
		if _, err := plainScaler("http://127.0.0.1:1").executePromQuery(context.Background(), "up"); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestShouldScaleToZero(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    bool
		wantErr bool
	}{
		{name: "below threshold scales to zero", body: okVal("0"), want: true},
		{name: "at or above threshold stays", body: okVal("2"), want: false},
		{name: "unavailable metric does not scale", body: `{"data":{"result":[{"value":[1,null]}]}}`, want: false},
		{name: "query error", body: "bad", wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := promServer(t, 200, tt.body)
			s := plainScaler(srv.URL)
			s.metadata.Query = "up"
			s.metadata.Threshold = 1
			got, err := s.ShouldScaleToZero(context.Background())
			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("ShouldScaleToZero() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestShouldScaleFromZero(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		want    bool
		wantErr bool
	}{
		{name: "at or above threshold scales", body: okVal("2"), want: true},
		{name: "below threshold does not scale", body: okVal("0"), want: false},
		{name: "unavailable metric scales", body: `{"data":{"result":[{"value":[1,null]}]}}`, want: true},
		{name: "query error scales and returns error", body: "bad", want: true, wantErr: true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			srv := promServer(t, 200, tt.body)
			s := plainScaler(srv.URL)
			s.metadata.Query = "up"
			s.metadata.Threshold = 1
			got, err := s.ShouldScaleFromZero(context.Background())
			if got != tt.want {
				t.Errorf("ShouldScaleFromZero() = %v, want %v", got, tt.want)
			}
			if tt.wantErr && err == nil {
				t.Fatal("expected error, got nil")
			}
			if !tt.wantErr && err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestIsHealthy(t *testing.T) {
	t.Run("healthy when uptime is 1 (default filter)", func(t *testing.T) {
		srv := promServer(t, 200, okVal("1"))
		s := plainScaler(srv.URL)
		s.cooldownPeriod = time.Minute
		ok, err := s.IsHealthy(context.Background())
		if err != nil || !ok {
			t.Fatalf("IsHealthy() = %v, %v; want true, nil", ok, err)
		}
	})

	t.Run("unhealthy when uptime is not 1 (custom filter)", func(t *testing.T) {
		srv := promServer(t, 200, okVal("0"))
		s := plainScaler(srv.URL)
		s.cooldownPeriod = time.Minute
		s.metadata.UptimeFilter = `job="prometheus"`
		ok, err := s.IsHealthy(context.Background())
		if err != nil || ok {
			t.Fatalf("IsHealthy() = %v, %v; want false, nil", ok, err)
		}
	})

	t.Run("query error", func(t *testing.T) {
		srv := promServer(t, 500, "")
		s := plainScaler(srv.URL)
		s.cooldownPeriod = time.Minute
		if _, err := s.IsHealthy(context.Background()); err == nil {
			t.Fatal("expected error, got nil")
		}
	})
}

func TestClose(t *testing.T) {
	if err := (&prometheusScaler{httpClient: &http.Client{}}).Close(context.Background()); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if err := (&prometheusScaler{}).Close(context.Background()); err != nil {
		t.Fatalf("unexpected error with nil client: %v", err)
	}
}
