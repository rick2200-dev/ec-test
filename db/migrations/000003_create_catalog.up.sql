CREATE TABLE catalog_svc.categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    parent_id UUID REFERENCES catalog_svc.categories(id),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL,
    sort_order INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, slug)
);

CREATE TABLE catalog_svc.products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    seller_id UUID NOT NULL,
    name VARCHAR(500) NOT NULL,
    slug VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    attributes JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, seller_id, slug)
);

CREATE TABLE catalog_svc.skus (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL,
    product_id UUID NOT NULL REFERENCES catalog_svc.products(id),
    seller_id UUID NOT NULL,
    sku_code VARCHAR(100) NOT NULL,
    price_amount BIGINT NOT NULL,
    price_currency VARCHAR(3) NOT NULL DEFAULT 'JPY',
    attributes JSONB DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(tenant_id, seller_id, sku_code)
);

CREATE TABLE catalog_svc.product_categories (
    product_id UUID NOT NULL REFERENCES catalog_svc.products(id),
    category_id UUID NOT NULL REFERENCES catalog_svc.categories(id),
    tenant_id UUID NOT NULL,
    PRIMARY KEY (product_id, category_id)
);

-- Indexes
CREATE INDEX idx_categories_tenant ON catalog_svc.categories(tenant_id);
CREATE INDEX idx_products_tenant ON catalog_svc.products(tenant_id);
CREATE INDEX idx_products_seller ON catalog_svc.products(tenant_id, seller_id);
CREATE INDEX idx_products_status ON catalog_svc.products(tenant_id, status) WHERE status = 'active';
CREATE INDEX idx_skus_tenant ON catalog_svc.skus(tenant_id);
CREATE INDEX idx_skus_product ON catalog_svc.skus(product_id);
CREATE INDEX idx_product_categories_tenant ON catalog_svc.product_categories(tenant_id);

-- Row-Level Security
ALTER TABLE catalog_svc.categories ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalog_svc.products ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalog_svc.skus ENABLE ROW LEVEL SECURITY;
ALTER TABLE catalog_svc.product_categories ENABLE ROW LEVEL SECURITY;

CREATE POLICY tenant_isolation ON catalog_svc.categories
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE POLICY tenant_isolation ON catalog_svc.products
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE POLICY tenant_isolation ON catalog_svc.skus
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);

CREATE POLICY tenant_isolation ON catalog_svc.product_categories
    USING (tenant_id = current_setting('app.current_tenant_id')::uuid);
