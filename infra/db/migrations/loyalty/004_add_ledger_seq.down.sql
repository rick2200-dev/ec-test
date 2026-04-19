DROP INDEX IF EXISTS loyalty_svc.point_transactions_buyer_seq_idx;
ALTER TABLE loyalty_svc.point_transactions
    DROP COLUMN IF EXISTS ledger_seq;
DROP SEQUENCE IF EXISTS loyalty_svc.point_transactions_ledger_seq_seq;
