DROP INDEX IF EXISTS coupon_svc.coupon_reservations_applicable_seller_idx;
ALTER TABLE coupon_svc.coupon_reservations
    DROP COLUMN IF EXISTS applicable_seller_id;
