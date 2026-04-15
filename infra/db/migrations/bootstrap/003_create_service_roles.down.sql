-- Dropping roles requires reassigning or dropping owned objects first. In
-- practice we recreate the DB for local dev, so the down is a best-effort
-- no-op guarded by IF EXISTS; prod should migrate via a dedicated DBA path.
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
    EXECUTE format('DROP ROLE IF EXISTS %I', role_name);
  END LOOP;
END$$;
