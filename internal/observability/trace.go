package observability

import (
	"context"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// TraceAgentExecution wraps fn in an OTel span named "agent.execute".
// Records agent_name and task_id as span attributes; sets error status on failure.
func TraceAgentExecution(ctx context.Context, agentName, taskID string, fn func(context.Context) error) error {
	tracer := otel.Tracer(tracerName)
	ctx, span := tracer.Start(ctx, "agent.execute")
	defer span.End()

	span.SetAttributes(
		attribute.String("agent.name", agentName),
		attribute.String("task.id", taskID),
	)

	start := time.Now()
	err := fn(ctx)
	elapsed := time.Since(start)

	AgentDuration.WithLabelValues(agentName, statusLabel(err)).Observe(elapsed.Seconds())

	if err != nil {
		span.RecordError(err)
		span.SetStatus(codes.Error, err.Error())
	} else {
		span.SetStatus(codes.Ok, "")
	}
	return err
}

// TraceLLMCall records a span + Prometheus counter for a single LLM invocation.
func TraceLLMCall(ctx context.Context, modelName string, inputTokens, outputTokens int, latency time.Duration) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "llm.call")
	defer span.End()

	span.SetAttributes(
		attribute.String("llm.model", modelName),
		attribute.Int("llm.input_tokens", inputTokens),
		attribute.Int("llm.output_tokens", outputTokens),
		attribute.Float64("llm.latency_ms", float64(latency.Milliseconds())),
	)

	LLMTokensTotal.WithLabelValues(modelName, "input").Add(float64(inputTokens))
	LLMTokensTotal.WithLabelValues(modelName, "output").Add(float64(outputTokens))
}

// TraceStorageAccess records a span for a storage operation.
func TraceStorageAccess(ctx context.Context, storeType, operation string, latency time.Duration) {
	tracer := otel.Tracer(tracerName)
	_, span := tracer.Start(ctx, "storage.access")
	defer span.End()

	span.SetAttributes(
		attribute.String("storage.type", storeType),
		attribute.String("storage.operation", operation),
		attribute.Float64("storage.latency_ms", float64(latency.Milliseconds())),
	)
}

func statusLabel(err error) string {
	if err != nil {
		return "error"
	}
	return "ok"
}
