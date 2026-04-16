-- Reverse of 002.up: drop subscription_svc tables + schema. The old
-- down-script also recreated auth_svc.* tables and the catalog_svc
-- matview; both are obsolete (see 002.up header) so we don't resurrect
-- them here.

DROP TABLE IF EXISTS subscription_svc.buyer_subscriptions;
DROP TABLE IF EXISTS subscription_svc.buyer_plans;
DROP TABLE IF EXISTS subscription_svc.seller_subscriptions;
DROP TABLE IF EXISTS subscription_svc.subscription_plans;
DROP SCHEMA IF EXISTS subscription_svc;
