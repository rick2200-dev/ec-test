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

// BuyerSubscriptionRepository implements port.BuyerSubscriptionStore.
type BuyerSubscriptionRepository struct {
	pool *pgxpool.Pool
}

// NewBuyerSubscriptionRepository creates a new BuyerSubscriptionRepository.
func NewBuyerSubscriptionRepository(pool *pgxpool.Pool) *BuyerSubscriptionRepository {
	return &BuyerSubscriptionRepository{pool: pool}
}

func (r *BuyerSubscriptionRepository) CreateBuyerPlan(ctx context.Context, p *domain.BuyerPlan) error {
	p.ID = uuid.New()

	featuresJSON, err := json.Marshal(p.Features)
	if err != nil {
		return fmt.Errorf("marshal buyer plan features: %w", err)
	}

	return database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO subscription_svc.buyer_plans
			 (id, name, slug, price_amount, price_currency, features, stripe_price_id, status)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			 RETURNING created_at, updated_at`,
			p.ID, p.Name, p.Slug,
			p.PriceAmount, p.PriceCurrency, featuresJSON, p.StripePriceID, p.Status,
		).Scan(&p.CreatedAt, &p.UpdatedAt)
	})
}

func (r *BuyerSubscriptionRepository) GetBuyerPlanByID(ctx context.Context, id uuid.UUID) (*domain.BuyerPlan, error) {
	var p domain.BuyerPlan
	var featuresJSON []byte
	var found bool

	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT id, name, slug, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at
			 FROM subscription_svc.buyer_plans WHERE id = $1`,
			id,
		).Scan(&p.ID, &p.Name, &p.Slug,
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
		return nil, fmt.Errorf("get buyer plan by id: %w", err)
	}
	if !found {
		return nil, nil
	}
	if err := json.Unmarshal(featuresJSON, &p.Features); err != nil {
		return nil, fmt.Errorf("unmarshal buyer plan features: %w", err)
	}
	return &p, nil
}

func (r *BuyerSubscriptionRepository) ListBuyerPlans(ctx context.Context) ([]domain.BuyerPlan, error) {
	var plans []domain.BuyerPlan

	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		rows, err := tx.Query(ctx,
			`SELECT id, name, slug, price_amount, price_currency, features, stripe_price_id, status, created_at, updated_at
			 FROM subscription_svc.buyer_plans WHERE status = 'active'
			 ORDER BY price_amount ASC`,
		)
		if err != nil {
			return err
		}
		defer rows.Close()

		for rows.Next() {
			var p domain.BuyerPlan
			var featuresJSON []byte
			if err := rows.Scan(&p.ID, &p.Name, &p.Slug,
				&p.PriceAmount, &p.PriceCurrency, &featuresJSON, &p.StripePriceID, &p.Status,
				&p.CreatedAt, &p.UpdatedAt); err != nil {
				return err
			}
			if err := json.Unmarshal(featuresJSON, &p.Features); err != nil {
				return fmt.Errorf("unmarshal buyer plan features: %w", err)
			}
			plans = append(plans, p)
		}
		return rows.Err()
	})
	if err != nil {
		return nil, fmt.Errorf("list buyer plans: %w", err)
	}
	return plans, nil
}

func (r *BuyerSubscriptionRepository) UpdateBuyerPlan(ctx context.Context, p *domain.BuyerPlan) error {
	featuresJSON, err := json.Marshal(p.Features)
	if err != nil {
		return fmt.Errorf("marshal buyer plan features: %w", err)
	}

	return database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		tag, err := tx.Exec(ctx,
			`UPDATE subscription_svc.buyer_plans
			 SET name = $2, slug = $3, price_amount = $4, price_currency = $5,
			     features = $6, stripe_price_id = $7, status = $8, updated_at = NOW()
			 WHERE id = $1`,
			p.ID, p.Name, p.Slug,
			p.PriceAmount, p.PriceCurrency, featuresJSON, p.StripePriceID, p.Status,
		)
		if err != nil {
			return fmt.Errorf("update buyer plan: %w", err)
		}
		if tag.RowsAffected() == 0 {
			return fmt.Errorf("buyer plan not found")
		}
		return nil
	})
}

func (r *BuyerSubscriptionRepository) GetBuyerSubscription(ctx context.Context, buyerAuth0ID string) (*domain.BuyerSubscriptionWithPlan, error) {
	var sub domain.BuyerSubscriptionWithPlan
	var featuresJSON []byte
	var found bool

	err := database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		err := tx.QueryRow(ctx,
			`SELECT bs.id, bs.buyer_auth0_id, bs.plan_id,
			        bs.stripe_subscription_id, bs.stripe_customer_id, bs.status,
			        bs.current_period_start, bs.current_period_end, bs.canceled_at,
			        bs.created_at, bs.updated_at,
			        bp.name, bp.slug, bp.features
			 FROM subscription_svc.buyer_subscriptions bs
			 JOIN subscription_svc.buyer_plans bp ON bp.id = bs.plan_id
			 WHERE bs.buyer_auth0_id = $1`,
			buyerAuth0ID,
		).Scan(
			&sub.ID, &sub.BuyerAuth0ID, &sub.PlanID,
			&sub.StripeSubscriptionID, &sub.StripeCustomerID, &sub.Status,
			&sub.CurrentPeriodStart, &sub.CurrentPeriodEnd, &sub.CanceledAt,
			&sub.CreatedAt, &sub.UpdatedAt,
			&sub.PlanName, &sub.PlanSlug, &featuresJSON,
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
		return nil, fmt.Errorf("get buyer subscription: %w", err)
	}
	if !found {
		return nil, nil
	}
	if err := json.Unmarshal(featuresJSON, &sub.Features); err != nil {
		return nil, fmt.Errorf("unmarshal buyer plan features: %w", err)
	}
	return &sub, nil
}

func (r *BuyerSubscriptionRepository) UpsertBuyerSubscription(ctx context.Context, sub *domain.BuyerSubscription) error {
	return database.Tx(ctx, r.pool, func(tx pgx.Tx) error {
		return tx.QueryRow(ctx,
			`INSERT INTO subscription_svc.buyer_subscriptions
			 (id, buyer_auth0_id, plan_id, stripe_subscription_id, stripe_customer_id, status,
			  current_period_start, current_period_end, canceled_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
			 ON CONFLICT (buyer_auth0_id)
			 DO UPDATE SET plan_id = EXCLUDED.plan_id,
			              stripe_subscription_id = EXCLUDED.stripe_subscription_id,
			              stripe_customer_id = EXCLUDED.stripe_customer_id,
			              status = EXCLUDED.status,
			              current_period_start = EXCLUDED.current_period_start,
			              current_period_end = EXCLUDED.current_period_end,
			              canceled_at = EXCLUDED.canceled_at,
			              updated_at = NOW()
			 RETURNING created_at, updated_at`,
			sub.ID, sub.BuyerAuth0ID, sub.PlanID,
			sub.StripeSubscriptionID, sub.StripeCustomerID, sub.Status,
			sub.CurrentPeriodStart, sub.CurrentPeriodEnd, sub.CanceledAt,
		).Scan(&sub.CreatedAt, &sub.UpdatedAt)
	})
}
