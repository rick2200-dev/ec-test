DROP TABLE IF EXISTS auth_svc.rbac_audit_log;

DROP INDEX IF EXISTS auth_svc.idx_seller_users_role;
ALTER TABLE auth_svc.seller_users DROP CONSTRAINT IF EXISTS seller_users_role_check;

DROP TABLE IF EXISTS auth_svc.platform_admins;
