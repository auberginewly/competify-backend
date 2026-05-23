package analyzer

import (
	"context"
	"testing"

	"github.com/competify-ai/competify-backend/internal/schema"
	"github.com/stretchr/testify/assert"
)

func TestFeatureAnalyzer_MockFallback(t *testing.T) {
	// nil model triggers stub fallback.
	a := NewFeatureAnalyzer(nil)
	ds := &schema.NormalizedDataset{TaskID: "t1", CleanedText: "test data", VikingURI: "viking://test"}
	out, err := a.Execute(context.Background(), ds)
	assert.NoError(t, err)

	result := out.(*schema.AnalysisResult)
	assert.Equal(t, "feature", result.Dimension)
	assert.NotEmpty(t, result.Payload)
	assert.Greater(t, result.Score, 0.0)
	assert.Equal(t, "openai", result.ModelName)
}
