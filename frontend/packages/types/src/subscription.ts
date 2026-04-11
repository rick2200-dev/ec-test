/**
 * Subscription plan features. Mirrors the `PlanFeatures` JSON column
 * stored by the billing service — used both by the seller app (to
 * render plan selection) and the admin app (to manage plans).
 *
 * `SubscriptionPlan` itself is deliberately NOT exported here because
 * the admin app carries extra back-office fields (stripe_price_id,
 * status, created_at, ...) that the seller app doesn't need. Each app
 * defines its own plan shape around this shared feature bag.
 */
export interface PlanFeatures {
  max_products: number;
  search_boost: number;
  featured_slots: number;
  promoted_results: number;
}
