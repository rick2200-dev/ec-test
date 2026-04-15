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
	catalogv1 "github.com/Riku-KANO/ec-test/services/catalog/api/gen/go/catalog/v1"
	"github.com/Riku-KANO/ec-test/services/search/internal/domain"
	"github.com/Riku-KANO/ec-test/services/search/internal/engine"
)

// SubscriptionID is the Pub/Sub subscription for product events.
const SubscriptionID = "product-events-search"

// ProductSubscriber subscribes to catalog product events and updates the
// search index. The payload contract is defined in
// services/catalog/api/proto/catalog/v1/events.proto.
type ProductSubscriber struct {
	engine     engine.SearchEngine
	subscriber pubsub.Subscriber
}

func NewProductSubscriber(eng engine.SearchEngine, sub pubsub.Subscriber) *ProductSubscriber {
	return &ProductSubscriber{engine: eng, subscriber: sub}
}

func (s *ProductSubscriber) Start(ctx context.Context) error {
	slog.Info("starting product event subscriber", "subscription", SubscriptionID)
	return s.subscriber.Subscribe(ctx, SubscriptionID, s.handleEvent)
}

func (s *ProductSubscriber) handleEvent(ctx context.Context, event pubsub.Event) error {
	slog.Info("received product event", "type", event.Type, "event_id", event.ID)

	switch event.Type {
	case "product.created":
		var p catalogv1.ProductCreated
		if err := decodeProtoEventData(event.Data, &p); err != nil {
			return err
		}
		return s.indexFromCreated(ctx, &p, event.Type)
	case "product.updated":
		var p catalogv1.ProductUpdated
		if err := decodeProtoEventData(event.Data, &p); err != nil {
			return err
		}
		return s.indexFromUpdated(ctx, &p, event.Type)
	case "product.deleted":
		var p catalogv1.ProductDeleted
		if err := decodeProtoEventData(event.Data, &p); err != nil {
			return err
		}
		id, err := uuid.Parse(p.Id)
		if err != nil {
			return fmt.Errorf("parse product id %q: %w", p.Id, err)
		}
		if err := s.engine.DeleteProduct(ctx, id); err != nil {
			return fmt.Errorf("delete product %s: %w", id, err)
		}
		slog.Info("product removed from index", "product_id", id)
		return nil
	default:
		slog.Warn("ignoring unknown event type", "type", event.Type)
		return nil
	}
}

func (s *ProductSubscriber) indexFromCreated(ctx context.Context, p *catalogv1.ProductCreated, eventType string) error {
	return s.indexProduct(ctx, p.Id, p.SellerId, p.Name, p.Slug, p.Description, p.Status, eventType)
}

func (s *ProductSubscriber) indexFromUpdated(ctx context.Context, p *catalogv1.ProductUpdated, eventType string) error {
	return s.indexProduct(ctx, p.Id, p.SellerId, p.Name, p.Slug, p.Description, p.Status, eventType)
}

func (s *ProductSubscriber) indexProduct(ctx context.Context, idStr, sellerIDStr, name, slug, description, status, eventType string) error {
	id, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("parse product id %q: %w", idStr, err)
	}
	sellerID, err := uuid.Parse(sellerIDStr)
	if err != nil {
		return fmt.Errorf("parse seller_id %q: %w", sellerIDStr, err)
	}
	product := domain.ProductEvent{
		ID:          id,
		SellerID:    sellerID,
		Name:        name,
		Slug:        slug,
		Description: description,
		Status:      status,
	}
	if err := s.engine.IndexProduct(ctx, product); err != nil {
		return fmt.Errorf("index product %s: %w", id, err)
	}
	slog.Info("product indexed", "product_id", id, "event_type", eventType)
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
