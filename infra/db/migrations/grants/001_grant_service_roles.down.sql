-- Revoke all grants by simply dropping the privileges. ALTER DEFAULT
-- PRIVILEGES needs symmetric REVOKE to clean up.

REVOKE ALL ON ALL TABLES IN SCHEMA auth_svc FROM auth_role, order_role, catalog_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA auth_svc FROM auth_role;
REVOKE USAGE ON SCHEMA auth_svc FROM auth_role, order_role, catalog_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth_svc REVOKE ALL ON TABLES FROM auth_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA auth_svc REVOKE ALL ON SEQUENCES FROM auth_role;

REVOKE ALL ON ALL TABLES IN SCHEMA catalog_svc FROM catalog_role, search_role, recommend_role, order_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA catalog_svc FROM catalog_role;
REVOKE USAGE ON SCHEMA catalog_svc FROM catalog_role, search_role, recommend_role, order_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA catalog_svc REVOKE ALL ON TABLES FROM catalog_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA catalog_svc REVOKE ALL ON SEQUENCES FROM catalog_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA catalog_svc REVOKE SELECT ON TABLES FROM search_role, recommend_role;

REVOKE ALL ON ALL TABLES IN SCHEMA inventory_svc FROM inventory_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA inventory_svc FROM inventory_role;
REVOKE USAGE ON SCHEMA inventory_svc FROM inventory_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA inventory_svc REVOKE ALL ON TABLES FROM inventory_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA inventory_svc REVOKE ALL ON SEQUENCES FROM inventory_role;

REVOKE ALL ON ALL TABLES IN SCHEMA order_svc FROM order_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA order_svc FROM order_role;
REVOKE USAGE ON SCHEMA order_svc FROM order_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA order_svc REVOKE ALL ON TABLES FROM order_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA order_svc REVOKE ALL ON SEQUENCES FROM order_role;

REVOKE ALL ON ALL TABLES IN SCHEMA subscription_svc FROM subscription_role, catalog_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA subscription_svc FROM subscription_role;
REVOKE USAGE ON SCHEMA subscription_svc FROM subscription_role, catalog_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA subscription_svc REVOKE ALL ON TABLES FROM subscription_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA subscription_svc REVOKE ALL ON SEQUENCES FROM subscription_role;

REVOKE ALL ON ALL TABLES IN SCHEMA inquiry_svc FROM inquiry_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA inquiry_svc FROM inquiry_role;
REVOKE USAGE ON SCHEMA inquiry_svc FROM inquiry_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA inquiry_svc REVOKE ALL ON TABLES FROM inquiry_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA inquiry_svc REVOKE ALL ON SEQUENCES FROM inquiry_role;

REVOKE ALL ON ALL TABLES IN SCHEMA review_svc FROM review_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA review_svc FROM review_role;
REVOKE USAGE ON SCHEMA review_svc FROM review_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA review_svc REVOKE ALL ON TABLES FROM review_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA review_svc REVOKE ALL ON SEQUENCES FROM review_role;

REVOKE ALL ON ALL TABLES IN SCHEMA shipping_svc FROM shipping_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA shipping_svc FROM shipping_role;
REVOKE USAGE ON SCHEMA shipping_svc FROM shipping_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA shipping_svc REVOKE ALL ON TABLES FROM shipping_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA shipping_svc REVOKE ALL ON SEQUENCES FROM shipping_role;

REVOKE ALL ON ALL TABLES IN SCHEMA notification_svc FROM notification_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA notification_svc FROM notification_role;
REVOKE USAGE ON SCHEMA notification_svc FROM notification_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA notification_svc REVOKE ALL ON TABLES FROM notification_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA notification_svc REVOKE ALL ON SEQUENCES FROM notification_role;
