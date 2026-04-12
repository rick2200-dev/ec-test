package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/review/internal/domain"
)

const pgUniqueViolation = "23505"

func isUniqueViolation(err error, constraint string) bool {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return false
	}
	return pgErr.Code == pgUniqueViolation && pgErr.ConstraintName == constraint
}

// ReviewRepository persists reviews, replies, and aggregate ratings.
type ReviewRepository struct {
	pool *pgxpool.Pool
}

// NewReviewRepository constructs a ReviewRepository.
func NewReviewRepository(pool *pgxpool.Pool) *ReviewRepository {
	return &ReviewRepository{pool: pool}
}

// RunInTx executes fn within a tenant-scoped transaction, embedding the tx
// into the context. Repository methods called with the returned context will
// join this transaction via withTx.
func (r *ReviewRepository) RunInTx(ctx context.Context, tenantID uuid.UUID, fn func(ctx context.Context) error) error {
	return database.TenantTxCtx(ctx, r.pool, tenantID, fn)
}

// withTx reuses a transaction from context if one exists (set by RunInTx),
// otherwise starts a new TenantTx. This allows individual repo methods to
// work both standalone and inside a service-layer transaction.
func (r *ReviewRepository) withTx(ctx context.Context, tenantID uuid.UUID, fn func(tx pgx.Tx) error) error {
	if tx, ok := database.TxFromContext(ctx); ok {
		return fn(tx)
	}
	return database.TenantTx(ctx, r.pool, tenantID, fn)
}

// Create inserts a new review. Returns domain.ErrAlreadyReviewed if the
// buyer has already reviewed this product.
func (r *ReviewRepository) Create(ctx context.Context, tenantID uuid.UUID, review *domain.Review) error {
	return r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		review.ID = uuid.New()
		review.TenantID = tenantID

		err := tx.QueryRow(ctx,
			`INSERT INTO review_svc.reviews
			 (id, tenant_id, buyer_auth0_id, product_id, seller_id, product_name, rating, title, body)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 RETURNING created_at, updated_at`,
			review.ID, review.TenantID, review.BuyerAuth0ID, review.ProductID,
			review.SellerID, review.ProductName, review.Rating, review.Title, review.Body,
		).Scan(&review.CreatedAt, &review.UpdatedAt)
		if err != nil {
			if isUniqueViolation(err, "uq_review_per_product") {
				return domain.ErrAlreadyReviewed
			}
			return fmt.Errorf("insert review: %w", err)
		}
		return nil
	})
}

// GetByID retrieves a review with its reply (LEFT JOIN). Returns nil if not found.
func (r *ReviewRepository) GetByID(ctx context.Context, tenantID, reviewID uuid.UUID) (*domain.Review, error) {
	var review *domain.Review

	err := r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		rv, err := loadReviewWithReplyTx(ctx, tx, tenantID, reviewID)
		if err != nil {
			return err
		}
		review = rv
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get review: %w", err)
	}
	return review, nil
}

