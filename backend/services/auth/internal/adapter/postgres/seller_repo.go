package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// SellerRepository handles persistence of sellers.
type SellerRepository struct {
	pool *pgxpool.Pool
}

// NewSellerRepository creates a new SellerRepository.
func NewSellerRepository(pool *pgxpool.Pool) *SellerRepository {
	return &SellerRepository{pool: pool}
}

// withTx uses the transaction from ctx if one was placed there by
// database.WithTx, otherwise opens a new transaction.
func (r *SellerRepository) withTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	if tx, ok := database.TxFromContext(ctx); ok {
		return fn(tx)
	}
	return database.Tx(ctx, r.pool, fn)
}

// Create inserts a new seller. When ctx carries a transaction (placed there
// by database.WithTx), the insert joins that transaction so the caller can
// make the seller and its initial owner atomically.
func (r *SellerRepository) Create(ctx context.Context, s *domain.Seller) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}
	settings := json.RawMessage("{}")
	if s.Settings != nil {
		settings = s.Settings
	}
	return r.withTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO auth_svc.sellers (id, auth0_org_id, name, slug, status, stripe_account_id, commission_rate_bps, settings)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 RETURNING created_at, updated_at`,
			s.ID, s.Auth0OrgID, s.Name, s.Slug, s.Status, s.StripeAccountID, s.CommissionRateBPS, settings,
		).Scan(&s.CreatedAt, &s.UpdatedAt)
	})
}

// GetByID retrieves a seller by ID.
func (r *SellerRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Seller, error) {
	var s domain.Seller
	var found bool

	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, auth0_org_id, name, slug, status, stripe_account_id, commission_rate_bps, settings, created_at, updated_at
			 FROM auth_svc.sellers WHERE id = $1`, id,
		).Scan(&s.ID, &s.Auth0OrgID, &s.Name, &s.Slug, &s.Status, &s.StripeAccountID, &s.CommissionRateBPS, &s.Settings, &s.CreatedAt, &s.UpdatedAt)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get seller by id: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &s, nil
}

// GetBySlug retrieves a seller by slug.
func (r *SellerRepository) GetBySlug(ctx context.Context, slug string) (*domain.Seller, error) {
	var s domain.Seller
	var found bool

	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, auth0_org_id, name, slug, status, stripe_account_id, commission_rate_bps, settings, created_at, updated_at
			 FROM auth_svc.sellers WHERE slug = $1`, slug,
		).Scan(&s.ID, &s.Auth0OrgID, &s.Name, &s.Slug, &s.Status, &s.StripeAccountID, &s.CommissionRateBPS, &s.Settings, &s.CreatedAt, &s.UpdatedAt)
		if err == pgx.ErrNoRows {
			return nil
		}
		if err != nil {
			return err
		}
		found = true
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("get seller by slug: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &s, nil
}

// List returns a paginated list of sellers.
func (r *SellerRepository) List(ctx context.Context, limit, offset int) ([]domain.Seller, int, error) {
	var sellers []domain.Seller
	var total int

	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM auth_svc.sellers`,
		).Scan(&total); err != nil {
			return err
		}

		rows, err := tx.Query(ctx,
			`SELECT id, auth0_org_id, name, slug, status, stripe_account_id, commission_rate_bps, settings, created_at, updated_at
			 FROM auth_svc.sellers ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
			limit, offset,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var s domain.Seller
			if err := rows.Scan(&s.ID, &s.Auth0OrgID, &s.Name, &s.Slug, &s.Status, &s.StripeAccountID, &s.CommissionRateBPS, &s.Settings, &s.CreatedAt, &s.UpdatedAt); err != nil {
				return err
			}
			sellers = append(sellers, s)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, fmt.Errorf("list sellers: %w", err)
	}
	return sellers, total, nil
}

// UpdateStatus updates the status of a seller.
func (r *SellerRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status domain.SellerStatus) error {
	return database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE auth_svc.sellers SET status = $2, updated_at = NOW() WHERE id = $1`,
			id, status,
		)
		if err != nil {
			return fmt.Errorf("update seller status: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("seller not found")
		}
		return nil
	})
}

// Update modifies an existing seller.
func (r *SellerRepository) Update(ctx context.Context, s *domain.Seller) error {
	return database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE auth_svc.sellers SET name = $2, slug = $3, auth0_org_id = $4, stripe_account_id = $5, commission_rate_bps = $6, settings = $7, updated_at = NOW()
			 WHERE id = $1`,
			s.ID, s.Name, s.Slug, s.Auth0OrgID, s.StripeAccountID, s.CommissionRateBPS, s.Settings,
		)
		if err != nil {
			return fmt.Errorf("update seller: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("seller not found")
		}
		return nil
	})
}

// BatchGetByIDs returns sellers matching any of the given ids. Unknown ids
// are silently omitted. Intended for internal batch lookup (order at
// checkout time).
func (r *SellerRepository) BatchGetByIDs(ctx context.Context, ids []uuid.UUID) ([]domain.Seller, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	rows, err := r.pool.Query(ctx,
		`SELECT id, auth0_org_id, name, slug, status, stripe_account_id, commission_rate_bps, settings, created_at, updated_at
		 FROM auth_svc.sellers WHERE id = ANY($1)`,
		ids,
	)
	if err != nil {
		return nil, fmt.Errorf("batch get sellers: %w", err)
	}
	defer rows.Close()
	out := make([]domain.Seller, 0, len(ids))
	for rows.Next() {
		var s domain.Seller
		if err := rows.Scan(&s.ID, &s.Auth0OrgID, &s.Name, &s.Slug, &s.Status, &s.StripeAccountID, &s.CommissionRateBPS, &s.Settings, &s.CreatedAt, &s.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan seller: %w", err)
		}
		out = append(out, s)
	}
	return out, rows.Err()
}
