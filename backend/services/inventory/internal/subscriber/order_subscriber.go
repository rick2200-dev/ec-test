// Package subscriber wires Pub/Sub event subscriptions for the
// inventory service. Today it owns a single subscription consuming
// order-events so stock can be released when an order is cancelled.
//
// Events are part of the public contract defined by the order service
// (see backend/services/order/internal/cancellation/events.go and
// docs/order-cancellation.md). The payload field names here must stay
// in sync with that file.
package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/service"
)

// orderEventsSubscription is the dedicated subscription for the
// inventory service. It is provisioned separately from the notification
// service's subscription so each subscriber gets its own at-least-once
// delivery cursor.
const orderEventsSubscription = "order-events-inventory"

// Event type constants (mirrored from cancellation/events.go so this
// package does not import the order service).
const (
	eventTypeOrderCancelled = "order.cancelled"
)

// CancellationReleaser is the narrow view of InventoryService that
// OrderSubscriber depends on. Declared as an interface so the
// subscriber can be unit-tested without spinning up a real Postgres.
// *service.InventoryService is the production implementation.
type CancellationReleaser interface {
	ReleaseStockForOrderCancellation(
		ctx context.Context,
		tenantID, orderID uuid.UUID,
		lines []service.CancellationLine,
	) error
}

// OrderSubscriber consumes order-events and releases stock for cancelled
// orders via a CancellationReleaser.
type OrderSubscriber struct {
	subscriber pubsub.Subscriber
	svc        CancellationReleaser
}

// NewOrderSubscriber constructs an OrderSubscriber.
func NewOrderSubscriber(subscriber pubsub.Subscriber, svc CancellationReleaser) *OrderSubscriber {
	return &OrderSubscriber{subscriber: subscriber, svc: svc}
}

// Start begins consuming events. It blocks until ctx is cancelled, so
// main.go should call it inside its own goroutine.
func (s *OrderSubscriber) Start(ctx context.Context) error {
	slog.Info("starting inventory order event subscriber", "subscription", orderEventsSubscription)
	return s.subscriber.Subscribe(ctx, orderEventsSubscription, s.handleEvent)
}

func (s *OrderSubscriber) handleEvent(ctx context.Context, event pubsub.Event) error {
	switch event.Type {
	case eventTypeOrderCancelled:
		return s.handleOrderCancelled(ctx, event)
	default:
		// The inventory service only cares about cancellations today.
		// All other order-events are ignored silently so they don't
		// get nacked and redelivered forever.
		return nil
	}
}

// cancelledLine mirrors cancellation.CancelledLineItem. Re-declared
// locally so this package does not import the order service.
type cancelledLine struct {
	SKUID       string `json:"sku_id"`
	ProductName string `json:"product_name"`
	SKUCode     string `json:"sku_code"`
	Quantity    int    `json:"quantity"`
}

// orderCancelledData mirrors the payload published by
// cancellation.publishOrderCancelled.
type orderCancelledData struct {
	OrderID      string          `json:"order_id"`
	TenantID     string          `json:"tenant_id"`
	SellerID     string          `json:"seller_id"`
	BuyerAuth0ID string          `json:"buyer_auth0_id"`
	RequestID    string          `json:"request_id"`
	Reason       string          `json:"reason"`
	LineItems    []cancelledLine `json:"line_items"`
}

func (s *OrderSubscriber) handleOrderCancelled(ctx context.Context, event pubsub.Event) error {
	var data orderCancelledData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.cancelled data: %w", err)
	}

	tenantID, err := uuid.Parse(data.TenantID)
	if err != nil {
		return fmt.Errorf("invalid tenant_id %q in order.cancelled: %w", data.TenantID, err)
	}
	orderID, err := uuid.Parse(data.OrderID)
	if err != nil {
		return fmt.Errorf("invalid order_id %q in order.cancelled: %w", data.OrderID, err)
	}

	lines := make([]service.CancellationLine, 0, len(data.LineItems))
	for _, li := range data.LineItems {
		if li.Quantity <= 0 {
			continue
		}
		skuID, err := uuid.Parse(li.SKUID)
		if err != nil {
			// One bad line shouldn't poison the whole batch — log
			// and skip. Idempotency is still correct: the movement
			// row for the order_id acts as the guard, and rerunning
			// will skip this line again the same way.
			slog.Warn("skipping cancellation line with invalid sku_id",
				"order_id", data.OrderID,
				"sku_id", li.SKUID,
				"error", err,
			)
			continue
		}
		lines = append(lines, service.CancellationLine{
			SKUID:    skuID,
			Quantity: li.Quantity,
		})
	}

	slog.Info("releasing stock for cancelled order",
		"order_id", data.OrderID,
		"tenant_id", data.TenantID,
		"line_count", len(lines),
	)

	if err := s.svc.ReleaseStockForOrderCancellation(ctx, tenantID, orderID, lines); err != nil {
		return fmt.Errorf("release stock for order %s: %w", data.OrderID, err)
	}
	return nil
}

// decodeEventData is the same normalization helper the notification
// service uses — event.Data comes back from JSON as a map[string]any,
// so round-trip through JSON to land it in a typed struct.
func decodeEventData(eventData any, target any) error {
	raw, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}
	if err := json.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("unmarshal event data: %w", err)
	}
	return nil
}
