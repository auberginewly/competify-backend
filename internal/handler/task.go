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

// CreateTaskHandler returns a handler for POST /api/v1/tasks.
func CreateTaskHandler(deps *Deps) func(context.Context, *app.RequestContext) {
	return func(ctx context.Context, c *app.RequestContext) {
		var req schema.UserQuery
		if err := c.Bind(&req); err != nil {
			c.JSON(400, map[string]string{"message": "invalid request body"})
			return
		}

		taskID := fmt.Sprintf("task_%s_%d", req.CompetitorName, time.Now().Unix())

		// Persist task status.
		deps.TaskStore.Create(taskID)

		// Publish to NATS task queue for Worker consumption.
		if err := deps.TaskPublisher.PublishTask(ctx, taskID, req); err != nil {
			c.JSON(500, map[string]string{"message": "failed to enqueue task"})
			return
		}

		c.JSON(201, createTaskResponse{TaskID: taskID})
	}
}

// GetTaskHandler returns a handler for GET /api/v1/tasks/:id.
func GetTaskHandler(deps *Deps) func(context.Context, *app.RequestContext) {
	return func(ctx context.Context, c *app.RequestContext) {
		id := c.Param("id")
		if ts, ok := deps.TaskStore.Get(id); ok {
			c.JSON(200, ts)
			return
		}
		c.JSON(200, taskStatus{
			TaskID:    id,
			Status:    "unknown",
			CreatedAt: time.Now().UTC().Format(time.RFC3339),
		})
	}
}
