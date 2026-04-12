package port

import (
	"context"

	"github.com/google/uuid"

	"github.com/Riku-KANO/ec-test/services/review/internal/domain"
)

// ReviewUseCase is the driving port (inbound) for review operations.
// Handlers depend on this interface; *app.ReviewService satisfies it.
type ReviewUseCase interface {
	// CreateReview creates a new review after verifying the buyer's purchase.
	CreateReview(ctx context.Context, tenantID uuid.UUID, buyerAuth0ID string, in domain.CreateReviewInput) (*domain.Review, error)
	// UpdateReview updates the caller's own review.
	UpdateReview(ctx context.Context, tenantID, reviewID uuid.UUID, buyerAuth0ID string, in domain.UpdateReviewInput) (*domain.Review, error)
	// DeleteReview deletes the caller's own review.
	DeleteReview(ctx context.Context, tenantID, reviewID uuid.UUID, buyerAuth0ID string) error
	// GetReview retrieves a single review with its reply.
	GetReview(ctx context.Context, tenantID, reviewID uuid.UUID) (*domain.Review, error)

	// ListByProduct returns paginated reviews for a product, ordered by newest first.
	ListByProduct(ctx context.Context, tenantID, productID uuid.UUID, limit, offset int) ([]domain.Review, int, error)
	// GetProductRating returns the aggregate rating for a product.
	GetProductRating(ctx context.Context, tenantID, productID uuid.UUID) (*domain.ProductRating, error)

	// ListForSeller returns paginated reviews on the seller's products.
	ListForSeller(ctx context.Context, tenantID, sellerID uuid.UUID, limit, offset int) ([]domain.Review, int, error)
	// CreateReply creates a seller reply on a review.
	CreateReply(ctx context.Context, tenantID, reviewID uuid.UUID, sellerAuth0ID string, sellerID uuid.UUID, in domain.CreateReplyInput) (*domain.ReviewReply, error)
	// UpdateReply updates a seller reply.
	UpdateReply(ctx context.Context, tenantID, reviewID uuid.UUID, sellerAuth0ID string, sellerID uuid.UUID, in domain.UpdateReplyInput) (*domain.ReviewReply, error)
	// DeleteReply deletes a seller reply.
	DeleteReply(ctx context.Context, tenantID, reviewID uuid.UUID, sellerID uuid.UUID) error
}
