package subscriber

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/notification/internal/email"
	"github.com/Riku-KANO/ec-test/services/notification/internal/templates"
)

const inventorySubscription = "inventory-events-notification"

// InventorySubscriber handles inventory-related events and sends notifications.
type InventorySubscriber struct {
	subscriber pubsub.Subscriber
	sender     email.Sender
}

// NewInventorySubscriber creates a new InventorySubscriber.
func NewInventorySubscriber(subscriber pubsub.Subscriber, sender email.Sender) *InventorySubscriber {
	return &InventorySubscriber{
		subscriber: subscriber,
		sender:     sender,
	}
}

// Start begins listening for inventory events. Blocks until context is cancelled.
func (s *InventorySubscriber) Start(ctx context.Context) error {
	slog.Info("starting inventory event subscriber", "subscription", inventorySubscription)
	return s.subscriber.Subscribe(ctx, inventorySubscription, s.handleEvent)
}

func (s *InventorySubscriber) handleEvent(ctx context.Context, event pubsub.Event) error {
	slog.Info("received inventory event",
		"event_id", event.ID,
		"event_type", event.Type,
		"tenant_id", event.TenantID,
	)

	switch event.Type {
	case "inventory.low_stock":
		return s.handleLowStock(ctx, event)
	default:
		slog.Warn("unhandled inventory event type", "event_type", event.Type)
		return nil
	}
}

type lowStockData struct {
	SKUCode      string `json:"sku_code"`
	ProductName  string `json:"product_name"`
	CurrentStock int    `json:"current_stock"`
	Threshold    int    `json:"threshold"`
	SellerEmail  string `json:"seller_email"`
}

func (s *InventorySubscriber) handleLowStock(ctx context.Context, event pubsub.Event) error {
	var data lowStockData
	if err := decodeEventData(event.Data, &data); err != nil {
		return fmt.Errorf("decode inventory.low_stock data: %w", err)
	}

	subject, body := templates.LowStockAlert(data.SKUCode, data.ProductName, data.CurrentStock, data.Threshold)

	slog.Info("sending low stock alert",
		"sku_code", data.SKUCode,
		"product_name", data.ProductName,
		"current_stock", data.CurrentStock,
		"threshold", data.Threshold,
		"seller_email", data.SellerEmail,
		"tenant_id", event.TenantID,
	)

	if err := s.sender.Send(ctx, data.SellerEmail, subject, body); err != nil {
		return fmt.Errorf("send low stock alert: %w", err)
	}
	return nil
}
