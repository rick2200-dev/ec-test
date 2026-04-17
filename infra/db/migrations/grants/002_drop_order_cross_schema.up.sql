-- Phase 1.2: order service no longer reads auth_svc / catalog_svc directly —
-- auth.BatchGetSellers (HTTP /internal/sellers/batch-get) and
-- catalog.BatchGetSKUs (gRPC) now provide those snapshots at checkout time.
-- Revoke the transitional grants that were added by grants/001.
--
-- Phase 3 splits auth + catalog to their own DBs. Bootstrap still creates
-- empty auth_svc/catalog_svc schemas, so check for actual tables.

DO $$
BEGIN
    IF to_regclass('auth_svc.sellers') IS NOT NULL THEN
        REVOKE SELECT ON auth_svc.sellers FROM order_role;
        REVOKE USAGE ON SCHEMA auth_svc FROM order_role;
    END IF;
END$$;

DO $$
BEGIN
    IF to_regclass('catalog_svc.products') IS NOT NULL THEN
        REVOKE SELECT ON catalog_svc.products, catalog_svc.skus FROM order_role;
        REVOKE USAGE ON SCHEMA catalog_svc FROM order_role;
    END IF;
END$$;
