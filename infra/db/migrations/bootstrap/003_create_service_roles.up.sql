-- Phase 1.1: create one LOGIN role per service. Grants are applied later by
-- the grants/ migration dir after every service schema has been created.
--
-- The migration superuser (ecmarket) continues to own all objects and runs
-- migrations; these roles are only used by running services at runtime.

DO $$
DECLARE
  svc TEXT;
  role_name TEXT;
BEGIN
  FOREACH svc IN ARRAY ARRAY[
    'auth','catalog','inventory','order','subscription',
    'inquiry','review','shipping','notification','search','recommend'
  ] LOOP
    role_name := svc || '_role';
    IF NOT EXISTS (SELECT 1 FROM pg_roles WHERE rolname = role_name) THEN
      EXECUTE format('CREATE ROLE %I LOGIN PASSWORD %L', role_name, 'localdev');
    END IF;
  END LOOP;
END$$;
