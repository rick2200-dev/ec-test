package subscriber

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	catalogv1 "github.com/Riku-KANO/ec-test/services/catalog/api/gen/go/catalog/v1"
)

// SellerNameResolver is the narrow driven port the ProductSubscriber uses
// to snapshot seller_name onto the search_svc.products projection.
// *httpclient.SellerClient satisfies this interface.
type SellerNameResolver interface {
	BatchGetSellerNames(ctx context.Context, ids []uuid.UUID) (map[uuid.UUID]string, error)
}

// SubscriptionID is the Pub/Sub subscription for product events.
const SubscriptionID = "product-events-search"

// ProductSubscriber maintains search_svc.products by reacting to catalog
// product lifecycle events. Replaces the former cross-schema JOINs the
// search engine used to run against catalog_svc + auth_svc tables.
//
// seller_name is resolved via a HTTP call to auth's /internal/sellers/
// batch-get endpoint because auth doesn't emit seller events yet. Once
// it does, swap for a search_svc.sellers table + seller event subscriber
// and drop the per-event HTTP lookup.
type ProductSubscriber struct {
	pool       *pgxpool.Pool
	subscriber pubsub.Subscriber
	sellers    SellerNameResolver
}

// NewProductSubscriber wires a subscriber to the pool + sellers resolver.
func NewProductSubscriber(pool *pgxpool.Pool, sub pubsub.Subscriber, sellers SellerNameResolver) *ProductSubscriber {
	return &ProductSubscriber{pool: pool, subscriber: sub, sellers: sellers}
}

// Start begins consuming product-events-search. Blocks until ctx cancels.
func (s *ProductSubscriber) Start(ctx context.Context) error {
	slog.Info("starting product event subscriber", "subscription", SubscriptionID)
	return s.subscriber.Subscribe(ctx, SubscriptionID, s.handle)
}

func (s *ProductSubscriber) handle(ctx context.Context, event pubsub.Event) error {
	switch event.Type {
	case "product.created":
		var p catalogv1.ProductCreated
		if err := decodeProtoEvent(event.Data, &p); err != nil {
			return fmt.Errorf("decode product.created: %w", err)
		}
		return s.upsert(ctx,
			p.Id, p.SellerId, p.Name, p.Slug, p.Description, p.Status,
			firstOrEmpty(p.CategoryIds), p.CategoryName,
			p.CheapestPriceAmount, p.CheapestPriceCurrency)
	case "product.updated":
		var p catalogv1.ProductUpdated
		if err := decodeProtoEvent(event.Data, &p); err != nil {
			return fmt.Errorf("decode product.updated: %w", err)
		}
		return s.upsert(ctx,
			p.Id, p.SellerId, p.Name, p.Slug, p.Description, p.Status,
			firstOrEmpty(p.CategoryIds), p.CategoryName,
			p.CheapestPriceAmount, p.CheapestPriceCurrency)
	case "product.deleted":
		var p catalogv1.ProductDeleted
		if err := decodeProtoEvent(event.Data, &p); err != nil {
			return fmt.Errorf("decode product.deleted: %w", err)
		}
		return s.delete(ctx, p.Id)
	default:
		slog.Warn("ignoring unknown event type", "type", event.Type)
		return nil
	}
}

func (s *ProductSubscriber) upsert(
	ctx context.Context,
	idStr, sellerIDStr, name, slug, description, status, categoryIDStr, categoryName string,
	priceAmount int64, priceCurrency string,
) error {
	productID, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("parse product id %q: %w", idStr, err)
	}
	sellerID, err := uuid.Parse(sellerIDStr)
	if err != nil {
		return fmt.Errorf("parse seller_id %q: %w", sellerIDStr, err)
	}

	var categoryID *uuid.UUID
	if categoryIDStr != "" {
		cid, err := uuid.Parse(categoryIDStr)
		if err != nil {
			slog.Warn("skipping invalid category_id on product event",
				"product_id", productID, "category_id", categoryIDStr, "error", err)
		} else {
			categoryID = &cid
		}
	}

	// Resolve seller_name via auth. A transient failure should not block
	// the upsert — degrade to empty name and let a future event correct
	// the row. Search queries COALESCE the column anyway.
	sellerName := ""
	if s.sellers != nil {
		if names, err := s.sellers.BatchGetSellerNames(ctx, []uuid.UUID{sellerID}); err != nil {
			slog.Warn("seller name lookup failed; indexing product without it",
				"product_id", productID, "seller_id", sellerID, "error", err)
		} else {
			sellerName = names[sellerID]
		}
	}

	var amountArg any = priceAmount
	var currencyArg any = priceCurrency
	if priceCurrency == "" {
		amountArg = nil
		currencyArg = nil
	}

	_, err = s.pool.Exec(ctx,
		`INSERT INTO search_svc.products
		   (id, seller_id, seller_name, category_id, category_name, name, slug, description, status, price_amount, price_currency, search_vector, updated_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11,
		         setweight(to_tsvector('simple', COALESCE($6, '')), 'A') ||
		         setweight(to_tsvector('simple', COALESCE($8, '')), 'B'),
		         NOW())
		 ON CONFLICT (id) DO UPDATE SET
		   seller_id       = EXCLUDED.seller_id,
		   seller_name     = EXCLUDED.seller_name,
		   category_id     = EXCLUDED.category_id,
		   category_name   = EXCLUDED.category_name,
		   name            = EXCLUDED.name,
		   slug            = EXCLUDED.slug,
		   description     = EXCLUDED.description,
		   status          = EXCLUDED.status,
		   price_amount    = EXCLUDED.price_amount,
		   price_currency  = EXCLUDED.price_currency,
		   search_vector   = EXCLUDED.search_vector,
		   updated_at      = NOW()`,
		productID, sellerID, sellerName, categoryID, categoryName, name, slug, description, status, amountArg, currencyArg,
	)
	if err != nil {
		return fmt.Errorf("upsert search_svc.products: %w", err)
	}
	slog.Debug("product indexed", "product_id", productID, "seller_id", sellerID)
	return nil
}

func (s *ProductSubscriber) delete(ctx context.Context, idStr string) error {
	productID, err := uuid.Parse(idStr)
	if err != nil {
		return fmt.Errorf("parse product id %q: %w", idStr, err)
	}
	_, err = s.pool.Exec(ctx, `DELETE FROM search_svc.products WHERE id = $1`, productID)
	return err
}

func firstOrEmpty(xs []string) string {
	if len(xs) == 0 {
		return ""
	}
	return xs[0]
}
