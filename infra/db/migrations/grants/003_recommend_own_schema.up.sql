-- Phase 2.2 (partial): recommend now owns recommend_svc.user_events.
-- Grant full privileges on the new schema and revoke the transitional
-- INSERT/UPDATE/DELETE on catalog_svc.user_events that grants/001 handed
-- out before the move. SELECT on catalog_svc.products/skus stays for the
-- remaining read-only joins until the product projection (the other half
-- of Phase 2.2) lands.

GRANT USAGE ON SCHEMA recommend_svc TO recommend_role;
GRANT ALL ON ALL TABLES IN SCHEMA recommend_svc TO recommend_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA recommend_svc TO recommend_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA recommend_svc GRANT ALL ON TABLES TO recommend_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA recommend_svc GRANT ALL ON SEQUENCES TO recommend_role;

REVOKE INSERT, UPDATE, DELETE ON catalog_svc.user_events FROM recommend_role;
