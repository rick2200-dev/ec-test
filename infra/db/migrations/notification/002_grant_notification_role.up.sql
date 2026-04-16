-- Phase 3 pilot: on the notification-only DB, the shared grants/ dir
-- never runs, so give notification_role its schema privileges here.
-- Safe to re-run on the shared cluster too (just re-GRANTs).

GRANT USAGE ON SCHEMA notification_svc TO notification_role;
GRANT ALL ON ALL TABLES IN SCHEMA notification_svc TO notification_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA notification_svc TO notification_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA notification_svc GRANT ALL ON TABLES TO notification_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA notification_svc GRANT ALL ON SEQUENCES TO notification_role;
