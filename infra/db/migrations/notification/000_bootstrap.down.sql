-- Reverse of 000_bootstrap.up.sql. DROP ROLE ... IF EXISTS is safe even
-- on a shared cluster where the role also backs other DBs, because
-- Postgres blocks the drop when the role owns objects — the up-script
-- grants on notification_svc, which gets dropped before this migration
-- rolls back.
DROP ROLE IF EXISTS notification_role;
