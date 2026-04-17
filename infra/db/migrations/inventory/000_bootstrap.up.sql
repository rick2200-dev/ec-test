-- Phase 3: inventory runs against its own Postgres instance, so the
-- shared bootstrap doesn't apply here.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'inventory_role') THEN
        CREATE ROLE inventory_role LOGIN PASSWORD 'localdev';
    END IF;
END$$;
