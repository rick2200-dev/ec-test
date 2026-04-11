-- Purchase history feature: snapshot seller_name on orders and product_id on
-- order_lines so buyer-facing order history remains readable even if the
-- underlying seller or SKU is later deleted, and add image_url on products so
-- live items can render a thumbnail in the order detail view.

-- orders: seller_name snapshot.
ALTER TABLE order_svc.orders
    ADD COLUMN seller_name VARCHAR(255) NOT NULL DEFAULT '';

-- Backfill existing rows from auth_svc.sellers (cross-schema; migrations run
-- as superuser so RLS is bypassed during the UPDATE).
UPDATE order_svc.orders o
   SET seller_name = COALESCE(s.name, '')
  FROM auth_svc.sellers s
 WHERE o.seller_id = s.id
   AND o.tenant_id = s.tenant_id;

-- order_lines: product_id snapshot (for detail-page enrichment via catalog gRPC).
ALTER TABLE order_svc.order_lines
    ADD COLUMN product_id UUID;

-- Backfill from catalog_svc.skus using the existing sku_id.
UPDATE order_svc.order_lines ol
   SET product_id = sk.product_id
  FROM catalog_svc.skus sk
 WHERE ol.sku_id = sk.id
   AND ol.tenant_id = sk.tenant_id;

-- For any historical rows whose sku_id no longer exists in catalog_svc.skus
-- (no FK constraint binds those tables), stamp the nil UUID as a sentinel so
-- the NOT NULL constraint below can be enforced safely. The gateway's order
-- detail enrichment treats the nil UUID as "product deleted" and returns
-- is_deleted=true for those lines, preserving buyer-visible history with
-- only the snapshotted product_name and sku_code.
UPDATE order_svc.order_lines
   SET product_id = '00000000-0000-0000-0000-000000000000'
 WHERE product_id IS NULL;

-- Enforce NOT NULL now that every row has either a real product_id or the
-- sentinel nil UUID.
ALTER TABLE order_svc.order_lines
    ALTER COLUMN product_id SET NOT NULL;

CREATE INDEX idx_order_lines_product
    ON order_svc.order_lines(tenant_id, product_id);

-- catalog: product image URL (single primary image for now; a dedicated
-- product_images table can be added later if multiple images are needed).
ALTER TABLE catalog_svc.products
    ADD COLUMN image_url VARCHAR(1024);
