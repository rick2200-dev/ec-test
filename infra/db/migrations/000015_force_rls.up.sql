-- Harden tenant isolation by forcing row-level security on tenant tables.
--
-- Background: in the current single-role deployment the application and
-- the migrator both connect as `ecmarket`, which happens to own every
-- table with a `tenant_isolation` policy. Without FORCE ROW LEVEL
-- SECURITY, the owner silently bypasses RLS policies and the isolation we
-- think we have is effectively disabled — any query missing a
-- `tenant_id = ...` clause would return rows from every tenant.
--
-- Scope of this migration: we apply FORCE ROW LEVEL SECURITY only to
-- tables that are *always* accessed through `database.TenantTx` (which
-- sets `app.current_tenant_id` via `SET LOCAL`). Tables that still have
-- legitimate direct-query (non-TenantTx) code paths are intentionally
-- excluded below — applying FORCE RLS to them would break working code.
-- Those exclusions are tracked and should be removed as the call sites
-- are refactored or replaced with `SECURITY DEFINER` helper functions.
--
-- Excluded (with reason):
--   * auth_svc.sellers                — joined directly from search service
--   * auth_svc.seller_api_tokens      — gateway GetByLookup is a tenant
--                                       bootstrap lookup; cannot know the
--                                       tenant_id before the query
--   * catalog_svc.categories          — joined directly from search service
--   * catalog_svc.products            — queried directly from search and
--                                       recommend engines
--   * catalog_svc.skus                — joined directly from search service
--   * catalog_svc.product_categories  — queried directly from recommend
--   * catalog_svc.user_events         — queried directly from recommend
--   * order_svc.orders                — Stripe webhook handler looks up
--                                       orders cross-tenant by
--                                       stripe_payment_intent_id before a
--                                       tenant context exists

ALTER TABLE auth_svc.seller_users           FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.platform_admins        FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.rbac_audit_log         FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.subscription_plans     FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.seller_subscriptions   FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.buyer_plans            FORCE ROW LEVEL SECURITY;
ALTER TABLE auth_svc.buyer_subscriptions    FORCE ROW LEVEL SECURITY;

ALTER TABLE inventory_svc.inventory         FORCE ROW LEVEL SECURITY;
ALTER TABLE inventory_svc.stock_movements   FORCE ROW LEVEL SECURITY;

ALTER TABLE order_svc.order_lines           FORCE ROW LEVEL SECURITY;
ALTER TABLE order_svc.commission_rules      FORCE ROW LEVEL SECURITY;
ALTER TABLE order_svc.payouts               FORCE ROW LEVEL SECURITY;

ALTER TABLE inquiry_svc.inquiries           FORCE ROW LEVEL SECURITY;
ALTER TABLE inquiry_svc.inquiry_messages    FORCE ROW LEVEL SECURITY;
