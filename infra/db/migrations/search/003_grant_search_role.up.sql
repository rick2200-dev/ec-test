-- Phase 3: on the search-only DB, the shared grants/ dir never runs.

GRANT USAGE ON SCHEMA search_svc TO search_role;
GRANT ALL ON ALL TABLES IN SCHEMA search_svc TO search_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA search_svc TO search_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA search_svc GRANT ALL ON TABLES TO search_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA search_svc GRANT ALL ON SEQUENCES TO search_role;
