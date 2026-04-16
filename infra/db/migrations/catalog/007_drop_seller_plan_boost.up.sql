-- Phase 2.2 completion: search now owns its seller_plan_boost projection
-- (populated from subscription events), so the cross-schema matview in
-- catalog_svc is no longer read by anyone. Drop the matview and its
-- refresh() helper.
--
-- Deployments that haven't promoted search's schema yet will break at
-- this step; coordinate the two rollouts.

DROP FUNCTION IF EXISTS catalog_svc.refresh_seller_plan_boost();
DROP MATERIALIZED VIEW IF EXISTS catalog_svc.seller_plan_boost;
