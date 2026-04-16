-- Phase 3 pilot: notification runs against its own Postgres instance, so
-- the shared bootstrap (CREATE ROLE, CREATE EXTENSION) from the shared-DB
-- bootstrap/ directory doesn't apply here. Re-create just what
-- notification needs in its own DB, idempotent so re-running against a
-- half-migrated instance is safe.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'notification_role') THEN
        CREATE ROLE notification_role LOGIN PASSWORD 'localdev';
    END IF;
END$$;
