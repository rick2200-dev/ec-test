-- Phase 3: search runs against its own Postgres instance.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'search_role') THEN
        CREATE ROLE search_role LOGIN PASSWORD 'localdev';
    END IF;
END$$;
