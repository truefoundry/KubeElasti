package crdcache

import (
	"fmt"
	"sync"
	"time"

	"github.com/truefoundry/elasti/pkg/messages"
	"github.com/truefoundry/elasti/resolver/internal/operator"
	"go.uber.org/zap"
)

const defaultPollInterval = 5 * time.Minute

type Cache struct {
	logger       *zap.Logger
	operatorRPC  *operator.Client
	pollInterval time.Duration

	mu    sync.RWMutex
	cache *sync.Map // key: "namespace/service-name", value: *messages.ElastiServiceEntry

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
