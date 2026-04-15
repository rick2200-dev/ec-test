-- Restore the transitional grants (inverse of the up).
GRANT USAGE ON SCHEMA auth_svc TO order_role;
GRANT SELECT ON auth_svc.sellers TO order_role;

GRANT USAGE ON SCHEMA catalog_svc TO order_role;
GRANT SELECT ON catalog_svc.products, catalog_svc.skus TO order_role;
