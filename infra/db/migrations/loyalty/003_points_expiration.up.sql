-- 003_points_expiration: enable the expiration reaper.
--
-- Two changes:
--
-- 1. Partial index to make the expiration reaper's candidate query
--    (earn rows with expires_at < now() that haven't been expired
--    yet) cheap. The existing point_transactions_buyer_time_idx is
--    buyer-scoped and doesn't help the cross-buyer scan the reaper
--    needs.
--
-- 2. One-shot backfill: existing earn rows were inserted with
--    expires_at = NULL (the column existed but the policy didn't).
--    Backfill them with `NOW() + 365 days` — NOT `created_at + 365
--    days` — so the policy starts today rather than instantly
--    expiring anyone who earned a year ago. New earn writes stamp
--    their own expires_at from cfg.PointExpirationDays (source of
--    truth is the service config; this SQL hardcodes 365 as a
--    one-time snapshot of that default).
--
-- Operators running with a non-default PointExpirationDays should
-- adjust the interval in this migration BEFORE applying it — the
-- two values must match so new earns and backfilled earns have the
-- same window semantics.

CREATE INDEX point_transactions_earn_expiry_idx
    ON loyalty_svc.point_transactions (expires_at)
    WHERE type = 'earn' AND expires_at IS NOT NULL;

UPDATE loyalty_svc.point_transactions
   SET expires_at = NOW() + INTERVAL '365 days'
 WHERE type = 'earn'
   AND expires_at IS NULL;
