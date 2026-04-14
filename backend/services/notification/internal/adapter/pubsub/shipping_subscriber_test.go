package subscriber

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/notification/internal/port"
)

// ── fakes ─────────────────────────────────────────────────────────────────────

type fakeDedup struct {
	// claimResults controls TryClaim return values (indexed by call order;
	// last entry is repeated when calls exceed slice length).
	claimResults []claimResult
	claimCalls   int

	sentIDs     []string
	markSentErr error

	releasedIDs []string
	releaseErr  error
}

type claimResult struct {
	outcome port.ClaimOutcome
	err     error
}

func (d *fakeDedup) TryClaim(_ context.Context, eventID, _ string) (port.ClaimOutcome, error) {
	idx := d.claimCalls
	if idx >= len(d.claimResults) {
		idx = len(d.claimResults) - 1
	}
	d.claimCalls++
	r := d.claimResults[idx]
	return r.outcome, r.err
}

func (d *fakeDedup) MarkSent(_ context.Context, eventID string) error {
	d.sentIDs = append(d.sentIDs, eventID)
	return d.markSentErr
}

func (d *fakeDedup) Release(_ context.Context, eventID string) error {
	d.releasedIDs = append(d.releasedIDs, eventID)
	return d.releaseErr
}

// shorthand constructors

func acquiredDedup() *fakeDedup {
	return &fakeDedup{claimResults: []claimResult{{outcome: port.ClaimAcquired}}}
}

func alreadySentDedup() *fakeDedup {
	return &fakeDedup{claimResults: []claimResult{{outcome: port.ClaimAlreadySent}}}
}

func activeProcessingDedup() *fakeDedup {
	return &fakeDedup{claimResults: []claimResult{{outcome: port.ClaimActiveProcessing}}}
}

type fakeSender struct {
	calls []senderCall
	err   error
}

type senderCall struct {
	to      string
	subject string
}

func (s *fakeSender) Send(_ context.Context, to, subject, body string) error {
	s.calls = append(s.calls, senderCall{to: to, subject: subject})
	return s.err
}

// ── helpers ───────────────────────────────────────────────────────────────────

func shippedEvent(eventID string) pubsub.Event {
	return pubsub.Event{
		ID:        eventID,
		Type:      "shipment.shipped",
		TenantID:  uuid.New().String(),
		Timestamp: time.Now(),
		Data: map[string]any{
			"shipment_id":     uuid.New().String(),
			"order_id":        uuid.New().String(),
			"buyer_auth0_id":  "auth0|buyer1",
			"carrier":         "yamato",
			"tracking_number": "1234567890",
		},
	}
}

func deliveredEvent(eventID string) pubsub.Event {
	return pubsub.Event{
		ID:        eventID,
		Type:      "shipment.delivered",
		TenantID:  uuid.New().String(),
		Timestamp: time.Now(),
		Data: map[string]any{
			"shipment_id":    uuid.New().String(),
			"order_id":       uuid.New().String(),
			"buyer_auth0_id": "auth0|buyer2",
		},
	}
}

// ── tests ─────────────────────────────────────────────────────────────────────

// TestShippingSubscriber_AlreadySentIsAcked verifies that ClaimAlreadySent
// causes the handler to return nil (ack) without sending. The event was
// previously processed successfully; future redeliveries must be silently skipped.
func TestShippingSubscriber_AlreadySentIsAcked(t *testing.T) {
	dedup := alreadySentDedup()
	sender := &fakeSender{}
	sub := NewShippingSubscriber(nil, sender, dedup)

	if err := sub.handleEvent(context.Background(), shippedEvent(uuid.New().String())); err != nil {
		t.Fatalf("handleEvent returned error: %v", err)
	}
	if len(sender.calls) != 0 {
		t.Errorf("expected no sends on already_sent, got %d", len(sender.calls))
	}
	if len(dedup.sentIDs) != 0 || len(dedup.releasedIDs) != 0 {
		t.Errorf("MarkSent/Release must not be called on already_sent")
	}
}

// TestShippingSubscriber_ActiveProcessingNacks verifies that ClaimActiveProcessing
// causes the handler to return a non-nil error (nack). Pub/Sub must keep the
// message alive for redelivery — returning nil here would ack the message and
// could permanently lose the notification if the active worker crashes before
// MarkSent and before the processing_at timeout expires.
func TestShippingSubscriber_ActiveProcessingNacks(t *testing.T) {
	dedup := activeProcessingDedup()
	sender := &fakeSender{}
	sub := NewShippingSubscriber(nil, sender, dedup)

	err := sub.handleEvent(context.Background(), shippedEvent(uuid.New().String()))
	if err == nil {
		t.Fatal("handleEvent must return error on active_processing so Pub/Sub nacks and redelivers")
	}
	if len(sender.calls) != 0 {
		t.Errorf("expected no sends on active_processing, got %d", len(sender.calls))
	}
	if len(dedup.releasedIDs) != 0 {
		t.Errorf("Release must not be called on active_processing")
	}
}

// TestShippingSubscriber_SendSuccessCallsMarkSent verifies that on a successful
// send, MarkSent is called with the correct event_id and Release is NOT called.
func TestShippingSubscriber_SendSuccessCallsMarkSent(t *testing.T) {
	eventID := uuid.New().String()
	dedup := acquiredDedup()
	sender := &fakeSender{}
	sub := NewShippingSubscriber(nil, sender, dedup)

	if err := sub.handleEvent(context.Background(), shippedEvent(eventID)); err != nil {
		t.Fatalf("handleEvent returned error: %v", err)
	}
	if len(sender.calls) != 1 {
		t.Errorf("expected 1 send, got %d", len(sender.calls))
	}
	if len(dedup.sentIDs) != 1 || dedup.sentIDs[0] != eventID {
		t.Errorf("sentIDs = %v, want [%s]", dedup.sentIDs, eventID)
	}
	if len(dedup.releasedIDs) != 0 {
		t.Errorf("Release must not be called on success, got %v", dedup.releasedIDs)
	}
}

