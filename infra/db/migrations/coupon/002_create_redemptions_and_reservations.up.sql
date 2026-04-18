-- Pending state for a coupon that a buyer has committed to at checkout
-- but whose order has not yet been paid through Stripe. Reservations
-- expire via the reaper after expires_at so orphaned reservations
-- (order-svc crash, abandoned checkout) don't permanently inflate
-- coupons.usage_count.
CREATE TABLE coupon_svc.coupon_reservations (
    id                       UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coupon_id                UUID NOT NULL REFERENCES coupon_svc.coupons(id),
    buyer_auth0_id           VARCHAR(255) NOT NULL,
    order_candidate_id       UUID NULL,
    stripe_payment_intent_id VARCHAR(255) NULL,
    discount_amount          BIGINT NOT NULL,
    currency                 VARCHAR(3) NOT NULL,
    status                   VARCHAR(16) NOT NULL DEFAULT 'pending',
    expires_at               TIMESTAMPTZ NOT NULL,
    created_at               TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    committed_at             TIMESTAMPTZ NULL,
    released_at              TIMESTAMPTZ NULL,

    CONSTRAINT coupon_reservations_status_chk
        CHECK (status IN ('pending', 'committed', 'released', 'expired')),
    CONSTRAINT coupon_reservations_discount_nonneg_chk
        CHECK (discount_amount >= 0)
);

CREATE INDEX coupon_reservations_reaper_idx
    ON coupon_svc.coupon_reservations (status, expires_at)
    WHERE status = 'pending';
CREATE INDEX coupon_reservations_buyer_idx
    ON coupon_svc.coupon_reservations (buyer_auth0_id, coupon_id);

-- Finalized, immutable redemption log. One row per (coupon, order) pair
-- on successful Commit. The unique constraint on (coupon_id, order_id)
-- gives Stripe webhook replays idempotent Commit semantics without
-- needing distributed locking.
CREATE TABLE coupon_svc.coupon_redemptions (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    coupon_id        UUID NOT NULL REFERENCES coupon_svc.coupons(id),
    buyer_auth0_id   VARCHAR(255) NOT NULL,
    order_id         UUID NOT NULL,
    discount_applied BIGINT NOT NULL,
    reservation_id   UUID NOT NULL REFERENCES coupon_svc.coupon_reservations(id),
    committed_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT coupon_redemptions_coupon_order_unique
        UNIQUE (coupon_id, order_id)
);

CREATE INDEX coupon_redemptions_buyer_idx
    ON coupon_svc.coupon_redemptions (buyer_auth0_id, coupon_id);
CREATE INDEX coupon_redemptions_order_idx
    ON coupon_svc.coupon_redemptions (order_id);
