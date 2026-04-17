// Package subscriber wires Pub/Sub event subscriptions for the inventory
// service. It owns a subscription consuming order-events so stock can be
// released when an order is cancelled. Payload schemas are defined in
// services/order/api/proto/order/v1/events.proto.
package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/inventory/internal/domain"
	orderv1 "github.com/Riku-KANO/ec-test/services/order/api/gen/go/order/v1"
)

const orderEventsSubscription = "order-events-inventory"

const eventTypeOrderCancelled = "order.cancelled"

// CancellationReleaser is the narrow view of InventoryService that
// OrderSubscriber depends on. Declared as an interface so the subscriber can
// be unit-tested without spinning up a real Postgres.
type CancellationReleaser interface {
	ReleaseStockForOrderCancellation(
		ctx context.Context,
		orderID uuid.UUID,
		lines []domain.CancellationLine,
	) error
}

type OrderSubscriber struct {
	subscriber pubsub.Subscriber
	svc        CancellationReleaser
}

func NewOrderSubscriber(subscriber pubsub.Subscriber, svc CancellationReleaser) *OrderSubscriber {
	return &OrderSubscriber{subscriber: subscriber, svc: svc}
}

func (s *OrderSubscriber) Start(ctx context.Context) error {
	slog.Info("starting inventory order event subscriber", "subscription", orderEventsSubscription)
	return s.subscriber.Subscribe(ctx, orderEventsSubscription, s.handleEvent)
}

func (s *OrderSubscriber) handleEvent(ctx context.Context, event pubsub.Event) error {
	switch event.Type {
	case eventTypeOrderCancelled:
		return s.handleOrderCancelled(ctx, event)
	default:
		return nil
	}
}

func (s *OrderSubscriber) handleOrderCancelled(ctx context.Context, event pubsub.Event) error {
	var data orderv1.OrderCancelled
	if err := decodeProtoEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.cancelled data: %w", err)
	}

	// The checkout flow does not yet call ReserveStock, so
	// quantity_reserved is never decremented at purchase time.
	// Releasing stock here without a prior reserve would inflate
	// quantity_available beyond the real physical count. Log the
	// event for observability but skip the actual release until the
	// full reserve→confirm→release cycle is wired up (requires
	// ReserveStock at checkout + ConfirmSold on order.paid).
	slog.Info("order.cancelled received; stock release skipped (no prior reserve)",
		"order_id", data.OrderId,
		"line_count", len(data.LineItems),
	)
	return nil
}

func decodeProtoEventData(eventData any, target protoreflect.ProtoMessage) error {
	raw, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("unmarshal proto event data: %w", err)
	}
	return nil
}
