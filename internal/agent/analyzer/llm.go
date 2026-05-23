// Package analyzer provides shared LLM calling logic for all dimension analyzers.
package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	einoSchema "github.com/cloudwego/eino/schema"
)

// analyzeWithLLM calls the shared ChatModel with a dimension-specific prompt.
// It returns structured findings, reasoning, and a confidence score.
// If the model is nil or MOCK_LLM is set, it returns an error so the caller can fall back to stub data.
func analyzeWithLLM(
	ctx context.Context,
	model *openai.ChatModel,
	dimension string,
	competitor string,
	cleanedText string,
) (payload string, reasoning string, score float64, err error) {
	if os.Getenv("MOCK_LLM") == "true" || model == nil {
		return "", "", 0, fmt.Errorf("llm: mock mode or model not initialized")
	}

	systemPrompt := fmt.Sprintf(
		`You are a competitive-intelligence %s analyst.
Analyze the following data about "%s" and output a JSON object with exactly these fields:
- "findings": concise summary (max 200 characters)
- "reasoning": short chain-of-thought (max 300 characters)
- "score": confidence score from 0.0 to 1.0
Output JSON only, no markdown fences.`,
		dimension, competitor,
	)

	resp, err := model.Generate(ctx, []*einoSchema.Message{
		{Role: einoSchema.System, Content: systemPrompt},
		{Role: einoSchema.User, Content: cleanedText},
	})
	if err != nil {
		return "", "", 0, fmt.Errorf("llm generate: %w", err)
	}

	content := strings.TrimSpace(resp.Content)
	// Strip optional markdown fences.
	content = stripMarkdownFences(content)

	var parsed struct {
		Findings  string  `json:"findings"`
		Reasoning string  `json:"reasoning"`
		Score     float64 `json:"score"`
	}
	if err := json.Unmarshal([]byte(content), &parsed); err != nil {
		// Fallback: treat the entire response as payload with neutral score.
		return content, "Raw LLM output (JSON parse failed)", 0.5, nil
	}

	return parsed.Findings, parsed.Reasoning, parsed.Score, nil
}

func stripMarkdownFences(s string) string {
	s = strings.TrimSpace(s)
	if strings.HasPrefix(s, "```json") {
		s = strings.TrimPrefix(s, "```json")
		s = strings.TrimSuffix(s, "```")
	} else if strings.HasPrefix(s, "```") {
		s = strings.TrimPrefix(s, "```")
		s = strings.TrimSuffix(s, "```")
	}
	return strings.TrimSpace(s)
}
