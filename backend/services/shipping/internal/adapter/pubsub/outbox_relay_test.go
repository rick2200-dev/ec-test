package subscriber

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	pkgpubsub "github.com/Riku-KANO/ec-test/pkg/pubsub"
)

// ── fakes ────────────────────────────────────────────────────────────────────

type fakeStore struct {
	batch          []outboxRow
	claimErr       error
	markPublishedErr error
	recordFailureErr error

	// recorded calls
	markedIDs     []uuid.UUID
	failuresCalls []failureCall
}

type failureCall struct {
	id     uuid.UUID
	errMsg string
}

func (f *fakeStore) claimBatch(_ context.Context) ([]outboxRow, error) {
	if f.claimErr != nil {
		return nil, f.claimErr
	}
	return f.batch, nil
}

func (f *fakeStore) markPublished(_ context.Context, ids []uuid.UUID) error {
	if f.markPublishedErr != nil {
		return f.markPublishedErr
	}
	f.markedIDs = append(f.markedIDs, ids...)
	return nil
}

func (f *fakeStore) recordFailure(_ context.Context, id uuid.UUID, errMsg string) error {
	if f.recordFailureErr != nil {
		return f.recordFailureErr
	}
	f.failuresCalls = append(f.failuresCalls, failureCall{id: id, errMsg: errMsg})
	return nil
}

type fakePublisher struct {
	// topicErrs maps topic to error; missing = success
	topicErrs map[string]error
	published []pkgpubsub.Event
}

func (p *fakePublisher) Publish(_ context.Context, topic string, event pkgpubsub.Event) error {
	if err, ok := p.topicErrs[topic]; ok {
		return err
	}
	p.published = append(p.published, event)
	return nil
}

func (p *fakePublisher) Close() error { return nil }

// ── helpers ───────────────────────────────────────────────────────────────────

func makeRow(topic string) outboxRow {
	return outboxRow{
		id:        uuid.New(),
		eventType: "shipment.shipped",
		topic:     topic,
		tenantID:  uuid.New(),
		payload:   json.RawMessage(`{"order_id":"test"}`),
		createdAt: time.Date(2024, 1, 15, 10, 0, 0, 0, time.UTC),
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

func TestOutboxRelay_EmptyBatch(t *testing.T) {
	store := &fakeStore{}
	pub := &fakePublisher{}
	relay := newOutboxRelayWithStore(store, pub)

	if err := relay.Run(context.Background()); err != nil {
		t.Fatalf("Run() with empty batch returned error: %v", err)
	}
	if len(pub.published) != 0 {
		t.Errorf("expected 0 published events, got %d", len(pub.published))
	}
}

func TestOutboxRelay_ClaimError(t *testing.T) {
	store := &fakeStore{claimErr: errors.New("db offline")}
	pub := &fakePublisher{}
	relay := newOutboxRelayWithStore(store, pub)

	err := relay.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when claim fails, got nil")
	}
}

// TestOutboxRelay_StableEventID verifies that the Pub/Sub envelope event_id
// equals the outbox row's UUID regardless of when the relay runs.
// A new uuid.New() call on retry must NOT generate a different event_id.
func TestOutboxRelay_StableEventID(t *testing.T) {
	row := makeRow("shipping-events")
	store := &fakeStore{batch: []outboxRow{row}}
	pub := &fakePublisher{}
	relay := newOutboxRelayWithStore(store, pub)

	if err := relay.Run(context.Background()); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(pub.published))
	}
	if pub.published[0].ID != row.id.String() {
		t.Errorf("event_id = %q, want %q (outbox row id)", pub.published[0].ID, row.id.String())
	}
}

// TestOutboxRelay_StableTimestamp verifies that the envelope timestamp comes
// from outbox.created_at, not from time.Now() at publish time.
func TestOutboxRelay_StableTimestamp(t *testing.T) {
	row := makeRow("shipping-events")
	store := &fakeStore{batch: []outboxRow{row}}
	pub := &fakePublisher{}
	relay := newOutboxRelayWithStore(store, pub)

	if err := relay.Run(context.Background()); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if len(pub.published) != 1 {
		t.Fatalf("expected 1 published event, got %d", len(pub.published))
	}
	got := pub.published[0].Timestamp.UTC()
	want := row.createdAt.UTC()
	if !got.Equal(want) {
		t.Errorf("envelope Timestamp = %v, want outbox created_at = %v", got, want)
	}
}

