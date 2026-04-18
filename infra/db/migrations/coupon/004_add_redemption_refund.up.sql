-- Add the refund columns so the order.cancelled subscriber can mark
-- a redemption as reversed without dropping the audit trail. We keep
-- the row and just stamp refunded_at / refunded_reason — that lets
-- stats-dashboard queries exclude refunded redemptions cleanly and
-- also makes the UNIQUE (coupon_id, order_id) constraint still reject
-- a retry-after-refund as "already handled".

ALTER TABLE coupon_svc.coupon_redemptions
    ADD COLUMN refunded_at      TIMESTAMPTZ NULL,
    ADD COLUMN refunded_reason  TEXT NULL;

CREATE INDEX coupon_redemptions_refunded_idx
    ON coupon_svc.coupon_redemptions (refunded_at)
    WHERE refunded_at IS NOT NULL;
