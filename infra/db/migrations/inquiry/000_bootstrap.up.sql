-- Phase 3: inquiry runs against its own Postgres instance.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'inquiry_role') THEN
        CREATE ROLE inquiry_role LOGIN PASSWORD 'localdev';
    END IF;
END$$;
