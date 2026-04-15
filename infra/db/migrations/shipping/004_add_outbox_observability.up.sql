-- Add observability columns to shipping_svc.outbox_events.
--
-- attempt_count: incremented each time the relay tries (and fails) to publish a row.
--   Lets operators identify stuck rows (e.g. high count = persistent Pub/Sub error).
-- last_error: stores the most-recent publish error message so operators can diagnose
--   failures without digging through logs. NULL means no attempt has failed yet.

ALTER TABLE shipping_svc.outbox_events
    ADD COLUMN IF NOT EXISTS attempt_count INT         NOT NULL DEFAULT 0,
    ADD COLUMN IF NOT EXISTS last_error    TEXT;
