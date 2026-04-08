package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/domain"
	"github.com/Riku-KANO/ec-test/services/recommend/internal/service"
)

// EventSubscriber handles Pub/Sub event subscriptions for the recommend service.
type EventSubscriber struct {
	svc        *service.RecommendService
	subscriber pubsub.Subscriber
}

// NewEventSubscriber creates a new EventSubscriber.
func NewEventSubscriber(svc *service.RecommendService, sub pubsub.Subscriber) *EventSubscriber {
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

	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		return fmt.Errorf("parse tenant_id: %w", err)
	}

	productID, err := uuid.Parse(data.ProductID)
	if err != nil {
		return fmt.Errorf("parse product_id: %w", err)
	}

	ue := domain.UserEvent{
		TenantID:  tenantID,
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

// orderEventData represents the data payload for order events.
type orderEventData struct {
	OrderID string          `json:"order_id"`
	UserID  string          `json:"user_id"`
	Lines   []orderLineData `json:"lines"`
}

// orderLineData represents a single line item in an order.
type orderLineData struct {
	ProductID string `json:"product_id"`
	Quantity  int    `json:"quantity"`
}

// handleOrderEvent processes order.paid events and records "purchased" events for
// all order line items.
func (s *EventSubscriber) handleOrderEvent(ctx context.Context, event pubsub.Event) error {
	if event.Type != "order.paid" {
		return nil // only process paid orders
	}

	dataBytes, err := json.Marshal(event.Data)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}

	var data orderEventData
	if err := json.Unmarshal(dataBytes, &data); err != nil {
		slog.Error("failed to decode order event data", "error", err)
		return fmt.Errorf("unmarshal order event data: %w", err)
	}

	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		return fmt.Errorf("parse tenant_id: %w", err)
	}

	for _, line := range data.Lines {
		productID, err := uuid.Parse(line.ProductID)
		if err != nil {
			slog.Error("failed to parse product_id in order line", "product_id", line.ProductID, "error", err)
			continue
		}

		ue := domain.UserEvent{
			TenantID:  tenantID,
			UserID:    data.UserID,
			EventType: domain.Purchased,
			ProductID: productID,
		}

		if err := s.svc.RecordUserEvent(ctx, ue); err != nil {
			slog.Error("failed to record purchase event", "product_id", line.ProductID, "error", err)
			// Continue processing other lines even if one fails.
		}
	}

	slog.Info("processed order.paid event", "order_id", data.OrderID, "lines", len(data.Lines))
	return nil
}
