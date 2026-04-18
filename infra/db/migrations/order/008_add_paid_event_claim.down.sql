DROP INDEX IF EXISTS order_svc.idx_orders_paid_event_claim_stale;

ALTER TABLE order_svc.orders
    DROP COLUMN IF EXISTS paid_event_claim_at;
