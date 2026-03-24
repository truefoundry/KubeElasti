package messages

import "encoding/json"

type RequestCount struct {
	Count     int    `json:"count"`
	Svc       string `json:"svc"`
	Namespace string `json:"namespace"`
}

type ElastiServiceEntry struct {
	Name   string          `json:"name"`
	Spec   json.RawMessage `json:"spec"`
	Status json.RawMessage `json:"status"`
}

type ElastiServiceCacheResponse struct {
	Services map[string]ElastiServiceEntry `json:"services"`
}
