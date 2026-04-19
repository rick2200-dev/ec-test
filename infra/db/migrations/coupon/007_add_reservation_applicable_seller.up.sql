-- Track the seller a reservation applies to when the coupon was
-- seller-issued. NULL for platform (cart-wide) coupons. Kept on the
-- reservation row so the commit / refund paths can enforce per-seller
-- share routing without re-fetching the coupon row (which the admin
-- could mutate between Reserve and Commit, changing issuer attribution
-- retroactively — not what we want).
ALTER TABLE coupon_svc.coupon_reservations
    ADD COLUMN applicable_seller_id UUID NULL;

-- Partial index: only seller-scoped reservations benefit from an index
-- since platform coupons (NULL) are the common case and platform
-- queries don't filter on this column.
CREATE INDEX coupon_reservations_applicable_seller_idx
    ON coupon_svc.coupon_reservations (applicable_seller_id)
    WHERE applicable_seller_id IS NOT NULL;