// TestShippingSubscriber_SendFailureReleasesAndErrors verifies that when the
// email send fails, Release is called (so Pub/Sub can retry) and the error is
// propagated. MarkSent must NOT be called.
func TestShippingSubscriber_SendFailureReleasesAndErrors(t *testing.T) {
	eventID := uuid.New().String()
	dedup := acquiredDedup()
	sender := &fakeSender{err: errors.New("smtp offline")}
	sub := NewShippingSubscriber(nil, sender, dedup)

	if err := sub.handleEvent(context.Background(), shippedEvent(eventID)); err == nil {
		t.Fatal("expected error on send failure, got nil")
	}
	if len(dedup.releasedIDs) != 1 || dedup.releasedIDs[0] != eventID {
		t.Errorf("releasedIDs = %v, want [%s]", dedup.releasedIDs, eventID)
	}
	if len(dedup.sentIDs) != 0 {
		t.Errorf("MarkSent must not be called on send failure")
	}
}

// TestShippingSubscriber_MarkSentFailureReturnsNil verifies that a MarkSent DB
// error after a successful send does NOT cause an error return. Returning an
// error here would nack an already-sent notification; a warning log is enough.
func TestShippingSubscriber_MarkSentFailureReturnsNil(t *testing.T) {
	dedup := acquiredDedup()
	dedup.markSentErr = errors.New("db write failed")
	sender := &fakeSender{}
	sub := NewShippingSubscriber(nil, sender, dedup)

	if err := sub.handleEvent(context.Background(), shippedEvent(uuid.New().String())); err != nil {
		t.Fatalf("handleEvent must return nil when MarkSent fails (send succeeded), got: %v", err)
	}
	if len(sender.calls) != 1 {
		t.Errorf("expected 1 send, got %d", len(sender.calls))
	}
}

// TestShippingSubscriber_TryClaimErrorProceedsAndSends verifies that a DB error
// from TryClaim does not drop the event — the subscriber falls through (fail
// open) and sends the notification.
func TestShippingSubscriber_TryClaimErrorProceedsAndSends(t *testing.T) {
	dedup := &fakeDedup{claimResults: []claimResult{{outcome: port.ClaimAcquired, err: errors.New("db timeout")}}}
	sender := &fakeSender{}
	sub := NewShippingSubscriber(nil, sender, dedup)

	if err := sub.handleEvent(context.Background(), shippedEvent(uuid.New().String())); err != nil {
		t.Fatalf("handleEvent returned error: %v", err)
	}
	if len(sender.calls) != 1 {
		t.Errorf("expected 1 send on dedup error (fail open), got %d", len(sender.calls))
	}
}

// TestShippingSubscriber_ShipmentDeliveredSends verifies the delivered path.
func TestShippingSubscriber_ShipmentDeliveredSends(t *testing.T) {
	eventID := uuid.New().String()
	dedup := acquiredDedup()
	sender := &fakeSender{}
	sub := NewShippingSubscriber(nil, sender, dedup)

	if err := sub.handleEvent(context.Background(), deliveredEvent(eventID)); err != nil {
		t.Fatalf("handleEvent returned error: %v", err)
	}
	if len(sender.calls) != 1 {
		t.Errorf("expected 1 send for delivered event, got %d", len(sender.calls))
	}
	if len(dedup.sentIDs) != 1 || dedup.sentIDs[0] != eventID {
		t.Errorf("sentIDs = %v, want [%s]", dedup.sentIDs, eventID)
	}
}

// TestShippingSubscriber_UnknownEventTypeReleasesAndReturnsNil verifies that an
// unrecognised event type releases the claim and returns nil (no retry storm).
func TestShippingSubscriber_UnknownEventTypeReleasesAndReturnsNil(t *testing.T) {
	eventID := uuid.New().String()
	dedup := acquiredDedup()
	sender := &fakeSender{}
	sub := NewShippingSubscriber(nil, sender, dedup)

	err := sub.handleEvent(context.Background(), pubsub.Event{
		ID: eventID, Type: "shipment.unknown",
		TenantID: uuid.New().String(), Data: map[string]any{},
	})
	if err != nil {
		t.Fatalf("handleEvent returned error for unknown type: %v", err)
	}
	if len(sender.calls) != 0 {
		t.Errorf("unexpected send for unknown event type")
	}
	if len(dedup.releasedIDs) != 1 || dedup.releasedIDs[0] != eventID {
		t.Errorf("releasedIDs = %v, want [%s]", dedup.releasedIDs, eventID)
	}
	if len(dedup.sentIDs) != 0 {
		t.Errorf("MarkSent must not be called for unknown event type")
	}
}

// TestShippingSubscriber_NoopDedup verifies that nil dedup (→ noopDeduplicator)
// processes every delivery without error.
func TestShippingSubscriber_NoopDedup(t *testing.T) {
	sender := &fakeSender{}
	sub := NewShippingSubscriber(nil, sender, nil)
	event := shippedEvent(uuid.New().String())
	for range 2 {
		if err := sub.handleEvent(context.Background(), event); err != nil {
			t.Fatalf("handleEvent returned error: %v", err)
		}
	}
	if len(sender.calls) != 2 {
		t.Errorf("noop dedup: expected 2 sends, got %d", len(sender.calls))
	}
}
