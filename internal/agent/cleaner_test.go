package agent

import (
	"context"
	"strings"
	"testing"

	"github.com/competify-ai/competify-backend/internal/schema"
	"github.com/stretchr/testify/assert"
)

func TestCleaner_Deduplicate(t *testing.T) {
	c := NewCleaner()
	packs := map[string]*schema.RawDataPack{
		"web":    {TaskID: "t1", RawContent: "Hello world"},
		"api":    {TaskID: "t1", RawContent: "Hello world"}, // duplicate
		"social": {TaskID: "t1", RawContent: "Different content"},
	}

	out, err := c.Execute(context.Background(), packs)
	assert.NoError(t, err)

	ds := out.(*schema.NormalizedDataset)
	assert.NotNil(t, ds)
	assert.Equal(t, "t1", ds.TaskID)
	assert.NotZero(t, ds.SimHashValue)

	// Should dedup identical web + api content, leaving 2 unique sources.
	assert.Equal(t, 2, ds.Structured["unique_sources"])
	assert.Equal(t, 3, ds.Structured["total_sources"])
	assert.True(t, strings.Contains(ds.CleanedText, "Different content"))
}

func TestCleaner_SinglePack(t *testing.T) {
	c := NewCleaner()
	pack := &schema.RawDataPack{TaskID: "t2", RawContent: "Single pack content"}
	out, err := c.Execute(context.Background(), pack)
	assert.NoError(t, err)

	ds := out.(*schema.NormalizedDataset)
	assert.Equal(t, "Single pack content", ds.CleanedText)
	assert.Equal(t, "aggregated", ds.SourceType)
}

func TestCleaner_EmptyMap(t *testing.T) {
	c := NewCleaner()
	_, err := c.Execute(context.Background(), map[string]*schema.RawDataPack{})
	assert.Error(t, err)
}
