-- Historical no-op (see .up.sql). Only the shipping_fee column belongs
-- to order; the buyer plan/subscription tables are owned by subscription.
ALTER TABLE order_svc.orders DROP COLUMN IF EXISTS shipping_fee;
