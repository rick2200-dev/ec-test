-- Phase 3: on the order-only DB, the shared grants/ dir never runs.

GRANT USAGE ON SCHEMA order_svc TO order_role;
GRANT ALL ON ALL TABLES IN SCHEMA order_svc TO order_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA order_svc TO order_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA order_svc GRANT ALL ON TABLES TO order_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA order_svc GRANT ALL ON SEQUENCES TO order_role;
