DROP INDEX IF EXISTS order_svc.idx_orders_point_reservation;
DROP INDEX IF EXISTS order_svc.idx_orders_coupon_reservation;

ALTER TABLE order_svc.orders
    DROP COLUMN IF EXISTS points_earned,
    DROP COLUMN IF EXISTS point_reservation_id,
    DROP COLUMN IF EXISTS coupon_reservation_id,
    DROP COLUMN IF EXISTS coupon_id,
    DROP COLUMN IF EXISTS point_discount_amount,
    DROP COLUMN IF EXISTS coupon_discount_amount;
