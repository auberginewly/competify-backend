package observability

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Package-level Prometheus metrics. Registered with promauto on import.
// Naming convention: competify_<subsystem>_<metric>_<unit>.
var (
	// AgentDuration measures each Agent.Execute duration (labels: agent_name, status).
	AgentDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "competify_agent_execution_duration_seconds",
			Help:    "Histogram of Agent Execute() latency.",
			Buckets: []float64{0.1, 0.5, 1, 2, 5, 10, 30, 60},
		},
		[]string{"agent_name", "status"},
	)

	// LLMTokensTotal counts input/output tokens per LLM call (labels: model_name, token_type).
	LLMTokensTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "competify_llm_tokens_total",
			Help: "Cumulative LLM token usage.",
		},
		[]string{"model_name", "token_type"},
	)

	// DAGNodeStatus tracks running/done/error per DAG node (labels: task_id, node_name, status).
	DAGNodeStatus = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "competify_dag_node_status",
			Help: "Current status gauge for each DAG node (1=active, 0=idle).",
		},
		[]string{"task_id", "node_name", "status"},
	)

	// ProvenanceConfidence records per-conclusion confidence scores.
	ProvenanceConfidence = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "competify_provenance_confidence",
			Help:    "Distribution of per-conclusion confidence scores.",
			Buckets: []float64{0, 0.2, 0.4, 0.6, 0.7, 0.8, 0.9, 1.0},
		},
	)

	// MerkleTreeDepth tracks the depth of the Merkle tree per report.
	MerkleTreeDepth = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "competify_merkle_tree_depth",
			Help: "Depth of the last generated Merkle tree.",
		},
	)

	// CompetitorEventsTotal counts published NATS competitor events by type.
	CompetitorEventsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "competify_competitor_events_total",
			Help: "Total NATS CompetitorEvents published by event type.",
		},
		[]string{"event_type"},
	)
)
