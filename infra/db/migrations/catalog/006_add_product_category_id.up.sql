-- catalog_svc.products.category_id — the column catalog's Go repository
-- has always referenced but that migration 001 forgot to create. A
-- single nullable FK matches the existing domain.Product shape
-- (CategoryID *uuid.UUID). The catalog_svc.product_categories join table
-- created in 001 stays untouched for backwards compatibility; it is
-- currently unused but removing it is out of scope here.
ALTER TABLE catalog_svc.products
    ADD COLUMN IF NOT EXISTS category_id UUID REFERENCES catalog_svc.categories(id);

CREATE INDEX IF NOT EXISTS idx_products_category_id
    ON catalog_svc.products (category_id)
    WHERE category_id IS NOT NULL;
