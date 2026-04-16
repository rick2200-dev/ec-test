-- Phase 2.2 completion: search owns its full product projection.
-- Replaces the cross-schema reads (catalog_svc.products/skus/categories +
-- auth_svc.sellers) that Search's query layer used to join at request
-- time. Maintained by ProductSubscriber (upsert on product events, with
-- seller_name resolved via a HTTP call to auth's internal batch-get).

CREATE TABLE IF NOT EXISTS search_svc.products (
    id                      UUID PRIMARY KEY,
    seller_id               UUID NOT NULL,
    seller_name             VARCHAR(255) NOT NULL DEFAULT '',
    category_id             UUID,
    category_name           VARCHAR(255) NOT NULL DEFAULT '',
    name                    VARCHAR(500) NOT NULL,
    slug                    VARCHAR(500) NOT NULL,
    description             TEXT,
    status                  VARCHAR(20) NOT NULL,
    price_amount            BIGINT,
    price_currency          VARCHAR(3),
    search_vector           tsvector,
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_search_products_status
    ON search_svc.products (status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_search_products_seller
    ON search_svc.products (seller_id);
CREATE INDEX IF NOT EXISTS idx_search_products_category
    ON search_svc.products (category_id) WHERE category_id IS NOT NULL;
CREATE INDEX IF NOT EXISTS idx_search_products_vector
    ON search_svc.products USING GIN (search_vector);

-- Backfill from catalog so existing products are searchable immediately.
-- Seller name is left blank here; ProductSubscriber fills it on the next
-- product event (or a one-shot admin backfill can call auth's batch-get).
INSERT INTO search_svc.products
       (id, seller_id, seller_name, category_id, category_name, name, slug, description, status, price_amount, price_currency, search_vector)
SELECT p.id, p.seller_id, '' AS seller_name,
       p.category_id, COALESCE(c.name, '') AS category_name,
       p.name, p.slug, p.description, p.status,
       pr.price_amount, pr.price_currency,
       setweight(to_tsvector('simple', COALESCE(p.name, '')), 'A') ||
       setweight(to_tsvector('simple', COALESCE(p.description, '')), 'B')
FROM catalog_svc.products p
LEFT JOIN catalog_svc.categories c ON c.id = p.category_id
LEFT JOIN LATERAL (
    SELECT price_amount, price_currency
    FROM catalog_svc.skus
    WHERE product_id = p.id
    ORDER BY price_amount ASC
    LIMIT 1
) pr ON TRUE
ON CONFLICT (id) DO NOTHING;
