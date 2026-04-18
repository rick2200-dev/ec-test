-- Phase 3: on the coupon-only DB, the shared grants/ dir never runs.

GRANT USAGE ON SCHEMA coupon_svc TO coupon_role;
GRANT ALL ON ALL TABLES IN SCHEMA coupon_svc TO coupon_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA coupon_svc TO coupon_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA coupon_svc GRANT ALL ON TABLES TO coupon_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA coupon_svc GRANT ALL ON SEQUENCES TO coupon_role;
