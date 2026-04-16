-- Phase 2.1: catalog outbox. Mirrors shipping_svc.outbox_events so product
-- lifecycle events (product.created/updated/deleted) can be published
-- inside the same transaction as the business write. An external relay
-- worker drains the table into Pub/Sub with at-least-once semantics.

CREATE TABLE IF NOT EXISTS catalog_svc.outbox_events (
    id            UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type    TEXT        NOT NULL,
    topic         TEXT        NOT NULL,
    payload       JSONB       NOT NULL,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    -- NULL = unpublished; non-NULL = relay has successfully published.
    published_at  TIMESTAMPTZ,
    -- Relay sets this while processing to prevent duplicate pickup across
    -- replicas. Auto-expires after 5 minutes so crashed replicas do not
    -- block retry.
    processing_at TIMESTAMPTZ,
    -- Retry observability: how many times the relay attempted to publish
    -- this row, and the last error if any.
    attempt_count INT         NOT NULL DEFAULT 0,
    last_error    TEXT
);

CREATE INDEX idx_catalog_outbox_events_pending
    ON catalog_svc.outbox_events (created_at)
    WHERE published_at IS NULL;
