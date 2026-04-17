-- Restore the transitional grants (inverse of the up).
-- Guard for split DBs where auth_svc/catalog_svc tables don't exist.

DO $$
BEGIN
    IF to_regclass('auth_svc.sellers') IS NOT NULL THEN
        GRANT USAGE ON SCHEMA auth_svc TO order_role;
        GRANT SELECT ON auth_svc.sellers TO order_role;
    END IF;
END$$;

DO $$
BEGIN
    IF to_regclass('catalog_svc.products') IS NOT NULL THEN
        GRANT USAGE ON SCHEMA catalog_svc TO order_role;
        GRANT SELECT ON catalog_svc.products, catalog_svc.skus TO order_role;
    END IF;
END$$;
