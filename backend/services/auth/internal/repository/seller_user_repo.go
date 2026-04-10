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

// SellerUserRepository handles persistence of seller_users within a tenant scope.
type SellerUserRepository struct {
	pool *pgxpool.Pool
}

// NewSellerUserRepository creates a new SellerUserRepository.
func NewSellerUserRepository(pool *pgxpool.Pool) *SellerUserRepository {
	return &SellerUserRepository{pool: pool}
}

// Create inserts a new seller_user row. The caller must ensure the
// SellerUser's TenantID matches the RLS tenant scope.
func (r *SellerUserRepository) Create(ctx context.Context, su *domain.SellerUser) error {
	if su.ID == uuid.Nil {
		su.ID = uuid.New()
	}
	return database.TenantTx(ctx, r.pool, su.TenantID, func(tx pgx.Tx) error {
		return r.CreateTx(ctx, tx, su)
	})
}

// CreateTx inserts a new seller_user row using an existing transaction. This
// enables atomic multi-step operations (e.g. seller creation with initial
// owner).
func (r *SellerUserRepository) CreateTx(ctx context.Context, tx pgx.Tx, su *domain.SellerUser) error {
	if su.ID == uuid.Nil {
		su.ID = uuid.New()
	}
	return tx.QueryRow(ctx,
		`INSERT INTO auth_svc.seller_users (id, tenant_id, seller_id, auth0_user_id, role)
		 VALUES ($1, $2, $3, $4, $5)
		 RETURNING created_at`,
		su.ID, su.TenantID, su.SellerID, su.Auth0UserID, su.Role,
	).Scan(&su.CreatedAt)
}

// GetByID retrieves a seller_user by its primary key within a tenant scope.
func (r *SellerUserRepository) GetByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SellerUser, error) {
	var su domain.SellerUser
	var found bool
	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, seller_id, auth0_user_id, role, created_at
			 FROM auth_svc.seller_users
			 WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&su.ID, &su.TenantID, &su.SellerID, &su.Auth0UserID, &su.Role, &su.CreatedAt)
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
		return nil, fmt.Errorf("get seller_user by id: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &su, nil
}

// GetByAuth0ID retrieves a seller_user by (tenant, seller, auth0_user_id).
func (r *SellerUserRepository) GetByAuth0ID(ctx context.Context, tenantID, sellerID uuid.UUID, auth0UserID string) (*domain.SellerUser, error) {
	var su domain.SellerUser
	var found bool
	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, seller_id, auth0_user_id, role, created_at
			 FROM auth_svc.seller_users
			 WHERE tenant_id = $1 AND seller_id = $2 AND auth0_user_id = $3`,
			tenantID, sellerID, auth0UserID,
		).Scan(&su.ID, &su.TenantID, &su.SellerID, &su.Auth0UserID, &su.Role, &su.CreatedAt)
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
		return nil, fmt.Errorf("get seller_user by auth0 id: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &su, nil
}

// ListBySeller returns all users belonging to a seller organization.
func (r *SellerUserRepository) ListBySeller(ctx context.Context, tenantID, sellerID uuid.UUID) ([]domain.SellerUser, error) {
	var users []domain.SellerUser
	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, seller_id, auth0_user_id, role, created_at
			 FROM auth_svc.seller_users
			 WHERE tenant_id = $1 AND seller_id = $2
			 ORDER BY created_at ASC`,
			tenantID, sellerID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()
		for rows.Next() {
			var su domain.SellerUser
			if err := rows.Scan(&su.ID, &su.TenantID, &su.SellerID, &su.Auth0UserID, &su.Role, &su.CreatedAt); err != nil {
				return err
			}
			users = append(users, su)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("list seller_users: %w", err)
	}
	return users, nil
}

// UpdateRole changes the role of a seller_user.
func (r *SellerUserRepository) UpdateRole(ctx context.Context, tenantID, id uuid.UUID, role domain.SellerUserRole) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return r.UpdateRoleTx(ctx, tx, tenantID, id, role)
	})
}

// UpdateRoleTx updates a seller_user's role using an existing transaction.
func (r *SellerUserRepository) UpdateRoleTx(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID, role domain.SellerUserRole) error {
	tag, err := tx.Exec(ctx,
		`UPDATE auth_svc.seller_users SET role = $3 WHERE id = $1 AND tenant_id = $2`,
		id, tenantID, role,
	)
	if err != nil {
		return fmt.Errorf("update seller_user role: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("seller_user not found")
	}
	return nil
}

// Delete removes a seller_user row.
func (r *SellerUserRepository) Delete(ctx context.Context, tenantID, id uuid.UUID) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return r.DeleteTx(ctx, tx, tenantID, id)
	})
}

// DeleteTx removes a seller_user row using an existing transaction.
func (r *SellerUserRepository) DeleteTx(ctx context.Context, tx pgx.Tx, tenantID, id uuid.UUID) error {
	tag, err := tx.Exec(ctx,
		`DELETE FROM auth_svc.seller_users WHERE id = $1 AND tenant_id = $2`,
		id, tenantID,
	)
	if err != nil {
		return fmt.Errorf("delete seller_user: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("seller_user not found")
	}
	return nil
}

// CountByRole returns the number of seller_users in a seller organization
// that have the given role. Used for "last owner" safeguard checks.
func (r *SellerUserRepository) CountByRole(ctx context.Context, tenantID, sellerID uuid.UUID, role domain.SellerUserRole) (int, error) {
	var n int
	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return r.CountByRoleTx(ctx, tx, tenantID, sellerID, role, &n)
	})
	if err != nil {
		return 0, fmt.Errorf("count seller_users by role: %w", err)
	}
	return n, nil
}

// CountByRoleTx is the transaction-scoped variant of CountByRole.
func (r *SellerUserRepository) CountByRoleTx(ctx context.Context, tx pgx.Tx, tenantID, sellerID uuid.UUID, role domain.SellerUserRole, out *int) error {
	return tx.QueryRow(ctx,
		`SELECT COUNT(*) FROM auth_svc.seller_users
		 WHERE tenant_id = $1 AND seller_id = $2 AND role = $3`,
		tenantID, sellerID, role,
	).Scan(out)
}
