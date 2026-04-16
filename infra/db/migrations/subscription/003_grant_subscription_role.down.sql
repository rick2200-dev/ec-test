ALTER DEFAULT PRIVILEGES IN SCHEMA subscription_svc REVOKE ALL ON SEQUENCES FROM subscription_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA subscription_svc REVOKE ALL ON TABLES FROM subscription_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA subscription_svc FROM subscription_role;
REVOKE ALL ON ALL TABLES IN SCHEMA subscription_svc FROM subscription_role;
REVOKE USAGE ON SCHEMA subscription_svc FROM subscription_role;
