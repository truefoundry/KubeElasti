package prom

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// HostExtractionCounter deliberately carries no request-derived free-text labels
	// (raw Host header, error string). Those are attacker-controlled and would both leak
	// service/namespace info through /metrics and blow up label cardinality (CWE-200).
	// "reason" is a fixed, bounded classification set by the resolver.
	HostExtractionCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "elasti_resolver_host_extraction_count",
			Help: "Counter for host extraction",
		},
		[]string{
			"extractionType",
			"reason",
		},
	)

	QueuedRequestGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "elasti_resolver_queued_count",
			Help: "Gauge for queued requests",
		},
		[]string{
			"source",
			"namespace",
		},
	)

	// IncomingRequestHistogram intentionally omits raw request-derived labels
	// (sourceHost, targetHost, requestURI). requestURI is fully attacker-controlled and
	// unbounded; the host URLs echo internal DNS/naming conventions. "source"/"target"/
	// "namespace" are validated against a known ElastiService before this is recorded, so
	// they are bounded and safe to expose.
	IncomingRequestHistogram = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "elasti_resolver_incoming_requests",
			Help:    "Histogram of response latency (seconds) for every request resolved",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10},
		},
		[]string{"source",
			"target",
			"namespace",
			"method",
			"status",
			"error"},
	)

	TrafficSwitchCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "elasti_resolver_traffic_switch_count",
			Help: "Counter for traffic switch",
		},
		[]string{"source", "enabled"},
	)

	OperatorRPCCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "elasti_resolver_operator_rpc_count",
			Help: "Counter for operator RPC",
		},
		[]string{"error"},
	)
)
