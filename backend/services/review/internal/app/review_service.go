package app

import (
	"context"
	"log/slog"
	"strings"

	"github.com/google/uuid"

	apperrors "github.com/Riku-KANO/ec-test/pkg/errors"
	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	reviewv1 "github.com/Riku-KANO/ec-test/services/review/api/gen/go/review/v1"
	"github.com/Riku-KANO/ec-test/services/review/internal/domain"
	"github.com/Riku-KANO/ec-test/services/review/internal/port"
)

const reviewEventTopic = "review-events"

// ReviewService holds the business logic for product reviews.
type ReviewService struct {
	repo      port.ReviewStore
	catalog   port.CatalogClient
	purchase  port.PurchaseChecker
	publisher pubsub.Publisher
}

// NewReviewService constructs a ReviewService.
func NewReviewService(
	repo port.ReviewStore,
	catalog port.CatalogClient,
	purchase port.PurchaseChecker,
	publisher pubsub.Publisher,
) *ReviewService {
	return &ReviewService{
		repo:      repo,
		catalog:   catalog,
		purchase:  purchase,
		publisher: publisher,
	}
}

// CreateReview creates a new review after verifying the buyer's purchase.
func (s *ReviewService) CreateReview(
	ctx context.Context,
	buyerAuth0ID string,
	in domain.CreateReviewInput,
) (*domain.Review, error) {
	if buyerAuth0ID == "" {
		return nil, apperrors.BadRequest("buyer_auth0_id is required")
	}
	if in.ProductID == uuid.Nil {
		return nil, apperrors.BadRequest("product_id is required")
	}
	if in.Rating < 1 || in.Rating > 5 {
		return nil, domain.ErrInvalidRating
	}
	in.Title = strings.TrimSpace(in.Title)
	in.Body = strings.TrimSpace(in.Body)
	if in.Title == "" {
		return nil, apperrors.BadRequest("title is required")
	}
	if in.Body == "" {
		return nil, apperrors.BadRequest("body is required")
	}
	if len(in.Title) > 255 {
		return nil, apperrors.BadRequest("title must be at most 255 characters")
	}
	if len(in.Body) > 4000 {
		return nil, apperrors.BadRequest("body must be at most 4000 characters")
	}

	// Look up product details from catalog.
	product, err := s.catalog.GetProduct(ctx, in.ProductID)
	if err != nil {
		return nil, err
	}

	// Verify purchase: check each SKU until one passes.
	// Track errors separately from "not purchased" to distinguish between
	// service failures and actual non-purchase.
	purchased := false
	var lastErr error
	errCount := 0
	for _, skuID := range product.SKUIDs {
		result, err := s.purchase.CheckPurchase(ctx, buyerAuth0ID, product.SellerID, skuID)
		if err != nil {
			lastErr = err
			errCount++
			slog.Warn("purchase check failed for SKU",
				"sku_id", skuID,
				"product_id", in.ProductID,
				"error", err,
			)
			continue
		}
		if result.Purchased {
			purchased = true
			break
		}
	}
	if !purchased {
		// If any SKU check failed with an error we cannot be sure whether
		// the buyer actually purchased — surface a 5xx so the caller retries
		// instead of incorrectly rejecting with "purchase required".
		if errCount > 0 {
			return nil, apperrors.Internal("purchase check partially failed", lastErr)
		}
		return nil, domain.ErrPurchaseRequired
	}

	review := &domain.Review{
		BuyerAuth0ID: buyerAuth0ID,
		ProductID:    in.ProductID,
		SellerID:     product.SellerID,
		ProductName:  product.ProductName,
		Rating:       in.Rating,
		Title:        in.Title,
		Body:         in.Body,
	}

	// Create review and update aggregate rating in a single transaction so
	// they cannot diverge.
	if err := s.repo.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Create(txCtx, review); err != nil {
			return err
		}
		return s.repo.UpsertProductRating(txCtx, in.ProductID, in.Rating, 1)
	}); err != nil {
		return nil, err
	}

	pubsub.PublishProtoEvent(ctx, s.publisher, "review.created", reviewEventTopic, &reviewv1.ReviewCreated{
		ReviewId:     review.ID.String(),
		ProductId:    review.ProductID.String(),
		SellerId:     review.SellerID.String(),
		BuyerAuth0Id: review.BuyerAuth0ID,
		Rating:       int32(review.Rating),
		Title:        review.Title,
		ProductName:  review.ProductName,
	})

	return review, nil
}

