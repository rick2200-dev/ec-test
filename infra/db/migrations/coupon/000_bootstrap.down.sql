DROP SCHEMA IF EXISTS coupon_svc CASCADE;
-- Intentionally keep the coupon_role around: dropping roles that are
-- still owners of objects in other databases would cascade-fail.
