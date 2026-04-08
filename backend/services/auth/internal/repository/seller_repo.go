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

// SellerRepository handles persistence of sellers within a tenant scope.
type SellerRepository struct {
	pool *pgxpool.Pool
}

// NewSellerRepository creates a new SellerRepository.
func NewSellerRepository(pool *pgxpool.Pool) *SellerRepository {
	return &SellerRepository{pool: pool}
}

// Create inserts a new seller within a tenant-scoped transaction.
func (r *SellerRepository) Create(ctx context.Context, tenantID uuid.UUID, s *domain.Seller) error {
	s.ID = uuid.New()
	s.TenantID = tenantID
	settings := json.RawMessage("{}")
	if s.Settings != nil {
		settings = s.Settings
	}

	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO auth_svc.sellers (id, tenant_id, auth0_org_id, name, slug, status, stripe_account_id, commission_rate_bps, settings)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 RETURNING created_at, updated_at`,
			s.ID, s.TenantID, s.Auth0OrgID, s.Name, s.Slug, s.Status, s.StripeAccountID, s.CommissionRateBPS, settings,
		).Scan(&s.CreatedAt, &s.UpdatedAt)
	})
}

// GetByID retrieves a seller by ID within a tenant scope.
func (r *SellerRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.Seller, error) {
	var s domain.Seller
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, auth0_org_id, name, slug, status, stripe_account_id, commission_rate_bps, settings, created_at, updated_at
			 FROM auth_svc.sellers WHERE id = $1 AND tenant_id = $2`, id, tenantID,
		).Scan(&s.ID, &s.TenantID, &s.Auth0OrgID, &s.Name, &s.Slug, &s.Status, &s.StripeAccountID, &s.CommissionRateBPS, &s.Settings, &s.CreatedAt, &s.UpdatedAt)
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

// GetBySlug retrieves a seller by slug within a tenant scope.
func (r *SellerRepository) GetBySlug(ctx context.Context, tenantID uuid.UUID, slug string) (*domain.Seller, error) {
	var s domain.Seller
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, auth0_org_id, name, slug, status, stripe_account_id, commission_rate_bps, settings, created_at, updated_at
			 FROM auth_svc.sellers WHERE slug = $1 AND tenant_id = $2`, slug, tenantID,
		).Scan(&s.ID, &s.TenantID, &s.Auth0OrgID, &s.Name, &s.Slug, &s.Status, &s.StripeAccountID, &s.CommissionRateBPS, &s.Settings, &s.CreatedAt, &s.UpdatedAt)
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

// List returns a paginated list of sellers for a tenant.
func (r *SellerRepository) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.Seller, int, error) {
	var sellers []domain.Seller
	var total int

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM auth_svc.sellers WHERE tenant_id = $1`, tenantID,
		).Scan(&total); err != nil {
			return err
		}

		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, auth0_org_id, name, slug, status, stripe_account_id, commission_rate_bps, settings, created_at, updated_at
			 FROM auth_svc.sellers WHERE tenant_id = $1 ORDER BY created_at DESC LIMIT $2 OFFSET $3`,
			tenantID, limit, offset,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var s domain.Seller
			if err := rows.Scan(&s.ID, &s.TenantID, &s.Auth0OrgID, &s.Name, &s.Slug, &s.Status, &s.StripeAccountID, &s.CommissionRateBPS, &s.Settings, &s.CreatedAt, &s.UpdatedAt); err != nil {
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
func (r *SellerRepository) UpdateStatus(ctx context.Context, tenantID, id uuid.UUID, status domain.SellerStatus) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE auth_svc.sellers SET status = $3, updated_at = NOW() WHERE id = $1 AND tenant_id = $2`,
			id, tenantID, status,
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
func (r *SellerRepository) Update(ctx context.Context, tenantID uuid.UUID, s *domain.Seller) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE auth_svc.sellers SET name = $3, slug = $4, auth0_org_id = $5, stripe_account_id = $6, commission_rate_bps = $7, settings = $8, updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2`,
			s.ID, tenantID, s.Name, s.Slug, s.Auth0OrgID, s.StripeAccountID, s.CommissionRateBPS, s.Settings,
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
