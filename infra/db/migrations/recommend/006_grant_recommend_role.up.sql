-- Phase 3: on the recommend-only DB, the shared grants/ dir never runs.

GRANT USAGE ON SCHEMA recommend_svc TO recommend_role;
GRANT ALL ON ALL TABLES IN SCHEMA recommend_svc TO recommend_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA recommend_svc TO recommend_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA recommend_svc GRANT ALL ON TABLES TO recommend_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA recommend_svc GRANT ALL ON SEQUENCES TO recommend_role;
