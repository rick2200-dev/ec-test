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

// Create inserts a new platform_admin row.
func (r *PlatformAdminRepository) Create(ctx context.Context, pa *domain.PlatformAdmin) error {
	if pa.ID == uuid.Nil {
		pa.ID = uuid.New()
	}
	return database.TenantTx(ctx, r.pool, pa.TenantID, func(tx pgx.Tx) error {
		return r.CreateTx(ctx, tx, pa)
	})
}

// CreateTx inserts a new platform_admin row using an existing transaction.
func (r *PlatformAdminRepository) CreateTx(ctx context.Context, tx pgx.Tx, pa *domain.PlatformAdmin) error {
	if pa.ID == uuid.Nil {
		pa.ID = uuid.New()
	}
	return tx.QueryRow(ctx,
		`INSERT INTO auth_svc.platform_admins (id, tenant_id, auth0_user_id, role)
		 VALUES ($1, $2, $3, $4)
		 RETURNING created_at, updated_at`,
		pa.ID, pa.TenantID, pa.Auth0UserID, pa.Role,
	).Scan(&pa.CreatedAt, &pa.UpdatedAt)
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

// GetByAuth0ID retrieves a platform_admin by (tenant, auth0_user_id).
func (r *PlatformAdminRepository) GetByAuth0ID(ctx context.Context, tenantID uuid.UUID, auth0UserID string) (*domain.PlatformAdmin, error) {
	var pa domain.PlatformAdmin
	var found bool
	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
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

// UpdateRole changes the role of a platform_admin.
func (r *PlatformAdminRepository) UpdateRole(ctx context.Context, tenantID, id uuid.UUID, role domain.PlatformAdminRole) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return r.UpdateRoleTx(ctx, tx, tenantID, id, role)
	})
}

// UpdateRoleTx updates a platform_admin's role using an existing transaction.
func (r *PlatformAdminRepository) UpdateRoleTx(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID, role domain.PlatformAdminRole) error {
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
}

// Delete removes a platform_admin row.
func (r *PlatformAdminRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return r.DeleteTx(ctx, tx, tenantID, id)
	})
}

// DeleteTx removes a platform_admin row using an existing transaction.
func (r *PlatformAdminRepository) DeleteTx(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID) error {
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
}

// CountByRole returns the number of platform_admins in a tenant with the
// given role. Used for "last super_admin" safeguard checks.
func (r *PlatformAdminRepository) CountByRole(ctx context.Context, tenantID uuid.UUID, role domain.PlatformAdminRole) (int, error) {
	var n int
	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return r.CountByRoleTx(ctx, tx, tenantID, role, &n)
	})
	if err != nil {
		return 0, fmt.Errorf("count platform_admins by role: %w", err)
	}
	return n, nil
}

// CountByRoleTx is the transaction-scoped variant of CountByRole.
func (r *PlatformAdminRepository) CountByRoleTx(ctx context.Context, tx pgx.Tx, tenantID uuid.UUID, role domain.PlatformAdminRole, out *int) error {
	return tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM auth_svc.platform_admins
		 WHERE tenant_id = $1 AND role = $2`,
		tenantID, role,
	).Scan(out)
}
