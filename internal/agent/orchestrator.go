// Orchestrator parses UserQuery into TaskDAGPlan.
package agent

import (
	"context"
	"fmt"
	"time"

	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
)

// Orchestrator generates the execution plan from a user's natural-language request.
type Orchestrator struct {
	BaseAgent
}

// NewOrchestrator creates an Orchestrator with an optional audit chain.
func NewOrchestrator(auditChain ...*provenance.AuditChain) *Orchestrator {
	ba := BaseAgent{Role: "orchestrator"}
	if len(auditChain) > 0 {
		ba.AuditChain = auditChain[0]
	}
	return &Orchestrator{BaseAgent: ba}
}

func (o *Orchestrator) Name() string { return "orchestrator" }

// Execute parses the UserQuery and returns a TaskDAGPlan.
// In Phase 2 stub it returns a deterministic plan so the pipeline can run end-to-end.
func (o *Orchestrator) Execute(ctx context.Context, input interface{}) (interface{}, error) {
	q, ok := input.(*schema.UserQuery)
	if !ok {
		return nil, fmt.Errorf("orchestrator: expected *schema.UserQuery, got %T", input)
	}

	plan := &schema.TaskDAGPlan{
		TaskID:         fmt.Sprintf("task_%s_%d", q.CompetitorName, time.Now().Unix()),
		CompetitorName: q.CompetitorName,
		Dimensions:     q.Dimensions,
		TargetSchema:   "v1.0",
		CreatedAt:      time.Now().UTC(),
		ExpiresAt:      time.Now().UTC().Add(24 * time.Hour),
	}

	// Light-weight rule engine: map requested dimensions to required collectors.
	for _, d := range q.Dimensions {
		switch d {
		case "feature", "pricing":
			plan.RequiresWebScrape = true
		case "tech":
			plan.RequiresAPI = true
			plan.RequiresWebScrape = true
		case "market":
			plan.RequiresSocial = true
			plan.RequiresWebScrape = true
		}
	}
	plan.RequiresReview = true // always run cross-reviewer for quality
	plan.RequiresFinancial = q.CompetitorName != ""

	o.RecordAudit(plan.TaskID, o.GenerateID(plan.TaskID),
		fmt.Sprintf("%+v", q), fmt.Sprintf("%+v", plan),
		"Orchestrator generated plan", 0.95)

	return plan, nil
}

func (o *Orchestrator) HealthCheck(ctx context.Context) error { return nil }
