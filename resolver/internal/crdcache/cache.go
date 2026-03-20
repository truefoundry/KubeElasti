package crdcache

import (
	"context"
	"encoding/json"
	"net"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/truefoundry/elasti/pkg/config"
	"github.com/truefoundry/elasti/pkg/messages"
	"go.uber.org/zap"
)

const (
	defaultPollInterval = 5 * time.Minute
	crdCachePath        = "/crd-cache"
)

type Cache struct {
	logger       *zap.Logger
	operatorURL  string
	pollInterval time.Duration
	client       *http.Client
	cache        *sync.Map // key: "namespace/name", value: *messages.CRDCacheEntry
	stopCh       chan struct{}
	stopOnce     sync.Once
}

// New creates a CRD cache that polls the operator every pollInterval.
func New(logger *zap.Logger, pollInterval time.Duration) *Cache {
	if pollInterval <= 0 {
		pollInterval = defaultPollInterval
	}
	operatorConfig := config.GetOperatorConfig()
	operatorHost := operatorConfig.ServiceName + "." + operatorConfig.Namespace + ".svc." + config.GetKubernetesClusterDomain()
	operatorHostPort := net.JoinHostPort(operatorHost, strconv.Itoa(int(operatorConfig.Port)))

	return &Cache{
		logger:       logger.With(zap.String("component", "crdcache")),
		operatorURL:  "http://" + operatorHostPort + crdCachePath,
		pollInterval: pollInterval,
		client:       &http.Client{Timeout: 30 * time.Second},
		cache:        &sync.Map{},
		stopCh:       make(chan struct{}),
	}
}

// Start begins polling the operator for CRD cache updates.
func (c *Cache) Start() {
	c.logger.Info("Starting CRD cache poller", zap.Duration("interval", c.pollInterval))
	c.fetch()
	ticker := time.NewTicker(c.pollInterval)
	defer ticker.Stop()
	for {
		select {
		case <-c.stopCh:
			c.logger.Info("CRD cache poller stopped")
			return
		case <-ticker.C:
			c.fetch()
		}
	}
}

func (c *Cache) StartBackground() {
	go c.Start()
}

func (c *Cache) Stop() {
	c.stopOnce.Do(func() {
		close(c.stopCh)
	})
}

// fetch retrieves the CRD cache from the operator and updates the local cache.
func (c *Cache) fetch() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.operatorURL, nil)
	if err != nil {
		c.logger.Error("Failed to create CRD cache request", zap.Error(err))
		return
	}
	resp, err := c.client.Do(req)
	if err != nil {
		c.logger.Error("Failed to fetch CRD cache from operator", zap.Error(err))
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		c.logger.Error("CRD cache fetch returned non-OK status", zap.Int("status", resp.StatusCode))
		return
	}
	var cacheResp messages.CRDCacheResponse
	if err := json.NewDecoder(resp.Body).Decode(&cacheResp); err != nil {
		c.logger.Error("Failed to decode CRD cache response", zap.Error(err))
		return
	}

	// Build new cache then swap atomically to avoid partial reads
	newCache := &sync.Map{}
	for k, v := range cacheResp.CRDCache {
		entry := v
		newCache.Store(k, &entry)
	}
	c.cache = newCache
	c.logger.Debug("CRD cache updated", zap.Int("count", len(cacheResp.CRDCache)))
}

// GetCRD returns the cached CRD entry for the given namespaced name (namespace/name).
func (c *Cache) GetCRD(namespacedName string) (*messages.CRDCacheEntry, bool) {
	val, ok := c.cache.Load(namespacedName)
	if !ok {
		return nil, false
	}
	return val.(*messages.CRDCacheEntry), true
}

// GetCRDByService returns the cached CRD entry for namespace and service name.
func (c *Cache) GetCRDByService(namespace, service string) (*messages.CRDCacheEntry, bool) {
	return c.GetCRD(namespace + "/" + service)
}

type Provider interface {
	GetCRD(namespacedName string) (*messages.CRDCacheEntry, bool)
}

func NewProvider(c *Cache) Provider {
	return c
}
