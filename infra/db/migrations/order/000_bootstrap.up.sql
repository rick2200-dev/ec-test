-- Phase 3: order runs against its own Postgres instance.

CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'order_role') THEN
        CREATE ROLE order_role LOGIN PASSWORD 'localdev';
    END IF;
END$$;
