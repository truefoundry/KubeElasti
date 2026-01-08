package scalers

import (
	"context"
	"encoding/json"
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

type ErrorCode string
type ExecutionError struct {
	error error
	code  ErrorCode
}

var (
	ErrCreatePrometheusScaler ErrorCode = "error creating prometheus scaler"
	ErrParseMetadata          ErrorCode = "failed to parse metadata"
	ErrCreateHTTPRequest      ErrorCode = "failed to create HTTP request"
	ErrExecuteHTTPRequest     ErrorCode = "failed to execute HTTP request"
	ErrUnexpectedHTTPStatus   ErrorCode = "unexpected HTTP status"
	ErrDecodePrometheusResp   ErrorCode = "failed to decode Prometheus response"
	ErrEmptyQueryResult       ErrorCode = "prometheus query result is empty, prometheus metrics 'prometheus' target may be lost"
	ErrMultipleQueryResults   ErrorCode = "prometheus query returned multiple elements"
	ErrEmptyValueList         ErrorCode = "prometheus query value list in result is empty, prometheus metrics 'prometheus' target may be lost"
	ErrInsufficientValues     ErrorCode = "prometheus query didn't return enough values"
	ErrParseMetricValue       ErrorCode = "failed to parse metric value"
	ErrInfiniteValue          ErrorCode = "prometheus query returns infinite value"
	ErrExecutePrometheusQuery ErrorCode = "failed to execute prometheus query"
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

func (s *prometheusScaler) executePromQuery(ctx context.Context, query string) (float64, ExecutionError) {
	t := time.Now().UTC().Format(time.RFC3339)
	queryEscaped := queryEscape(query)
	queryURL := fmt.Sprintf("%s/api/v1/query?query=%s&time=%s", s.metadata.ServerAddress, queryEscaped, t)

	req, err := http.NewRequestWithContext(ctx, "GET", queryURL, nil)
	if err != nil {
		return -1, ExecutionError{fmt.Errorf("%w: %w", ErrCreateHTTPRequest, err), ErrCreateHTTPRequest}
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return -1, ExecutionError{
			error: fmt.Errorf("%s: %w", ErrExecuteHTTPRequest, err),
			code:  ErrExecuteHTTPRequest,
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return -1, ExecutionError{
			error: fmt.Errorf("%s: %s", ErrUnexpectedHTTPStatus, resp.Status),
			code:  ErrUnexpectedHTTPStatus,
		}
	}

	if err := json.NewDecoder(resp.Body).Decode(&promQueryResponse); err != nil {
		return -1, ExecutionError{
			error: fmt.Errorf("%s: %w", ErrDecodePrometheusResp, err),
			code:  ErrDecodePrometheusResp,
		}
	}

	var v float64 = -1

	if len(promQueryResponse.Data.Result) == 0 {
		return -1, ExecutionError{
			error: fmt.Errorf("%s: %s", ErrEmptyQueryResult, query),
			code:  ErrEmptyQueryResult,
		}
	} else if len(promQueryResponse.Data.Result) > 1 {
		return -1, ExecutionError{
			error: fmt.Errorf("%s: %s", ErrMultipleQueryResults, query),
			code:  ErrMultipleQueryResults,
		}
	}

	valueLen := len(promQueryResponse.Data.Result[0].Value)
	if valueLen == 0 {
		return -1, ExecutionError{
			error: fmt.Errorf("%s: %s", ErrEmptyValueList, s.metadata.Query),
			code:  ErrEmptyValueList,
		}
	} else if valueLen < 2 {
		return -1, ExecutionError{
			error: fmt.Errorf("%s: %s", ErrInsufficientValues, s.metadata.Query),
			code:  ErrInsufficientValues,
		}
	}

	val := promQueryResponse.Data.Result[0].Value[1]
	if val != nil {
		str := val.(string)
		v, err = strconv.ParseFloat(str, 64)
		if err != nil {
			return -1, ExecutionError{
				error: fmt.Errorf("%s: %w", ErrParseMetricValue, err),
				code:  ErrParseMetricValue,
			}
		}
	}

	if math.IsInf(v, 0) {
		return -1, ExecutionError{
			error: fmt.Errorf("%s: %f", ErrInfiniteValue, v),
			code:  ErrInfiniteValue,
		}
	}

	return v, ExecutionError{}
}

func (s *prometheusScaler) ShouldScaleToZero(ctx context.Context) (bool, error) {
	metricValue, err := s.executePromQuery(ctx, s.metadata.Query)
	if err.error != nil {
		return false, fmt.Errorf("%s %s: %w", ErrExecutePrometheusQuery, s.metadata.Query, err.error)
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
	if err.error != nil {
		return true, fmt.Errorf("%s %s: %w", ErrExecutePrometheusQuery, s.metadata.Query, err.error)
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
	if err.error != nil {
		// Only return HTTP request and status errors, ignore all other errors
		errCode := err.code
		if errCode == ErrCreateHTTPRequest ||
			errCode == ErrExecuteHTTPRequest ||
			errCode == ErrUnexpectedHTTPStatus {
			return false, fmt.Errorf("%s %s: %w", errCode, finalUptimeQuery, err.error)
		}
		// Ignore all other errors and return healthy
		return true, nil
	}
	return metricValue == 1, nil
}
