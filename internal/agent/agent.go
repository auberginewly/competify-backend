// Package agent defines the unified Agent interface and BaseAgent.
// Every Agent role (Orchestrator / Collector / Cleaner / ... ) embeds BaseAgent.
package agent

import (
	"context"
	"fmt"

	"github.com/competify-ai/competify-backend/internal/provenance"
)

// Agent is the unified interface every Agent role must implement.
type Agent interface {
	Name() string
	Execute(ctx context.Context, input interface{}) (interface{}, error)
	HealthCheck(ctx context.Context) error
}

// BaseAgent provides shared infrastructure for all agents.
type BaseAgent struct {
	ID         string
	Role       string
	AuditChain *provenance.AuditChain
}

// RecordAudit wraps the execution result into the append-only audit chain.
// Concrete agents should defer-call this at the end of Execute.
func (b *BaseAgent) RecordAudit(taskID, nodeID, inputData, outputData, reasoning string, confidence float64) {
	if b.AuditChain == nil {
		return
	}
	b.AuditChain.Append(b.Role, taskID, nodeID, inputData, outputData, reasoning, confidence)
}

// GenerateID creates a deterministic node ID for a given task.
func (b *BaseAgent) GenerateID(taskID string) string {
	return fmt.Sprintf("%s_%s", b.Role, taskID)
}
