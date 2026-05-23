package handler

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/websocket"
	"github.com/competify-ai/competify-backend/internal/messaging"
	"github.com/nats-io/nats.go"
)

// Allow all origins in dev — browser sends Origin: localhost:5173 while
// the backend host is localhost:8080, and the default same-origin check rejects it.
var upgrader = websocket.HertzUpgrader{
	CheckOrigin: func(ctx *app.RequestContext) bool { return true },
}

// DAGWebSocketHandler returns a handler for GET /api/v1/tasks/:id/dag (WebSocket upgrade).
func DAGWebSocketHandler(deps *Deps) func(context.Context, *app.RequestContext) {
	return func(ctx context.Context, c *app.RequestContext) {
		taskID := c.Param("id")

		err := upgrader.Upgrade(c, func(conn *websocket.Conn) {
			defer conn.Close()

			// Subscribe to NATS DAG status events for this task.
			unsub, err := messaging.SubscribeDAGStatus(deps.NATSConn, taskID, func(ev messaging.DAGEvent) {
				_ = conn.WriteJSON(ev)
			})
			if err != nil {
				return
			}
			defer unsub()

			// Send initial pending state for all agents.
			for _, name := range agentNames {
				_ = conn.WriteJSON(messaging.NewDAGEvent(name, "pending", 0))
			}

			// Keep connection open until the client disconnects or context cancelled.
			for {
				_, _, err := conn.ReadMessage()
				if err != nil {
					break
				}
			}
		})
		if err != nil {
			return
		}
	}
}

var agentNames = []string{
	"orchestrator",
	"collector_web",
	"collector_api",
	"collector_fin",
	"collector_rev",
	"collector_soc",
	"cleaner",
	"analyzer_feat",
	"analyzer_price",
	"analyzer_tech",
	"analyzer_mkt",
	"cross_reviewer",
	"writer",
	"final_reviewer",
}

// simulateProgress publishes mock DAG status events to NATS for a given task.
// Used by the Worker to drive the frontend progress bar while the real DAG runs.
func SimulateProgress(natsConn *nats.Conn, taskID string) {
	// wave -> list of agent indices
	waveMap := make(map[int][]int)
	for i, name := range agentNames {
		wave := executionWave(name)
		waveMap[wave] = append(waveMap[wave], i)
	}

	maxWave := 6
	for w := 0; w <= maxWave; w++ {
		agents := waveMap[w]

		// Transition to running.
		for _, idx := range agents {
			_ = messaging.PublishDAGEvent(natsConn, taskID, messaging.NewDAGEvent(agentNames[idx], "running", 0.1, agentNames[idx]+" started"))
		}

		// Simulate progress ticks within the wave.
		for tick := 1; tick <= 4; tick++ {
			time.Sleep(400 * time.Millisecond)
			for _, idx := range agents {
				progress := float64(tick) / 5.0
				if progress > 0.95 {
					progress = 0.95
				}
				_ = messaging.PublishDAGEvent(natsConn, taskID, messaging.NewDAGEvent(agentNames[idx], "running", progress, agentNames[idx]+" in progress..."))
			}
		}

		// Transition to done.
		time.Sleep(200 * time.Millisecond)
		for _, idx := range agents {
			_ = messaging.PublishDAGEvent(natsConn, taskID, messaging.NewDAGEvent(agentNames[idx], "done", 1.0, agentNames[idx]+" completed"))
		}
	}
}

func executionWave(id string) int {
	switch {
	case id == "orchestrator":
		return 0
	case len(id) >= 9 && id[:9] == "collector":
		return 1
	case id == "cleaner":
		return 2
	case len(id) >= 8 && id[:8] == "analyzer":
		return 3
	case id == "cross_reviewer":
		return 4
	case id == "writer":
		return 5
	case id == "final_reviewer":
		return 6
	}
	return 0
}
