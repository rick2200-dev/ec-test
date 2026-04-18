-- Adds partial-refund tracking to coupon_redemptions so a multi-seller
-- checkout can refund per seller order share when that order is
-- individually cancelled, without releasing the coupon seat until every
-- cart order has been cancelled.
--
-- Before: refunded_at was a boolean-ish timestamp stamped on the first
-- refund, which made cancelling one order-of-three either
--   (a) refund nothing (if it wasn't the anchor), or
--   (b) refund the full cart discount and restore the seat (if it was
--       the anchor) even when other seller orders still held the discount.
-- Neither matches "refund what you actually cancelled".
--
-- After: refunded_amount accumulates per-cancel, refunded_at gets stamped
-- only when the full discount has been refunded, and usage_count is
-- released only on that same transition. A 3-order cart where 2 are
-- cancelled leaves the redemption partially refunded and the seat still
-- held until the 3rd cancel (or never, if the 3rd order ships).

ALTER TABLE coupon_svc.coupon_redemptions
    ADD COLUMN refunded_amount BIGINT NOT NULL DEFAULT 0;

-- Invariant: can't refund more than was applied, can't go below zero.
ALTER TABLE coupon_svc.coupon_redemptions
    ADD CONSTRAINT coupon_redemptions_refund_bounds_chk
        CHECK (refunded_amount >= 0 AND refunded_amount <= discount_applied);
