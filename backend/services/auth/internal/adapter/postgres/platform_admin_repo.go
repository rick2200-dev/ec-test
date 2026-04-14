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

// PlatformAdminRepository handles persistence of platform_admins.
type PlatformAdminRepository struct {
	pool *pgxpool.Pool
}

// NewPlatformAdminRepository creates a new PlatformAdminRepository.
func NewPlatformAdminRepository(pool *pgxpool.Pool) *PlatformAdminRepository {
	return &PlatformAdminRepository{pool: pool}
}

// withTx uses the transaction from ctx if one was placed there by
// database.WithTx, otherwise opens a new transaction.
func (r *PlatformAdminRepository) withTx(ctx context.Context, fn func(tx pgx.Tx) error) error {
	if tx, ok := database.TxFromContext(ctx); ok {
		return fn(tx)
	}
	return database.Tx(ctx, r.pool, fn)
}

// Create inserts a new platform_admin row. When ctx carries a transaction it
// joins that transaction; otherwise opens its own.
func (r *PlatformAdminRepository) Create(ctx context.Context, pa *domain.PlatformAdmin) error {
	if pa.ID == uuid.Nil {
		pa.ID = uuid.New()
	}
	return r.withTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO auth_svc.platform_admins (id, auth0_user_id, role)
			 VALUES ($1, $2, $3)
			 RETURNING created_at, updated_at`,
			pa.ID, pa.Auth0UserID, pa.Role,
		).Scan(&pa.CreatedAt, &pa.UpdatedAt)
	})
}

// GetByID retrieves a platform_admin by its primary key.
func (r *PlatformAdminRepository) GetByID(ctx context.Context, id uuid.UUID) (*domain.PlatformAdmin, error) {
	var pa domain.PlatformAdmin
	var found bool
	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, auth0_user_id, role, created_at, updated_at
			 FROM auth_svc.platform_admins
			 WHERE id = $1`,
			id,
		).Scan(&pa.ID, &pa.Auth0UserID, &pa.Role, &pa.CreatedAt, &pa.UpdatedAt)
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

// GetByAuth0ID retrieves a platform_admin by auth0_user_id. When ctx
// carries a transaction it joins that transaction; otherwise opens its own.
func (r *PlatformAdminRepository) GetByAuth0ID(ctx context.Context, auth0UserID string) (*domain.PlatformAdmin, error) {
	var pa domain.PlatformAdmin
	var found bool
	err := r.withTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, auth0_user_id, role, created_at, updated_at
			 FROM auth_svc.platform_admins
			 WHERE auth0_user_id = $1`,
			auth0UserID,
		).Scan(&pa.ID, &pa.Auth0UserID, &pa.Role, &pa.CreatedAt, &pa.UpdatedAt)
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

// List returns all platform_admins.
func (r *PlatformAdminRepository) List(ctx context.Context) ([]domain.PlatformAdmin, error) {
	var admins []domain.PlatformAdmin
	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, auth0_user_id, role, created_at, updated_at
			 FROM auth_svc.platform_admins
			 ORDER BY created_at ASC`,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var pa domain.PlatformAdmin
			if err := rows.Scan(&pa.ID, &pa.Auth0UserID, &pa.Role, &pa.CreatedAt, &pa.UpdatedAt); err != nil {
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
func (r *PlatformAdminRepository) UpdateRole(ctx context.Context, id uuid.UUID, role domain.PlatformAdminRole) error {
	return r.withTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE auth_svc.platform_admins SET role = $2, updated_at = NOW()
			 WHERE id = $1`,
			id, role,
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
func (r *PlatformAdminRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.withTx(ctx, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`DELETE FROM auth_svc.platform_admins WHERE id = $1`,
			id,
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

// CountByRole returns the number of platform_admins with the given role.
// When ctx carries a transaction it joins that transaction; otherwise opens its own.
func (r *PlatformAdminRepository) CountByRole(ctx context.Context, role domain.PlatformAdminRole) (int, error) {
	var n int
	err := r.withTx(ctx, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM auth_svc.platform_admins
			 WHERE role = $1`,
			role,
		).Scan(&n)
	})
	if err != nil {
		return 0, fmt.Errorf("count platform_admins by role: %w", err)
	}
	return n, nil
}

// CheckRole returns the role of the given Auth0 user as a platform admin,
// or an empty string if the user is not an admin. When ctx carries a
// transaction it joins that transaction; otherwise opens its own.
// Returns ("", nil) when the user is not found; returns ("", err) on
// unexpected database errors.
func (r *PlatformAdminRepository) CheckRole(ctx context.Context, auth0UserID string) (domain.PlatformAdminRole, error) {
	var role domain.PlatformAdminRole
	err := r.withTx(ctx, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT role FROM auth_svc.platform_admins
			 WHERE auth0_user_id = $1`,
			auth0UserID,
		).Scan(&role)
		if err == pgx.ErrNoRows {
			return nil
		}
		return err
	})
	return role, err
}
