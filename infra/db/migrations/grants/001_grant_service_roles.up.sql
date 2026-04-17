-- Phase 1.1: grant each service role strict privileges on its own schema,
-- plus a small explicitly-documented set of transitional cross-schema reads
-- that Phase 1.2 will remove once legacy direct reads become gRPC calls.
--
-- This migration runs LAST (migrate.sh iterates SERVICES with "grants" at the
-- end) so every schema created by any per-service migration already exists.
--
-- All GRANTs are idempotent; re-running is safe. ALTER DEFAULT PRIVILEGES
-- ensures tables created by future migrations inherit the right grants as
-- long as the migration runner keeps using the ecmarket superuser.

-- ─── Owned-schema grants ──────────────────────────────────────

-- auth lives on its own Postgres instance (Phase 3).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'auth_svc') THEN
        GRANT USAGE ON SCHEMA auth_svc TO auth_role;
        GRANT ALL ON ALL TABLES IN SCHEMA auth_svc TO auth_role;
        GRANT ALL ON ALL SEQUENCES IN SCHEMA auth_svc TO auth_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA auth_svc GRANT ALL ON TABLES TO auth_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA auth_svc GRANT ALL ON SEQUENCES TO auth_role;
    END IF;
END$$;

-- catalog lives on its own Postgres instance (Phase 3).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'catalog_svc') THEN
        GRANT USAGE ON SCHEMA catalog_svc TO catalog_role;
        GRANT ALL ON ALL TABLES IN SCHEMA catalog_svc TO catalog_role;
        GRANT ALL ON ALL SEQUENCES IN SCHEMA catalog_svc TO catalog_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA catalog_svc GRANT ALL ON TABLES TO catalog_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA catalog_svc GRANT ALL ON SEQUENCES TO catalog_role;
    END IF;
END$$;

-- inventory lives on its own Postgres instance (Phase 3). Guard for
-- fresh shared-cluster DBs that no longer have the schema.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'inventory_svc') THEN
        GRANT USAGE ON SCHEMA inventory_svc TO inventory_role;
        GRANT ALL ON ALL TABLES IN SCHEMA inventory_svc TO inventory_role;
        GRANT ALL ON ALL SEQUENCES IN SCHEMA inventory_svc TO inventory_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA inventory_svc GRANT ALL ON TABLES TO inventory_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA inventory_svc GRANT ALL ON SEQUENCES TO inventory_role;
    END IF;
END$$;

-- order (note: "order" is a reserved word so the role uses an underscore suffix)
GRANT USAGE ON SCHEMA order_svc TO order_role;
GRANT ALL ON ALL TABLES IN SCHEMA order_svc TO order_role;
GRANT ALL ON ALL SEQUENCES IN SCHEMA order_svc TO order_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA order_svc GRANT ALL ON TABLES TO order_role;
ALTER DEFAULT PRIVILEGES IN SCHEMA order_svc GRANT ALL ON SEQUENCES TO order_role;

-- subscription lives on its own Postgres instance (Phase 3), so on a
-- fresh shared-cluster DB the subscription_svc schema is not created
-- here. The grants that used to live in this block moved to the
-- subscription migration dir (subscription/003_grant_subscription_role)
-- so they run on the split DB instead. Kept as a guarded DO block for
-- existing shared-cluster DBs that still have the schema lingering from
-- the pre-split layout.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'subscription_svc') THEN
        GRANT USAGE ON SCHEMA subscription_svc TO subscription_role;
        GRANT ALL ON ALL TABLES IN SCHEMA subscription_svc TO subscription_role;
        GRANT ALL ON ALL SEQUENCES IN SCHEMA subscription_svc TO subscription_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA subscription_svc GRANT ALL ON TABLES TO subscription_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA subscription_svc GRANT ALL ON SEQUENCES TO subscription_role;
    END IF;
END$$;

-- inquiry lives on its own Postgres instance (Phase 3).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'inquiry_svc') THEN
        GRANT USAGE ON SCHEMA inquiry_svc TO inquiry_role;
        GRANT ALL ON ALL TABLES IN SCHEMA inquiry_svc TO inquiry_role;
        GRANT ALL ON ALL SEQUENCES IN SCHEMA inquiry_svc TO inquiry_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA inquiry_svc GRANT ALL ON TABLES TO inquiry_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA inquiry_svc GRANT ALL ON SEQUENCES TO inquiry_role;
    END IF;
END$$;

-- review lives on its own Postgres instance (Phase 3).
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'review_svc') THEN
        GRANT USAGE ON SCHEMA review_svc TO review_role;
        GRANT ALL ON ALL TABLES IN SCHEMA review_svc TO review_role;
        GRANT ALL ON ALL SEQUENCES IN SCHEMA review_svc TO review_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA review_svc GRANT ALL ON TABLES TO review_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA review_svc GRANT ALL ON SEQUENCES TO review_role;
    END IF;
