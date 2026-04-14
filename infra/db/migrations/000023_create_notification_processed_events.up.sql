-- Notification service schema and processed-events inbox table.
--
-- processed_events is a status-based inbox that prevents duplicate notification
-- sends under at-least-once Pub/Sub delivery. Each event passes through two
-- states before it is permanently deduplicated:
--
--   processing → sent
--
-- Claim protocol (see EventDeduplicator.TryClaim):
--   INSERT (event_id, status='processing', processing_at=NOW())
--   ON CONFLICT DO UPDATE SET processing_at=NOW(), status='processing'
--     WHERE status='processing' AND processing_at < NOW()-INTERVAL '10 min'
--
--   'acquired'         → this caller owns the claim; proceed to send.
--   'active_processing'→ another caller holds an active (non-expired) claim;
--                        return a retryable error (nack) so the message stays
--                        alive for redelivery — do NOT return nil here.
--   'sent'             → notification was already delivered; return nil (ack)
--                        without sending again.
--
-- After a successful send:
--   UPDATE status='sent'   (MarkSent)
--   Future redeliveries see status='sent' → ON CONFLICT WHERE is false → skipped.
--
-- After a failed send:
--   DELETE the row         (Release)
--   Pub/Sub redelivers; the next delivery can TryClaim again.
--
-- Crash recovery (closes the pre-send crash window):
--   If the process crashes after TryClaim but before MarkSent/Release,
--   processing_at stays at the claim time. While the claim is active, Pub/Sub
--   redeliveries receive ClaimActiveProcessing and must return a retryable error
--   (nack) — NOT nil — so the message stays alive in the Pub/Sub queue.
--   After 10 minutes the WHERE condition becomes true and the next redelivery
--   re-claims the event and retries the send.
--
-- Delivery guarantee:
--   - At-least-once: no notification is permanently lost as long as:
--     (a) Pub/Sub redelivers at least once after the 10-min timeout expires, and
--     (b) callers return a retryable error (not nil) on ClaimActiveProcessing.
--     Returning nil on active_processing would ack the message before the
--     timeout, leaving nothing to redeliver — this is the critical invariant
--     that callers must uphold.
--   - Concurrent-redelivery safe: DB INSERT/UPDATE atomicity ensures only one
--     caller holds an active claim at a time.
--
-- Cleanup: rows with status='sent' can be pruned once they are older than the
-- maximum time a Pub/Sub message can be redelivered. That window is bounded by
-- max(Pub/Sub retention, outbox relay recovery time). Pub/Sub retains messages
-- for up to 7 days; an outbox row stuck due to a DB/Pub/Sub outage could
-- remain undelivered for hours to days. Using 30 days as the safe prune
-- threshold covers both cases with comfortable margin.
-- The index on processed_at supports periodic cleanup jobs.

CREATE SCHEMA IF NOT EXISTS notification_svc;

CREATE TABLE IF NOT EXISTS notification_svc.processed_events (
    event_id       TEXT        PRIMARY KEY,
    event_type     TEXT        NOT NULL,
    -- 'processing': claim is held; send is in progress or the process crashed.
    -- 'sent':       notification was delivered; future redeliveries are skipped.
    status         TEXT        NOT NULL DEFAULT 'processing'
                               CHECK (status IN ('processing', 'sent')),
    -- Updated to NOW() only when an expired processing claim is re-claimed
    -- (WHERE status='processing' AND processing_at < NOW()-INTERVAL '10 min').
    -- Active non-expired claims leave this column unchanged; the ON CONFLICT
    -- WHERE is false so the UPDATE does not fire. Used for the 10-minute
    -- crash-recovery timeout: an expired processing claim can be re-claimed.
    processing_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- When the row was first inserted. Used only for cleanup age calculations.
    processed_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_processed_events_cleanup
    ON notification_svc.processed_events (processed_at);
