-- Phase 1.2: order service no longer reads auth_svc / catalog_svc directly —
-- auth.BatchGetSellers (HTTP /internal/sellers/batch-get) and
-- catalog.BatchGetSKUs (gRPC) now provide those snapshots at checkout time.
-- Revoke the transitional grants that were added by grants/001.

REVOKE SELECT ON auth_svc.sellers FROM order_role;
REVOKE USAGE ON SCHEMA auth_svc FROM order_role;

REVOKE SELECT ON catalog_svc.products, catalog_svc.skus FROM order_role;
REVOKE USAGE ON SCHEMA catalog_svc FROM order_role;
