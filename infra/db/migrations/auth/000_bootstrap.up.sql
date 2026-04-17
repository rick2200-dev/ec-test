-- Phase 3: auth runs against its own Postgres instance.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'auth_role') THEN
        CREATE ROLE auth_role LOGIN PASSWORD 'localdev';
    END IF;
END$$;
