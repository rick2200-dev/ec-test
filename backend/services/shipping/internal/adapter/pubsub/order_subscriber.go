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
	orderv1 "github.com/Riku-KANO/ec-test/services/order/api/gen/go/order/v1"
	"github.com/Riku-KANO/ec-test/services/shipping/internal/port"
)

const orderSubscription = "order-events-shipping"

// OrderSubscriber listens for order events and manages shipment lifecycle.
// Payload schemas are defined in services/order/api/proto/order/v1/events.proto.
type OrderSubscriber struct {
	subscriber pubsub.Subscriber
	svc        port.ShipmentUseCase
}

// NewOrderSubscriber creates a new OrderSubscriber.
func NewOrderSubscriber(sub pubsub.Subscriber, svc port.ShipmentUseCase) *OrderSubscriber {
	return &OrderSubscriber{subscriber: sub, svc: svc}
}

// Start begins listening for order events. Blocks until ctx is cancelled.
func (s *OrderSubscriber) Start(ctx context.Context) error {
	slog.Info("starting order event subscriber", "subscription", orderSubscription)
	return s.subscriber.Subscribe(ctx, orderSubscription, s.handleEvent)
}

func (s *OrderSubscriber) handleEvent(ctx context.Context, event pubsub.Event) error {
	slog.Info("received order event",
		"event_id", event.ID,
		"event_type", event.Type,
	)
	switch event.Type {
	case "order.paid":
		return s.handleOrderPaid(ctx, event)
	case "order.cancelled":
		return s.handleOrderCancelled(ctx, event)
	default:
		slog.Debug("ignoring unhandled order event", "event_type", event.Type)
		return nil
	}
}

func (s *OrderSubscriber) handleOrderPaid(ctx context.Context, event pubsub.Event) error {
	var data orderv1.OrderPaid
	if err := decodeProtoEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.paid data: %w", err)
	}

	sellerID, err := uuid.Parse(data.SellerId)
	if err != nil {
		return fmt.Errorf("parse seller_id: %w", err)
	}
	orderID, err := uuid.Parse(data.OrderId)
	if err != nil {
		return fmt.Errorf("parse order_id: %w", err)
	}

	if err := s.svc.CreateShipment(ctx, port.CreateShipmentInput{
		SellerID:        sellerID,
		OrderID:         orderID,
		BuyerAuth0ID:    data.BuyerAuth0Id,
		ShippingAddress: []byte(data.ShippingAddressJson),
	}); err != nil {
		return fmt.Errorf("create shipment for order %s: %w", data.OrderId, err)
	}

	slog.Info("shipment created for order", "order_id", data.OrderId)
	return nil
}

func (s *OrderSubscriber) handleOrderCancelled(ctx context.Context, event pubsub.Event) error {
	var data orderv1.OrderCancelled
	if err := decodeProtoEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.cancelled data: %w", err)
	}

	orderID, err := uuid.Parse(data.OrderId)
	if err != nil {
		return fmt.Errorf("parse order_id: %w", err)
	}

	if err := s.svc.CancelShipment(ctx, orderID); err != nil {
		return fmt.Errorf("cancel shipment for order %s: %w", data.OrderId, err)
	}

	slog.Info("shipment cancelled for order", "order_id", data.OrderId)
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
