-- Phase 3: recommend runs against its own Postgres instance.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'recommend_role') THEN
        CREATE ROLE recommend_role LOGIN PASSWORD 'localdev';
    END IF;
END$$;
