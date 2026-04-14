package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/shipping/internal/port"
)

const orderSubscription = "order-events-shipping"

// OrderSubscriber listens for order events and manages shipment lifecycle.
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
		"tenant_id", event.TenantID,
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

// orderPaidData mirrors the payload published by the order service for order.paid.
// Field names here must stay in sync with order/internal/domain/events.go.
type orderPaidData struct {
	OrderID             string `json:"order_id"`
	SellerID            string `json:"seller_id"`
	BuyerAuth0ID        string `json:"buyer_auth0_id"`
	ShippingAddressJSON string `json:"shipping_address_json"`
}

func (s *OrderSubscriber) handleOrderPaid(ctx context.Context, event pubsub.Event) error {
	var data orderPaidData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.paid data: %w", err)
	}

	// TenantID comes from the pubsub envelope, not the data payload.
	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		return fmt.Errorf("parse tenant_id from envelope: %w", err)
	}

	sellerID, err := uuid.Parse(data.SellerID)
	if err != nil {
		return fmt.Errorf("parse seller_id: %w", err)
	}
	orderID, err := uuid.Parse(data.OrderID)
	if err != nil {
		return fmt.Errorf("parse order_id: %w", err)
	}

	if err := s.svc.CreateShipment(ctx, port.CreateShipmentInput{
		TenantID:        tenantID,
		SellerID:        sellerID,
		OrderID:         orderID,
		BuyerAuth0ID:    data.BuyerAuth0ID,
		ShippingAddress: []byte(data.ShippingAddressJSON),
	}); err != nil {
		return fmt.Errorf("create shipment for order %s: %w", data.OrderID, err)
	}

	slog.Info("shipment created for order", "order_id", data.OrderID, "tenant_id", tenantID)
	return nil
}

// orderCancelledData mirrors the cancellation payload from the order service.
type orderCancelledData struct {
	OrderID string `json:"order_id"`
}

func (s *OrderSubscriber) handleOrderCancelled(ctx context.Context, event pubsub.Event) error {
	var data orderCancelledData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.cancelled data: %w", err)
	}

	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		return fmt.Errorf("parse tenant_id from envelope: %w", err)
	}

	orderID, err := uuid.Parse(data.OrderID)
	if err != nil {
		return fmt.Errorf("parse order_id: %w", err)
	}

	if err := s.svc.CancelShipment(ctx, tenantID, orderID); err != nil {
		return fmt.Errorf("cancel shipment for order %s: %w", data.OrderID, err)
	}

	slog.Info("shipment cancelled for order", "order_id", data.OrderID)
	return nil
}

func decodeEventData(eventData any, target any) error {
	raw, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}
	return json.Unmarshal(raw, target)
}
