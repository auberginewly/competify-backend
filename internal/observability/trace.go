package observability

import (
	"context"
	"time"
)

// TraceAgentExecution wraps an Agent's Execute call in a span.
// Phase 8: real OTel implementation.
func TraceAgentExecution(ctx context.Context, agentName, taskID string, fn func(context.Context) error) error {
	// TODO Phase 8: tracer.Start(ctx, "agent.execute", ...) + span.End
	return fn(ctx)
}

// TraceLLMCall records LLM invocation metrics (model / tokens / latency).
func TraceLLMCall(ctx context.Context, modelName string, inputTokens, outputTokens int, latency time.Duration) {
	// TODO Phase 8
}

// TraceStorageAccess records DB/cache access timing.
func TraceStorageAccess(ctx context.Context, storeType, operation string, latency time.Duration) {
	// TODO Phase 8
}
