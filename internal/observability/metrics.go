package observability

// Prometheus 指标定义。Phase 8 实现。
//
// 计划指标：
//   - competify_agent_execution_duration_seconds (Histogram, labels: agent_name, status)
//   - competify_llm_tokens_total (Counter, labels: model_name, token_type)
//   - competify_dag_node_status (Gauge, labels: task_id, node_name)
//   - competify_provenance_confidence (Histogram, buckets: 0/0.2/0.4/0.6/0.7/0.8/0.9/1.0)
//   - competify_merkle_tree_depth (Gauge)
