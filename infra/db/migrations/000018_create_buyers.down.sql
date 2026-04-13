DROP POLICY IF EXISTS tenant_isolation ON auth_svc.buyers;
DROP INDEX IF EXISTS auth_svc.idx_buyers_tenant;
DROP TABLE IF EXISTS auth_svc.buyers;
