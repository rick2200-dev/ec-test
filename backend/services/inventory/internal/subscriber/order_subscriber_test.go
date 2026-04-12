package subscriber

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/service"
)

// fakeReleaser captures every ReleaseStockForOrderCancellation call so
// the idempotency test can verify "the second delivery of the same
// order.cancelled event becomes a no-op at the repository layer, not
// at the subscriber layer". We test the subscriber side: it must
// faithfully forward both calls to the service. The repository-level
// idempotency (via the stock_movements reference_id guard) is exercised
// in inventory_service_integration_test where a real Postgres is
// available.
type fakeReleaser struct {
	calls   []fakeReleaserCall
	err     error
	errOnce bool // return err only on the first call if set
}

type fakeReleaserCall struct {
	tenantID uuid.UUID
	orderID  uuid.UUID
	lines    []service.CancellationLine
}

func (f *fakeReleaser) ReleaseStockForOrderCancellation(
	ctx context.Context,
	tenantID, orderID uuid.UUID,
	lines []service.CancellationLine,
) error {
	f.calls = append(f.calls, fakeReleaserCall{
		tenantID: tenantID,
		orderID:  orderID,
		lines:    append([]service.CancellationLine(nil), lines...),
	})
	if f.err != nil {
		err := f.err
		if f.errOnce {
			f.err = nil
		}
		return err
	}
	return nil
}

// buildOrderCancelledEvent constructs a pubsub.Event payload that
// mirrors what the order service's publishOrderCancelled helper
// emits on the "order-events" topic.
func buildOrderCancelledEvent(tenantID, orderID, skuID uuid.UUID, quantity int) pubsub.Event {
	return pubsub.Event{
		ID:        uuid.New().String(),
		Type:      eventTypeOrderCancelled,
		TenantID:  tenantID.String(),
		Timestamp: time.Now().UTC(),
		Data: map[string]any{
			"order_id":       orderID.String(),
			"tenant_id":      tenantID.String(),
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
		ID:       uuid.New().String(),
		Type:     "order.paid", // not order.cancelled
		TenantID: uuid.New().String(),
		Data:     map[string]any{},
	})
	if err != nil {
		t.Fatalf("handleEvent(order.paid) returned error: %v", err)
	}
	if len(releaser.calls) != 0 {
		t.Errorf("unrelated event reached releaser: %d calls", len(releaser.calls))
	}
}

func TestOrderSubscriber_InvokesReleaseForOrderCancelled(t *testing.T) {
	releaser := &fakeReleaser{}
	sub := NewOrderSubscriber(nil, releaser)

	tenantID := uuid.New()
	orderID := uuid.New()
	skuID := uuid.New()

	err := sub.handleEvent(context.Background(), buildOrderCancelledEvent(tenantID, orderID, skuID, 3))
	if err != nil {
		t.Fatalf("handleEvent returned error: %v", err)
	}
	if len(releaser.calls) != 1 {
		t.Fatalf("expected 1 releaser call, got %d", len(releaser.calls))
	}
	got := releaser.calls[0]
	if got.tenantID != tenantID {
		t.Errorf("tenant id = %s, want %s", got.tenantID, tenantID)
	}
	if got.orderID != orderID {
		t.Errorf("order id = %s, want %s", got.orderID, orderID)
	}
	if len(got.lines) != 1 {
		t.Fatalf("lines = %d, want 1", len(got.lines))
	}
	if got.lines[0].SKUID != skuID || got.lines[0].Quantity != 3 {
		t.Errorf("line = %+v, want sku=%s qty=3", got.lines[0], skuID)
	}
}

