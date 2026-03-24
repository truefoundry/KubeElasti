package crdcache

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/truefoundry/elasti/pkg/messages"
	"github.com/truefoundry/elasti/resolver/internal/operator"
	"go.uber.org/zap"
)

// swappableHandler lets tests replace the HTTP handler mid-test.
type swappableHandler struct {
	mu sync.RWMutex
	fn http.HandlerFunc
}

func (h *swappableHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.mu.RLock()
	fn := h.fn
	h.mu.RUnlock()
	fn(w, r)
}

func (h *swappableHandler) set(fn http.HandlerFunc) {
	h.mu.Lock()
	h.fn = fn
	h.mu.Unlock()
}

func respondWith(resp *messages.ElastiServiceCacheResponse) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp) //nolint:errcheck
	}
}

func newTestCache(t *testing.T, server *httptest.Server, interval time.Duration) *Cache {
	t.Helper()
	rpc := operator.NewOperatorClientWithURL(zap.NewNop(), 0, server.URL)
	return New(zap.NewNop(), rpc, interval)
}

// TestFetchPopulatesCache checks that a successful fetch stores entries.
func TestFetchPopulatesCache(t *testing.T) {
	srv := httptest.NewServer(respondWith(&messages.ElastiServiceCacheResponse{
		Services: map[string]messages.ElastiServiceEntry{
			"default/my-svc": {Name: "my-elastiservice", Spec: json.RawMessage(`{"service":"my-svc"}`)},
		},
	}))
	defer srv.Close()

	c := newTestCache(t, srv, time.Minute)
	if err := c.fetch(); err != nil {
		t.Fatalf("fetch() unexpected error: %v", err)
	}

	entry, ok := c.GetElastiService("default/my-svc")
	if !ok {
		t.Fatal("expected entry to be in cache")
	}
	if entry.Name != "my-elastiservice" {
		t.Errorf("Name = %q; want %q", entry.Name, "my-elastiservice")
	}
}

// TestFetchErrorDoesNotClearCache checks that a failed fetch leaves the previous cache intact.
func TestFetchErrorDoesNotClearCache(t *testing.T) {
	h := &swappableHandler{}
	h.set(respondWith(&messages.ElastiServiceCacheResponse{
		Services: map[string]messages.ElastiServiceEntry{
			"ns/svc": {Name: "es"},
		},
	}))
	srv := httptest.NewServer(h)
	defer srv.Close()

	c := newTestCache(t, srv, time.Minute)
	if err := c.fetch(); err != nil {
		t.Fatalf("first fetch error: %v", err)
	}

	// Inject a server error.
	h.set(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "operator down", http.StatusInternalServerError)
	})

	if err := c.fetch(); err == nil {
		t.Fatal("expected error from fetch, got nil")
	}

	// Previous data must still be accessible.
	_, ok := c.GetElastiService("ns/svc")
	if !ok {
		t.Fatal("cache was cleared on error; expected previous entry to survive")
	}
}

// TestGetElastiServiceMiss checks that missing keys return false.
func TestGetElastiServiceMiss(t *testing.T) {
	srv := httptest.NewServer(respondWith(&messages.ElastiServiceCacheResponse{
		Services: map[string]messages.ElastiServiceEntry{},
	}))
	defer srv.Close()

	c := newTestCache(t, srv, time.Minute)
	_ = c.fetch()

	_, ok := c.GetElastiService("ns/missing")
	if ok {
		t.Fatal("expected miss for unknown key")
	}
}

// TestConcurrentFetchAndGet exercises the RWMutex guarding the cache pointer.
func TestConcurrentFetchAndGet(t *testing.T) {
	srv := httptest.NewServer(respondWith(&messages.ElastiServiceCacheResponse{
		Services: map[string]messages.ElastiServiceEntry{
			"ns/svc": {Name: "es"},
		},
	}))
	defer srv.Close()

	c := newTestCache(t, srv, time.Minute)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(2)
		go func() {
			defer wg.Done()
			_ = c.fetch()
		}()
		go func() {
			defer wg.Done()
			c.GetElastiService("ns/svc")
		}()
	}
	wg.Wait()
}

// TestStopCancelsPoller checks that Stop() prevents further ticks.
func TestStopCancelsPoller(t *testing.T) {
	var calls atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		calls.Add(1)
		json.NewEncoder(w).Encode(&messages.ElastiServiceCacheResponse{ //nolint:errcheck
			Services: map[string]messages.ElastiServiceEntry{},
		})
	}))
	defer srv.Close()

	c := newTestCache(t, srv, 50*time.Millisecond)
	c.StartBackground()
	time.Sleep(160 * time.Millisecond)
	c.Stop()

	callsAfterStop := calls.Load()
	time.Sleep(150 * time.Millisecond)

	if got := calls.Load(); got != callsAfterStop {
		t.Errorf("calls continued after Stop(): before=%d after=%d", callsAfterStop, got)
	}
}
