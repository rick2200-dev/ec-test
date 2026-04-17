-- Guard: catalog_svc.products may not exist on split DBs (Phase 3).
-- Bootstrap creates the empty schema, so check for the table itself.
DO $$
BEGIN
    IF to_regclass('catalog_svc.products') IS NOT NULL THEN
        ALTER TABLE catalog_svc.products DROP COLUMN IF EXISTS image_url;
    END IF;
END$$;

DROP INDEX IF EXISTS order_svc.idx_order_lines_product;
ALTER TABLE order_svc.order_lines DROP COLUMN IF EXISTS product_id;

ALTER TABLE order_svc.orders DROP COLUMN IF EXISTS seller_name;
