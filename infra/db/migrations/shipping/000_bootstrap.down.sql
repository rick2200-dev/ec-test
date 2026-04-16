-- Reverse of 000_bootstrap.up.sql. DROP ROLE ... IF EXISTS is safe on
-- shared clusters: Postgres blocks the drop if the role still owns
-- objects, and the up-side grants on shipping_svc get dropped before
-- this migration rolls back.
DROP ROLE IF EXISTS shipping_role;
