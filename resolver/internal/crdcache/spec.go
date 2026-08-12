package crdcache

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/kubeelasti/kubeelasti/operator/api/v1alpha1"
	"go.uber.org/zap"
)

const (
	MatchTypePathPrefix        = "PathPrefix"
	MatchTypeExact             = "Exact"
	MatchTypeRegularExpression = "RegularExpression"
)

// MatchProbeResponseFromSpec returns the configured response body and HTTP status when spec JSON
// matches the request using ElastiService CRD probeResponse semantics (path, headers, queryParams, method ANDed).
// Rules are tried in order; first match wins.
func MatchProbeResponseFromSpec(specJSON []byte, req *http.Request, logger *zap.Logger) (body string, status int, matched bool) {
	if req == nil || len(specJSON) == 0 {
		return "", 0, false
	}
	var s v1alpha1.ElastiServiceSpec
	if err := json.Unmarshal(specJSON, &s); err != nil {
		return "", 0, false
	}
	for _, r := range s.ProbeResponse {
		// Method Match
		if !methodMatchesProbeResponse(r.Method, req.Method) {
			continue
		}
		// Path Match
		if pathMatched, err := pathMatches(req.URL.Path, r.Path); err != nil {
			logger.Error("error matching path", zap.Error(err))
			continue
		} else if !pathMatched {
			continue
		}
		// Headers Match
		if headersMatched, err := headersMatch(r.Headers, req.Header); err != nil {
			logger.Error("error matching headers", zap.Error(err))
			continue
		} else if !headersMatched {
			continue
		}
		if !queryParamsMatch(r.QueryParams, req.URL.Query()) {
			continue
		}
		// Status is enum-validated on the CRD; 0 only appears if JSON omits status (unmarshal zero value).
		st := r.Response.Status
		if st == 0 {
			st = http.StatusOK
		}
		return r.Response.Body, st, true
	}
	return "", 0, false
}

func methodMatchesProbeResponse(ruleMethod *string, requestMethod string) bool {
	if ruleMethod == nil || strings.TrimSpace(*ruleMethod) == "" {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(*ruleMethod), requestMethod)
}

func pathMatches(reqPath string, p *v1alpha1.ProbeResponsePathMatch) (bool, error) {
	pm := normalizeProbePathMatch(p)
	switch canonicalPathMatchType(pm.Type) {
	case "Exact":
		return reqPath == pm.Value, nil
	case "RegularExpression":
		re, err := regexp.Compile(pm.Value)
		if err != nil {
			return false, fmt.Errorf("probeResponse path regex invalid: %w", err)
		}
		return re.MatchString(reqPath), nil
	default: // PathPrefix
		return pathPrefix(reqPath, pm.Value), nil
	}
}

type probePathMatch struct {
	Type  string
	Value string
}

func normalizeProbePathMatch(p *v1alpha1.ProbeResponsePathMatch) probePathMatch {
	if p == nil {
		return probePathMatch{Type: MatchTypePathPrefix, Value: "/"}
	}
	typ := strings.TrimSpace(p.Type)
	if typ == "" {
		typ = MatchTypePathPrefix
	}
	val := p.Value
	if canonicalPathMatchType(typ) == MatchTypePathPrefix && val == "" {
		val = "/"
	}
	return probePathMatch{Type: typ, Value: val}
}

func canonicalPathMatchType(t string) string {
	ts := strings.TrimSpace(t)
	switch {
	case ts == "", strings.EqualFold(ts, MatchTypePathPrefix):
		return MatchTypePathPrefix
	case strings.EqualFold(ts, MatchTypeExact):
		return MatchTypeExact
	case strings.EqualFold(ts, MatchTypeRegularExpression):
		return MatchTypeRegularExpression
	default:
		return MatchTypePathPrefix
	}
}

// pathPrefix implements segment-aware PathPrefix matching.
// HTTPPathMatch: /foo matches /foo and /foo/bar but not /foobar; "/" matches any path with a
// leading slash.
func pathPrefix(path, prefix string) bool {
	if prefix == "" || prefix == "/" {
		return strings.HasPrefix(path, "/")
	}
	if path == prefix {
		return true
	}
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	if strings.HasSuffix(prefix, "/") {
		return true
	}
	return len(path) > len(prefix) && path[len(prefix)] == '/'
}

// headersMatch implements Gateway-style AND semantics: every rule in the slice must match the
// request (same header name may appear once per rule; multiple values for a name are ORed inside
// headerValueMatches).
func headersMatch(rules []v1alpha1.ProbeResponseHeaderMatch, h http.Header) (bool, error) {
	for _, rule := range rules {
		if strings.TrimSpace(rule.Name) == "" {
			return false, fmt.Errorf("header name is empty")
		}
		ok, err := headerValueMatches(rule, h)
		if err != nil {
			return false, err
		}
		if !ok {
			// This probe rule does not match
			return false, nil
		}
	}
	return true, nil
}

func headerValueMatches(rule v1alpha1.ProbeResponseHeaderMatch, h http.Header) (bool, error) {
	key := http.CanonicalHeaderKey(strings.TrimSpace(rule.Name))
	values := h.Values(key)
	if len(values) == 0 {
		return false, nil
	}
	typ := strings.TrimSpace(rule.Type)
	switch {
	case typ == "" || strings.EqualFold(typ, MatchTypeExact):
		for _, v := range values {
			if v == rule.Value {
				return true, nil
			}
		}
		return false, nil
	case strings.EqualFold(typ, MatchTypeRegularExpression):
		re, err := regexp.Compile(rule.Value)
		if err != nil {
			return false, fmt.Errorf("probeResponse header regex invalid: %w: header: %s, pattern: %s", err, rule.Name, rule.Value)
		}
		for _, v := range values {
			if re.MatchString(v) {
				return true, nil
			}
		}
		return false, nil
	default:
		for _, v := range values {
			if v == rule.Value {
				return true, nil
			}
		}
		return false, nil
	}
}

func queryParamsMatch(rules []v1alpha1.ProbeResponseQueryParamMatch, q url.Values) bool {
	for _, rule := range rules {
		if strings.TrimSpace(rule.Name) == "" {
			return false
		}
		if !queryParamValueMatches(rule, q) {
			return false
		}
	}
	return true
}

func queryParamValueMatches(rule v1alpha1.ProbeResponseQueryParamMatch, q url.Values) bool {
	values := q[rule.Name]
	if len(values) == 0 {
		return false
	}
	typ := strings.TrimSpace(rule.Type)
	switch {
	case typ == "" || strings.EqualFold(typ, MatchTypeExact):
		for _, v := range values {
			if v == rule.Value {
				return true
			}
		}
		return false
	case strings.EqualFold(typ, MatchTypeRegularExpression):
		re, err := regexp.Compile(rule.Value)
		if err != nil {
			return false
		}
		for _, v := range values {
			if re.MatchString(v) {
				return true
			}
		}
		return false
	default:
		for _, v := range values {
			if v == rule.Value {
				return true
			}
		}
		return false
	}
}
