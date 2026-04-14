package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
	"github.com/Riku-KANO/ec-test/services/order/internal/port"
)

const shippingSubscription = "shipping-events-order"

// ShippingSubscriber listens for shipping events and updates order status.
//
// When the shipping service marks a shipment as shipped or delivered, it
// publishes events to "shipping-events". This subscriber consumes them and
// advances the corresponding order status so that order queries remain
// consistent with the shipment state.
//
// Idempotency: the UpdateStatus call in OrderStore uses a conditional UPDATE
// so duplicate events are safe.
type ShippingSubscriber struct {
	subscriber pubsub.Subscriber
	orderStore port.OrderStore
}

// NewShippingSubscriber creates a new ShippingSubscriber.
func NewShippingSubscriber(sub pubsub.Subscriber, orderStore port.OrderStore) *ShippingSubscriber {
	return &ShippingSubscriber{
		subscriber: sub,
		orderStore: orderStore,
	}
}

// Start begins listening for shipping events. Blocks until ctx is cancelled.
func (s *ShippingSubscriber) Start(ctx context.Context) error {
	slog.Info("starting shipping event subscriber", "subscription", shippingSubscription)
	return s.subscriber.Subscribe(ctx, shippingSubscription, s.handleEvent)
}

func (s *ShippingSubscriber) handleEvent(ctx context.Context, event pubsub.Event) error {
	slog.Info("received shipping event",
		"event_id", event.ID,
		"event_type", event.Type,
		"tenant_id", event.TenantID,
	)

	switch event.Type {
	case "shipment.shipped":
		return s.handleShipmentShipped(ctx, event)
	case "shipment.delivered":
		return s.handleShipmentDelivered(ctx, event)
	default:
		slog.Debug("ignoring unhandled shipping event", "event_type", event.Type)
		return nil
	}
}

// shipmentShippedData mirrors domain.ShipmentShippedEvent from the shipping service.
// Field names must stay in sync with shipping/internal/domain/events.go.
type shipmentShippedData struct {
	OrderID string `json:"order_id"`
}

func (s *ShippingSubscriber) handleShipmentShipped(ctx context.Context, event pubsub.Event) error {
	var data shipmentShippedData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode shipment.shipped data: %w", err)
	}

	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		return fmt.Errorf("parse tenant_id: %w", err)
	}
	orderID, err := uuid.Parse(data.OrderID)
	if err != nil {
		return fmt.Errorf("parse order_id: %w", err)
	}

	if err := s.orderStore.UpdateStatus(ctx, tenantID, orderID, domain.StatusShipped); err != nil {
		return fmt.Errorf("update order status to shipped: %w", err)
	}

	slog.Info("order status updated to shipped", "order_id", data.OrderID)
	return nil
}

// shipmentDeliveredData mirrors domain.ShipmentDeliveredEvent.
type shipmentDeliveredData struct {
	OrderID string `json:"order_id"`
}

func (s *ShippingSubscriber) handleShipmentDelivered(ctx context.Context, event pubsub.Event) error {
	var data shipmentDeliveredData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode shipment.delivered data: %w", err)
	}

	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		return fmt.Errorf("parse tenant_id: %w", err)
	}
	orderID, err := uuid.Parse(data.OrderID)
	if err != nil {
		return fmt.Errorf("parse order_id: %w", err)
	}

	if err := s.orderStore.UpdateStatus(ctx, tenantID, orderID, domain.StatusDelivered); err != nil {
		return fmt.Errorf("update order status to delivered: %w", err)
	}

	slog.Info("order status updated to delivered", "order_id", data.OrderID)
	return nil
}

func decodeEventData(eventData any, target any) error {
	raw, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}
	return json.Unmarshal(raw, target)
}
