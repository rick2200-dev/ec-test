-- Phase 3: on the inquiry-only DB, the shared grants/ dir never runs.

GRANT USAGE ON SCHEMA inquiry_svc TO inquiry_role;
GRANT ALL ON ALL TABLES IN SCHEMA inquiry_svc TO inquiry_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA inquiry_svc TO inquiry_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA inquiry_svc GRANT ALL ON TABLES TO inquiry_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA inquiry_svc GRANT ALL ON SEQUENCES TO inquiry_role;
