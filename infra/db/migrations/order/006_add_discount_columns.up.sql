-- Add discount / loyalty fields so order-svc can record coupon and point
-- redemptions applied at checkout. All new columns default to 0 / NULL so
-- pre-existing orders remain consistent without a backfill.
--
-- Ownership stays with order-svc (not coupon-svc or loyalty-svc) because
-- the authoritative total lives here and needs to reconcile with the
-- Stripe PaymentIntent amount atomically during checkout.

ALTER TABLE order_svc.orders
    ADD COLUMN coupon_discount_amount BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN point_discount_amount  BIGINT NOT NULL DEFAULT 0,
    ADD COLUMN coupon_id              UUID NULL,
    ADD COLUMN coupon_reservation_id  UUID NULL,
    ADD COLUMN point_reservation_id   UUID NULL,
    ADD COLUMN points_earned          BIGINT NOT NULL DEFAULT 0;

-- Helpful for coupon-svc reconciliation dashboards that join by reservation.
CREATE INDEX idx_orders_coupon_reservation ON order_svc.orders(coupon_reservation_id)
    WHERE coupon_reservation_id IS NOT NULL;
CREATE INDEX idx_orders_point_reservation ON order_svc.orders(point_reservation_id)
    WHERE point_reservation_id IS NOT NULL;
