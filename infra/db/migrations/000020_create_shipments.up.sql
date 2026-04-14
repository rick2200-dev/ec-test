-- Shipping service tables.
--
-- A shipment is a 1:1 counterpart to an order (v1). It is created automatically
-- when an order.paid event is received and tracks the physical delivery lifecycle:
--   pending → ready_to_ship → shipped → delivered
--                   │
--                   └── cancelled  (when order.cancelled is received before shipment)
--
-- Design notes:
--   * UNIQUE (order_id) enforces the 1-order-1-shipment invariant at the DB level
--     and makes the order.paid consumer idempotent via ON CONFLICT DO NOTHING.
--   * shipping_address is a JSONB snapshot copied from orders.shipping_address at
--     shipment creation time. This prevents address changes from affecting in-flight
--     deliveries and keeps the shipping service independent of the order schema.
--   * buyer_auth0_id is denormalised here so the shipping service can authorise
--     buyer reads without calling out to the order service.
--   * FORCE ROW LEVEL SECURITY is applied because all access paths go through
--     database.RunTenantTx which sets app.current_tenant_id. There is no webhook
--     or bootstrap code path that needs to bypass RLS.

CREATE SCHEMA IF NOT EXISTS shipping_svc;

CREATE TABLE shipping_svc.shipments (
    id               UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id        UUID          NOT NULL,
    seller_id        UUID          NOT NULL,
    order_id         UUID          NOT NULL,
    buyer_auth0_id   VARCHAR(255)  NOT NULL,
    status           VARCHAR(20)   NOT NULL DEFAULT 'ready_to_ship'
                         CHECK (status IN ('pending','ready_to_ship','shipped','delivered','cancelled')),
    shipping_address JSONB         NOT NULL DEFAULT '{}',
    carrier          VARCHAR(64),
    tracking_number  VARCHAR(128),
    shipped_at       TIMESTAMPTZ,
    delivered_at     TIMESTAMPTZ,
    note             TEXT,
    created_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ   NOT NULL DEFAULT NOW()
);

-- 1 order → at most 1 shipment. Also makes ON CONFLICT DO NOTHING idempotent.
CREATE UNIQUE INDEX ux_shipments_order_id
    ON shipping_svc.shipments(order_id);

CREATE INDEX idx_shipments_tenant
    ON shipping_svc.shipments(tenant_id);

CREATE INDEX idx_shipments_seller_status
    ON shipping_svc.shipments(tenant_id, seller_id, status, created_at DESC);

CREATE INDEX idx_shipments_buyer
    ON shipping_svc.shipments(tenant_id, buyer_auth0_id, created_at DESC);

-- Row-Level Security
ALTER TABLE shipping_svc.shipments ENABLE ROW LEVEL SECURITY;
ALTER TABLE shipping_svc.shipments FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON shipping_svc.shipments
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

-- Audit log: one row per status transition.
-- actor_type distinguishes system-driven transitions (order events) from
-- seller-initiated ones (register / deliver endpoints).
CREATE TABLE shipping_svc.shipment_events (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id    UUID         NOT NULL,
    shipment_id  UUID         NOT NULL REFERENCES shipping_svc.shipments(id),
    from_status  VARCHAR(20),
    to_status    VARCHAR(20)  NOT NULL,
    actor_type   VARCHAR(20)  NOT NULL CHECK (actor_type IN ('seller','system')),
    actor_id     VARCHAR(255),
    payload      JSONB,
    created_at   TIMESTAMPTZ  NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_shipment_events_shipment
    ON shipping_svc.shipment_events(shipment_id, created_at DESC);

CREATE INDEX idx_shipment_events_tenant
    ON shipping_svc.shipment_events(tenant_id);

ALTER TABLE shipping_svc.shipment_events ENABLE ROW LEVEL SECURITY;
ALTER TABLE shipping_svc.shipment_events FORCE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON shipping_svc.shipment_events
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
