package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go/jetstream"
	"github.com/competify-ai/competify-backend/internal/schema"
)

const (
	taskStreamName    = "COMPETIFY_TASKS"
	taskStreamSubject = "competify.task.>"
	taskConsumerName  = "dag-worker"
)

// TaskPublisher publishes task messages to NATS JetStream.
type TaskPublisher struct {
	js jetstream.JetStream
}

// NewTaskPublisher creates a TaskPublisher from an existing JetStream context.
func NewTaskPublisher(js jetstream.JetStream) (*TaskPublisher, error) {
	return &TaskPublisher{js: js}, nil
}

// EnsureTaskStream idempotently creates the task JetStream Stream.
func (p *TaskPublisher) EnsureTaskStream(ctx context.Context) error {
	_, err := p.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      taskStreamName,
		Subjects:  []string{taskStreamSubject},
		Retention: jetstream.WorkQueuePolicy,
		MaxMsgs:   10000,
	})
	if err != nil {
		return fmt.Errorf("messaging.EnsureTaskStream: %w", err)
	}
	return nil
}

// PublishTask serializes a UserQuery and publishes it to the task queue.
func (p *TaskPublisher) PublishTask(ctx context.Context, taskID string, query schema.UserQuery) error {
	payload := map[string]interface{}{
		"task_id": taskID,
		"query":   query,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("messaging.PublishTask: marshal: %w", err)
	}
	_, err = p.js.Publish(ctx, subjectForTask(taskID), data, jetstream.WithMsgID(taskID))
	if err != nil {
		return fmt.Errorf("messaging.PublishTask: publish: %w", err)
	}
	return nil
}

func subjectForTask(taskID string) string {
	return fmt.Sprintf("competify.task.%s", taskID)
}

// TaskSubscriber consumes task messages from NATS JetStream.
type TaskSubscriber struct {
	js jetstream.JetStream
}

// NewTaskSubscriber creates a TaskSubscriber from an existing JetStream context.
func NewTaskSubscriber(js jetstream.JetStream) (*TaskSubscriber, error) {
	return &TaskSubscriber{js: js}, nil
}

// SubscribeTask begins consuming task messages. Blocks until ctx is cancelled.
func (s *TaskSubscriber) SubscribeTask(ctx context.Context, handler func(taskID string, query schema.UserQuery)) error {
	_, err := s.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      taskStreamName,
		Subjects:  []string{taskStreamSubject},
		Retention: jetstream.WorkQueuePolicy,
		MaxMsgs:   10000,
	})
	if err != nil {
		// Stream is already ensured by main(); proceed to create consumer.
		log.Printf("[TaskSubscriber] CreateOrUpdateStream warning (stream may exist): %v", err)
	}

	cons, err := s.js.CreateOrUpdateConsumer(ctx, taskStreamName, jetstream.ConsumerConfig{
		Name:       taskConsumerName,
		Durable:    taskConsumerName,
		AckPolicy:  jetstream.AckExplicitPolicy,
		MaxDeliver: 3,
	})
	if err != nil {
		return fmt.Errorf("messaging.SubscribeTask: create consumer: %w", err)
	}

	cc, err := cons.Consume(func(msg jetstream.Msg) {
		var payload struct {
			TaskID string          `json:"task_id"`
			Query  schema.UserQuery `json:"query"`
		}
		if err := json.Unmarshal(msg.Data(), &payload); err != nil {
			msg.Nak()
			return
		}
		handler(payload.TaskID, payload.Query)
		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("messaging.SubscribeTask: consume: %w", err)
	}

	<-ctx.Done()
	cc.Stop()
	return ctx.Err()
}