// TestOrderSubscriber_IdempotentRedelivery verifies that a Pub/Sub
// at-least-once redelivery of the SAME order.cancelled event does not
// crash the subscriber: it calls ReleaseStockForOrderCancellation a
// second time with identical arguments, and the service/repository
// layer is expected to short-circuit via the stock_movements reference
// guard. From the subscriber's point of view both acks must succeed.
func TestOrderSubscriber_IdempotentRedelivery(t *testing.T) {
	releaser := &fakeReleaser{}
	sub := NewOrderSubscriber(nil, releaser)

	tenantID := uuid.New()
	orderID := uuid.New()
	skuID := uuid.New()
	event := buildOrderCancelledEvent(tenantID, orderID, skuID, 2)

	// First delivery.
	if err := sub.handleEvent(context.Background(), event); err != nil {
		t.Fatalf("first handleEvent returned error: %v", err)
	}
	// Second delivery of the very same event.
	if err := sub.handleEvent(context.Background(), event); err != nil {
		t.Fatalf("second handleEvent returned error: %v", err)
	}

	if len(releaser.calls) != 2 {
		t.Fatalf("expected 2 releaser calls, got %d", len(releaser.calls))
	}
	// Both calls must carry identical arguments — the reference id
	// (orderID) is what the repository uses to dedupe.
	if releaser.calls[0].orderID != releaser.calls[1].orderID {
		t.Errorf("redelivery orderID mismatch: %s vs %s",
			releaser.calls[0].orderID, releaser.calls[1].orderID)
	}
	if releaser.calls[0].tenantID != releaser.calls[1].tenantID {
		t.Errorf("redelivery tenantID mismatch: %s vs %s",
			releaser.calls[0].tenantID, releaser.calls[1].tenantID)
	}
}

func TestOrderSubscriber_InvalidTenantIDErrors(t *testing.T) {
	releaser := &fakeReleaser{}
	sub := NewOrderSubscriber(nil, releaser)

	event := pubsub.Event{
		ID:   uuid.New().String(),
		Type: eventTypeOrderCancelled,
		Data: map[string]any{
			"order_id":   uuid.New().String(),
			"tenant_id":  "not-a-uuid",
			"line_items": []map[string]any{},
		},
	}
	err := sub.handleEvent(context.Background(), event)
	if err == nil {
		t.Fatal("expected error on invalid tenant_id, got nil")
	}
	if len(releaser.calls) != 0 {
		t.Errorf("releaser should not be called on parse error, got %d calls", len(releaser.calls))
	}
}

func TestOrderSubscriber_SkipsLinesWithBadSKUID(t *testing.T) {
	releaser := &fakeReleaser{}
	sub := NewOrderSubscriber(nil, releaser)

	tenantID := uuid.New()
	orderID := uuid.New()
	goodSKU := uuid.New()

	event := pubsub.Event{
		ID:   uuid.New().String(),
		Type: eventTypeOrderCancelled,
		Data: map[string]any{
			"order_id":  orderID.String(),
			"tenant_id": tenantID.String(),
			"line_items": []map[string]any{
				// Bad sku_id — gets skipped, but delivery still acks.
				{"sku_id": "not-a-uuid", "quantity": 1},
				// Zero qty — also skipped by the validator.
				{"sku_id": uuid.New().String(), "quantity": 0},
				// Valid line.
				{"sku_id": goodSKU.String(), "quantity": 5},
			},
		},
	}

	if err := sub.handleEvent(context.Background(), event); err != nil {
		t.Fatalf("handleEvent returned error: %v", err)
	}
	if len(releaser.calls) != 1 {
		t.Fatalf("releaser calls = %d, want 1", len(releaser.calls))
	}
	lines := releaser.calls[0].lines
	if len(lines) != 1 {
		t.Fatalf("lines = %d, want 1 (the valid one)", len(lines))
	}
	if lines[0].SKUID != goodSKU || lines[0].Quantity != 5 {
		t.Errorf("kept line = %+v, want sku=%s qty=5", lines[0], goodSKU)
	}
}

func TestOrderSubscriber_PropagatesReleaseError(t *testing.T) {
	releaser := &fakeReleaser{err: errors.New("db offline")}
	sub := NewOrderSubscriber(nil, releaser)

	event := buildOrderCancelledEvent(uuid.New(), uuid.New(), uuid.New(), 1)
	err := sub.handleEvent(context.Background(), event)
	if err == nil {
		t.Fatal("expected error propagation, got nil")
	}
}
