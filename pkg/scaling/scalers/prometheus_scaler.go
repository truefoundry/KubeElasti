package scalers

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

const (
	httpClientTimeout   = 5 * time.Second
	uptimeQuery         = "min_over_time((max(up{%s}) or vector(0))[%ds:])"
	defaultUptimeFilter = "container=\"prometheus\""
)

type prometheusScaler struct {
	httpClient           *http.Client
	metadata             *prometheusMetadata
	cooldownPeriod       time.Duration
	defaultServerAddress string
	defaultHeaders       map[string]string
	// allowedServerAddresses is an optional admin-configured allowlist of hosts
	// (host or host:port) that a CRD-supplied serverAddress must match. When
	// empty, any serverAddress is accepted (still subject to the dial guard).
	allowedServerAddresses []string
}

type prometheusMetadata struct {
	ServerAddress string            `json:"serverAddress"`
	Query         string            `json:"query"`
	Threshold     float64           `json:"threshold,string"`
	UptimeFilter  string            `json:"uptimeFilter"`
	Headers       map[string]string `json:"headers"`
}

var promQueryResponse struct {
	Status string `json:"status"`
	Data   struct {
		ResultType string `json:"resultType"`
		Result     []struct {
			Metric map[string]string `json:"metric"`
			Value  []interface{}     `json:"value"`
		} `json:"result"`
	} `json:"data"`
}

func NewPrometheusScaler(metadata json.RawMessage, cooldownPeriod time.Duration) (Scaler, error) {
	parsedMetadata, err := parsePrometheusMetadata(metadata)
	if err != nil {
		return nil, fmt.Errorf("error creating prometheus scaler: %w", err)
	}

	allowedServerAddresses := fetchAllowedServerAddresses()

	return &prometheusScaler{
		metadata: parsedMetadata,
		// When an admin allowlist is configured it is the sole gate on the
		// destination (allowlist beats blocklist), so the dial guard is disabled
		// and an explicitly-allowed host is reachable even if it is otherwise
		// blocked. Without an allowlist, the dial guard is the protection.
		httpClient:             newHTTPClient(len(allowedServerAddresses) == 0),
		cooldownPeriod:         cooldownPeriod,
		defaultServerAddress:   os.Getenv("PROMETHEUS_TRIGGER_SERVER_ADDRESS"),
		defaultHeaders:         fetchDefaultHeaders(),
		allowedServerAddresses: allowedServerAddresses,
	}, nil
}

// newHTTPClient builds an http.Client hardened against SSRF. Redirects are
// rejected outright since the Prometheus query API never issues them. When
// dialGuard is true, the dialer also inspects the concrete IP being connected
// to (after DNS resolution, so hostname- and redirect-based bypasses are
// covered too) and refuses never-legitimate targets such as loopback,
// link-local, and multicast addresses.
func newHTTPClient(dialGuard bool) *http.Client {
	transport := &http.Transport{}
	if dialGuard {
		dialer := &net.Dialer{
			Control: func(_, address string, _ syscall.RawConn) error {
				host, _, err := net.SplitHostPort(address)
				if err != nil {
					return fmt.Errorf("failed to parse dial address %q: %w", address, err)
				}
				ip := net.ParseIP(host)
				if ip == nil {
					return fmt.Errorf("failed to parse dial IP %q", host)
				}
				if isBlockedIP(ip) {
					return fmt.Errorf("connection to %s blocked to prevent SSRF", ip)
				}
				return nil
			},
		}
		transport.DialContext = dialer.DialContext
	}

	return &http.Client{
		Timeout:   httpClientTimeout,
		Transport: transport,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return fmt.Errorf("redirects are not allowed")
		},
	}
}

// isBlockedIP reports whether an IP is a never-legitimate Prometheus target.
// Private ranges are intentionally allowed: an in-cluster Prometheus is reached
// via a private ClusterIP or a *.svc.cluster.local name resolving to one.
func isBlockedIP(ip net.IP) bool {
	return ip.IsLoopback() ||
		ip.IsUnspecified() ||
		ip.IsLinkLocalUnicast() ||
		ip.IsLinkLocalMulticast() ||
		ip.IsMulticast()
}

// fetchAllowedServerAddresses parses the optional operator allowlist.
func fetchAllowedServerAddresses() []string {
	raw := os.Getenv("PROMETHEUS_TRIGGER_ALLOWED_SERVER_ADDRESSES")
	if raw == "" {
		return nil
	}

	allowed := make([]string, 0)
	for _, addr := range strings.Split(raw, ",") {
		if trimmed := strings.TrimSpace(addr); trimmed != "" {
			allowed = append(allowed, trimmed)
		}
	}
	return allowed
}

func fetchDefaultHeaders() map[string]string {
	headers := make(map[string]string)

	authorizationHeader := os.Getenv("PROMETHEUS_TRIGGER_AUTHORIZATION_HEADER")
	if authorizationHeader != "" {
		headers["Authorization"] = authorizationHeader
	}

	return headers
}

func parsePrometheusMetadata(jsonMetadata json.RawMessage) (*prometheusMetadata, error) {
	metadata := &prometheusMetadata{}
	err := json.Unmarshal(jsonMetadata, metadata)
	if err != nil {
		return nil, fmt.Errorf("failed to parse metadata: %w", err)
	}
	return metadata, nil
}

// golang issue: https://github.com/golang/go/issues/4013
func queryEscape(query string) string {
	queryEscaped := url.QueryEscape(query)
	plusEscaped := strings.ReplaceAll(queryEscaped, "+", "%20")

	return plusEscaped
}

