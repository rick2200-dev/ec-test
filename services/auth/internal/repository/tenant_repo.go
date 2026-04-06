package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// TenantRepository handles persistence of tenants.
type TenantRepository struct {
	pool *pgxpool.Pool
}

// NewTenantRepository creates a new TenantRepository.
func NewTenantRepository(pool *pgxpool.Pool) *TenantRepository {
	return &TenantRepository{pool: pool}
}

// Create inserts a new tenant (global operation, not tenant-scoped).
func (r *TenantRepository) Create(ctx context.Context, t *domain.Tenant) error {
	t.ID = uuid.New()
	settings := json.RawMessage("{}")
	if t.Settings != nil {
		settings = t.Settings
	}

	err := r.pool.QueryRow(ctx,
		`INSERT INTO auth_svc.tenants (id, name, slug, status, settings)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING created_at, updated_at`,
		t.ID, t.Name, t.Slug, t.Status, settings,
	).Scan(&t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		return fmt.Errorf("insert tenant: %w", err)
	}
	return nil
}

// GetByID retrieves a tenant by its ID.
func (r *TenantRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.Tenant, error) {
	var t domain.Tenant
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, status, settings, created_at, updated_at
		 FROM auth_svc.tenants WHERE id = $1`, id,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.Settings, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get tenant by id: %w", err)
	}
	return &t, nil
}

// GetBySlug retrieves a tenant by its slug.
func (r *TenantRepository) GetBySlug(ctx context.Context, slug string) (*domain.Tenant, error) {
	var t domain.Tenant
	err := r.pool.QueryRow(ctx,
		`SELECT id, name, slug, status, settings, created_at, updated_at
		 FROM auth_svc.tenants WHERE slug = $1`, slug,
	).Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.Settings, &t.CreatedAt, &t.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get tenant by slug: %w", err)
	}
	return &t, nil
}

// List returns a paginated list of tenants.
func (r *TenantRepository) List(ctx context.Context, limit, offset int) ([]domain.Tenant, int, error) {
	var total int
	err := r.pool.QueryRow(ctx, `SELECT COUNT(*) FROM auth_svc.tenants`).Scan(&total)
	if err != nil {
		return nil, 0, fmt.Errorf("count tenants: %w", err)
	}

	rows, err := r.pool.Query(ctx,
		`SELECT id, name, slug, status, settings, created_at, updated_at
		 FROM auth_svc.tenants ORDER BY created_at DESC LIMIT $1 OFFSET $2`,
		limit, offset,
	)
	if err != nil {
		return nil, 0, fmt.Errorf("list tenants: %w", err)
	}
	defer rows.Close()

	var tenants []domain.Tenant
	for rows.Next() {
		var t domain.Tenant
		if err := rows.Scan(&t.ID, &t.Name, &t.Slug, &t.Status, &t.Settings, &t.CreatedAt, &t.UpdatedAt); err != nil {
			return nil, 0, fmt.Errorf("scan tenant: %w", err)
		}
		tenants = append(tenants, t)
	}
	return tenants, total, nil
}

// Update modifies an existing tenant.
func (r *TenantRepository) Update(ctx context.Context, t *domain.Tenant) error {
	tag, err := r.pool.Exec(ctx,
		`UPDATE auth_svc.tenants SET name = $2, slug = $3, status = $4, settings = $5, updated_at = NOW()
		 WHERE id = $1`,
		t.ID, t.Name, t.Slug, t.Status, t.Settings,
	)
	if err != nil {
		return fmt.Errorf("update tenant: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("tenant not found")
	}
	return nil
}
