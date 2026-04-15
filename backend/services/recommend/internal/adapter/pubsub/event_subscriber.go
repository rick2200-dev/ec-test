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
	"github.com/Riku-KANO/ec-test/services/recommend/internal/domain"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/port"
)

// EventSubscriber handles Pub/Sub event subscriptions for the recommend service.
type EventSubscriber struct {
	svc        port.RecommendUseCase
	subscriber pubsub.Subscriber
}

// NewEventSubscriber creates a new EventSubscriber.
func NewEventSubscriber(svc port.RecommendUseCase, sub pubsub.Subscriber) *EventSubscriber {
	return &EventSubscriber{svc: svc, subscriber: sub}
}

// Start subscribes to all relevant event topics.
func (s *EventSubscriber) Start(ctx context.Context) error {
	// Subscribe to user behavior events.
	go func() {
		if err := s.subscriber.Subscribe(ctx, "user-events-recommend", s.handleUserEvent); err != nil {
			slog.Error("user-events-recommend subscription error", "error", err)
		}
	}()

	// Subscribe to order events.
	go func() {
		if err := s.subscriber.Subscribe(ctx, "order-events-recommend", s.handleOrderEvent); err != nil {
			slog.Error("order-events-recommend subscription error", "error", err)
		}
	}()

	return nil
}

// userEventData represents the data payload for user behavior events.
type userEventData struct {
	UserID    string `json:"user_id"`
	EventType string `json:"event_type"`
	ProductID string `json:"product_id"`
}

// handleUserEvent processes incoming user behavior events and records them.
func (s *EventSubscriber) handleUserEvent(ctx context.Context, event pubsub.Event) error {
	dataBytes, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}

	var data userEventData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		slog.Error("failed to decode user event data", "error", err)
		return fmt.Errorf("unmarshal user event data: %w", err)
	}

	productID, err := uuid.Parse(data.ProductID)
	if err != nil {
		return fmt.Errorf("parse product_id: %w", err)
	}

	ue := domain.UserEvent{
		UserID:    data.UserID,
		EventType: domain.UserEventType(data.EventType),
		ProductID: productID,
	}

	if err := s.svc.RecordUserEvent(ctx, ue); err != nil {
		slog.Error("failed to record user event from pubsub", "error", err)
		return err
	}

	slog.Debug("recorded user event", "type", data.EventType, "product_id", data.ProductID)
	return nil
}

// handleOrderEvent processes order.paid events as a "purchased" signal for the
// buyer. The canonical payload is orderv1.OrderPaid which carries order_id +
// buyer_auth0_id + seller_id + total_amount but no line items — per-line
// product affinity would require the order service to include lines on
// order.paid or a callback to order. For now we record a single purchased
// event keyed on (buyer_auth0_id, order_id) so the recommender still knows a
// purchase happened.
func (s *EventSubscriber) handleOrderEvent(ctx context.Context, event pubsub.Event) error {
	if event.Type != "order.paid" {
		return nil
	}

	var data orderv1.OrderPaid
	if err := decodeProtoEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode order.paid data: %w", err)
	}

	orderID, err := uuid.Parse(data.OrderId)
	if err != nil {
		return fmt.Errorf("parse order_id %q: %w", data.OrderId, err)
	}

	ue := domain.UserEvent{
		UserID:    data.BuyerAuth0Id,
		EventType: domain.Purchased,
		ProductID: orderID, // Order-scoped signal until line-level data is in the contract.
	}
	if err := s.svc.RecordUserEvent(ctx, ue); err != nil {
		slog.Error("failed to record purchase event", "order_id", data.OrderId, "error", err)
		return err
	}

	slog.Info("processed order.paid event", "order_id", data.OrderId, "buyer_auth0_id", data.BuyerAuth0Id)
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