// UpdateReview updates the caller's own review.
func (s *ReviewService) UpdateReview(
	ctx context.Context,
	reviewID uuid.UUID,
	buyerAuth0ID string,
	in domain.UpdateReviewInput,
) (*domain.Review, error) {
	review, err := s.repo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, apperrors.Internal("failed to load review", err)
	}
	if review == nil {
		return nil, domain.ErrReviewNotFound
	}
	if review.BuyerAuth0ID != buyerAuth0ID {
		return nil, domain.ErrNotReviewOwner
	}

	oldRating := review.Rating

	if in.Rating != nil {
		if *in.Rating < 1 || *in.Rating > 5 {
			return nil, domain.ErrInvalidRating
		}
		review.Rating = *in.Rating
	}
	if in.Title != nil {
		title := strings.TrimSpace(*in.Title)
		if title == "" {
			return nil, apperrors.BadRequest("title is required")
		}
		if len(title) > 255 {
			return nil, apperrors.BadRequest("title must be at most 255 characters")
		}
		review.Title = title
	}
	if in.Body != nil {
		body := strings.TrimSpace(*in.Body)
		if body == "" {
			return nil, apperrors.BadRequest("body is required")
		}
		if len(body) > 4000 {
			return nil, apperrors.BadRequest("body must be at most 4000 characters")
		}
		review.Body = body
	}

	// Update review and adjust aggregate rating in a single transaction.
	ratingDelta := review.Rating - oldRating
	if err := s.repo.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Update(txCtx, review); err != nil {
			return err
		}
		if ratingDelta != 0 {
			return s.repo.UpsertProductRating(txCtx, review.ProductID, ratingDelta, 0)
		}
		return nil
	}); err != nil {
		return nil, apperrors.Internal("failed to update review", err)
	}

	pubsub.PublishProtoEvent(ctx, s.publisher, "review.updated", reviewEventTopic, &reviewv1.ReviewUpdated{
		ReviewId:     review.ID.String(),
		ProductId:    review.ProductID.String(),
		SellerId:     review.SellerID.String(),
		BuyerAuth0Id: review.BuyerAuth0ID,
		OldRating:    int32(oldRating),
		NewRating:    int32(review.Rating),
		ProductName:  review.ProductName,
	})

	return review, nil
}

// DeleteReview deletes the caller's own review.
func (s *ReviewService) DeleteReview(
	ctx context.Context,
	reviewID uuid.UUID,
	buyerAuth0ID string,
) error {
	review, err := s.repo.GetByID(ctx, reviewID)
	if err != nil {
		return apperrors.Internal("failed to load review", err)
	}
	if review == nil {
		return domain.ErrReviewNotFound
	}
	if review.BuyerAuth0ID != buyerAuth0ID {
		return domain.ErrNotReviewOwner
	}

	// Delete review and adjust aggregate rating in a single transaction.
	if err := s.repo.RunInTx(ctx, func(txCtx context.Context) error {
		if err := s.repo.Delete(txCtx, reviewID); err != nil {
			return err
		}
		return s.repo.UpsertProductRating(txCtx, review.ProductID, -review.Rating, -1)
	}); err != nil {
		return apperrors.Internal("failed to delete review", err)
	}

	pubsub.PublishProtoEvent(ctx, s.publisher, "review.deleted", reviewEventTopic, &reviewv1.ReviewDeleted{
		ReviewId:     review.ID.String(),
		ProductId:    review.ProductID.String(),
		SellerId:     review.SellerID.String(),
		BuyerAuth0Id: review.BuyerAuth0ID,
		ProductName:  review.ProductName,
	})

	return nil
}

// GetReview retrieves a single review with its reply.
func (s *ReviewService) GetReview(ctx context.Context, reviewID uuid.UUID) (*domain.Review, error) {
	review, err := s.repo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, apperrors.Internal("failed to load review", err)
	}
	if review == nil {
		return nil, domain.ErrReviewNotFound
	}
	return review, nil
}

// ListByProduct returns paginated reviews for a product.
func (s *ReviewService) ListByProduct(
	ctx context.Context,
	productID uuid.UUID,
	limit, offset int,
) ([]domain.Review, int, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	items, total, err := s.repo.ListByProduct(ctx, productID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list reviews", err)
	}
	return items, total, nil
}

