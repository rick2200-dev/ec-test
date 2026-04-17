-- Phase 3: on the auth-only DB, the shared grants/ dir never runs.

GRANT USAGE ON SCHEMA auth_svc TO auth_role;
GRANT ALL ON ALL TABLES IN SCHEMA auth_svc TO auth_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA auth_svc TO auth_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth_svc GRANT ALL ON TABLES TO auth_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth_svc GRANT ALL ON SEQUENCES TO auth_role;
