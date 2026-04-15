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

	orderID, err := uuid.Parse(data.OrderId)
	if err != nil {
		return fmt.Errorf("invalid order_id %q in order.cancelled: %w", data.OrderId, err)
	}

	lines := make([]domain.CancellationLine, 0, len(data.LineItems))
	for _, li := range data.LineItems {
		if li.Quantity <= 0 {
			continue
		}
		skuID, err := uuid.Parse(li.SkuId)
		if err != nil {
			// One bad line shouldn't poison the whole batch. Idempotency is
			// preserved via the order-scoped movement guard in the service.
			slog.Warn("skipping cancellation line with invalid sku_id",
				"order_id", data.OrderId,
				"sku_id", li.SkuId,
				"error", err,
			)
			continue
		}
		lines = append(lines, domain.CancellationLine{
			SKUID:    skuID,
			Quantity: int(li.Quantity),
		})
	}

	slog.Info("releasing stock for cancelled order",
		"order_id", data.OrderId,
		"line_count", len(lines),
	)

	if err := s.svc.ReleaseStockForOrderCancellation(ctx, orderID, lines); err != nil {
		return fmt.Errorf("release stock for order %s: %w", data.OrderId, err)
	}
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
