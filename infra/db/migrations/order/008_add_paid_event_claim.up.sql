-- Adds a TTL-based claim column used by HandlePaymentSuccess to
-- serialize concurrent webhook deliveries through the publish block.
--
-- Race being closed: two webhook retries arriving near-simultaneously
-- both read paid_event_published_at IS NULL, both publish order.paid
-- to Pub/Sub (subscribers see duplicates), only one MarkPaidEventPublished
-- wins the UPDATE. Subscribers should be idempotent by order_id, but
-- some side effects (notification emails) benefit from "publish once".
--
-- Claim pattern: the handler acquires a time-bounded exclusive claim
-- by atomically setting paid_event_claim_at = NOW() when the previous
-- claim is either NULL or older than a TTL. Only the winning claimer
-- proceeds to publish. A crash between claim and publish eventually
-- unsticks itself once the TTL expires, so we never need manual
-- intervention to recover.

ALTER TABLE order_svc.orders
    ADD COLUMN paid_event_claim_at TIMESTAMPTZ NULL;

-- Finding stale claims (for the re-claim path) wants a fast scan on
-- the filter expression, partial-index it.
CREATE INDEX idx_orders_paid_event_claim_stale
    ON order_svc.orders (paid_event_claim_at)
    WHERE paid_event_published_at IS NULL AND paid_event_claim_at IS NOT NULL;
