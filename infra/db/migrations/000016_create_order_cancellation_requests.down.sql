ALTER TABLE order_svc.payouts
    DROP COLUMN IF EXISTS stripe_reversal_id,
    DROP COLUMN IF EXISTS reversed_at;

ALTER TABLE order_svc.orders
    DROP COLUMN IF EXISTS cancellation_reason,
    DROP COLUMN IF EXISTS cancelled_at;

DROP POLICY IF EXISTS tenant_isolation ON order_svc.order_cancellation_requests;
DROP TABLE IF EXISTS order_svc.order_cancellation_requests;
