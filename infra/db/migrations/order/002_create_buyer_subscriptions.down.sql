-- Remove shipping_fee column from orders.
ALTER TABLE order_svc.orders DROP COLUMN IF EXISTS shipping_fee;

-- Drop buyer subscription tables.
DROP TABLE IF EXISTS auth_svc.buyer_subscriptions;
DROP TABLE IF EXISTS auth_svc.buyer_plans;
