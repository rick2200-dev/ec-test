-- Phase 3: catalog runs against its own Postgres instance.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'catalog_role') THEN
        CREATE ROLE catalog_role LOGIN PASSWORD 'localdev';
    END IF;
END$$;
