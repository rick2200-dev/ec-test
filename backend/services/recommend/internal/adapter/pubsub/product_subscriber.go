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
		return s.applyProductCreated(ctx, &p)
	case "product.updated":
		var p catalogv1.ProductUpdated
		if err := decodeProtoEvent(event.Data, &p); err != nil {
			return fmt.Errorf("decode product.updated: %w", err)
		}
		return s.applyProductUpdated(ctx, &p)
	case "product.deleted":
		var p catalogv1.ProductDeleted
		if err := decodeProtoEvent(event.Data, &p); err != nil {
			return fmt.Errorf("decode product.deleted: %w", err)
		}
		return s.applyProductDeleted(ctx, p.Id)
	default:
		slog.Debug("ignoring unhandled product event type", "type", event.Type)
		return nil
	}
}

// applyProductCreated / applyProductUpdated upsert both the products row
// and its category assignments in a single Tx so the projection never
// shows a half-applied update to query readers.
func (s *ProductSubscriber) applyProductCreated(ctx context.Context, p *catalogv1.ProductCreated) error {
	return s.applyProductFields(ctx, p.Id, p.SellerId, p.Name, p.Slug, p.Description, p.Status,
		p.CategoryIds, p.CheapestPriceAmount, p.CheapestPriceCurrency)
}

func (s *ProductSubscriber) applyProductUpdated(ctx context.Context, p *catalogv1.ProductUpdated) error {
	return s.applyProductFields(ctx, p.Id, p.SellerId, p.Name, p.Slug, p.Description, p.Status,
		p.CategoryIds, p.CheapestPriceAmount, p.CheapestPriceCurrency)
}

func (s *ProductSubscriber) applyProductFields(
	ctx context.Context,
	idStr, sellerIDStr, name, slug, description, status string,
	categoryIDs []string,
	priceAmount int64,
	priceCurrency string,
) error {
	productID, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("parse product_id %q: %w", idStr, err)
	}
	sellerID, err := uuid.Parse(sellerIDStr)
	if err != nil {
		return fmt.Errorf("parse seller_id %q: %w", sellerIDStr, err)
	}
	parsedCats := make([]uuid.UUID, 0, len(categoryIDs))
	for _, raw := range categoryIDs {
		id, err := uuid.Parse(raw)
		if err != nil {
			slog.Warn("skipping invalid category_id",
				"product_id", productID, "category_id", raw, "error", err)
			continue
		}
		parsedCats = append(parsedCats, id)
	}

	// Empty currency + zero amount from the publisher mean "no SKUs yet".
	// Store NULL rather than 0/'' so downstream filters can distinguish.
	var amountArg any = priceAmount
	var currencyArg any = priceCurrency
	if priceCurrency == "" {
		amountArg = nil
		currencyArg = nil
	}

	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`INSERT INTO recommend_svc.products
			   (id, seller_id, name, slug, description, status, cheapest_price_amount, cheapest_price_currency, updated_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, NOW())
			 ON CONFLICT (id) DO UPDATE SET
			   seller_id               = EXCLUDED.seller_id,
			   name                    = EXCLUDED.name,
			   slug                    = EXCLUDED.slug,
			   description             = EXCLUDED.description,
			   status                  = EXCLUDED.status,
			   cheapest_price_amount   = EXCLUDED.cheapest_price_amount,
			   cheapest_price_currency = EXCLUDED.cheapest_price_currency,
			   updated_at              = NOW()`,
			productID, sellerID, name, slug, description, status, amountArg, currencyArg,
		); err != nil {
			return fmt.Errorf("upsert recommend_svc.products: %w", err)
		}
		if _, err := tx.Exec(ctx,
			`DELETE FROM recommend_svc.product_categories
			  WHERE product_id = $1
			    AND (cardinality($2::uuid[]) = 0 OR category_id <> ALL($2::uuid[]))`,
			productID, parsedCats,
		); err != nil {
			return fmt.Errorf("prune product_categories: %w", err)
		}
		for _, cid := range parsedCats {
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

func (s *ProductSubscriber) applyProductDeleted(ctx context.Context, idStr string) error {
	productID, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("parse product_id %q: %w", idStr, err)
	}
	return pgx.BeginFunc(ctx, s.pool, func(tx pgx.Tx) error {
		if _, err := tx.Exec(ctx,
			`DELETE FROM recommend_svc.product_categories WHERE product_id = $1`,
			productID); err != nil {
			return err
		}
		_, err := tx.Exec(ctx,
			`DELETE FROM recommend_svc.products WHERE id = $1`,
			productID)
		return err
	})
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
