-- Adds a flag column used by HandlePaymentSuccess to publish the
-- `order.paid` + `payout.completed` event pair exactly once per order,
-- independently of whether the Stripe transfer just happened or was
-- already completed on a prior attempt.
--
-- Without this, a webhook retry that passes the transfer block via
-- payout.Status=completed could succeed at coupon/loyalty commit yet
-- never republish `order.paid` — starving shipping / notification /
-- recommend / loyalty-earn-fanin consumers of the order entirely.

ALTER TABLE order_svc.orders
    ADD COLUMN paid_event_published_at TIMESTAMPTZ NULL;

-- The handler's guard is "publish if paid_event_published_at IS NULL",
-- so a partial index on the null rows is the natural shape for the
-- eventual catch-up query.
CREATE INDEX idx_orders_paid_event_unpublished
    ON order_svc.orders (paid_at)
    WHERE paid_event_published_at IS NULL AND paid_at IS NOT NULL;
