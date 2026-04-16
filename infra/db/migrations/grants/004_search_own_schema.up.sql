-- Phase 2.2: search now owns search_svc (currently seller_plan_boost,
-- products projection to follow). Give search_role full privileges on
-- the schema so it can upsert the projection rows it maintains from
-- subscription + product events.

GRANT USAGE ON SCHEMA search_svc TO search_role;
GRANT ALL ON ALL TABLES IN SCHEMA search_svc TO search_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA search_svc TO search_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA search_svc GRANT ALL ON TABLES TO search_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA search_svc GRANT ALL ON SEQUENCES TO search_role;
