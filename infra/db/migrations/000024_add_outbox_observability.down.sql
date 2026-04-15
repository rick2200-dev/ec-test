ALTER TABLE shipping_svc.outbox_events
    DROP COLUMN IF EXISTS attempt_count,
    DROP COLUMN IF EXISTS last_error;
