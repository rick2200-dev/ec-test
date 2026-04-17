-- Phase 3: review runs against its own Postgres instance.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'review_role') THEN
        CREATE ROLE review_role LOGIN PASSWORD 'localdev';
    END IF;
END$$;
