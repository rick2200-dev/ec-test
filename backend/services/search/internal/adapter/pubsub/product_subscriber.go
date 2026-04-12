package subscriber

import (
	"context"
	"encoding/json"
	"log/slog"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	"github.com/Riku-KANO/ec-test/services/search/internal/domain"
	"github.com/Riku-KANO/ec-test/services/search/internal/engine"
)

const (
	// SubscriptionID is the Pub/Sub subscription for product events.
	SubscriptionID = "product-events-search"
)

// ProductSubscriber subscribes to product events and updates the search index.
type ProductSubscriber struct {
	engine     engine.SearchEngine
	subscriber pubsub.Subscriber
}

// NewProductSubscriber creates a new ProductSubscriber.
func NewProductSubscriber(eng engine.SearchEngine, sub pubsub.Subscriber) *ProductSubscriber {
	return &ProductSubscriber{
		engine:     eng,
		subscriber: sub,
	}
}

// Start begins listening for product events.
func (s *ProductSubscriber) Start(ctx context.Context) error {
	slog.Info("starting product event subscriber", "subscription", SubscriptionID)
	return s.subscriber.Subscribe(ctx, SubscriptionID, s.handleEvent)
}

// handleEvent processes a single product event.
func (s *ProductSubscriber) handleEvent(ctx context.Context, event pubsub.Event) error {
	slog.Info("received product event", "type", event.Type, "tenant_id", event.TenantID, "event_id", event.ID)

	switch event.Type {
	case "product.created", "product.updated":
		return s.handleProductUpsert(ctx, event)
	case "product.deleted":
		return s.handleProductDeleted(ctx, event)
	default:
		slog.Warn("ignoring unknown event type", "type", event.Type)
		return nil
	}
}

// handleProductUpsert indexes or updates a product in the search engine.
func (s *ProductSubscriber) handleProductUpsert(ctx context.Context, event pubsub.Event) error {
	var product domain.ProductEvent
	data, err := json.Marshal(event.Data)
	if err != nil {
		slog.Error("failed to marshal event data", "error", err)
		return err
	}
	if err := json.Unmarshal(data, &product); err != nil {
		slog.Error("failed to unmarshal product event", "error", err)
		return err
	}

	if err := s.engine.IndexProduct(ctx, product); err != nil {
		slog.Error("failed to index product", "error", err, "product_id", product.ID)
		return err
	}

	slog.Info("product indexed", "product_id", product.ID, "event_type", event.Type)
	return nil
}

// handleProductDeleted removes a product from the search index.
func (s *ProductSubscriber) handleProductDeleted(ctx context.Context, event pubsub.Event) error {
	tenantID, err := uuid.Parse(event.TenantID)
	if err != nil {
		slog.Error("invalid tenant_id in event", "error", err, "tenant_id", event.TenantID)
		return err
	}

	// Extract product ID from the event data
	var data struct {
		ID uuid.UUID `json:"id"`
	}
	rawData, err := json.Marshal(event.Data)
	if err != nil {
		slog.Error("failed to marshal event data", "error", err)
		return err
	}
	if err := json.Unmarshal(rawData, &data); err != nil {
		slog.Error("failed to unmarshal delete event data", "error", err)
		return err
	}

	if err := s.engine.DeleteProduct(ctx, tenantID, data.ID); err != nil {
		slog.Error("failed to delete product from index", "error", err, "product_id", data.ID)
		return err
	}

	slog.Info("product removed from index", "product_id", data.ID, "event_type", event.Type)
	return nil
}