END$$;

-- shipping lives on its own Postgres instance (Phase 3), so on a fresh
-- shared-cluster DB the shipping_svc schema is not created here. The
-- grants that used to live in this block moved to the shipping
-- migration dir (shipping/005_grant_shipping_role) so they run on the
-- split DB instead. Kept as a guarded DO block for existing
-- shared-cluster DBs that still have the schema lingering from the
-- pre-split layout.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'shipping_svc') THEN
        GRANT USAGE ON SCHEMA shipping_svc TO shipping_role;
        GRANT ALL ON ALL TABLES IN SCHEMA shipping_svc TO shipping_role;
        GRANT ALL ON ALL SEQUENCES IN SCHEMA shipping_svc TO shipping_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA shipping_svc GRANT ALL ON TABLES TO shipping_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA shipping_svc GRANT ALL ON SEQUENCES TO shipping_role;
    END IF;
END$$;

-- notification lives on its own Postgres instance (Phase 3), so on a
-- fresh shared-cluster DB the notification_svc schema is not created
-- here. The grants that used to live in this block moved to the
-- notification migration dir (notification/002_grant_notification_role)
-- so they run on the split DB instead. Kept as a guarded DO block for
-- existing shared-cluster DBs that still have the schema lingering from
-- the pre-split layout.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'notification_svc') THEN
        GRANT USAGE ON SCHEMA notification_svc TO notification_role;
        GRANT ALL ON ALL TABLES IN SCHEMA notification_svc TO notification_role;
        GRANT ALL ON ALL SEQUENCES IN SCHEMA notification_svc TO notification_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA notification_svc GRANT ALL ON TABLES TO notification_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA notification_svc GRANT ALL ON SEQUENCES TO notification_role;
    END IF;
END$$;

-- ─── Read-only catalog consumers ──────────────────────────────
-- Historical: search and recommend used to read catalog_svc directly.
-- Phase 2.2 gave each service its own projection populated by events,
-- and Phase 3 split them to their own DBs. Guard in case catalog_svc
-- still exists on a legacy shared cluster that hasn't been cleaned up.

DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'catalog_svc') THEN
        GRANT USAGE ON SCHEMA catalog_svc TO search_role, recommend_role;
        GRANT SELECT ON ALL TABLES IN SCHEMA catalog_svc TO search_role, recommend_role;
        ALTER DEFAULT PRIVILEGES IN SCHEMA catalog_svc GRANT SELECT ON TABLES TO search_role, recommend_role;
        IF EXISTS (
            SELECT 1 FROM pg_tables
            WHERE schemaname = 'catalog_svc' AND tablename = 'user_events'
        ) THEN
            GRANT INSERT, UPDATE, DELETE ON catalog_svc.user_events TO recommend_role;
        END IF;
    END IF;
END$$;

-- ─── Transitional cross-schema reads (to be removed in Phase 1.2/2.1) ──

-- order reads auth_svc.sellers (seller_name snapshot at checkout) and
-- catalog_svc.products/skus during order creation. Phase 1.2 replaces these
-- with auth.BatchGetSellers and catalog.BatchGetSKUs gRPC calls.
-- auth_svc may live on a split DB (Phase 3); guard both blocks.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'auth_svc') THEN
        GRANT USAGE ON SCHEMA auth_svc TO order_role;
        GRANT SELECT ON auth_svc.sellers TO order_role;
    END IF;
END$$;
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'catalog_svc') THEN
        GRANT USAGE ON SCHEMA catalog_svc TO order_role;
        GRANT SELECT ON catalog_svc.products, catalog_svc.skus TO order_role;
    END IF;
END$$;

-- catalog matview seller_plan_boost joins auth_svc.sellers +
-- subscription_svc.subscription_plans. Phase 2.1 moves this to a local
-- projection fed by events.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'auth_svc') THEN
        GRANT USAGE ON SCHEMA auth_svc TO catalog_role;
        GRANT SELECT ON auth_svc.sellers TO catalog_role;
        IF EXISTS (
            SELECT 1 FROM pg_tables
            WHERE schemaname = 'auth_svc' AND tablename = 'seller_subscriptions'
        ) THEN
            GRANT SELECT ON auth_svc.seller_subscriptions TO catalog_role;
        END IF;
    END IF;
END$$;
-- subscription_svc may live on a split DB (Phase 3). Guard so fresh
-- shared-cluster DBs without the schema don't fail this migration.
DO $$
BEGIN
    IF EXISTS (SELECT 1 FROM pg_namespace WHERE nspname = 'subscription_svc') THEN
        GRANT USAGE ON SCHEMA subscription_svc TO catalog_role;
        GRANT SELECT ON subscription_svc.subscription_plans TO catalog_role;
    END IF;
END$$;