// Update persists changes to an existing review.
func (r *ReviewRepository) Update(ctx context.Context, tenantID uuid.UUID, review *domain.Review) error {
	return r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE review_svc.reviews
			    SET rating = $3, title = $4, body = $5, updated_at = NOW()
			  WHERE id = $1 AND tenant_id = $2`,
			review.ID, tenantID, review.Rating, review.Title, review.Body,
		)
		if err != nil {
			return fmt.Errorf("update review: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrReviewNotFound
		}
		return nil
	})
}

// Delete removes a review (CASCADE deletes the reply).
func (r *ReviewRepository) Delete(ctx context.Context, tenantID, reviewID uuid.UUID) error {
	return r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`DELETE FROM review_svc.reviews WHERE id = $1 AND tenant_id = $2`,
			reviewID, tenantID,
		)
		if err != nil {
			return fmt.Errorf("delete review: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrReviewNotFound
		}
		return nil
	})
}

// ListByProduct returns paginated reviews with replies for a product.
func (r *ReviewRepository) ListByProduct(
	ctx context.Context,
	tenantID, productID uuid.UUID,
	limit, offset int,
) ([]domain.Review, int, error) {
	var out []domain.Review
	var total int

	err := r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM review_svc.reviews WHERE tenant_id = $1 AND product_id = $2`,
			tenantID, productID,
		).Scan(&total); err != nil {
			return fmt.Errorf("count reviews: %w", err)
		}

		rows, err := tx.Query(ctx,
			`SELECT r.id, r.tenant_id, r.buyer_auth0_id, r.product_id, r.seller_id,
			        r.product_name, r.rating, r.title, r.body, r.created_at, r.updated_at,
			        rp.id, rp.tenant_id, rp.review_id, rp.seller_auth0_id, rp.body,
			        rp.created_at, rp.updated_at
			   FROM review_svc.reviews r
			   LEFT JOIN review_svc.review_replies rp ON rp.review_id = r.id AND rp.tenant_id = r.tenant_id
			  WHERE r.tenant_id = $1 AND r.product_id = $2
			  ORDER BY r.created_at DESC
			  LIMIT $3 OFFSET $4`,
			tenantID, productID, limit, offset,
		)
		if err != nil {
			return fmt.Errorf("list reviews: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			rv, err := scanReviewWithReply(rows)
			if err != nil {
				return err
			}
			out = append(out, *rv)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// ListBySeller returns paginated reviews on the seller's products with replies.
func (r *ReviewRepository) ListBySeller(
	ctx context.Context,
	tenantID, sellerID uuid.UUID,
	limit, offset int,
) ([]domain.Review, int, error) {
	var out []domain.Review
	var total int

	err := r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM review_svc.reviews WHERE tenant_id = $1 AND seller_id = $2`,
			tenantID, sellerID,
		).Scan(&total); err != nil {
			return fmt.Errorf("count reviews: %w", err)
		}

		rows, err := tx.Query(ctx,
			`SELECT r.id, r.tenant_id, r.buyer_auth0_id, r.product_id, r.seller_id,
			        r.product_name, r.rating, r.title, r.body, r.created_at, r.updated_at,
			        rp.id, rp.tenant_id, rp.review_id, rp.seller_auth0_id, rp.body,
			        rp.created_at, rp.updated_at
			   FROM review_svc.reviews r
			   LEFT JOIN review_svc.review_replies rp ON rp.review_id = r.id AND rp.tenant_id = r.tenant_id
			  WHERE r.tenant_id = $1 AND r.seller_id = $2
			  ORDER BY r.created_at DESC
			  LIMIT $3 OFFSET $4`,
			tenantID, sellerID, limit, offset,
		)
		if err != nil {
			return fmt.Errorf("list reviews: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			rv, err := scanReviewWithReply(rows)
			if err != nil {
				return err
			}
			out = append(out, *rv)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return out, total, nil
}

// CreateReply inserts a new seller reply. Returns domain.ErrAlreadyReplied if
// a reply already exists.
func (r *ReviewRepository) CreateReply(ctx context.Context, tenantID uuid.UUID, reply *domain.ReviewReply) error {
	return r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		reply.ID = uuid.New()
		reply.TenantID = tenantID

		err := tx.QueryRow(ctx,
			`INSERT INTO review_svc.review_replies
			 (id, tenant_id, review_id, seller_auth0_id, body)
			 VALUES ($1, $2, $3, $4, $5)
			 RETURNING created_at, updated_at`,
			reply.ID, reply.TenantID, reply.ReviewID, reply.SellerAuth0ID, reply.Body,
		).Scan(&reply.CreatedAt, &reply.UpdatedAt)
		if err != nil {
			if isUniqueViolation(err, "uq_reply_per_review") {
				return domain.ErrAlreadyReplied
			}
			return fmt.Errorf("insert reply: %w", err)
		}
		return nil
	})
}

// UpdateReply persists changes to a seller reply, populating all fields on
// the reply via RETURNING so the caller gets a complete object.
func (r *ReviewRepository) UpdateReply(ctx context.Context, tenantID uuid.UUID, reply *domain.ReviewReply) error {
	return r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`UPDATE review_svc.review_replies
			    SET body = $3, updated_at = NOW()
			  WHERE review_id = $1 AND tenant_id = $2
			  RETURNING id, tenant_id, review_id, seller_auth0_id, body, created_at, updated_at`,
			reply.ReviewID, tenantID, reply.Body,
		).Scan(&reply.ID, &reply.TenantID, &reply.ReviewID, &reply.SellerAuth0ID, &reply.Body, &reply.CreatedAt, &reply.UpdatedAt)
		if err == pgx.ErrNoRows {
			return domain.ErrReplyNotFound
		}
		if err != nil {
			return fmt.Errorf("update reply: %w", err)
		}
		return nil
	})
}

// DeleteReply removes a seller reply.
func (r *ReviewRepository) DeleteReply(ctx context.Context, tenantID, reviewID uuid.UUID) error {
	return r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`DELETE FROM review_svc.review_replies WHERE review_id = $1 AND tenant_id = $2`,
			reviewID, tenantID,
		)
		if err != nil {
			return fmt.Errorf("delete reply: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return domain.ErrReplyNotFound
		}
		return nil
	})
}

// GetReplyByReview retrieves the seller reply for a review (nil if none).
func (r *ReviewRepository) GetReplyByReview(ctx context.Context, tenantID, reviewID uuid.UUID) (*domain.ReviewReply, error) {
	var reply *domain.ReviewReply

	err := r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		var rp domain.ReviewReply
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, review_id, seller_auth0_id, body, created_at, updated_at
			   FROM review_svc.review_replies
			  WHERE review_id = $1 AND tenant_id = $2`,
			reviewID, tenantID,
		).Scan(&rp.ID, &rp.TenantID, &rp.ReviewID, &rp.SellerAuth0ID, &rp.Body, &rp.CreatedAt, &rp.UpdatedAt)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return fmt.Errorf("load reply: %w", err)
		}
		reply = &rp
		return nil
	})
	if err != nil {
		return nil, err
	}
	return reply, nil
}

