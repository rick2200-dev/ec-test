CREATE TABLE order_svc.orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    seller_id UUID NOT NULL,
    buyer_auth0_id VARCHAR(255) NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'pending',
    subtotal_amount BIGINT NOT NULL,
    commission_amount BIGINT NOT NULL,
    total_amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'JPY',
    shipping_address JSONB,
    stripe_payment_intent_id VARCHAR(255),
    paid_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE order_svc.order_lines (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    order_id UUID NOT NULL REFERENCES order_svc.orders(id),
    sku_id UUID NOT NULL,
    product_name VARCHAR(500) NOT NULL,
    sku_code VARCHAR(100) NOT NULL,
    quantity INT NOT NULL,
    unit_price BIGINT NOT NULL,
    line_total BIGINT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE order_svc.commission_rules (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    seller_id UUID,
    category_id UUID,
    rate_bps INT NOT NULL,
    priority INT NOT NULL DEFAULT 0,
    valid_from TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    valid_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE order_svc.payouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    seller_id UUID NOT NULL,
    order_id UUID NOT NULL REFERENCES order_svc.orders(id),
    amount BIGINT NOT NULL,
    currency VARCHAR(3) NOT NULL DEFAULT 'JPY',
    stripe_transfer_id VARCHAR(255),
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMPTZ
);

-- Indexes
CREATE INDEX idx_orders_tenant ON order_svc.orders(tenant_id);
CREATE INDEX idx_orders_buyer ON order_svc.orders(tenant_id, buyer_auth0_id);
CREATE INDEX idx_orders_seller ON order_svc.orders(tenant_id, seller_id);
CREATE INDEX idx_orders_status ON order_svc.orders(tenant_id, status);
CREATE INDEX idx_order_lines_order ON order_svc.order_lines(order_id);
CREATE INDEX idx_order_lines_tenant ON order_svc.order_lines(tenant_id);
CREATE INDEX idx_commission_rules_tenant ON order_svc.commission_rules(tenant_id);
CREATE INDEX idx_payouts_tenant ON order_svc.payouts(tenant_id);
CREATE INDEX idx_payouts_seller ON order_svc.payouts(tenant_id, seller_id);
CREATE INDEX idx_payouts_order ON order_svc.payouts(order_id);

-- Row-Level Security
ALTER TABLE order_svc.orders ENABLE ROW LEVEL SECURITY;
ALTER TABLE order_svc.order_lines ENABLE ROW LEVEL SECURITY;
ALTER TABLE order_svc.commission_rules ENABLE ROW LEVEL SECURITY;
ALTER TABLE order_svc.payouts ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON order_svc.orders
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE POLICY tenant_isolation ON order_svc.order_lines
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE POLICY tenant_isolation ON order_svc.commission_rules
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE POLICY tenant_isolation ON order_svc.payouts
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
