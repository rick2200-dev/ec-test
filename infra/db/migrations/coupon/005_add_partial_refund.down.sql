ALTER TABLE coupon_svc.coupon_redemptions
    DROP CONSTRAINT IF EXISTS coupon_redemptions_refund_bounds_chk;

ALTER TABLE coupon_svc.coupon_redemptions
    DROP COLUMN IF EXISTS refunded_amount;
