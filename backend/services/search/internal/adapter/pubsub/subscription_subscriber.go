package subscriber

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/reflect/protoreflect"

	"github.com/Riku-KANO/ec-test/pkg/pubsub"
	subscriptionv1 "github.com/Riku-KANO/ec-test/services/subscription/api/gen/go/subscription/v1"
)

// SubscriptionSubscriber maintains search_svc.seller_plan_boost from the
// subscription service's plan lifecycle events. Replaces the former
// catalog_svc.seller_plan_boost materialized view that both catalog and
// subscription used to read/refresh cross-schema.
type SubscriptionSubscriber struct {
	pool       *pgxpool.Pool
	subscriber pubsub.Subscriber
}

// SubscriptionEventsSubscription is the pub/sub subscription name.
const SubscriptionEventsSubscription = "subscription-events-search"

// NewSubscriptionSubscriber wires the subscriber to the pool it writes to.
func NewSubscriptionSubscriber(pool *pgxpool.Pool, sub pubsub.Subscriber) *SubscriptionSubscriber {
	return &SubscriptionSubscriber{pool: pool, subscriber: sub}
}

// Start begins consuming subscription-events-search. Blocks until ctx done.
func (s *SubscriptionSubscriber) Start(ctx context.Context) error {
	slog.Info("starting subscription event subscriber", "subscription", SubscriptionEventsSubscription)
	return s.subscriber.Subscribe(ctx, SubscriptionEventsSubscription, s.handle)
}

func (s *SubscriptionSubscriber) handle(ctx context.Context, event pubsub.Event) error {
	switch event.Type {
	case "seller.subscribed":
		var e subscriptionv1.SellerSubscribed
		if err := decodeProtoEvent(event.Data, &e); err != nil {
			return fmt.Errorf("decode seller.subscribed: %w", err)
		}
		return s.upsertBoost(ctx, e.SellerId, e.PlanTier, e.PlanSlug, e.SearchBoost, e.PromotedResults)
	case "seller_subscription.cancelled":
		var e subscriptionv1.SellerSubscriptionCancelled
		if err := decodeProtoEvent(event.Data, &e); err != nil {
			return fmt.Errorf("decode seller_subscription.cancelled: %w", err)
		}
		return s.resetBoost(ctx, e.SellerId)
	case "subscription_plan.updated":
		// Re-applies plan features to every seller currently on the plan.
		// The event carries no cohort list, so the subscriber scans the
		// projection for rows matching this plan_tier via plan_slug. A
		// more accurate approach would track plan_id per row, but tier +
		// slug have always been the only search-relevant fields.
		var e subscriptionv1.SubscriptionPlanUpdated
		if err := decodeProtoEvent(event.Data, &e); err != nil {
			return fmt.Errorf("decode subscription_plan.updated: %w", err)
		}
		return s.applyPlanUpdate(ctx, e.PlanId, e.PlanTier, e.SearchBoost, e.PromotedResults)
	default:
		slog.Debug("ignoring unhandled subscription event type", "type", event.Type)
		return nil
	}
}

func (s *SubscriptionSubscriber) upsertBoost(ctx context.Context, sellerIDStr string, tier int32, slug string, boost float64, promoted int32) error {
	sellerID, err := uuid.Parse(sellerIDStr)
	if err != nil {
		return fmt.Errorf("parse seller_id %q: %w", sellerIDStr, err)
	}
	if boost == 0 {
		boost = 1.0
	}
	if slug == "" {
		slug = "free"
	}
	_, err = s.pool.Exec(ctx,
		`INSERT INTO search_svc.seller_plan_boost (seller_id, plan_tier, plan_slug, search_boost, promoted_results, updated_at)
		 VALUES ($1, $2, $3, $4, $5, NOW())
		 ON CONFLICT (seller_id) DO UPDATE SET
		   plan_tier        = EXCLUDED.plan_tier,
		   plan_slug        = EXCLUDED.plan_slug,
		   search_boost     = EXCLUDED.search_boost,
		   promoted_results = EXCLUDED.promoted_results,
		   updated_at       = NOW()`,
		sellerID, int(tier), slug, boost, int(promoted),
	)
	return err
}

func (s *SubscriptionSubscriber) resetBoost(ctx context.Context, sellerIDStr string) error {
	sellerID, err := uuid.Parse(sellerIDStr)
	if err != nil {
		return fmt.Errorf("parse seller_id %q: %w", sellerIDStr, err)
	}
	// Reset to free-tier defaults rather than deleting, so the JOIN in
	// search queries still finds a row and returns baseline ranking.
	_, err = s.pool.Exec(ctx,
		`UPDATE search_svc.seller_plan_boost
		 SET plan_tier = 0, plan_slug = 'free', search_boost = 1.0, promoted_results = 0, updated_at = NOW()
		 WHERE seller_id = $1`,
		sellerID,
	)
	return err
}

// applyPlanUpdate is a coarse-grained refresh for every seller whose
// plan_tier matches the updated plan. Called when the admin bumps a plan's
// search_boost. Small cardinality in practice so a bulk UPDATE is fine.
func (s *SubscriptionSubscriber) applyPlanUpdate(ctx context.Context, planIDStr string, tier int32, boost float64, promoted int32) error {
	if _, err := uuid.Parse(planIDStr); err != nil {
		return fmt.Errorf("parse plan_id %q: %w", planIDStr, err)
	}
	if boost == 0 {
		boost = 1.0
	}
	_, err := s.pool.Exec(ctx,
		`UPDATE search_svc.seller_plan_boost
		 SET search_boost = $1, promoted_results = $2, updated_at = NOW()
		 WHERE plan_tier = $3`,
		boost, int(promoted), int(tier),
	)
	return err
}

func decodeProtoEvent(eventData any, target protoreflect.ProtoMessage) error {
	raw, err := json.Marshal(eventData)
	if err != nil {
		return fmt.Errorf("marshal event data: %w", err)
	}
	opts := protojson.UnmarshalOptions{DiscardUnknown: true}
	if err := opts.Unmarshal(raw, target); err != nil {
		return fmt.Errorf("unmarshal proto event data: %w", err)
	}
	return nil
}
