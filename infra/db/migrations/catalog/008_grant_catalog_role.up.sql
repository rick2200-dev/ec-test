-- Phase 3: on the catalog-only DB, the shared grants/ dir never runs.

GRANT USAGE ON SCHEMA catalog_svc TO catalog_role;
GRANT ALL ON ALL TABLES IN SCHEMA catalog_svc TO catalog_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA catalog_svc TO catalog_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA catalog_svc GRANT ALL ON TABLES TO catalog_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA catalog_svc GRANT ALL ON SEQUENCES TO catalog_role;