// TestOutboxRelay_SuccessMarksPublished verifies that successfully published
// rows are passed to markPublished and NOT to recordFailure.
func TestOutboxRelay_SuccessMarksPublished(t *testing.T) {
	row := makeRow("shipping-events")
	store := &fakeStore{batch: []outboxRow{row}}
	pub := &fakePublisher{}
	relay := newOutboxRelayWithStore(store, pub)

	if err := relay.Run(context.Background()); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if len(store.markedIDs) != 1 || store.markedIDs[0] != row.id {
		t.Errorf("markedIDs = %v, want [%s]", store.markedIDs, row.id)
	}
	if len(store.failuresCalls) != 0 {
		t.Errorf("expected no failure calls on success, got %d", len(store.failuresCalls))
	}
}

// TestOutboxRelay_PublishFailureRecordsAttempt verifies that when publish
// fails, the row is passed to recordFailure with the error message and NOT
// passed to markPublished.
func TestOutboxRelay_PublishFailureRecordsAttempt(t *testing.T) {
	row := makeRow("bad-topic")
	store := &fakeStore{batch: []outboxRow{row}}
	pubErr := errors.New("PERMISSION_DENIED")
	pub := &fakePublisher{topicErrs: map[string]error{"bad-topic": pubErr}}
	relay := newOutboxRelayWithStore(store, pub)

	if err := relay.Run(context.Background()); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if len(store.markedIDs) != 0 {
		t.Errorf("failed row must not be marked published, got markedIDs=%v", store.markedIDs)
	}
	if len(store.failuresCalls) != 1 {
		t.Fatalf("expected 1 failure call, got %d", len(store.failuresCalls))
	}
	fc := store.failuresCalls[0]
	if fc.id != row.id {
		t.Errorf("failure id = %s, want %s", fc.id, row.id)
	}
	if fc.errMsg == "" {
		t.Error("failure errMsg must not be empty")
	}
}

// TestOutboxRelay_PartialSuccess verifies that in a mixed batch, only the
// successfully published rows are marked and only failed rows get attempt
// increments.
func TestOutboxRelay_PartialSuccess(t *testing.T) {
	good := makeRow("shipping-events")
	bad := makeRow("bad-topic")
	store := &fakeStore{batch: []outboxRow{good, bad}}
	pub := &fakePublisher{topicErrs: map[string]error{"bad-topic": errors.New("no auth")}}
	relay := newOutboxRelayWithStore(store, pub)

	if err := relay.Run(context.Background()); err != nil {
		t.Fatalf("Run() returned error: %v", err)
	}
	if len(store.markedIDs) != 1 || store.markedIDs[0] != good.id {
		t.Errorf("markedIDs = %v, want [%s]", store.markedIDs, good.id)
	}
	if len(store.failuresCalls) != 1 || store.failuresCalls[0].id != bad.id {
		t.Errorf("failuresCalls = %v, want [{%s ...}]", store.failuresCalls, bad.id)
	}
}

// TestOutboxRelay_MarkPublishedFailureReturnsError verifies that a Phase 3a
// DB error is propagated so the relay's Start loop can log it and retry on
// the next tick. The rows remain in processing_at state and will be retried
// after the 5-minute expiry — at-least-once delivery is preserved.
func TestOutboxRelay_MarkPublishedFailureReturnsError(t *testing.T) {
	row := makeRow("shipping-events")
	store := &fakeStore{
		batch:            []outboxRow{row},
		markPublishedErr: errors.New("db write failed"),
	}
	pub := &fakePublisher{}
	relay := newOutboxRelayWithStore(store, pub)

	err := relay.Run(context.Background())
	if err == nil {
		t.Fatal("expected error when markPublished fails, got nil")
	}
}
