-- Order cancellation request feature.
--
-- Adds a buyer-initiated cancellation request that sellers approve or reject.
-- On approval the order service refunds the Stripe PaymentIntent, reverses the
-- per-seller Transfer(s), marks the order cancelled, and publishes an event
-- that causes the inventory service to release reserved stock.
--
-- Design notes:
--   * Partial unique index `ux_cancellation_pending_per_order` ensures at most
--     one pending request per order — this is the concurrency guardrail that
--     prevents two racing buyer clicks from creating duplicate requests.
--   * The `failed` status exists for the case where the seller clicked approve
--     but the Stripe refund or transfer reversal irrecoverably failed; the row
--     is kept so the request is no longer counted as pending (freeing the
--     partial unique index) and an operator can reconcile from the Stripe
--     dashboard. Idempotency keys on Stripe calls are deterministic per
--     request id so retries are safe.
--   * FORCE ROW LEVEL SECURITY is applied because this table is only ever
--     accessed through database.TenantTx (there is no webhook or bootstrap
--     code path that needs to bypass RLS).

CREATE TABLE order_svc.order_cancellation_requests (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    order_id UUID NOT NULL REFERENCES order_svc.orders(id),
    requested_by_auth0_id VARCHAR(255) NOT NULL,
    reason TEXT NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending'
        CHECK (status IN ('pending', 'approved', 'rejected', 'failed')),
    seller_comment TEXT,
    processed_by_seller_id UUID,
    processed_at TIMESTAMPTZ,
    stripe_refund_id VARCHAR(255),
    failure_reason TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- At most one pending request per order. This is the authoritative concurrency
-- guardrail against two buyer clicks racing to create duplicate requests.
CREATE UNIQUE INDEX ux_cancellation_pending_per_order
    ON order_svc.order_cancellation_requests(order_id)
    WHERE status = 'pending';

CREATE INDEX idx_cancellation_requests_tenant
    ON order_svc.order_cancellation_requests(tenant_id);

CREATE INDEX idx_cancellation_requests_order
    ON order_svc.order_cancellation_requests(tenant_id, order_id);

CREATE INDEX idx_cancellation_requests_status
    ON order_svc.order_cancellation_requests(tenant_id, status, created_at DESC);

CREATE INDEX idx_cancellation_requests_buyer
    ON order_svc.order_cancellation_requests(tenant_id, requested_by_auth0_id, created_at DESC);

-- Row-Level Security
ALTER TABLE order_svc.order_cancellation_requests ENABLE ROW LEVEL SECURITY;
ALTER TABLE order_svc.order_cancellation_requests FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON order_svc.order_cancellation_requests
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Denormalized cancellation fields on orders. Copied from the approved request
-- so order detail reads don't need to join against the request table.
ALTER TABLE order_svc.orders
    ADD COLUMN cancelled_at TIMESTAMPTZ,
    ADD COLUMN cancellation_reason TEXT;

-- Payout reversal tracking. `status` reuses the existing VARCHAR(20) column;
-- a new literal 'reversed' is introduced alongside pending/completed/failed.
-- The original stripe_transfer_id is preserved so the reversal can be linked
-- back to its origin transfer for reconciliation.
ALTER TABLE order_svc.payouts
    ADD COLUMN reversed_at TIMESTAMPTZ,
    ADD COLUMN stripe_reversal_id VARCHAR(255);
