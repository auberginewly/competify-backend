// Package messaging implements NATS JetStream event-driven ontology evolution.
// When a Collector detects a competitor change (new feature, price update, funding),
// it publishes a CompetitorEvent; ReactiveOntologyWatcher subscribes and updates Dgraph.
package messaging

import "time"

// CompetitorEvent represents a detected competitor change.
type CompetitorEvent struct {
	EventID        string    `json:"event_id"` // SHA256 idempotency key
	CompetitorName string    `json:"competitor_name"`
	EventType      string    `json:"event_type"` // NEW_FEATURE / PRICE_CHANGE / FUNDING / ACQUISITION
	Payload        string    `json:"payload"`
	SourceURL      string    `json:"source_url"`
	DetectedAt     time.Time `json:"detected_at"`
	Confidence     float64   `json:"confidence"`
}

// EventType constants
const (
	EventNewFeature    = "NEW_FEATURE"
	EventPriceChange   = "PRICE_CHANGE"
	EventFunding       = "FUNDING"
	EventAcquisition   = "ACQUISITION"
	EventProductLaunch = "PRODUCT_LAUNCH"
	EventPartnership   = "PARTNERSHIP"
)

// ReactiveOntologyWatcher subscribes to competitor.event.> and reacts.
// Phase 5 implementation.
type ReactiveOntologyWatcher struct {
	natsURL string
	// nc *nats.Conn
	// js nats.JetStreamContext
}

// NewReactiveOntologyWatcher creates a watcher; call StartEventLoop to subscribe.
func NewReactiveOntologyWatcher(natsURL string) (*ReactiveOntologyWatcher, error) {
	// TODO Phase 5: nats.Connect + jetstream.AddStream
	return &ReactiveOntologyWatcher{natsURL: natsURL}, nil
}
