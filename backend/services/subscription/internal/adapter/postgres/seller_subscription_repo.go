// Package repository implements the subscription service's Postgres
// adapters. All tables live under the subscription_svc schema; the
// catalog_svc.seller_plan_boost materialized view continues to live in
// catalog_svc but now reads from subscription_svc tables.
package repository

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/Riku-KANO/ec-test/pkg/database"
	"github.com/Riku-KANO/ec-test/services/subscription/internal/domain"
)

// SellerSubscriptionRepository implements port.SellerSubscriptionStore.
type SellerSubscriptionRepository struct {
	pool *pgxpool.Pool
}

// NewSellerSubscriptionRepository creates a new SellerSubscriptionRepository.
func NewSellerSubscriptionRepository(pool *pgxpool.Pool) *SellerSubscriptionRepository {
	return &SellerSubscriptionRepository{pool: pool}
}

func (r *SellerSubscriptionRepository) CreatePlan(ctx context.Context, tenantID uuid.UUID, p *domain.SubscriptionPlan) error {
	p.ID = uuid.New()
	p.TenantID = tenantID

	featuresJSON, err := json.Marshal(p.Features)
	if err != nil {
		return fmt.Errorf("marshal plan features: %w", err)
	}

	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO subscription_svc.subscription_plans
			 (id, tenant_id, name, slug, tier, price_amount, price_currency, features, stripe_price_id, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			 RETURNING created_at, updated_at`,
			p.ID, p.TenantID, p.Name, p.Slug, p.Tier,
			p.PriceAmount, p.PriceCurrency, featuresJSON, p.StripePriceID, p.Status,
		).Scan(&p.CreatedAt, &p.UpdatedAt)
	})
}

func (r *SellerSubscriptionRepository) GetPlanByID(ctx context.Context, tenantID, id uuid.UUID) (*domain.SubscriptionPlan, error) {
	var p domain.SubscriptionPlan
	var featuresJSON []byte
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, tenant_id, name, slug, tier, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at
			 FROM subscription_svc.subscription_plans WHERE id = $1 AND tenant_id = $2`,
			id, tenantID,
		).Scan(&p.ID, &p.TenantID, &p.Name, &p.Slug, &p.Tier,
			&p.PriceAmount, &p.PriceCurrency, &featuresJSON, &p.StripePriceID, &p.Status,
			&p.CreatedAt, &p.UpdatedAt)
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
		return nil, fmt.Errorf("get plan by id: %w", err)
	}
	if !found {
		return nil, nil
	}
	if err := json.Unmarshal(featuresJSON, &p.Features); err != nil {
		return nil, fmt.Errorf("unmarshal plan features: %w", err)
	}
	return &p, nil
}

func (r *SellerSubscriptionRepository) ListPlans(ctx context.Context, tenantID uuid.UUID) ([]domain.SubscriptionPlan, error) {
	var plans []domain.SubscriptionPlan

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, tenant_id, name, slug, tier, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at
			 FROM subscription_svc.subscription_plans WHERE tenant_id = $1 AND status = 'active'
			 ORDER BY tier ASC`,
			tenantID,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var p domain.SubscriptionPlan
			var featuresJSON []byte
			if err := rows.Scan(&p.ID, &p.TenantID, &p.Name, &p.Slug, &p.Tier,
				&p.PriceAmount, &p.PriceCurrency, &featuresJSON, &p.StripePriceID, &p.Status,
				&p.CreatedAt, &p.UpdatedAt); err != nil {
				return err
			}
			if err := json.Unmarshal(featuresJSON, &p.Features); err != nil {
				return fmt.Errorf("unmarshal plan features: %w", err)
			}
			plans = append(plans, p)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("list plans: %w", err)
	}
	return plans, nil
}

func (r *SellerSubscriptionRepository) UpdatePlan(ctx context.Context, tenantID uuid.UUID, p *domain.SubscriptionPlan) error {
	featuresJSON, err := json.Marshal(p.Features)
	if err != nil {
		return fmt.Errorf("marshal plan features: %w", err)
	}

	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE subscription_svc.subscription_plans
			 SET name = $3, slug = $4, tier = $5, price_amount = $6, price_currency = $7,
			     features = $8, stripe_price_id = $9, status = $10, updated_at = NOW()
			 WHERE id = $1 AND tenant_id = $2`,
			p.ID, tenantID, p.Name, p.Slug, p.Tier,
			p.PriceAmount, p.PriceCurrency, featuresJSON, p.StripePriceID, p.Status,
		)
		if err != nil {
			return fmt.Errorf("update plan: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("plan not found")
		}
		return nil
	})
}

