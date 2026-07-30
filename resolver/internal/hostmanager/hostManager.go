package hostmanager

import (
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/truefoundry/elasti/resolver/internal/prom"

	"github.com/truefoundry/elasti/pkg/logger"
	"github.com/truefoundry/elasti/pkg/utils"

	"github.com/truefoundry/elasti/pkg/messages"
	"go.uber.org/zap"
)

// ElastiServiceChecker reports whether an ElastiService CR exists for a given
// "namespace/service" key. It is satisfied by the resolver's crdcache.Cache and lets the
// HostManager reject Host headers that don't map to a real ElastiService before they are
// cached, proxied, or exported as metrics.
type ElastiServiceChecker interface {
	GetElastiService(namespacedServiceName string) (*messages.ElastiServiceEntry, bool)
}

// HostManager is to manage the hosts, and their traffic
// It is used to process incoming requests and cache the host details in "hosts" map
// For further requests, the cache is used to get the host details
type HostManager struct {
	logger                      *zap.Logger
	hosts                       sync.Map
	trafficReEnableDuration     time.Duration
	trafficDisableGraceDuration time.Duration
	headerForHost               string
	crdChecker                  ElastiServiceChecker
}

// NewHostManager returns a new HostManager.
// crdChecker validates that a parsed (namespace, service) belongs to a known ElastiService;
// pass nil to disable that check (used in tests).
func NewHostManager(logger *zap.Logger, trafficReEnableDuration, trafficDisableGraceDuration time.Duration, headerForHost string, crdChecker ElastiServiceChecker) *HostManager {
	return &HostManager{
		logger:                      logger.With(zap.String("component", "hostManager")),
		hosts:                       sync.Map{},
		trafficReEnableDuration:     trafficReEnableDuration,
		trafficDisableGraceDuration: trafficDisableGraceDuration,
		headerForHost:               headerForHost,
		crdChecker:                  crdChecker,
	}
}

// GetHost returns the host details for incoming and outgoing requests
func (hm *HostManager) GetHost(req *http.Request) (*messages.Host, error) {
	incomingHost := req.Host
	if values, ok := req.Header[hm.headerForHost]; ok {
		incomingHost = values[0]
	}
	// Normalize the key by dropping the request path/wildcard (the actual path is taken from
	// req.RequestURI at proxy time). This collapses the otherwise-unbounded set of path
	// variations for a given host:port into a single cache entry, so a caller cannot grow
	// hm.hosts without bound by sending many distinct paths for the same service.
	incomingHost = hm.removeTrailingWildcardIfNeeded(incomingHost)
	incomingHost = hm.removeTrailingPathIfNeeded(incomingHost)
	host, ok := hm.hosts.Load(incomingHost)
	if !ok {
		sourceService, namespace, err := hm.extractNamespaceAndService(incomingHost)
		if err != nil {
			prom.HostExtractionCounter.WithLabelValues("error", "invalid-format").Inc()
			return &messages.Host{}, err
		}
		// Reject Host headers that don't map to a known ElastiService. Without this an
		// unauthenticated caller could make the resolver cache, proxy to, and probe
		// arbitrary in-cluster services across namespaces (CWE-200 / cross-namespace SSRF).
		if hm.crdChecker != nil {
			if _, exists := hm.crdChecker.GetElastiService(namespace + "/" + sourceService); !exists {
				prom.HostExtractionCounter.WithLabelValues("rejected", "unknown-service").Inc()
				return &messages.Host{}, fmt.Errorf("no ElastiService registered for host: %s", logger.MaskMiddle(incomingHost, 4, 4))
			}
		}
		targetService := utils.GetPrivateServiceName(sourceService)
		sourceHost := hm.addHTTPIfNeeded(incomingHost)
		targetHost := hm.replaceServiceName(sourceHost, targetService)
		targetHost = hm.addHTTPIfNeeded(targetHost)
		newHost := &messages.Host{
			IncomingHost:   incomingHost,
			Namespace:      namespace,
			SourceService:  sourceService,
			TargetService:  targetService,
			SourceHost:     sourceHost,
			TargetHost:     targetHost,
			TrafficAllowed: true,
		}
		hm.hosts.Store(incomingHost, newHost)
		prom.HostExtractionCounter.WithLabelValues("cache-miss", "").Inc()
		return newHost, nil
	}
	prom.HostExtractionCounter.WithLabelValues("cache-hit", "").Inc()
	return host.(*messages.Host), nil
}

// DisableTrafficForHost disables the traffic for the host
func (hm *HostManager) disableTrafficForHost(hostName string) {
	if host, ok := hm.hosts.Load(hostName); ok && host.(*messages.Host).TrafficAllowed {
		host.(*messages.Host).TrafficAllowed = false
		hm.hosts.Store(hostName, host)
		hm.logger.Debug("Disabled traffic for host",
			zap.String("hostName", logger.MaskMiddle(hostName, 4, 4)),
			zap.Duration("trafficReEnableDuration", hm.trafficReEnableDuration))
		go time.AfterFunc(hm.trafficReEnableDuration, func() {
			hm.enableTrafficForHost(hostName)
		})
		prom.TrafficSwitchCounter.WithLabelValues(hostName, "disabled").Inc()
	}
}

