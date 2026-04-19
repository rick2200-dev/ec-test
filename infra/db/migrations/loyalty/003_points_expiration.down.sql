-- The backfill UPDATE is not reversible: once expires_at is set we
-- can't safely guess which rows were originally NULL vs written with
-- a value. Only the index is rolled back here.
DROP INDEX IF EXISTS loyalty_svc.point_transactions_earn_expiry_idx;
