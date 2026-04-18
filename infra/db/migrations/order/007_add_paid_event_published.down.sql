DROP INDEX IF EXISTS order_svc.idx_orders_paid_event_unpublished;

ALTER TABLE order_svc.orders
    DROP COLUMN IF EXISTS paid_event_published_at;
