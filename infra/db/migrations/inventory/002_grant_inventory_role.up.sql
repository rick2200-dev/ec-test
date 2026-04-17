-- Phase 3: on the inventory-only DB, the shared grants/ dir never runs,
-- so give inventory_role its schema privileges here.

GRANT USAGE ON SCHEMA inventory_svc TO inventory_role;
GRANT ALL ON ALL TABLES IN SCHEMA inventory_svc TO inventory_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA inventory_svc TO inventory_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA inventory_svc GRANT ALL ON TABLES TO inventory_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA inventory_svc GRANT ALL ON SEQUENCES TO inventory_role;
