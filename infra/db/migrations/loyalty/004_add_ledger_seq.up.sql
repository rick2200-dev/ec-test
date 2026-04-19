-- Add a monotonic ledger_seq so the expiration reaper's replay order
-- is stable. The expiration reaper rebuilds per-buyer earn buckets by
-- walking the ledger in order; FIFO correctness depends on that order
-- matching the true insertion sequence.
--
-- We can't rely on created_at: it defaults to NOW() which in Postgres
-- returns the transaction-start timestamp, so every row written in one
-- transaction carries the same value. Falling back to UUID (id ASC)
-- is unstable — the order becomes random, which is wrong the moment a
-- single transaction inserts more than one row (e.g. the
-- CommitPointsReservation path that writes both redeem + earn).
-- Switching to clock_timestamp() would still collide at high enough
-- write rates.
--
-- ledger_seq is written by nextval() on every INSERT. Because
-- nextval() advances inside the writing transaction but the increments
-- happen in a strict total order across concurrent transactions,
-- the column gives us a replay order that exactly matches commit
-- order — which is what FIFO accounting requires.

-- Step 1: add column (nullable initially so the backfill has something
-- to fill).
ALTER TABLE loyalty_svc.point_transactions
    ADD COLUMN ledger_seq BIGINT;

-- Step 2: backfill existing rows deterministically.
--
-- created_at collides for rows written in the same transaction
-- (Postgres NOW() is the transaction-start timestamp), and UUID is
-- random within the collision — replaying in that order would put
-- an earn's ledger_seq BEFORE the redeem that originally came first
-- in code. To keep the FIFO walk semantically consistent we force
-- consumption-type rows ahead of creation-type rows within the same
-- timestamp. Specifically: CommitPointsReservation writes redeem
-- followed by earn in a single tx; the CASE below recovers that
-- insertion order from rows that only carry collided created_at.
--
-- Caveat: this is a best-effort reconstruction, not a faithful
-- replay. If any historical path wrote two earns or two consumption
-- rows in the same transaction, the relative order among same-
-- bucket-affecting rows is still UUID-random. New rows written
-- after this migration use nextval() from the sequence below, which
-- is immune to the collision. Operators running expiration on
-- heavily reverse_earn / adjust histories should spot-check the
-- invariant warnings that the reaper emits on the first few passes
-- post-enable.
WITH ordered AS (
    SELECT
        id,
        ROW_NUMBER() OVER (
            ORDER BY
                created_at,
                CASE type
                    -- consumption rows (any kind of debit) first
                    WHEN 'redeem'       THEN 0
                    WHEN 'reverse_earn' THEN 0
                    WHEN 'refund'       THEN 0
                    WHEN 'adjust'       THEN 0
                    WHEN 'expire'       THEN 0
                    -- credit-creating rows last so a same-tx redeem
                    -- doesn't appear to consume an earn that wasn't
                    -- yet written when the redeem was emitted
                    WHEN 'earn'         THEN 1
                    ELSE 2
                END,
                id
        ) AS rn
    FROM loyalty_svc.point_transactions
)
UPDATE loyalty_svc.point_transactions AS pt
   SET ledger_seq = ordered.rn
  FROM ordered
 WHERE pt.id = ordered.id;

-- Step 3: create the sequence, seeded past the highest backfilled
-- value so freshly-inserted rows don't collide with historical ones.
CREATE SEQUENCE IF NOT EXISTS loyalty_svc.point_transactions_ledger_seq_seq;
SELECT setval(
    'loyalty_svc.point_transactions_ledger_seq_seq',
    COALESCE((SELECT MAX(ledger_seq) FROM loyalty_svc.point_transactions), 0) + 1,
    false -- start at exactly that value on nextval()
);

-- Step 4: make nextval() the default + lock in NOT NULL.
ALTER TABLE loyalty_svc.point_transactions
    ALTER COLUMN ledger_seq SET DEFAULT nextval('loyalty_svc.point_transactions_ledger_seq_seq');
ALTER TABLE loyalty_svc.point_transactions
    ALTER COLUMN ledger_seq SET NOT NULL;

-- Index used by the reaper's chronological replay query.
CREATE INDEX point_transactions_buyer_seq_idx
    ON loyalty_svc.point_transactions (buyer_auth0_id, ledger_seq);
