CREATE TABLE inventory_svc.inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    sku_id UUID NOT NULL,
    seller_id UUID NOT NULL,
    quantity_available INT NOT NULL DEFAULT 0,
    quantity_reserved INT NOT NULL DEFAULT 0,
    low_stock_threshold INT DEFAULT 10,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, sku_id)
);

CREATE TABLE inventory_svc.stock_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    sku_id UUID NOT NULL,
    movement_type VARCHAR(20) NOT NULL,
    quantity INT NOT NULL,
    reference_type VARCHAR(50),
    reference_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_inventory_tenant ON inventory_svc.inventory(tenant_id);
CREATE INDEX idx_inventory_sku ON inventory_svc.inventory(tenant_id, sku_id);
CREATE INDEX idx_inventory_seller ON inventory_svc.inventory(tenant_id, seller_id);
CREATE INDEX idx_stock_movements_tenant ON inventory_svc.stock_movements(tenant_id);
CREATE INDEX idx_stock_movements_sku ON inventory_svc.stock_movements(tenant_id, sku_id);

-- Row-Level Security
ALTER TABLE inventory_svc.inventory ENABLE ROW LEVEL SECURITY;
ALTER TABLE inventory_svc.stock_movements ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON inventory_svc.inventory
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE POLICY tenant_isolation ON inventory_svc.stock_movements
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
