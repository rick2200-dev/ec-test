-- Phase 3: coupon runs against its own Postgres instance (or schema),
-- so the shared bootstrap (CREATE ROLE, CREATE EXTENSION) from the
-- shared-DB bootstrap/ directory doesn't necessarily apply here.
-- Re-create just what coupon needs, idempotent so re-running against a
-- half-migrated instance is safe.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'coupon_role') THEN
        CREATE ROLE coupon_role LOGIN PASSWORD 'localdev';
    END IF;
END$$;

CREATE SCHEMA IF NOT EXISTS coupon_svc;
