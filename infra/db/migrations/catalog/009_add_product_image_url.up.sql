-- Phase 3 fix: the image_url column was historically added by order/003.
-- On a fresh catalog-only DB, order's migrations don't run, so the column
-- must be owned by catalog's migration chain.
ALTER TABLE catalog_svc.products ADD COLUMN IF NOT EXISTS image_url VARCHAR(1024);
