package messaging

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
)

// DAGEvent is the real-time status update for a single DAG node.
type DAGEvent struct {
	NodeName  string   `json:"node_name"`
	Status    string   `json:"status"`
	Progress  float64  `json:"progress"`
	Timestamp string   `json:"timestamp"`
	Logs      []string `json:"logs"`
}

// NewDAGEvent creates a DAGEvent with the current timestamp.
func NewDAGEvent(nodeName, status string, progress float64, logs ...string) DAGEvent {
	return DAGEvent{
		NodeName:  nodeName,
		Status:    status,
		Progress:  progress,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Logs:      logs,
	}
}

func subjectForDAGStatus(taskID string) string {
	return fmt.Sprintf("dag.status.%s", taskID)
}

// PublishDAGEvent publishes a DAG status event to NATS.
func PublishDAGEvent(nc *nats.Conn, taskID string, ev DAGEvent) error {
	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("messaging.PublishDAGEvent: marshal: %w", err)
	}
	if err := nc.Publish(subjectForDAGStatus(taskID), data); err != nil {
		return fmt.Errorf("messaging.PublishDAGEvent: publish: %w", err)
	}
	return nil
}

// SubscribeDAGStatus subscribes to DAG status events for a specific task.
// Returns an unsubscribe function and an error.
func SubscribeDAGStatus(nc *nats.Conn, taskID string, handler func(ev DAGEvent)) (func(), error) {
	sub, err := nc.Subscribe(subjectForDAGStatus(taskID), func(msg *nats.Msg) {
		var ev DAGEvent
		if err := json.Unmarshal(msg.Data, &ev); err != nil {
			return
		}
		handler(ev)
	})
	if err != nil {
		return nil, fmt.Errorf("messaging.SubscribeDAGStatus: %w", err)
	}
	return func() { sub.Unsubscribe() }, nil
}
