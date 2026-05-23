// Package analyzer provides shared LLM calling logic for all dimension analyzers.
package analyzer

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/cloudwego/eino-ext/components/model/openai"
	einoSchema "github.com/cloudwego/eino/schema"
)

// competitorFromTaskID extracts the competitor name from a task ID.
// Task IDs have the form "task_<CompetitorName>_<unix_timestamp>".
func competitorFromTaskID(taskID string) string {
	s := strings.TrimPrefix(taskID, "task_")
	if idx := strings.LastIndex(s, "_"); idx > 0 {
		return s[:idx]
	}
	return s
}

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
		`你是一名竞品情报%s分析师。
分析以下关于"%s"的数据，输出包含以下字段的JSON对象：
- "findings": 简洁分析结论（最多200字，使用中文）
- "reasoning": 简短推理过程（最多150字，使用中文）
- "score": 置信度分数，0.0到1.0之间
只输出JSON，不要markdown代码块。`,
		dimension, competitor,
	)

	resp, err := model.Generate(ctx, []*einoSchema.Message{
		{Role: einoSchema.System, Content: systemPrompt},
		{Role: einoSchema.User, Content: cleanedText},
	})
	if err != nil {
		log.Printf("[LLM] analyzeWithLLM failed for %s/%s: %v", dimension, competitor, err)
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
