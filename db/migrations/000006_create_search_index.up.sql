-- Add full-text search vector column to products for local search fallback
ALTER TABLE catalog_svc.products ADD COLUMN IF NOT EXISTS search_vector tsvector;

-- Create GIN index for full-text search
CREATE INDEX IF NOT EXISTS idx_products_search ON catalog_svc.products USING GIN(search_vector);

-- Create function to update search vector
CREATE OR REPLACE FUNCTION catalog_svc.update_product_search_vector()
RETURNS TRIGGER AS $$
BEGIN
    NEW.search_vector :=
        setweight(to_tsvector('simple', COALESCE(NEW.name, '')), 'A') ||
        setweight(to_tsvector('simple', COALESCE(NEW.description, '')), 'B');
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Create trigger
DROP TRIGGER IF EXISTS trg_products_search_vector ON catalog_svc.products;
CREATE TRIGGER trg_products_search_vector
    BEFORE INSERT OR UPDATE OF name, description ON catalog_svc.products
    FOR EACH ROW
    EXECUTE FUNCTION catalog_svc.update_product_search_vector();

-- Backfill existing products
UPDATE catalog_svc.products SET search_vector =
    setweight(to_tsvector('simple', COALESCE(name, '')), 'A') ||
    setweight(to_tsvector('simple', COALESCE(description, '')), 'B');
