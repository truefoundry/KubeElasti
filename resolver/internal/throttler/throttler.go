package throttler

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/truefoundry/elasti/pkg/k8shelper"
	"github.com/truefoundry/elasti/pkg/messages"
	"go.uber.org/zap"
)

type (
	Throttler struct {
		logger                  *zap.Logger
		breaker                 *Breaker
		k8sUtil                 *k8shelper.Ops
		retryDuration           time.Duration
		TrafficReEnableDuration time.Duration
		serviceReadyMap         sync.Map
		queueSizeMap            sync.Map
	}

	Params struct {
		QueueRetryDuration      time.Duration
		TrafficReEnableDuration time.Duration
		K8sUtil                 *k8shelper.Ops
		QueueDepth              int
		MaxConcurrency          int
		InitialCapacity         int
		Logger                  *zap.Logger
	}
)

func NewThrottler(param *Params) *Throttler {
	breaker := NewBreaker(BreakerParams{
		QueueDepth:      param.QueueDepth,
		MaxConcurrency:  param.MaxConcurrency,
		InitialCapacity: param.InitialCapacity,
		Logger:          param.Logger,
	})

	return &Throttler{
		logger:                  param.Logger.With(zap.String("component", "throttler")),
		breaker:                 breaker,
		k8sUtil:                 param.K8sUtil,
		TrafficReEnableDuration: param.TrafficReEnableDuration,
		retryDuration:           param.QueueRetryDuration,
	}
}

func (t *Throttler) Try(ctx context.Context, host *messages.Host, resolve func(int) error, tryErrCallback func()) error {
	reenqueue := true
	tryCount := 1
	var tryErr error

	t.incrementQueueSize(host.Namespace, host.SourceService)
	defer t.decrementQueueSize(host.Namespace, host.SourceService)

	for reenqueue {
		tryErr = nil
		breakErr := t.breaker.Maybe(ctx, func() {
			if isPodActive, err := t.checkIfServiceReady(host.Namespace, host.TargetService); err != nil {
				tryErr = err
				go tryErrCallback()
			} else if isPodActive {
				if res := resolve(tryCount); res != nil {
					tryErr = fmt.Errorf("resolve error: %w", res)
				}
				// We don't reenqueue if the POD is active, but request failed to resolve
				reenqueue = false
			}

			select {
			case <-ctx.Done():
				tryErr = fmt.Errorf("context done error: %w", ctx.Err())
				reenqueue = false
			default:
				if reenqueue {
					tryCount++
					time.Sleep(t.retryDuration)
				}
			}
		})
		if breakErr != nil {
			return fmt.Errorf("breaker error: %w", breakErr)
		}
	}
	if tryErr != nil {
		return fmt.Errorf("thunk error: %w", tryErr)
	}
	return nil
}

func (t *Throttler) checkIfServiceReady(namespace, service string) (bool, error) {
	key := fmt.Sprintf("%s/%s", namespace, service)
	if ready, ok := t.serviceReadyMap.Load(key); ok {
		return ready.(bool), nil
	}

	isPodActive, err := t.k8sUtil.CheckIfServiceEndpointActive(namespace, service)
	if err != nil {
		return false, fmt.Errorf("unable to get target active endpoints: %w", err)
	}
	if !isPodActive {
		return false, fmt.Errorf("no active endpoints found for namespace: %v service: %v", namespace, service)
	}

	t.serviceReadyMap.Store(key, true)
	// release the memory after sometime
	time.AfterFunc(t.TrafficReEnableDuration, func() {
		t.serviceReadyMap.Delete(key)
	})
	return true, nil
}

func (t *Throttler) GetQueueSize(namespace, service string) int {
	key := fmt.Sprintf("%s/%s", namespace, service)
	if size, ok := t.queueSizeMap.Load(key); ok {
		return size.(int)
	}
	return 0
}

func (t *Throttler) incrementQueueSize(namespace, service string) {
	key := fmt.Sprintf("%s/%s", namespace, service)
	t.queueSizeMap.Store(key, t.GetQueueSize(namespace, service)+1)
}

func (t *Throttler) decrementQueueSize(namespace, service string) {
	key := fmt.Sprintf("%s/%s", namespace, service)
	t.queueSizeMap.Store(key, t.GetQueueSize(namespace, service)-1)
}
