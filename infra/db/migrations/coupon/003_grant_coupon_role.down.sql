-- Revoke in the reverse direction. Safe to rerun.
ALTER DEFAULT PRIVILEGES IN SCHEMA coupon_svc REVOKE ALL ON SEQUENCES FROM coupon_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA coupon_svc REVOKE ALL ON TABLES FROM coupon_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA coupon_svc FROM coupon_role;
REVOKE ALL ON ALL TABLES IN SCHEMA coupon_svc FROM coupon_role;
REVOKE USAGE ON SCHEMA coupon_svc FROM coupon_role;
