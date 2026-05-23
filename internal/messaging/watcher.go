package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
	"github.com/competify-ai/competify-backend/internal/storage/dgraph"
)

// OntologyUpdater is the subset of dgraph.Client needed by Watcher.
// Implemented by *dgraph.Client; mock it in tests.
type OntologyUpdater interface {
	AddFeature(ctx context.Context, competitorName, featureName string) error
	UpdatePricing(ctx context.Context, competitorName, payload string) error
	RecordFunding(ctx context.Context, competitorName, payload string) error
}

// Watcher subscribes to competitor events and updates the ontology in Dgraph.
type Watcher struct {
	js     jetstream.JetStream
	dgraph OntologyUpdater
}

// NewWatcher creates a Watcher. Call Start to begin consuming.
func NewWatcher(nc *nats.Conn, dg OntologyUpdater) (*Watcher, error) {
	js, err := jetstream.New(nc)
	if err != nil {
		return nil, fmt.Errorf("messaging.NewWatcher: %w", err)
	}
	return &Watcher{js: js, dgraph: dg}, nil
}

// Start begins the event consumption loop. Blocks until ctx is cancelled.
func (w *Watcher) Start(ctx context.Context) error {
	// Ensure stream exists.
	_, err := w.js.CreateOrUpdateStream(ctx, jetstream.StreamConfig{
		Name:      streamName,
		Subjects:  []string{streamSubject},
		Retention: jetstream.WorkQueuePolicy,
		MaxMsgs:   10000,
	})
	if err != nil {
		return fmt.Errorf("messaging.Watcher.Start: create stream: %w", err)
	}

	// Create a durable consumer.
	cons, err := w.js.CreateOrUpdateConsumer(ctx, streamName, jetstream.ConsumerConfig{
		Name:       consumerName,
		Durable:    consumerName,
		AckPolicy:  jetstream.AckExplicitPolicy,
		MaxDeliver: 3,
	})
	if err != nil {
		return fmt.Errorf("messaging.Watcher.Start: create consumer: %w", err)
	}

	// Consume messages.
	cc, err := cons.Consume(func(msg jetstream.Msg) {
		if err := w.handleMsg(ctx, msg); err != nil {
			log.Printf("[Watcher] handle msg failed: %v", err)
			msg.Nak()
			return
		}
		msg.Ack()
	})
	if err != nil {
		return fmt.Errorf("messaging.Watcher.Start: consume: %w", err)
	}

	<-ctx.Done()
	cc.Stop()
	return ctx.Err()
}

func (w *Watcher) handleMsg(ctx context.Context, msg jetstream.Msg) error {
	var ev CompetitorEvent
	if err := json.Unmarshal(msg.Data(), &ev); err != nil {
		return fmt.Errorf("unmarshal: %w", err)
	}
	log.Printf("[Watcher] event=%s competitor=%s", ev.EventType, ev.CompetitorName)
	return w.handleEvent(ctx, &ev)
}

func (w *Watcher) handleEvent(ctx context.Context, ev *CompetitorEvent) error {
	switch ev.EventType {
	case EventNewFeature:
		return w.dgraph.AddFeature(ctx, ev.CompetitorName, ev.Payload)
	case EventPriceChange:
		return w.dgraph.UpdatePricing(ctx, ev.CompetitorName, ev.Payload)
	case EventFunding:
		return w.dgraph.RecordFunding(ctx, ev.CompetitorName, ev.Payload)
	default:
		return fmt.Errorf("unknown event type: %s", ev.EventType)
	}
}

// PublishInternalRecalculation publishes a lightweight recalculation trigger.
// Used by Watcher to cascade updates to downstream analyzers.
func (w *Watcher) PublishInternalRecalculation(ctx context.Context, competitorName, eventType string) error {
	data, _ := json.Marshal(map[string]string{
		"competitor": competitorName,
		"event_type": eventType,
	})
	_, err := w.js.Publish(ctx, "internal.recalculate", data)
	return err
}

// Ensure Watcher's compile-time contract: *dgraph.Client satisfies OntologyUpdater.
var _ OntologyUpdater = (*dgraph.Client)(nil)
