-- Phase 3: on the review-only DB, the shared grants/ dir never runs.

GRANT USAGE ON SCHEMA review_svc TO review_role;
GRANT ALL ON ALL TABLES IN SCHEMA review_svc TO review_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA review_svc TO review_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA review_svc GRANT ALL ON TABLES TO review_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA review_svc GRANT ALL ON SEQUENCES TO review_role;
