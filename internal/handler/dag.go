package handler

import (
	"context"
	"time"

	"github.com/cloudwego/hertz/pkg/app"
	"github.com/hertz-contrib/websocket"
)

// dagEvent is the JSON payload pushed to the frontend.
type dagEvent struct {
	NodeName  string   `json:"node_name"`
	Status    string   `json:"status"`
	Progress  float64  `json:"progress"`
	Timestamp string   `json:"timestamp"`
	Logs      []string `json:"logs"`
}

var upgrader = websocket.HertzUpgrader{}

// DAGWebSocket handles GET /api/v1/tasks/:id/dag (WebSocket upgrade).
func DAGWebSocket(ctx context.Context, c *app.RequestContext) {
	taskID := c.Param("id")

	err := upgrader.Upgrade(c, func(conn *websocket.Conn) {
		defer conn.Close()

		simulateDAG(taskID, conn)
	})
	if err != nil {
		// HertzUpgrader writes its own error response; no extra handling needed.
		return
	}
}

var agentWave = []struct {
	name string
	wave int
}{
	{"orchestrator", 0},
	{"collector_web", 1},
	{"collector_api", 1},
	{"collector_fin", 1},
	{"collector_rev", 1},
	{"collector_soc", 1},
	{"cleaner", 2},
	{"analyzer_feat", 3},
	{"analyzer_price", 3},
	{"analyzer_tech", 3},
	{"analyzer_mkt", 3},
	{"cross_reviewer", 4},
	{"writer", 5},
	{"final_reviewer", 6},
}

func simulateDAG(taskID string, conn *websocket.Conn) {
	// wave -> list of agent indices
	waveMap := make(map[int][]int)
	for i, a := range agentWave {
		waveMap[a.wave] = append(waveMap[a.wave], i)
	}

	status := make([]string, len(agentWave))
	for i := range status {
		status[i] = "pending"
	}

	// Send initial pending state for all agents.
	for i, a := range agentWave {
		_ = conn.WriteJSON(dagEvent{
			NodeName:  a.name,
			Status:    status[i],
			Progress:  0,
			Timestamp: time.Now().UTC().Format(time.RFC3339),
			Logs:      []string{},
		})
	}

	maxWave := 6
	for w := 0; w <= maxWave; w++ {
		agents := waveMap[w]

		// Transition to running.
		for _, idx := range agents {
			status[idx] = "running"
			_ = conn.WriteJSON(dagEvent{
				NodeName:  agentWave[idx].name,
				Status:    "running",
				Progress:  0.1,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Logs:      []string{agentWave[idx].name + " started"},
			})
		}

		// Simulate progress ticks within the wave.
		for tick := 1; tick <= 4; tick++ {
			time.Sleep(400 * time.Millisecond)
			for _, idx := range agents {
				progress := float64(tick) / 5.0
				if progress > 0.95 {
					progress = 0.95
				}
				_ = conn.WriteJSON(dagEvent{
					NodeName:  agentWave[idx].name,
					Status:    "running",
					Progress:  progress,
					Timestamp: time.Now().UTC().Format(time.RFC3339),
					Logs:      []string{agentWave[idx].name + " in progress..."},
				})
			}
		}

		// Transition to done.
		time.Sleep(200 * time.Millisecond)
		for _, idx := range agents {
			status[idx] = "done"
			_ = conn.WriteJSON(dagEvent{
				NodeName:  agentWave[idx].name,
				Status:    "done",
				Progress:  1.0,
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Logs:      []string{agentWave[idx].name + " completed"},
			})
		}
	}

	// Keep connection alive briefly so the frontend can read the final state.
	time.Sleep(2 * time.Second)
}
