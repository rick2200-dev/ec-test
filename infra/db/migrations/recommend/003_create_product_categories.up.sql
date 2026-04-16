-- Phase 2.2 (partial): local projection of the product ↔ category mapping.
-- Populated by recommend's ProductSubscriber from catalog's ProductCreated /
-- ProductUpdated events so the recommendation queries stop reading
-- catalog_svc.product_categories directly.
--
-- Composite primary key matches catalog's own shape. No foreign keys to
-- catalog_svc — the whole point of the projection is to decouple.

CREATE TABLE IF NOT EXISTS recommend_svc.product_categories (
    product_id  UUID NOT NULL,
    category_id UUID NOT NULL,
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (product_id, category_id)
);

CREATE INDEX IF NOT EXISTS idx_recommend_product_categories_category
    ON recommend_svc.product_categories (category_id);

-- Backfill: copy the current (product_id, category_id) mapping out of
-- catalog_svc.products so existing products are searchable through
-- similar / personalized immediately after deploy, without waiting for
-- each product to emit its next product.updated event. Catalog stores
-- the mapping inline on products.category_id; future many-to-many work
-- would widen this SELECT.
INSERT INTO recommend_svc.product_categories (product_id, category_id)
SELECT id, category_id
FROM catalog_svc.products
WHERE category_id IS NOT NULL
ON CONFLICT (product_id, category_id) DO NOTHING;
