-- Historical migration.
--
-- The original up-script created auth_svc.buyer_plans +
-- auth_svc.buyer_subscriptions and seeded a buyer Premium plan.
-- subscription/002 later relocated those tables to subscription_svc
-- and dropped the auth_svc originals.
--
-- After Phase 3 (order split to its own Postgres instance), a fresh
-- order-only DB has no auth_svc schema, so the original CREATE TABLE
-- auth_svc.* statements would fail. The buyer plan/subscription tables
-- are fully owned by subscription now.
--
-- The shipping_fee column addition on order_svc.orders is preserved
-- since it does belong to the order schema.

ALTER TABLE order_svc.orders ADD COLUMN IF NOT EXISTS shipping_fee BIGINT NOT NULL DEFAULT 0;