func (s *prometheusScaler) executePromQuery(ctx context.Context, query string) (float64, error) {
	t := time.Now().UTC().Format(time.RFC3339)
	queryEscaped := queryEscape(query)
	serverAddress, err := s.getServerAddress()
	if err != nil {
		return -1, err
	}
	if err := s.validateServerAddress(serverAddress); err != nil {
		return -1, err
	}
	queryURL := fmt.Sprintf("%s/api/v1/query?query=%s&time=%s", serverAddress, queryEscaped, t)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return -1, fmt.Errorf("failed to create HTTP request: %w", err)
	}

	// Apply default headers, then per-trigger metadata headers (which can override defaults)
	for key, value := range s.defaultHeaders {
		req.Header.Set(key, value)
	}
	for key, value := range s.metadata.Headers {
		req.Header.Set(key, value)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return -1, fmt.Errorf("failed to execute HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("unexpected HTTP status: %s", resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&promQueryResponse); err != nil {
		return -1, fmt.Errorf("failed to decode Prometheus response: %w", err)
	}

	var v float64 = -1

	if len(promQueryResponse.Data.Result) == 0 {
		return -1, fmt.Errorf("prometheus query %s, result is empty, prometheus metrics 'prometheus' target may be lost", query)
	} else if len(promQueryResponse.Data.Result) > 1 {
		return -1, fmt.Errorf("prometheus query %s returned multiple elements", query)
	}

	valueLen := len(promQueryResponse.Data.Result[0].Value)
	if valueLen == 0 {
		return -1, fmt.Errorf("prometheus query %s, value list in result is empty, prometheus metrics 'prometheus' target may be lost", s.metadata.Query)
	} else if valueLen < 2 {
		return -1, fmt.Errorf("prometheus query %s didn't return enough values", s.metadata.Query)
	}

	val := promQueryResponse.Data.Result[0].Value[1]
	if val != nil {
		str := val.(string)
		v, err = strconv.ParseFloat(str, 64)
		if err != nil {
			return -1, fmt.Errorf("failed to parse metric value: %w", err)
		}
	}

	if math.IsInf(v, 0) {
		return -1, fmt.Errorf("prometheus query returns %f", v)
	}

	return v, nil
}

// getServerAddress returns the effective Prometheus address: the CRD-supplied
// serverAddress when present, otherwise the operator-configured default.
func (s *prometheusScaler) getServerAddress() (string, error) {
	if s.metadata.ServerAddress != "" {
		return s.metadata.ServerAddress, nil
	}
	if s.defaultServerAddress == "" {
		return "", fmt.Errorf("prometheus serverAddress not configured")
	}
	return s.defaultServerAddress, nil
}

// validateServerAddress enforces the optional admin allowlist against the
// effective server address, matching on either host or host:port. The operator
// default is always trusted, and an empty allowlist accepts any address; only a
// non-default address absent from a configured allowlist is rejected.
func (s *prometheusScaler) validateServerAddress(serverAddress string) error {
	if serverAddress == s.defaultServerAddress || len(s.allowedServerAddresses) == 0 {
		return nil
	}

	u, err := url.Parse(serverAddress)
	if err != nil {
		return fmt.Errorf("failed to parse serverAddress %q: %w", serverAddress, err)
	}
	host := u.Hostname()
	hostPort := u.Host

	for _, allowed := range s.allowedServerAddresses {
		if allowed == host || allowed == hostPort {
			return nil
		}
	}
	return fmt.Errorf("serverAddress %q is not in the allowed list", serverAddress)
}

func (s *prometheusScaler) ShouldScaleToZero(ctx context.Context) (bool, error) {
	metricValue, err := s.executePromQuery(ctx, s.metadata.Query)
	if err != nil {
		return false, fmt.Errorf("failed to execute prometheus query %s: %w", s.metadata.Query, err)
	}

	if metricValue == -1 {
		return false, nil
	}
	if metricValue < s.metadata.Threshold {
		return true, nil
	}
	return false, nil
}

func (s *prometheusScaler) ShouldScaleFromZero(ctx context.Context) (bool, error) {
	metricValue, err := s.executePromQuery(ctx, s.metadata.Query)
	if err != nil {
		return true, fmt.Errorf("failed to execute prometheus query %s: %w", s.metadata.Query, err)
	}
	if metricValue == -1 {
		return true, nil
	}

	if metricValue >= s.metadata.Threshold {
		return true, nil
	}
	return false, nil
}

func (s *prometheusScaler) Close(_ context.Context) error {
	if s.httpClient != nil {
		s.httpClient.CloseIdleConnections()
	}
	return nil
}

func (s *prometheusScaler) IsHealthy(ctx context.Context) (bool, error) {
	uptimeFilter := s.metadata.UptimeFilter
	if uptimeFilter == "" {
		uptimeFilter = defaultUptimeFilter
	}

	cooldownPeriodSeconds := int(math.Ceil(s.cooldownPeriod.Seconds()))
	finalUptimeQuery := fmt.Sprintf(uptimeQuery, uptimeFilter, cooldownPeriodSeconds)

	metricValue, err := s.executePromQuery(
		ctx,
		finalUptimeQuery,
	)
	if err != nil {
		return false, fmt.Errorf("failed to execute prometheus query %s: %w", finalUptimeQuery, err)
	}
	return metricValue == 1, nil
}
