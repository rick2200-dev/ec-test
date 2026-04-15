CREATE TABLE catalog_svc.categories (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id UUID REFERENCES catalog_svc.categories(id),
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(255) NOT NULL UNIQUE,
    sort_order INT DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE catalog_svc.products (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    seller_id UUID NOT NULL,
    name VARCHAR(500) NOT NULL,
    slug VARCHAR(500) NOT NULL,
    description TEXT,
    status VARCHAR(20) NOT NULL DEFAULT 'draft',
    attributes JSONB DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(seller_id, slug)
);

CREATE TABLE catalog_svc.skus (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES catalog_svc.products(id),
    seller_id UUID NOT NULL,
    sku_code VARCHAR(100) NOT NULL,
    price_amount BIGINT NOT NULL,
    price_currency VARCHAR(3) NOT NULL DEFAULT 'JPY',
    attributes JSONB DEFAULT '{}',
    status VARCHAR(20) NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(seller_id, sku_code)
);

CREATE TABLE catalog_svc.product_categories (
    product_id UUID NOT NULL REFERENCES catalog_svc.products(id),
    category_id UUID NOT NULL REFERENCES catalog_svc.categories(id),
    PRIMARY KEY (product_id, category_id)
);

-- Indexes
CREATE INDEX idx_products_seller ON catalog_svc.products(seller_id);
CREATE INDEX idx_products_status ON catalog_svc.products(status) WHERE status = 'active';
CREATE INDEX idx_skus_product ON catalog_svc.skus(product_id);
