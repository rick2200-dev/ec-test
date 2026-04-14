-- outbox_events is a system queue table used by the shipping service's
-- outbox relay worker. It is intentionally NOT protected by Row-Level Security
-- because the relay must read events across all tenants.
--
-- Events are written inside the same DB transaction as the business-data update
-- (shipment status change) so they are guaranteed to be published eventually.
-- The relay worker polls for unpublished events, publishes them to Pub/Sub, and
-- marks them as published. FOR UPDATE SKIP LOCKED lets multiple replicas pick
-- non-overlapping batches safely.

CREATE TABLE IF NOT EXISTS shipping_svc.outbox_events (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type    TEXT        NOT NULL,
    topic         TEXT        NOT NULL,
    tenant_id     UUID        NOT NULL,
    payload       JSONB       NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- NULL = unpublished; non-NULL = relay has successfully published.
    published_at  TIMESTAMPTZ,
    -- Relay sets this while processing to prevent duplicate pickup across replicas.
    -- Auto-expires after 5 minutes so crashed replicas do not block retry.
    processing_at TIMESTAMPTZ
);

CREATE INDEX idx_outbox_events_pending
    ON shipping_svc.outbox_events (created_at)
    WHERE published_at IS NULL;
