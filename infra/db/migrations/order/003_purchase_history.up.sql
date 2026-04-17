-- Purchase history feature: snapshot seller_name on orders and product_id on
-- order_lines so buyer-facing order history remains readable even if the
-- underlying seller or SKU is later deleted, and add image_url on products so
-- live items can render a thumbnail in the order detail view.

-- orders: seller_name snapshot.
ALTER TABLE order_svc.orders
    ADD COLUMN seller_name VARCHAR(255) NOT NULL DEFAULT '';

-- Backfill existing rows from auth_svc.sellers. On a fresh order-only
-- DB (Phase 3 split), auth_svc doesn't exist — skip.
DO $$
BEGIN
    IF to_regclass('auth_svc.sellers') IS NOT NULL THEN
        UPDATE order_svc.orders o
           SET seller_name = COALESCE(s.name, '')
          FROM auth_svc.sellers s
         WHERE o.seller_id = s.id;
    END IF;
END$$;

-- order_lines: product_id snapshot (for detail-page enrichment via catalog gRPC).
ALTER TABLE order_svc.order_lines
    ADD COLUMN product_id UUID;

-- Backfill from catalog_svc.skus using the existing sku_id. On a fresh
-- order-only DB, catalog_svc doesn't exist — skip.
DO $$
BEGIN
    IF to_regclass('catalog_svc.skus') IS NOT NULL THEN
        UPDATE order_svc.order_lines ol
           SET product_id = sk.product_id
          FROM catalog_svc.skus sk
         WHERE ol.sku_id = sk.id;
    END IF;
END$$;

-- For any historical rows whose sku_id no longer exists (no FK constraint
-- binds those tables), stamp the nil UUID as a sentinel.
UPDATE order_svc.order_lines
   SET product_id = '00000000-0000-0000-0000-000000000000'
 WHERE product_id IS NULL;

ALTER TABLE order_svc.order_lines
    ALTER COLUMN product_id SET NOT NULL;

CREATE INDEX idx_order_lines_product
    ON order_svc.order_lines(product_id);

-- catalog: product image URL. On a split DB without catalog_svc, this
-- column is added by catalog's own migration. Guard so it doesn't fail.
DO $$
BEGIN
    IF to_regclass('catalog_svc.products') IS NOT NULL THEN
        IF NOT EXISTS (
            SELECT 1 FROM information_schema.columns
            WHERE table_schema = 'catalog_svc' AND table_name = 'products'
              AND column_name = 'image_url'
        ) THEN
            ALTER TABLE catalog_svc.products ADD COLUMN image_url VARCHAR(1024);
        END IF;
    END IF;
END$$;
