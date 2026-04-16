-- Phase 3: subscription runs against its own Postgres instance, so the
-- shared bootstrap (CREATE ROLE, CREATE EXTENSION) from the shared-DB
-- bootstrap/ directory doesn't apply here. Re-create just what
-- subscription needs in its own DB, idempotent so re-running against a
-- half-migrated instance is safe.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'subscription_role') THEN
        CREATE ROLE subscription_role LOGIN PASSWORD 'localdev';
    END IF;
END$$;
