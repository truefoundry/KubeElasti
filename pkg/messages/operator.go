package messages

import "encoding/json"

type RequestCount struct {
	Count     int    `json:"count"`
	Svc       string `json:"svc"`
	Namespace string `json:"namespace"`
}

type CRDCacheEntry struct {
	CRDName string          `json:"crdName"`
	Spec    json.RawMessage `json:"spec"`
	Status  json.RawMessage `json:"status"`
}

type CRDCacheResponse struct {
	CRDCache map[string]CRDCacheEntry `json:"crdCache"`
}
