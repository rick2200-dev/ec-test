-- Phase 2.2: local product projection. Populated by recommend's
-- ProductSubscriber from catalog's ProductCreated / ProductUpdated
-- events so the recommendation queries stop reading catalog_svc.products
-- and catalog_svc.skus at request time.
--
-- cheapest_price_amount / cheapest_price_currency are denormalized from
-- catalog_svc.skus; catalog republishes product.updated whenever a SKU
-- mutation might change that snapshot.

CREATE TABLE IF NOT EXISTS recommend_svc.products (
    id                      UUID PRIMARY KEY,
    seller_id               UUID NOT NULL,
    name                    VARCHAR(500) NOT NULL,
    slug                    VARCHAR(500) NOT NULL,
    description             TEXT,
    status                  VARCHAR(20) NOT NULL,
    cheapest_price_amount   BIGINT,
    cheapest_price_currency VARCHAR(3),
    updated_at              TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_recommend_products_status   ON recommend_svc.products (status) WHERE status = 'active';
CREATE INDEX IF NOT EXISTS idx_recommend_products_seller   ON recommend_svc.products (seller_id);

-- Backfill from catalog. On a fresh recommend-only DB (Phase 3 split),
-- catalog_svc doesn't exist — skip; product events will populate this.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'catalog_svc') THEN
        INSERT INTO recommend_svc.products (id, seller_id, name, slug, description, status, cheapest_price_amount, cheapest_price_currency)
        SELECT p.id, p.seller_id, p.name, p.slug, p.description, p.status,
               pr.price_amount, pr.price_currency
        FROM catalog_svc.products p
        LEFT JOIN LATERAL (
            SELECT price_amount, price_currency
            FROM catalog_svc.skus
            WHERE product_id = p.id
            ORDER BY price_amount ASC
            LIMIT 1
        ) pr ON TRUE
        ON CONFLICT (id) DO NOTHING;
    END IF;
END$$;
