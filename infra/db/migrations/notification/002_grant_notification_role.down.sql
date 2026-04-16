ALTER DEFAULT PRIVILEGES IN SCHEMA notification_svc REVOKE ALL ON SEQUENCES FROM notification_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA notification_svc REVOKE ALL ON TABLES FROM notification_role;
REVOKE ALL ON ALL SEQUENCES IN SCHEMA notification_svc FROM notification_role;
REVOKE ALL ON ALL TABLES IN SCHEMA notification_svc FROM notification_role;
REVOKE USAGE ON SCHEMA notification_svc FROM notification_role;