// GetProductRating retrieves the aggregate rating for a product (nil if no reviews).
func (r *ReviewRepository) GetProductRating(ctx context.Context, tenantID, productID uuid.UUID) (*domain.ProductRating, error) {
	var rating *domain.ProductRating

	err := r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		var pr domain.ProductRating
		err := tx.QueryRow(ctx,
			`SELECT tenant_id, product_id, average_rating, review_count, updated_at
			   FROM review_svc.product_ratings
			  WHERE tenant_id = $1 AND product_id = $2`,
			tenantID, productID,
		).Scan(&pr.TenantID, &pr.ProductID, &pr.AverageRating, &pr.ReviewCount, &pr.UpdatedAt)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return fmt.Errorf("load product rating: %w", err)
		}
		rating = &pr
		return nil
	})
	if err != nil {
		return nil, err
	}
	return rating, nil
}

// UpsertProductRating atomically adjusts the aggregate rating for a product.
// ratingDelta is added to rating_sum; countDelta is added to review_count.
// The average is recomputed from the updated sum and count.
func (r *ReviewRepository) UpsertProductRating(
	ctx context.Context,
	tenantID, productID uuid.UUID,
	ratingDelta, countDelta int,
) error {
	return r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		_, err := tx.Exec(ctx,
			`INSERT INTO review_svc.product_ratings (tenant_id, product_id, rating_sum, review_count, average_rating, updated_at)
			 VALUES ($1, $2, $3, $4, CASE WHEN $4 > 0 THEN $3::numeric / $4 ELSE 0 END, NOW())
			 ON CONFLICT (tenant_id, product_id) DO UPDATE SET
			     rating_sum   = review_svc.product_ratings.rating_sum   + $5,
			     review_count = review_svc.product_ratings.review_count + $6,
			     average_rating = CASE
			         WHEN review_svc.product_ratings.review_count + $6 > 0
			         THEN (review_svc.product_ratings.rating_sum + $5)::numeric / (review_svc.product_ratings.review_count + $6)
			         ELSE 0
			     END,
			     updated_at = NOW()`,
			tenantID, productID, ratingDelta, countDelta, ratingDelta, countDelta,
		)
		if err != nil {
			return fmt.Errorf("upsert product rating: %w", err)
		}
		return nil
	})
}