func (r *SellerSubscriptionRepository) GetSellerSubscription(ctx context.Context, tenantID, sellerID uuid.UUID) (*domain.SellerSubscriptionWithPlan, error) {
	var sub domain.SellerSubscriptionWithPlan
	var found bool

	err := database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT ss.id, ss.tenant_id, ss.seller_id, ss.plan_id,
			        ss.stripe_subscription_id, ss.stripe_customer_id, ss.status,
			        ss.current_period_start, ss.current_period_end, ss.canceled_at,
			        ss.created_at, ss.updated_at,
			        sp.name, sp.slug, sp.tier
			 FROM subscription_svc.seller_subscriptions ss
			 JOIN subscription_svc.subscription_plans sp ON sp.id = ss.plan_id
			 WHERE ss.seller_id = $1 AND ss.tenant_id = $2`,
			sellerID, tenantID,
		).Scan(
			&sub.ID, &sub.TenantID, &sub.SellerID, &sub.PlanID,
			&sub.StripeSubscriptionID, &sub.StripeCustomerID, &sub.Status,
			&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CanceledAt,
			&sub.CreatedAt, &sub.UpdatedAt,
			&sub.PlanName, &sub.PlanSlug, &sub.PlanTier,
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
		return nil, fmt.Errorf("get seller subscription: %w", err)
	}
	if !found {
		return nil, nil
	}
	return &sub, nil
}

func (r *SellerSubscriptionRepository) UpsertSellerSubscription(ctx context.Context, tenantID uuid.UUID, sub *domain.SellerSubscription) error {
	return database.TenantTx(ctx, r.pool, tenantID, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO subscription_svc.seller_subscriptions
			 (id, tenant_id, seller_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
			  current_period_start, current_period_end, canceled_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
			 ON CONFLICT (tenant_id, seller_id)
			 DO UPDATE SET plan_id = EXCLUDED.plan_id,
			              stripe_subscription_id = EXCLUDED.stripe_subscription_id,
			              stripe_customer_id = EXCLUDED.stripe_customer_id,
			              status = EXCLUDED.status,
			              current_period_start = EXCLUDED.current_period_start,
			              current_period_end = EXCLUDED.current_period_end,
			              canceled_at = EXCLUDED.canceled_at,
			              updated_at = NOW()
			 RETURNING created_at, updated_at`,
			sub.ID, tenantID, sub.SellerID, sub.PlanID,
			sub.StripeSubscriptionID, sub.StripeCustomerID, sub.Status,
			sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.CanceledAt,
		).Scan(&sub.CreatedAt, &sub.UpdatedAt)
	})
}

// RefreshPlanBoostView refreshes the cross-schema materialized view used by
// the search engine. The view definition joins subscription_svc and auth_svc
// tables; we only own the REFRESH trigger point here.
//
// The refresh is delegated to the `catalog_svc.refresh_seller_plan_boost()`
// SECURITY DEFINER function (see migration 000020). A direct
// `REFRESH MATERIALIZED VIEW` from the application role would hit FORCE
// ROW LEVEL SECURITY on the underlying subscription_svc tables — no
// tenant context is set for a background refresh, so the view would be
// rebuilt against an empty row set. The helper function runs as a
// BYPASSRLS-owning role so the refresh sees every tenant's rows.
func (r *SellerSubscriptionRepository) RefreshPlanBoostView(ctx context.Context) error {
	_, err := r.pool.Exec(ctx, `SELECT catalog_svc.refresh_seller_plan_boost()`)
	if err != nil {
		return fmt.Errorf("refresh seller_plan_boost view: %w", err)
	}
	return nil
}
