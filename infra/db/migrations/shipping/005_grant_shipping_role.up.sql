-- Phase 3: on the shipping-only DB, the shared grants/ dir never runs,
-- so give shipping_role its schema privileges here. Safe to re-run on
-- the shared cluster (just re-GRANTs).

GRANT USAGE ON SCHEMA shipping_svc TO shipping_role;
GRANT ALL ON ALL TABLES IN SCHEMA shipping_svc TO shipping_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA shipping_svc TO shipping_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA shipping_svc GRANT ALL ON TABLES TO shipping_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA shipping_svc GRANT ALL ON SEQUENCES TO shipping_role;
