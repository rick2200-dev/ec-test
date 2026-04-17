package subscriber

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/domain"
)

// fakeReleaser captures every ReleaseStockForOrderCancellation call.
// With the current no-op handler it should never be called; the tests
// verify that invariant. When the full reserve→confirm→release cycle
// is wired up, these tests will need to be updated to assert calls.
type fakeReleaser struct {
	calls []fakeReleaserCall
	err   error
}

type fakeReleaserCall struct {
	orderID uuid.UUID
	lines   []domain.CancellationLine
}

func (f *fakeReleaser) ReleaseStockForOrderCancellation(
	ctx context.Context,
	orderID uuid.UUID,
	lines []domain.CancellationLine,
) error {
	f.calls = append(f.calls, fakeReleaserCall{
		orderID: orderID,
		lines:   append([]domain.CancellationLine(nil), lines...),
	})
	if f.err != nil {
		return f.err
	}
	return nil
}

// buildOrderCancelledEvent constructs a pubsub.Event payload that
// mirrors what the order service's publishOrderCancelled helper
// emits on the "order-events" topic.
func buildOrderCancelledEvent(orderID, skuID uuid.UUID, quantity int) pubsub.Event {
	return pubsub.Event{
		ID:        uuid.New().String(),
		Type:      eventTypeOrderCancelled,
		Timestamp: time.Now().UTC(),
		Data: map[string]any{
			"order_id":       orderID.String(),
			"seller_id":      uuid.New().String(),
			"buyer_auth0_id": "auth0|buyer-test",
			"request_id":     uuid.New().String(),
			"reason":         "changed mind",
			"line_items": []map[string]any{
				{
					"sku_id":       skuID.String(),
					"product_name": "Test Product",
					"sku_code":     "SKU-001",
					"quantity":     quantity,
				},
			},
		},
	}
}

func TestOrderSubscriber_IgnoresUnrelatedEvents(t *testing.T) {
	releaser := &fakeReleaser{}
	sub := NewOrderSubscriber(nil, releaser)

	err := sub.handleEvent(context.Background(), pubsub.Event{
		ID:   uuid.New().String(),
		Type: "order.paid", // not order.cancelled
		Data: map[string]any{},
	})
	if err != nil {
		t.Fatalf("handleEvent(order.paid) returned error: %v", err)
	}
	if len(releaser.calls) != 0 {
		t.Errorf("unrelated event reached releaser: %d calls", len(releaser.calls))
	}
}

func TestOrderSubscriber_NoOpRelease(t *testing.T) {
	// While checkout does not call ReserveStock, handleOrderCancelled
	// must decode+validate the payload but NOT call the releaser.
	releaser := &fakeReleaser{}
	sub := NewOrderSubscriber(nil, releaser)

	orderID := uuid.New()
	skuID := uuid.New()

	err := sub.handleEvent(context.Background(), buildOrderCancelledEvent(orderID, skuID, 3))
	if err != nil {
		t.Fatalf("handleEvent returned error: %v", err)
	}
	if len(releaser.calls) != 0 {
		t.Fatalf("releaser was called %d times; expected 0 (no-op until reserve cycle is wired)", len(releaser.calls))
	}
}

func TestOrderSubscriber_RejectsInvalidOrderID(t *testing.T) {
	releaser := &fakeReleaser{}
	sub := NewOrderSubscriber(nil, releaser)

	event := pubsub.Event{
		ID:   uuid.New().String(),
		Type: eventTypeOrderCancelled,
		Data: map[string]any{
			"order_id":   "not-a-uuid",
			"line_items": []map[string]any{},
		},
	}

	err := sub.handleEvent(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for invalid order_id, got nil")
	}
}

func TestOrderSubscriber_RejectsMalformedPayload(t *testing.T) {
	releaser := &fakeReleaser{}
	sub := NewOrderSubscriber(nil, releaser)

	event := pubsub.Event{
		ID:   uuid.New().String(),
		Type: eventTypeOrderCancelled,
		Data: "not a map", // invalid data type
	}

	err := sub.handleEvent(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for malformed payload, got nil")
	}
}

func TestOrderSubscriber_RejectsInvalidSKUID(t *testing.T) {
	releaser := &fakeReleaser{}
	sub := NewOrderSubscriber(nil, releaser)

	event := pubsub.Event{
		ID:   uuid.New().String(),
		Type: eventTypeOrderCancelled,
		Data: map[string]any{
			"order_id": uuid.New().String(),
			"line_items": []map[string]any{
				{"sku_id": "not-a-uuid", "quantity": 1},
			},
		},
	}

	err := sub.handleEvent(context.Background(), event)
	if err == nil {
		t.Fatal("expected error for invalid sku_id, got nil")
	}
}

func TestOrderSubscriber_IdempotentRedelivery(t *testing.T) {
	// Even with no-op release, redeliveries must succeed (ack).
	releaser := &fakeReleaser{}
	sub := NewOrderSubscriber(nil, releaser)

	event := buildOrderCancelledEvent(uuid.New(), uuid.New(), 2)

	if err := sub.handleEvent(context.Background(), event); err != nil {
		t.Fatalf("first delivery: %v", err)
	}
	if err := sub.handleEvent(context.Background(), event); err != nil {
		t.Fatalf("second delivery: %v", err)
	}
	if len(releaser.calls) != 0 {
		t.Errorf("releaser called %d times on redelivery; expected 0", len(releaser.calls))
	}
}