// GetProductRating returns the aggregate rating for a product.
func (s *ReviewService) GetProductRating(ctx context.Context, productID uuid.UUID) (*domain.ProductRating, error) {
	rating, err := s.repo.GetProductRating(ctx, productID)
	if err != nil {
		return nil, apperrors.Internal("failed to load product rating", err)
	}
	if rating == nil {
		// Return a zero rating if no reviews exist.
		return &domain.ProductRating{
			ProductID:     productID,
			AverageRating: 0,
			ReviewCount:   0,
		}, nil
	}
	return rating, nil
}

// ListForSeller returns paginated reviews on the seller's products.
func (s *ReviewService) ListForSeller(
	ctx context.Context,
	sellerID uuid.UUID,
	limit, offset int,
) ([]domain.Review, int, error) {
	if sellerID == uuid.Nil {
		return nil, 0, apperrors.BadRequest("seller_id is required")
	}
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	items, total, err := s.repo.ListBySeller(ctx, sellerID, limit, offset)
	if err != nil {
		return nil, 0, apperrors.Internal("failed to list reviews", err)
	}
	return items, total, nil
}

// CreateReply creates a seller reply on a review.
func (s *ReviewService) CreateReply(
	ctx context.Context,
	reviewID uuid.UUID,
	sellerAuth0ID string,
	sellerID uuid.UUID,
	in domain.CreateReplyInput,
) (*domain.ReviewReply, error) {
	in.Body = strings.TrimSpace(in.Body)
	if in.Body == "" {
		return nil, apperrors.BadRequest("body is required")
	}
	if len(in.Body) > 2000 {
		return nil, apperrors.BadRequest("body must be at most 2000 characters")
	}

	review, err := s.repo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, apperrors.Internal("failed to load review", err)
	}
	if review == nil {
		return nil, domain.ErrReviewNotFound
	}
	if review.SellerID != sellerID {
		return nil, domain.ErrNotSellerOfProduct
	}

	reply := &domain.ReviewReply{
		ReviewID:      reviewID,
		SellerAuth0ID: sellerAuth0ID,
		Body:          in.Body,
	}
	if err := s.repo.CreateReply(ctx, reply); err != nil {
		return nil, err
	}

	pubsub.PublishProtoEvent(ctx, s.publisher, "review.replied", reviewEventTopic, &reviewv1.ReviewReplied{
		ReviewId:     review.ID.String(),
		ProductId:    review.ProductID.String(),
		SellerId:     review.SellerID.String(),
		BuyerAuth0Id: review.BuyerAuth0ID,
		ProductName:  review.ProductName,
		ReplyPreview: truncate(in.Body, 200),
	})

	return reply, nil
}

// UpdateReply updates a seller reply.
func (s *ReviewService) UpdateReply(
	ctx context.Context,
	reviewID uuid.UUID,
	sellerAuth0ID string,
	sellerID uuid.UUID,
	in domain.UpdateReplyInput,
) (*domain.ReviewReply, error) {
	in.Body = strings.TrimSpace(in.Body)
	if in.Body == "" {
		return nil, apperrors.BadRequest("body is required")
	}
	if len(in.Body) > 2000 {
		return nil, apperrors.BadRequest("body must be at most 2000 characters")
	}

	review, err := s.repo.GetByID(ctx, reviewID)
	if err != nil {
		return nil, apperrors.Internal("failed to load review", err)
	}
	if review == nil {
		return nil, domain.ErrReviewNotFound
	}
	if review.SellerID != sellerID {
		return nil, domain.ErrNotSellerOfProduct
	}

	// Pass reviewID and new body to UpdateReply; the repo uses RETURNING
	// to populate all remaining fields (id, seller_auth0_id,
	// created_at, updated_at) so the caller gets a complete object.
	reply := &domain.ReviewReply{
		ReviewID: reviewID,
		Body:     in.Body,
	}
	if err := s.repo.UpdateReply(ctx, reply); err != nil {
		return nil, err
	}

	return reply, nil
}

// DeleteReply deletes a seller reply.
func (s *ReviewService) DeleteReply(
	ctx context.Context,
	reviewID uuid.UUID,
	sellerID uuid.UUID,
) error {
	review, err := s.repo.GetByID(ctx, reviewID)
	if err != nil {
		return apperrors.Internal("failed to load review", err)
	}
	if review == nil {
		return domain.ErrReviewNotFound
	}
	if review.SellerID != sellerID {
		return domain.ErrNotSellerOfProduct
	}
	return s.repo.DeleteReply(ctx, reviewID)
}

func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen]
}
