CREATE SCHEMA IF NOT EXISTS inventory_svc;

CREATE TABLE inventory_svc.inventory (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku_id UUID NOT NULL,
    seller_id UUID NOT NULL,
    quantity_available INT NOT NULL DEFAULT 0,
    quantity_reserved INT NOT NULL DEFAULT 0,
    low_stock_threshold INT DEFAULT 10,
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(sku_id)
);

CREATE TABLE inventory_svc.stock_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sku_id UUID NOT NULL,
    movement_type VARCHAR(20) NOT NULL,
    quantity INT NOT NULL,
    reference_type VARCHAR(50),
    reference_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Indexes
CREATE INDEX idx_inventory_sku ON inventory_svc.inventory(sku_id);
CREATE INDEX idx_inventory_seller ON inventory_svc.inventory(seller_id);
CREATE INDEX idx_stock_movements_sku ON inventory_svc.stock_movements(sku_id);
