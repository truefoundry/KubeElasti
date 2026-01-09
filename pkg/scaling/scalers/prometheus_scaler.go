package scalers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const (
	httpClientTimeout   = 5 * time.Second
	uptimeQuery         = "min_over_time((max(up{%s}) or vector(0))[%ds:])"
	defaultUptimeFilter = "container=\"prometheus\""
)

// Predefined error variables
var (
	ErrCreatePrometheusScaler = errors.New("error creating prometheus scaler")
	ErrParseMetadata          = errors.New("failed to parse metadata")
	ErrCreateHTTPRequest      = errors.New("failed to create HTTP request")
	ErrExecuteHTTPRequest     = errors.New("failed to execute HTTP request")
	ErrUnexpectedHTTPStatus   = errors.New("unexpected HTTP status")
	ErrDecodePrometheusResp   = errors.New("failed to decode Prometheus response")
	ErrEmptyQueryResult       = errors.New("prometheus query result is empty")
	ErrMultipleQueryResults   = errors.New("prometheus query returned multiple elements")
	ErrEmptyValueList         = errors.New("prometheus query value list is empty")
	ErrInsufficientValues     = errors.New("prometheus query didn't return enough values")
	ErrParseMetricValue       = errors.New("failed to parse metric value")
	ErrInfiniteValue          = errors.New("prometheus query returns infinite value")
	ErrExecutePrometheusQuery = errors.New("failed to execute prometheus query")
)

type prometheusScaler struct {
	httpClient     *http.Client
	metadata       *prometheusMetadata
	cooldownPeriod time.Duration
}

type prometheusMetadata struct {
	ServerAddress string  `json:"serverAddress"`
	Query         string  `json:"query"`
	Threshold     float64 `json:"threshold,string"`
	UptimeFilter  string  `json:"uptimeFilter"`
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
		return nil, fmt.Errorf("%w: %w", ErrCreatePrometheusScaler, err)
	}

	client := &http.Client{
		Timeout: httpClientTimeout,
	}

	return &prometheusScaler{
		metadata:       parsedMetadata,
		httpClient:     client,
		cooldownPeriod: cooldownPeriod,
	}, nil
}

func parsePrometheusMetadata(jsonMetadata json.RawMessage) (*prometheusMetadata, error) {
	metadata := &prometheusMetadata{}
	err := json.Unmarshal(jsonMetadata, metadata)
	if err != nil {
		return nil, fmt.Errorf("%w: %w", ErrParseMetadata, err)
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
	queryURL := fmt.Sprintf("%s/api/v1/query?query=%s&time=%s", s.metadata.ServerAddress, queryEscaped, t)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return -1, fmt.Errorf("%w: %w", ErrCreateHTTPRequest, err)
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return -1, fmt.Errorf("%w: %w", ErrExecuteHTTPRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return -1, fmt.Errorf("%w: %s", ErrUnexpectedHTTPStatus, resp.Status)
	}

	if err := json.NewDecoder(resp.Body).Decode(&promQueryResponse); err != nil {
		return -1, fmt.Errorf("%w: %w", ErrDecodePrometheusResp, err)
	}

	var v float64 = -1

	if len(promQueryResponse.Data.Result) == 0 {
		return -1, fmt.Errorf("%w: %s", ErrEmptyQueryResult, query)
	} else if len(promQueryResponse.Data.Result) > 1 {
		return -1, fmt.Errorf("%w: %s", ErrMultipleQueryResults, query)
	}

	valueLen := len(promQueryResponse.Data.Result[0].Value)
	if valueLen == 0 {
		return -1, fmt.Errorf("%w: %s", ErrEmptyValueList, s.metadata.Query)
	} else if valueLen < 2 {
		return -1, fmt.Errorf("%w: %s", ErrInsufficientValues, s.metadata.Query)
	}

	val := promQueryResponse.Data.Result[0].Value[1]
	if val != nil {
		str := val.(string)
		v, err = strconv.ParseFloat(str, 64)
		if err != nil {
			return -1, fmt.Errorf("%w: %w", ErrParseMetricValue, err)
		}
	}

	if math.IsInf(v, 0) {
		return -1, fmt.Errorf("%w: %f", ErrInfiniteValue, v)
	}

	return v, nil
}

func (s *prometheusScaler) ShouldScaleToZero(ctx context.Context) (bool, error) {
	metricValue, err := s.executePromQuery(ctx, s.metadata.Query)
	if err != nil {
		return false, fmt.Errorf("%s %s: %w", ErrExecutePrometheusQuery, s.metadata.Query, err)
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
		return true, fmt.Errorf("%s %s: %w", ErrExecutePrometheusQuery, s.metadata.Query, err)
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
		// Only return HTTP errors, ignore query and data parsing errors
		if errors.Is(err, ErrCreateHTTPRequest) ||
			errors.Is(err, ErrExecuteHTTPRequest) ||
			errors.Is(err, ErrUnexpectedHTTPStatus) {
			return false, fmt.Errorf("prometheus health check failed: %w", err)
		}
		// Ignore non-HTTP errors and return healthy
		return true, nil
	}
	return metricValue == 1, nil
}
