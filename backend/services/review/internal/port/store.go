// Package port defines the driven ports (outbound) and driving ports (inbound)
// for the review service.
package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/review/internal/domain"
)

// ReviewStore is the driven port for review persistence.
// *postgres.ReviewRepository satisfies this interface.
type ReviewStore interface {
	// RunInTx executes fn within a single tenant-scoped database transaction.
	// Repository methods called with the context passed to fn will join this
	// transaction instead of opening their own.
	RunInTx(ctx context.Context, tenantID uuid.UUID, fn func(ctx context.Context) error) error

	// Create persists a new review within the tenant.
	Create(ctx context.Context, tenantID uuid.UUID, review *domain.Review) error
	// GetByID retrieves a review with its reply by review ID.
	GetByID(ctx context.Context, tenantID, reviewID uuid.UUID) (*domain.Review, error)
	// Update persists changes to an existing review.
	Update(ctx context.Context, tenantID uuid.UUID, review *domain.Review) error
	// Delete removes a review (CASCADE deletes the reply).
	Delete(ctx context.Context, tenantID, reviewID uuid.UUID) error
	// ListByProduct returns paginated reviews for a product with replies, ordered by newest first.
	ListByProduct(ctx context.Context, tenantID, productID uuid.UUID, limit, offset int) ([]domain.Review, int, error)
	// ListBySeller returns paginated reviews on the seller's products.
	ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Review, int, error)

	// CreateReply persists a new seller reply.
	CreateReply(ctx context.Context, tenantID uuid.UUID, reply *domain.ReviewReply) error
	// UpdateReply persists changes to a seller reply.
	UpdateReply(ctx context.Context, tenantID uuid.UUID, reply *domain.ReviewReply) error
	// DeleteReply removes a seller reply.
	DeleteReply(ctx context.Context, tenantID, reviewID uuid.UUID) error
	// GetReplyByReview retrieves the seller reply for a review (nil if none).
	GetReplyByReview(ctx context.Context, tenantID, reviewID uuid.UUID) (*domain.ReviewReply, error)

	// GetProductRating retrieves the aggregate rating for a product (nil if none).
	GetProductRating(ctx context.Context, tenantID, productID uuid.UUID) (*domain.ProductRating, error)
	// UpsertProductRating atomically adjusts the aggregate rating for a product.
	// ratingDelta is added to rating_sum; countDelta is added to review_count.
	UpsertProductRating(ctx context.Context, tenantID, productID uuid.UUID, ratingDelta, countDelta int) error
}

// ProductLookup is the result of a product lookup from the catalog service.
// Defined in port so both the app layer and the httpclient adapter share the type.
type ProductLookup struct {
	ProductID   uuid.UUID   `json:"product_id"`
	SellerID    uuid.UUID   `json:"seller_id"`
	ProductName string      `json:"product_name"`
	SKUIDs      []uuid.UUID `json:"sku_ids"`
}

// CatalogClient looks up product details from the catalog service.
// *httpclient.CatalogClient satisfies this interface.
type CatalogClient interface {
	// GetProduct retrieves product metadata and SKU IDs.
	GetProduct(ctx context.Context, tenantID, productID uuid.UUID) (*ProductLookup, error)
}

// PurchaseCheckResult is the result of a purchase verification against the order service.
type PurchaseCheckResult struct {
	Purchased bool `json:"purchased"`
}

// PurchaseChecker verifies whether a buyer has purchased a given SKU from a seller.
// *httpclient.OrderClient satisfies this interface.
type PurchaseChecker interface {
	// CheckPurchase verifies whether the buyer has purchased the given SKU from the seller.
	CheckPurchase(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, sellerID, skuID uuid.UUID) (*PurchaseCheckResult, error)
}
