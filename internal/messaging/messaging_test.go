package messaging

import (
	"context"
	"testing"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats-server/v2/test"
)

// mockOntologyUpdater records which method was called and with what args.
type mockOntologyUpdater struct {
	addFeatureCalled    bool
	updatePricingCalled bool
	recordFundingCalled bool
	competitorName      string
	payload             string
}

func (m *mockOntologyUpdater) AddFeature(ctx context.Context, competitorName, featureName string) error {
	m.addFeatureCalled = true
	m.competitorName = competitorName
	m.payload = featureName
	return nil
}

func (m *mockOntologyUpdater) UpdatePricing(ctx context.Context, competitorName, payload string) error {
	m.updatePricingCalled = true
	m.competitorName = competitorName
	m.payload = payload
	return nil
}

func (m *mockOntologyUpdater) RecordFunding(ctx context.Context, competitorName, payload string) error {
	m.recordFundingCalled = true
	m.competitorName = competitorName
	m.payload = payload
	return nil
}

func startNATS(t *testing.T) (*nats.Conn, func()) {
	opts := test.DefaultTestOptions
	opts.JetStream = true
	opts.Port = -1
	srv := test.RunServer(&opts)
	t.Cleanup(func() { srv.Shutdown() })

	nc, err := nats.Connect(srv.ClientURL())
	if err != nil {
		t.Fatalf("nats connect: %v", err)
	}
	return nc, func() { nc.Close() }
}

// TestPublisher_PublishAndAck: publish an event and verify it is accepted by JetStream.
func TestPublisher_PublishAndAck(t *testing.T) {
	nc, cleanup := startNATS(t)
	defer cleanup()

	pub, err := NewPublisher(nc)
	if err != nil {
		t.Fatalf("NewPublisher: %v", err)
	}
	if err := pub.EnsureStream(context.Background()); err != nil {
		t.Fatalf("EnsureStream: %v", err)
	}

	ev := &CompetitorEvent{
		CompetitorName: "Cursor",
		EventType:      EventNewFeature,
		Payload:        "Agent Mode",
		SourceURL:      "https://cursor.com/changelog",
		Confidence:     0.92,
	}
	if err := pub.PublishCompetitorEvent(context.Background(), ev); err != nil {
		t.Fatalf("PublishCompetitorEvent: %v", err)
	}

	// Verify the event ID was generated.
	if ev.EventID == "" {
		t.Fatal("expected EventID to be auto-generated")
	}
}

// TestWatcher_ConsumesNewFeature: watcher calls AddFeature on NEW_FEATURE event.
func TestWatcher_ConsumesNewFeature(t *testing.T) {
	nc, cleanup := startNATS(t)
	defer cleanup()

	mock := &mockOntologyUpdater{}
	w, err := NewWatcher(nc, mock)
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}

	// Start watcher in background.
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		if err := w.Start(ctx); err != nil && err != context.Canceled {
			t.Logf("Watcher exited: %v", err)
		}
	}()

	// Give watcher time to set up stream and consumer.
	time.Sleep(200 * time.Millisecond)

	// Publish an event.
	pub, _ := NewPublisher(nc)
	_ = pub.EnsureStream(ctx)
	ev := &CompetitorEvent{
		CompetitorName: "Cursor",
		EventType:      EventNewFeature,
		Payload:        "Agent Mode",
		Confidence:     0.92,
	}
	if err := pub.PublishCompetitorEvent(ctx, ev); err != nil {
		t.Fatalf("PublishCompetitorEvent: %v", err)
	}

	// Wait for watcher to process.
	time.Sleep(500 * time.Millisecond)

	if !mock.addFeatureCalled {
		t.Fatal("expected AddFeature to be called")
	}
	if mock.competitorName != "Cursor" {
		t.Fatalf("expected competitor Cursor, got %s", mock.competitorName)
	}
	if mock.payload != "Agent Mode" {
		t.Fatalf("expected payload 'Agent Mode', got %s", mock.payload)
	}
}

// TestWatcher_ConsumesPriceChange: watcher calls UpdatePricing on PRICE_CHANGE event.
func TestWatcher_ConsumesPriceChange(t *testing.T) {
	nc, cleanup := startNATS(t)
	defer cleanup()

	mock := &mockOntologyUpdater{}
	w, err := NewWatcher(nc, mock)
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = w.Start(ctx)
	}()
	time.Sleep(200 * time.Millisecond)

	pub, _ := NewPublisher(nc)
	_ = pub.EnsureStream(ctx)
	ev := &CompetitorEvent{
		CompetitorName: "Cursor",
		EventType:      EventPriceChange,
		Payload:        "$20/month",
		Confidence:     0.85,
	}
	_ = pub.PublishCompetitorEvent(ctx, ev)

	time.Sleep(500 * time.Millisecond)

	if !mock.updatePricingCalled {
		t.Fatal("expected UpdatePricing to be called")
	}
}

// TestWatcher_ConsumesFunding: watcher calls RecordFunding on FUNDING event.
func TestWatcher_ConsumesFunding(t *testing.T) {
	nc, cleanup := startNATS(t)
	defer cleanup()

	mock := &mockOntologyUpdater{}
	w, err := NewWatcher(nc, mock)
	if err != nil {
		t.Fatalf("NewWatcher: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	go func() {
		_ = w.Start(ctx)
	}()
	time.Sleep(200 * time.Millisecond)

	pub, _ := NewPublisher(nc)
	_ = pub.EnsureStream(ctx)
	ev := &CompetitorEvent{
		CompetitorName: "Cursor",
		EventType:      EventFunding,
		Payload:        "series_c",
		Confidence:     0.90,
	}
	_ = pub.PublishCompetitorEvent(ctx, ev)

	time.Sleep(500 * time.Millisecond)

	if !mock.recordFundingCalled {
		t.Fatal("expected RecordFunding to be called")
	}
}

// TestGenerateEventID_Idempotent: same input -> same ID.
func TestGenerateEventID_Idempotent(t *testing.T) {
	ev := &CompetitorEvent{CompetitorName: "X", EventType: EventNewFeature, Payload: "Y"}
	id1 := generateEventID(ev)
	id2 := generateEventID(ev)
	if id1 != id2 {
		t.Fatal("expected idempotent event ID")
	}
}
