package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"sync/atomic"

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
	js            jetstream.JetStream
	dgraph        OntologyUpdater
	dgraphErrOnce atomic.Int32 // 1 after first Dgraph error is logged; suppresses repeats
}

// NewWatcher creates a Watcher from an existing JetStream context. Call Start to begin consuming.
func NewWatcher(js jetstream.JetStream, dg OntologyUpdater) (*Watcher, error) {
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
		// Stream is already ensured by main(); proceed to create consumer.
		log.Printf("[Watcher] CreateOrUpdateStream warning (stream may exist): %v", err)
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
	var err error
	switch ev.EventType {
	case EventNewFeature:
		err = w.dgraph.AddFeature(ctx, ev.CompetitorName, ev.Payload)
	case EventPriceChange:
		err = w.dgraph.UpdatePricing(ctx, ev.CompetitorName, ev.Payload)
	case EventFunding:
		err = w.dgraph.RecordFunding(ctx, ev.CompetitorName, ev.Payload)
	case EventProductLaunch:
		// Treat product launch as a feature update — adds to the ontology graph.
		err = w.dgraph.AddFeature(ctx, ev.CompetitorName, ev.Payload)
	default:
		// Unknown events are logged but not fatal — forward compatibility.
		log.Printf("[Watcher] skipping unhandled event type: %s", ev.EventType)
		return nil
	}
	if err != nil {
		// Log dgraph write failures only once — when Dgraph is down every event
		// would spam the log. Run `make infra-up` to enable persistent ontology.
		if w.dgraphErrOnce.CompareAndSwap(0, 1) {
			log.Printf("[Watcher] dgraph write failed (suppressing future errors — run `make infra-up` to enable): %v", err)
		}
		return nil // non-fatal: ontology update is best-effort
	}
	return nil
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
