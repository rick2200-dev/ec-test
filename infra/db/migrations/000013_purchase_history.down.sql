ALTER TABLE catalog_svc.products DROP COLUMN IF EXISTS image_url;

DROP INDEX IF EXISTS order_svc.idx_order_lines_product;
ALTER TABLE order_svc.order_lines DROP COLUMN IF EXISTS product_id;

ALTER TABLE order_svc.orders DROP COLUMN IF EXISTS seller_name;
