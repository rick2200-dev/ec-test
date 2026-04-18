# Transactional Outbox Pattern — EC Marketplace

## Problem

Every service that does "DB commit + Pub/Sub publish" has a gap:

```
1. BEGIN TX
2. UPDATE shipments SET status='shipped' ...
3. COMMIT  ← process can crash here
4. pubsub.Publish("shipping-events", ...)  ← event lost permanently
```

If the process dies between step 3 and 4, the DB is correct but downstream
consumers (order status update, buyer notification email) never receive the
event. Recovery requires manual inspection and re-triggering.

This affects: `shipping.RegisterShipment`, `shipping.MarkDelivered`, and
all equivalent operations in the order service.

## Current State (v1 — known limitation)

All services in this codebase use **best-effort publish**: `pubsub.PublishEvent`
logs a warning on failure but does not retry or persist the event. The GCP
Pub/Sub client has built-in retry for transient network errors, so single-call
failures are rare in steady state. The unrecoverable case is a process crash
between commit and publish.

This is documented as a known limitation. When this matters most:

- `shipment.shipped` → order status + buyer email (tracking number)
- `shipment.delivered` → order status + buyer email + review prompt
- `order.paid` → inventory reservation, shipment creation
- `order.cancelled` → refund, inventory release

## Outbox Pattern (v2 recommendation)

Add an `outbox_events` table to each schema that participates in event
publishing. Write events atomically in the same DB transaction as the business
data. A separate relay goroutine polls the outbox and publishes, deleting (or
marking) each row after successful publish.

### Schema (add once per service schema)

```sql
CREATE TABLE {schema}.outbox_events (
    id          UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    event_type  TEXT        NOT NULL,
    topic       TEXT        NOT NULL,
    tenant_id   UUID        NOT NULL,
    payload     JSONB       NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    published_at TIMESTAMPTZ          -- NULL = unpublished
);
CREATE INDEX idx_outbox_unpublished
    ON {schema}.outbox_events (created_at)
    WHERE published_at IS NULL;
```

### Write side (inside the business TX)

```go
// In app layer, inside RunTenantTx:
_, err = tx.Exec(ctx,
    `INSERT INTO shipping_svc.outbox_events (event_type, topic, tenant_id, payload)
     VALUES ($1, $2, $3, $4)`,
    domain.EventTypeShipmentShipped, "shipping-events", tenantID,
    mustJSON(domain.ShipmentShippedEvent{...}),
)
```

### Relay goroutine (cmd/server/main.go)

```go
ticker := time.NewTicker(5 * time.Second)
go func() {
    for {
        select {
        case <-bgCtx.Done():
            return
        case <-ticker.C:
            if err := relay.Run(bgCtx); err != nil {
                slog.Warn("outbox relay error", "error", err)
            }
        }
    }
}()
```

### Relay implementation

```go
// Fetch unpublished events in batches.
rows, _ := pool.Query(ctx,
    `SELECT id, event_type, topic, tenant_id, payload
     FROM shipping_svc.outbox_events
     WHERE published_at IS NULL
     ORDER BY created_at
     LIMIT 50
     FOR UPDATE SKIP LOCKED`)

for rows.Next() {
    // publish ...
    pool.Exec(ctx,
        `UPDATE shipping_svc.outbox_events SET published_at=NOW() WHERE id=$1`, id)
}
```

`FOR UPDATE SKIP LOCKED` makes multiple relay replicas safe — each locks its
own batch without blocking others.

## Idempotency on the Consumer Side

At-least-once delivery means consumers must deduplicate. Use `event.ID` (UUID)
as the dedup key. Check before processing:

```sql
INSERT INTO {schema}.processed_events (event_id, processed_at)
VALUES ($1, NOW())
ON CONFLICT (event_id) DO NOTHING
RETURNING event_id
```

If `RETURNING` returns no rows, the event was already processed — skip.

## Files to Modify When Implementing

- `infra/db/migrations/000XXX_add_outbox.up.sql` — add table
- `backend/services/{name}/internal/adapter/postgres/outbox_relay.go` — relay
- `backend/services/{name}/internal/app/{name}_service.go` — write to outbox in TX instead of calling `pubsub.PublishEvent` after commit
- `backend/services/{name}/cmd/server/main.go` — start relay goroutine