// --- internal helpers ---

func loadReviewWithReplyTx(ctx context.Context, tx pgx.Tx, tenantID, reviewID uuid.UUID) (*domain.Review, error) {
	row := tx.QueryRow(ctx,
		`SELECT r.id, r.tenant_id, r.buyer_auth0_id, r.product_id, r.seller_id,
		        r.product_name, r.rating, r.title, r.body, r.created_at, r.updated_at,
		        rp.id, rp.tenant_id, rp.review_id, rp.seller_auth0_id, rp.body,
		        rp.created_at, rp.updated_at
		   FROM review_svc.reviews r
		   LEFT JOIN review_svc.review_replies rp ON rp.review_id = r.id AND rp.tenant_id = r.tenant_id
		  WHERE r.id = $1 AND r.tenant_id = $2`,
		reviewID, tenantID,
	)

	rv, err := scanReviewRow(row)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load review with reply: %w", err)
	}
	return rv, nil
}

// scanReviewRow scans a single row from the reviews LEFT JOIN review_replies query.
func scanReviewRow(row pgx.Row) (*domain.Review, error) {
	var rv domain.Review
	var rpID *uuid.UUID
	var rpTenantID *uuid.UUID
	var rpReviewID *uuid.UUID
	var rpSellerAuth0ID *string
	var rpBody *string
	var rpCreatedAt *time.Time
	var rpUpdatedAt *time.Time

	err := row.Scan(
		&rv.ID, &rv.TenantID, &rv.BuyerAuth0ID, &rv.ProductID, &rv.SellerID,
		&rv.ProductName, &rv.Rating, &rv.Title, &rv.Body, &rv.CreatedAt, &rv.UpdatedAt,
		&rpID, &rpTenantID, &rpReviewID, &rpSellerAuth0ID, &rpBody,
		&rpCreatedAt, &rpUpdatedAt,
	)
	if err != nil {
		return nil, err
	}

	if rpID != nil {
		rv.Reply = &domain.ReviewReply{
			ID:            *rpID,
			TenantID:      *rpTenantID,
			ReviewID:      *rpReviewID,
			SellerAuth0ID: *rpSellerAuth0ID,
			Body:          *rpBody,
			CreatedAt:     *rpCreatedAt,
			UpdatedAt:     *rpUpdatedAt,
		}
	}

	return &rv, nil
}

// scanReviewWithReply scans a row from the reviews LEFT JOIN review_replies query (multi-row).
func scanReviewWithReply(rows pgx.Rows) (*domain.Review, error) {
	var rv domain.Review
	var rpID *uuid.UUID
	var rpTenantID *uuid.UUID
	var rpReviewID *uuid.UUID
	var rpSellerAuth0ID *string
	var rpBody *string
	var rpCreatedAt *time.Time
	var rpUpdatedAt *time.Time

	if err := rows.Scan(
		&rv.ID, &rv.TenantID, &rv.BuyerAuth0ID, &rv.ProductID, &rv.SellerID,
		&rv.ProductName, &rv.Rating, &rv.Title, &rv.Body, &rv.CreatedAt, &rv.UpdatedAt,
		&rpID, &rpTenantID, &rpReviewID, &rpSellerAuth0ID, &rpBody,
		&rpCreatedAt, &rpUpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("scan review: %w", err)
	}

	if rpID != nil {
		rv.Reply = &domain.ReviewReply{
			ID:            *rpID,
			TenantID:      *rpTenantID,
			ReviewID:      *rpReviewID,
			SellerAuth0ID: *rpSellerAuth0ID,
			Body:          *rpBody,
			CreatedAt:     *rpCreatedAt,
			UpdatedAt:     *rpUpdatedAt,
		}
	}

	return &rv, nil
}
