DROP INDEX IF EXISTS catalog_svc.idx_products_category_id;
ALTER TABLE catalog_svc.products DROP COLUMN IF EXISTS category_id;
