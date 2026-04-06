package repository

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/order/internal/domain"
)

// CommissionRepository handles persistence of commission rules.
type CommissionRepository struct {
	pool *pgxpool.Pool
}

// NewCommissionRepository creates a new CommissionRepository.
func NewCommissionRepository(pool *pgxpool.Pool) *CommissionRepository {
	return &CommissionRepository{pool: pool}
}

// GetApplicableRule finds the highest-priority commission rule matching the given seller and category.
// It checks rules in order: specific seller+category, specific seller, specific category, then default.
func (r *CommissionRepository) GetApplicableRule(ctx context.Context, tenantID, sellerID uuid.UUID, categoryID *uuid.UUID) (*domain.CommissionRule, error) {
	var rule domain.CommissionRule
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		query := `SELECT id, tenant_id, seller_id, category_id, rate_bps, priority, valid_from, valid_until, created_at
			 FROM order_svc.commission_rules
			 WHERE tenant_id = $1
			   AND (seller_id = $2 OR seller_id IS NULL)
			   AND (category_id = $3 OR category_id IS NULL)
			   AND valid_from <= NOW()
			   AND (valid_until IS NULL OR valid_until > NOW())
			 ORDER BY priority DESC
			 LIMIT 1`

		err := tx.QueryRow(ctx, query, tenantID, sellerID, categoryID).Scan(
			&rule.ID, &rule.TenantID, &rule.SellerID, &rule.CategoryID,
			&rule.RateBps, &rule.Priority, &rule.ValidFrom, &rule.ValidUntil, &rule.CreatedAt,
		)
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
		return nil, fmt.Errorf("get applicable commission rule: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &rule, nil
}

// List returns a paginated list of commission rules for a tenant.
func (r *CommissionRepository) List(ctx context.Context, tenantID uuid.UUID, limit, offset int) ([]domain.CommissionRule, int, error) {
	var rules []domain.CommissionRule
	var total int

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		if err := tx.QueryRow(ctx,
			`SELECT COUNT(*) FROM order_svc.commission_rules WHERE tenant_id = $1`,
			tenantID,
		).Scan(&total); err != nil {
			return fmt.Errorf("count commission rules: %w", err)
		}

		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, seller_id, category_id, rate_bps, priority, valid_from, valid_until, created_at
			 FROM order_svc.commission_rules WHERE tenant_id = $1
			 ORDER BY priority DESC, created_at DESC LIMIT $2 OFFSET $3`,
			tenantID, limit, offset,
		)
		if err != nil {
			return fmt.Errorf("list commission rules: %w", err)
		}
		defer rows.Close()

		for rows.Next() {
			var rule domain.CommissionRule
			if err := rows.Scan(
				&rule.ID, &rule.TenantID, &rule.SellerID, &rule.CategoryID,
				&rule.RateBps, &rule.Priority, &rule.ValidFrom, &rule.ValidUntil, &rule.CreatedAt,
			); err != nil {
				return fmt.Errorf("scan commission rule: %w", err)
			}
			rules = append(rules, rule)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, 0, err
	}
	return rules, total, nil
}

// Create inserts a new commission rule.
func (r *CommissionRepository) Create(ctx context.Context, tenantID uuid.UUID, rule *domain.CommissionRule) error {
	rule.ID = uuid.New()
	rule.TenantID = tenantID

	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO order_svc.commission_rules
			 (id, tenant_id, seller_id, category_id, rate_bps, priority, valid_from, valid_until)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 RETURNING created_at`,
			rule.ID, rule.TenantID, rule.SellerID, rule.CategoryID,
			rule.RateBps, rule.Priority, rule.ValidFrom, rule.ValidUntil,
		).Scan(&rule.CreatedAt)
	})
}
