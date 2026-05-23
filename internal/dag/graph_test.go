package dag

import (
	"context"
	"testing"

	"github.com/competify-ai/competify-backend/internal/provenance"
	"github.com/competify-ai/competify-backend/internal/schema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCompetifyGraph_Compile(t *testing.T) {
	auditChain := provenance.NewAuditChain()
	agents, err := BuildAllAgents(nil, auditChain, nil)
	require.NoError(t, err)
	require.NotNil(t, agents)

	runnable, err := BuildRunner(agents)
	require.NoError(t, err)
	assert.NotNil(t, runnable)
}

func TestExecuteDAG_MockMode(t *testing.T) {
	auditChain := provenance.NewAuditChain()
	agents, err := BuildAllAgents(nil, auditChain, nil)
	require.NoError(t, err)

	runnable, err := BuildRunner(agents)
	require.NoError(t, err)

	output, err := ExecuteDAG(context.Background(), runnable, schema.UserQuery{
		CompetitorName: "TestCo",
		Dimensions:     []string{"feature"},
	})
	require.NoError(t, err)
	assert.NotNil(t, output)
	assert.NotEmpty(t, output.Content)
}
