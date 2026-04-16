-- Phase 3: on the subscription-only DB, the shared grants/ dir never
-- runs, so give subscription_role its schema privileges here.
-- Safe to re-run on the shared cluster too (just re-GRANTs).

GRANT USAGE ON SCHEMA subscription_svc TO subscription_role;
GRANT ALL ON ALL TABLES IN SCHEMA subscription_svc TO subscription_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA subscription_svc TO subscription_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA subscription_svc GRANT ALL ON TABLES TO subscription_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA subscription_svc GRANT ALL ON SEQUENCES TO subscription_role;
