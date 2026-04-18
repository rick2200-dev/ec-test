DROP INDEX IF EXISTS coupon_svc.coupon_redemptions_refunded_idx;

ALTER TABLE coupon_svc.coupon_redemptions
    DROP COLUMN IF EXISTS refunded_reason,
    DROP COLUMN IF EXISTS refunded_at;