// ScheduleDisableTrafficForHost schedules a delayed disable of traffic for the host.
// The disable fires after TrafficDisableGraceDuration, and only one timer is scheduled per host
// regardless of how many requests arrive during the delay window.
// This allows a grace period for EndpointSlice and CNI controller to converge on route changes
// before the resolver stops routing requests
func (hm *HostManager) ScheduleDisableTrafficForHost(hostName string) {
	host, ok := hm.hosts.Load(hostName)
	if !ok {
		return
	}
	h := host.(*messages.Host)
	if h.TrafficDisableScheduled {
		return
	}
	h.TrafficDisableScheduled = true
	hm.hosts.Store(hostName, h)
	hm.logger.Debug("Scheduled delayed disable for host",
		zap.String("hostName", logger.MaskMiddle(hostName, 4, 4)),
		zap.Duration("delay", hm.trafficDisableGraceDuration))
	go time.AfterFunc(hm.trafficDisableGraceDuration, func() {
		if host, ok := hm.hosts.Load(hostName); ok {
			h := host.(*messages.Host)
			h.TrafficDisableScheduled = false
			hm.hosts.Store(hostName, h)
		}
		hm.disableTrafficForHost(hostName)
	})
}

// enableTrafficForHost enables the traffic for the host
func (hm *HostManager) enableTrafficForHost(hostName string) {
	if host, ok := hm.hosts.Load(hostName); ok && !host.(*messages.Host).TrafficAllowed {
		host.(*messages.Host).TrafficAllowed = true
		hm.hosts.Store(hostName, host)
		hm.logger.Debug("Enabled traffic for host", zap.Any("hostName", logger.MaskMiddle(hostName, 4, 4)))
		prom.TrafficSwitchCounter.WithLabelValues(hostName, "enabled").Inc()
	}
}

// k8sServiceHostRe matches only the Kubernetes service DNS forms
// "<service>.<namespace>.svc" and "<service>.<namespace>.svc.cluster.local", where each
// of <service> and <namespace> is a valid RFC 1123 DNS label. The previous catch-all
// patterns accepted any two-segment or single-segment string, which let attackers inject
// arbitrary Host headers (CWE-200); those are intentionally gone.
var k8sServiceHostRe = regexp.MustCompile(
	`^([a-z0-9]([-a-z0-9]*[a-z0-9])?)\.([a-z0-9]([-a-z0-9]*[a-z0-9])?)\.svc(?:\.cluster\.local)?$`,
)

// maxDNSLabelLength is the RFC 1123 limit for a single DNS label (service/namespace name).
const maxDNSLabelLength = 63

func (hm *HostManager) extractNamespaceAndService(rawHost string) (string, string, error) {
	// Strip scheme, path/wildcard, and port so only the DNS name remains.
	host := strings.TrimPrefix(rawHost, "http://")
	host = strings.TrimPrefix(host, "https://")
	if idx := strings.IndexByte(host, '/'); idx != -1 {
		host = host[:idx]
	}
	if idx := strings.LastIndexByte(host, ':'); idx != -1 {
		host = host[:idx]
	}

	matches := k8sServiceHostRe.FindStringSubmatch(host)
	if matches == nil {
		return "", "", fmt.Errorf("invalid Kubernetes service host: %s", logger.MaskMiddle(rawHost, 4, 4))
	}
	serviceName, namespace := matches[1], matches[3]
	if len(serviceName) > maxDNSLabelLength || len(namespace) > maxDNSLabelLength {
		return "", "", fmt.Errorf("invalid Kubernetes service host: %s", logger.MaskMiddle(rawHost, 4, 4))
	}
	return serviceName, namespace, nil
}

// addHTTPIfNeeded adds http if not present in the service URL
func (hm *HostManager) addHTTPIfNeeded(serviceURL string) string {
	if !strings.HasPrefix(serviceURL, "http://") && !strings.HasPrefix(serviceURL, "https://") {
		return "http://" + serviceURL
	}
	return serviceURL
}

// removeTrailingWildcardIfNeeded removes the trailing wildcard if present in the service URL
func (hm *HostManager) removeTrailingWildcardIfNeeded(serviceURL string) string {
	if strings.HasSuffix(serviceURL, "/*") {
		return strings.TrimSuffix(serviceURL, "/*")
	}
	return serviceURL
}

func (hm *HostManager) removeTrailingPathIfNeeded(serviceURL string) string {
	if idx := strings.Index(serviceURL, "/"); idx != -1 {
		return serviceURL[:idx]
	}
	return serviceURL
}

// replaceServiceName replaces the service name in the service URL
func (hm *HostManager) replaceServiceName(serviceURL, newServiceName string) string {
	parts := strings.Split(serviceURL, ".")
	if len(parts) < 3 {
		return serviceURL
	}
	parts[0] = newServiceName
	return strings.Join(parts, ".")
}
