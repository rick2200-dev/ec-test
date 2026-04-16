package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	catalogv1 "github.com/Riku-KANO/ec-test/services/catalog/api/gen/go/catalog/v1"
)

// ProductSubscriber maintains recommend_svc.product_categories by reacting
// to catalog product lifecycle events. One subscription, three event types
// (product.created / product.updated / product.deleted). Keeps the local
// projection consistent with catalog by treating each event's category_ids
// as the full replacement set for the product.
type ProductSubscriber struct {
	pool       *pgxpool.Pool
	subscriber pubsub.Subscriber
}

// NewProductSubscriber wires a subscriber to the injected pool.
func NewProductSubscriber(pool *pgxpool.Pool, sub pubsub.Subscriber) *ProductSubscriber {
	return &ProductSubscriber{pool: pool, subscriber: sub}
}

// Start begins consuming product-events-recommend. Blocks until ctx is done.
func (s *ProductSubscriber) Start(ctx context.Context) error {
	slog.Info("starting product event subscriber", "subscription", "product-events-recommend")
	return s.subscriber.Subscribe(ctx, "product-events-recommend", s.handle)
}

func (s *ProductSubscriber) handle(ctx context.Context, event pubsub.Event) error {
	switch event.Type {
	case "product.created":
		var p catalogv1.ProductCreated
		if err := decodeProtoEvent(event.Data, &p); err != nil {
			return fmt.Errorf("decode product.created: %w", err)
		}
		return s.upsertCategories(ctx, p.Id, p.CategoryIds)
	case "product.updated":
		var p catalogv1.ProductUpdated
		if err := decodeProtoEvent(event.Data, &p); err != nil {
			return fmt.Errorf("decode product.updated: %w", err)
		}
		return s.upsertCategories(ctx, p.Id, p.CategoryIds)
	case "product.deleted":
		var p catalogv1.ProductDeleted
		if err := decodeProtoEvent(event.Data, &p); err != nil {
			return fmt.Errorf("decode product.deleted: %w", err)
		}
		return s.deleteCategories(ctx, p.Id)
	default:
		slog.Debug("ignoring unhandled product event type", "type", event.Type)
		return nil
	}
}

// upsertCategories applies the event's category_ids as the full replacement
// set for the product: delete rows that are no longer in the set, insert
// missing ones. The DELETE + INSERT runs in a Tx so readers never see a
// half-applied update.
func (s *ProductSubscriber) upsertCategories(ctx context.Context, productIDStr string, categoryIDs []string) error {
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		return fmt.Errorf("parse product_id %q: %w", productIDStr, err)
	}
	parsed := make([]uuid.UUID, 0, len(categoryIDs))
	for _, raw := range categoryIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			slog.Warn("skipping invalid category_id",
				"product_id", productID, "category_id", raw, "error", err)
			continue
		}
		parsed = append(parsed, id)
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`DELETE FROM recommend_svc.product_categories
			  WHERE product_id = $1
			    AND (cardinality($2::uuid[]) = 0 OR category_id <> ALL($2::uuid[]))`,
			productID, parsed,
		); err != nil {
			return fmt.Errorf("prune stale product_categories: %w", err)
		}
		for _, cid := range parsed {
			if _, err := tx.Exec(ctx,
				`INSERT INTO recommend_svc.product_categories (product_id, category_id, updated_at)
				 VALUES ($1, $2, NOW())
				 ON CONFLICT (product_id, category_id) DO UPDATE SET updated_at = NOW()`,
				productID, cid,
			); err != nil {
				return fmt.Errorf("upsert product_category (%s, %s): %w", productID, cid, err)
			}
		}
		return nil
	})
}

func (s *ProductSubscriber) deleteCategories(ctx context.Context, productIDStr string) error {
	productID, err := uuid.Parse(productIDStr)
	if err != nil {
		return fmt.Errorf("parse product_id %q: %w", productIDStr, err)
	}
	_, err = s.pool.Exec(ctx,
		`DELETE FROM recommend_svc.product_categories WHERE product_id = $1`,
		productID,
	)
	return err
}

func decodeProtoEvent(eventData any, target protoreflect.ProtoMessage) error {
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
