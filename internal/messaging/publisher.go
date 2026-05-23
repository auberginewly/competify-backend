package messaging

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	streamName     = "COMPETITOR_EVENTS"
	streamSubject  = "competitor.event.>"
	consumerName   = "ontology-worker"
)

// Publisher wraps a JetStream connection to publish CompetitorEvents.
type Publisher struct {
	js jetstream.JetStream
}

// NewPublisher creates a Publisher and ensures the target Stream exists.
func NewPublisher(nc *nats.Conn) (*Publisher, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("messaging.NewPublisher: %w", err)
	}
	return &Publisher{js: js}, nil
}

// EnsureStream idempotently creates the JetStream Stream.
func (p *Publisher) EnsureStream(ctx context.Context) error {
	_, err := p.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{streamSubject},
		Retention: jetstream.WorkQueuePolicy,
		MaxMsgs:   10000,
	})
	if err != nil {
		return fmt.Errorf("messaging.EnsureStream: %w", err)
	}
	return nil
}

// PublishCompetitorEvent serializes and publishes a CompetitorEvent.
// It generates an idempotency key (SHA256) if EventID is empty.
func (p *Publisher) PublishCompetitorEvent(ctx context.Context, ev *CompetitorEvent) error {
	if ev.EventID == "" {
		ev.EventID = generateEventID(ev)
	}
	if ev.DetectedAt.IsZero() {
		ev.DetectedAt = time.Now().UTC()
	}

	data, err := json.Marshal(ev)
	if err != nil {
		return fmt.Errorf("messaging.PublishCompetitorEvent: marshal: %w", err)
	}

	_, err = p.js.Publish(ctx, subjectForEvent(ev.EventType), data,
		jetstream.WithMsgID(ev.EventID),
	)
	if err != nil {
		return fmt.Errorf("messaging.PublishCompetitorEvent: publish: %w", err)
	}
	return nil
}

func generateEventID(ev *CompetitorEvent) string {
	h := sha256.Sum256([]byte(ev.CompetitorName + ev.EventType + ev.Payload))
	return hex.EncodeToString(h[:])
}

func subjectForEvent(eventType string) string {
	return fmt.Sprintf("competitor.event.%s", eventType)
}
