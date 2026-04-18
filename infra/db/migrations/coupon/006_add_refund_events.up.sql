-- Per-order refund ledger so redelivered order.cancelled events don't
-- double-refund the cart-wide coupon redemption.
--
-- The problem: the cancellation subscriber receives one event per
-- cancelled seller order in a multi-seller cart, each carrying that
-- order's share of the discount. Without a per-order ledger, a
-- Pub/Sub redelivery of the same cancel event passes through
-- ApplyPartialRefund a second time (because refunded_at is still
-- NULL while the redemption is only partially refunded), adding the
-- share again. Two redeliveries of a 300-share cancel against a
-- 1000-discount cart would record 600 refunded for that one order,
-- and three would prematurely stamp refunded_at and release the seat
-- before the other seller orders even cancelled.
--
-- The solution is a small refund-events table that mirrors the
-- ledger pattern loyalty uses for refund / reverse_earn. UNIQUE
-- (redemption_id, cancelled_order_id) makes the first INSERT the
-- arbiter; every subsequent redelivery hits the constraint and the
-- subscriber treats that as "already refunded this order's share."

CREATE TABLE coupon_svc.coupon_refund_events (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    redemption_id       UUID NOT NULL REFERENCES coupon_svc.coupon_redemptions(id),
    cancelled_order_id  UUID NOT NULL,
    share_applied       BIGINT NOT NULL,
    reason              TEXT,
    created_at          TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT coupon_refund_events_share_nonneg_chk
        CHECK (share_applied >= 0),
    CONSTRAINT coupon_refund_events_per_order_unique
        UNIQUE (redemption_id, cancelled_order_id)
);

CREATE INDEX coupon_refund_events_redemption_idx
    ON coupon_svc.coupon_refund_events (redemption_id);
