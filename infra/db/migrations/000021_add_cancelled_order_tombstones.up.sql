-- cancelled_order_tombstones records order IDs that received an
-- order.cancelled event before a shipment row existed. When order.paid
-- arrives out of order and the shipping subscriber tries to create a
-- ready_to_ship shipment, it checks this table first and skips creation
-- for any order that was already cancelled.
--
-- Rows in this table are logically permanent: once an order is cancelled,
-- it should never become shippable again. No TTL / cleanup is defined for v1.

CREATE TABLE IF NOT EXISTS shipping_svc.cancelled_order_tombstones (
    order_id   UUID        NOT NULL,
    tenant_id  UUID        NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (order_id)
);

ALTER TABLE shipping_svc.cancelled_order_tombstones ENABLE ROW LEVEL SECURITY;
ALTER TABLE shipping_svc.cancelled_order_tombstones FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation
    ON shipping_svc.cancelled_order_tombstones
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
