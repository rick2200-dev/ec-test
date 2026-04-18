CREATE EXTENSION IF NOT EXISTS pgcrypto;

DO $$
BEGIN
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = 'loyalty_role') THEN
        CREATE ROLE loyalty_role LOGIN PASSWORD 'localdev';
    END IF;
END$$;

CREATE SCHEMA IF NOT EXISTS loyalty_svc;
