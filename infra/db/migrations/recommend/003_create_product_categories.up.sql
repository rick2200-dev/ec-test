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

-- Backfill from catalog. On a fresh recommend-only DB (Phase 3 split),
-- catalog_svc doesn't exist — skip; product events will populate this.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'catalog_svc') THEN
        INSERT INTO recommend_svc.product_categories (product_id, category_id)
        SELECT id, category_id
        FROM catalog_svc.products
        WHERE category_id IS NOT NULL
        ON CONFLICT (product_id, category_id) DO NOTHING;
    END IF;
END$$;
