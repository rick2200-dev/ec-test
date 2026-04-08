DROP TRIGGER IF EXISTS trg_products_search_vector ON catalog_svc.products;
DROP FUNCTION IF EXISTS catalog_svc.update_product_search_vector();
DROP INDEX IF EXISTS catalog_svc.idx_products_search;
ALTER TABLE catalog_svc.products DROP COLUMN IF EXISTS search_vector;
