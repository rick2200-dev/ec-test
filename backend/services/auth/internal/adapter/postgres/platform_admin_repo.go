package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/auth/internal/domain"
)

// PlatformAdminRepository handles persistence of platform_admins within a
// tenant scope.
type PlatformAdminRepository struct {
	pool *pgxpool.Pool
}

// NewPlatformAdminRepository creates a new PlatformAdminRepository.
func NewPlatformAdminRepository(pool *pgxpool.Pool) *PlatformAdminRepository {
	return &PlatformAdminRepository{pool: pool}
}

// withTx uses the transaction from ctx if one was placed there by
// database.WithTx, otherwise opens a new tenant-scoped transaction.
func (r *PlatformAdminRepository) withTx(ctx context.Context, tenantID uuid.UUID, fn func(tx pgx.Tx) error) error {
	if tx, ok := database.TxFromContext(ctx); ok {
		return fn(tx)
	}
	return database.TenantTx(ctx, r.pool, tenantID, fn)
}

// Create inserts a new platform_admin row. When ctx carries a transaction it
// joins that transaction; otherwise opens its own.
func (r *PlatformAdminRepository) Create(ctx context.Context, pa *domain.PlatformAdmin) error {
	if pa.ID == uuid.Nil {
		pa.ID = uuid.New()
	}
	return r.withTx(ctx, pa.TenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO auth_svc.platform_admins (id, tenant_id, auth0_user_id, role)
			 VALUES ($1, $2, $3, $4)
			 RETURNING created_at, updated_at`,
			pa.ID, pa.TenantID, pa.Auth0UserID, pa.Role,
		).Scan(&pa.CreatedAt, &pa.UpdatedAt)
	})
}

// GetByID retrieves a platform_admin by its primary key within a tenant scope.
func (r *PlatformAdminRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.PlatformAdmin, error) {
	var pa domain.PlatformAdmin
	var found bool
	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, auth0_user_id, role, created_at, updated_at
			 FROM auth_svc.platform_admins
			 WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&pa.ID, &pa.TenantID, &pa.Auth0UserID, &pa.Role, &pa.CreatedAt, &pa.UpdatedAt)
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
		return nil, fmt.Errorf("get platform_admin by id: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &pa, nil
}

// GetByAuth0ID retrieves a platform_admin by (tenant, auth0_user_id). When ctx
// carries a transaction it joins that transaction; otherwise opens its own.
func (r *PlatformAdminRepository) GetByAuth0ID(ctx context.Context, tenantID uuid.UUID, auth0UserID string) (*domain.PlatformAdmin, error) {
	var pa domain.PlatformAdmin
	var found bool
	err := r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, auth0_user_id, role, created_at, updated_at
			 FROM auth_svc.platform_admins
			 WHERE tenant_id = $1 AND auth0_user_id = $2`,
			tenantID, auth0UserID,
		).Scan(&pa.ID, &pa.TenantID, &pa.Auth0UserID, &pa.Role, &pa.CreatedAt, &pa.UpdatedAt)
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
		return nil, fmt.Errorf("get platform_admin by auth0 id: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &pa, nil
}

// List returns all platform_admins in a tenant.
func (r *PlatformAdminRepository) List(ctx context.Context, tenantID uuid.UUID) ([]domain.PlatformAdmin, error) {
	var admins []domain.PlatformAdmin
	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, auth0_user_id, role, created_at, updated_at
			 FROM auth_svc.platform_admins
			 WHERE tenant_id = $1
			 ORDER BY created_at ASC`,
			tenantID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var pa domain.PlatformAdmin
			if err := rows.Scan(&pa.ID, &pa.TenantID, &pa.Auth0UserID, &pa.Role, &pa.CreatedAt, &pa.UpdatedAt); err != nil {
				return err
			}
			admins = append(admins, pa)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("list platform_admins: %w", err)
	}
	return admins, nil
}

// UpdateRole changes the role of a platform_admin. When ctx carries a
// transaction it joins that transaction; otherwise opens its own.
func (r *PlatformAdminRepository) UpdateRole(ctx context.Context, tenantID, id uuid.UUID, role domain.PlatformAdminRole) error {
	return r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE auth_svc.platform_admins SET role = $3, updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2`,
			id, tenantID, role,
		)
		if err != nil {
			return fmt.Errorf("update platform_admin role: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("platform_admin not found")
		}
		return nil
	})
}

// Delete removes a platform_admin row. When ctx carries a transaction it
// joins that transaction; otherwise opens its own.
func (r *PlatformAdminRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`DELETE FROM auth_svc.platform_admins WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		)
		if err != nil {
			return fmt.Errorf("delete platform_admin: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("platform_admin not found")
		}
		return nil
	})
}

// CountByRole returns the number of platform_admins in a tenant with the
// given role. When ctx carries a transaction it joins that transaction;
// otherwise opens its own.
func (r *PlatformAdminRepository) CountByRole(ctx context.Context, tenantID uuid.UUID, role domain.PlatformAdminRole) (int, error) {
	var n int
	err := r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM auth_svc.platform_admins
			 WHERE tenant_id = $1 AND role = $2`,
			tenantID, role,
		).Scan(&n)
	})
	if err != nil {
		return 0, fmt.Errorf("count platform_admins by role: %w", err)
	}
	return n, nil
}

// CheckRole returns the role of the given Auth0 user as a platform admin in
// the tenant, or an empty string if the user is not an admin. When ctx
// carries a transaction it joins that transaction; otherwise opens its own.
// Returns ("", nil) when the user is not found; returns ("", err) on
// unexpected database errors.
func (r *PlatformAdminRepository) CheckRole(ctx context.Context, tenantID uuid.UUID, auth0UserID string) (domain.PlatformAdminRole, error) {
	var role domain.PlatformAdminRole
	err := r.withTx(ctx, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT role FROM auth_svc.platform_admins
			 WHERE tenant_id = $1 AND auth0_user_id = $2`,
			tenantID, auth0UserID,
		).Scan(&role)
		if err == pgx.ErrNoRows {
			return nil
		}
		return err
	})
	return role, err
}
