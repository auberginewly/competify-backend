// Package handler implements HTTP and WebSocket routes.
// Routes call dag.Runnable or agent layers — never access storage directly.
// See docs/api.md for the full API contract.
package handler

import (
	"context"
	"fmt"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/competify-ai/competify-backend/internal/schema"
)

type createTaskResponse struct {
	TaskID string `json:"task_id"`
}

type taskStatus struct {
	TaskID    string `json:"task_id"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
}

// CreateTask handles POST /api/v1/tasks.
func CreateTask(ctx context.Context, c *app.RequestContext) {
	var req schema.UserQuery
	if err := c.Bind(&req); err != nil {
		c.JSON(400, map[string]string{"message": "invalid request body"})
		return
	}

	taskID := fmt.Sprintf("task_%s_%d", req.CompetitorName, time.Now().Unix())
	c.JSON(201, createTaskResponse{TaskID: taskID})
}

// GetTask handles GET /api/v1/tasks/:id.
func GetTask(ctx context.Context, c *app.RequestContext) {
	id := c.Param("id")
	c.JSON(200, taskStatus{
		TaskID:    id,
		Status:    "running",
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	})
}
