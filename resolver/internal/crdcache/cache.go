package crdcache

import (
	"fmt"
	"sort"
	"sync"
	"time"

	"github.com/truefoundry/elasti/pkg/messages"
	"github.com/truefoundry/elasti/resolver/internal/operator"
	"go.uber.org/zap"
)

const defaultPollInterval = 5 * time.Minute

// minOnDemandRefreshInterval bounds how often a cache miss may trigger a synchronous refresh
// from the operator. It lets a newly-created ElastiService be picked up on its first request
// (instead of waiting for the next poll) while preventing probing for unknown hosts from
// turning into an operator fetch storm.
const minOnDemandRefreshInterval = 2 * time.Second

type Cache struct {
	logger       *zap.Logger
	operatorRPC  *operator.Client
	pollInterval time.Duration

	mu        sync.RWMutex
	cache     *sync.Map // key: "namespace/service-name", value: *messages.ElastiServiceEntry
	lastFetch time.Time // time of the last successful fetch (poll or on-demand); guarded by mu

	refreshMu sync.Mutex // serializes on-demand refreshes triggered by GetElastiServiceFresh

	stopCh   chan struct{}
	stopOnce sync.Once
}

// New creates a Cache that polls the operator every pollInterval.
func New(logger *zap.Logger, operatorRPC *operator.Client, pollInterval time.Duration) *Cache {
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}
	return &Cache{
		logger:       logger.With(zap.String("component", "crdcache")),
		operatorRPC:  operatorRPC,
		pollInterval: pollInterval,
		cache:        &sync.Map{},
		stopCh:       make(chan struct{}),
	}
}

// Start begins polling the operator for CRD cache updates.
func (c *Cache) Start() {
	c.logger.Info("Starting ElastiService cache poller", zap.Duration("interval", c.pollInterval))
	if err := c.fetch(); err != nil {
		c.logger.Error("Initial ElastiService cache fetch failed", zap.Error(err))
	}
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			c.logger.Info("ElastiService cache poller stopped")
			return
		case <-ticker.C:
			if err := c.fetch(); err != nil {
				c.logger.Error("ElastiService cache fetch failed", zap.Error(err))
			}
		}
	}
}

func (c *Cache) StartBackground() {
	go c.Start()
}

func (c *Cache) Stop() {
	c.stopOnce.Do(func() { close(c.stopCh) })
}

// fetch retrieves the ElastiService cache from the operator and atomically replaces the local copy.
func (c *Cache) fetch() error {
	resp, err := c.operatorRPC.GetElastiServiceCache()
	if err != nil {
		return fmt.Errorf("fetch: %w", err)
	}

	newCache := &sync.Map{}
	for k, v := range resp.Services {
		entry := v
		newCache.Store(k, &entry)
	}

	c.mu.Lock()
	c.cache = newCache
	c.lastFetch = time.Now()
	c.mu.Unlock()

	c.logger.Debug("ElastiService cache updated", zap.Int("count", len(resp.Services)))
	return nil
}

// GetElastiService returns the cached entry for "namespace/service-name".
func (c *Cache) GetElastiService(namespacedServiceName string) (*messages.ElastiServiceEntry, bool) {
	c.mu.RLock()
	cm := c.cache
	c.mu.RUnlock()

	val, ok := cm.Load(namespacedServiceName)
	if !ok {
		return nil, false
	}
	return val.(*messages.ElastiServiceEntry), true
}

// GetElastiServiceFresh returns the entry for "namespace/service-name". On a miss it triggers
// a rate-limited synchronous refresh from the operator and re-checks, so an ElastiService
// created since the last poll is recognized on its first request rather than being rejected
// until the next poll interval. The refresh is throttled by minOnDemandRefreshInterval so
// requests for unknown hosts cannot flood the operator with fetches.
func (c *Cache) GetElastiServiceFresh(namespacedServiceName string) (*messages.ElastiServiceEntry, bool) {
	if entry, ok := c.GetElastiService(namespacedServiceName); ok {
		return entry, true
	}
	c.refreshOnDemand()
	return c.GetElastiService(namespacedServiceName)
}

// refreshOnDemand fetches from the operator at most once per minOnDemandRefreshInterval.
func (c *Cache) refreshOnDemand() {
	c.refreshMu.Lock()
	defer c.refreshMu.Unlock()

	c.mu.RLock()
	since := time.Since(c.lastFetch)
	c.mu.RUnlock()
	if since < minOnDemandRefreshInterval {
		return
	}
	if err := c.fetch(); err != nil {
		c.logger.Warn("on-demand ElastiService cache refresh failed", zap.Error(err))
	}
}

// CachedService is one ElastiService in the resolver's local cache (key is namespace/service-name).
type CachedService struct {
	NamespacedName string `json:"namespacedName"`
	messages.ElastiServiceEntry
}

// ListCachedServices returns a stable snapshot of cached ElastiService entries.
func (c *Cache) ListCachedServices() []CachedService {
	c.mu.RLock()
	defer c.mu.RUnlock()
	cm := c.cache

	var keys []string
	cm.Range(func(key, _ any) bool {
		keys = append(keys, key.(string))
		return true
	})
	sort.Strings(keys)
	out := make([]CachedService, 0, len(keys))
	for _, k := range keys {
		v, ok := cm.Load(k)
		if !ok {
			continue
		}
		entry := *(v.(*messages.ElastiServiceEntry))
		out = append(out, CachedService{NamespacedName: k, ElastiServiceEntry: entry})
	}
	return out
}
