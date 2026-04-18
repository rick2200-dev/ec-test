# Service Boundaries & Communication Patterns

## Table of Contents

1. [Choosing a Communication Mechanism](#choosing-a-communication-mechanism)
2. [Synchronous: gRPC vs HTTP](#synchronous-grpc-vs-http)
3. [Asynchronous: Pub/Sub Events](#asynchronous-pubsub-events)
4. [Bounded Contexts Within a Service](#bounded-contexts-within-a-service)
5. [Batch & Scheduled Work](#batch--scheduled-work)
6. [Adding a New Service Dependency](#adding-a-new-service-dependency)
7. [Current Service Communication Map](#current-service-communication-map)

---

## Choosing a Communication Mechanism

Ask these questions in order:

| Question                                                                          | Answer → Mechanism                                                      |
| --------------------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| Does the caller need the result immediately to serve the request?                 | Yes → **synchronous** (gRPC or HTTP)                                    |
| Is there an existing gRPC contract for this service pair?                         | Yes → **gRPC**                                                          |
| gRPC not yet set up, or it's a lightweight one-off call?                          | → **Internal HTTP** (with `X-Internal-Token`)                           |
| Is the caller fine if the downstream processes the event later (seconds/minutes)? | → **Pub/Sub**                                                           |
| Does multiple services need to react to the same fact?                            | → **Pub/Sub** (fan-out via separate subscriptions)                      |
| Is it a background sweep / nightly job?                                           | → **Pub/Sub trigger** from a scheduled job, or a standalone task runner |

---

## Synchronous: gRPC vs HTTP

### Use gRPC when

- The connection is `gateway → downstream` (gateway has a gRPC client pool)
- The API is complex (multiple RPCs, streaming, pagination)
- Catalog, Order, Inventory already have `.proto` definitions — prefer gRPC for those
- You are defining a new permanent API surface

### Use Internal HTTP when

- The gRPC contract doesn't exist yet and velocity matters
- The call is narrow / one-off (e.g., `GET /internal/purchase-check`)
- The service pair is: cart→catalog, cart→order, inquiry→order

### Internal HTTP conventions

- Endpoint prefix: `/internal/`
- Auth: `X-Internal-Token` header (shared secret per service, set via env var)
- Tenant: `X-Tenant-ID` header (UUID string)
- Caller puts the client in `adapter/httpclient/`, implements a `port/store.go` interface
- The called service validates the token in a middleware or at handler entry

---

## Asynchronous: Pub/Sub Events

### When to use events

- Downstream doesn't need to answer the request (fire-and-forget)
- Multiple services react to the same fact (e.g., `order.cancelled` → inventory AND notification)
- Loose coupling is desired (order service shouldn't know about inventory or notification)
- Eventual consistency is acceptable

### Topic naming

```
{domain}-events      e.g.  order-events, cart-events, product-events
```

One topic covers all events for a domain. Consumers filter by `event.Type`.

### Subscription naming

```
{topic}-{consuming-service}   e.g.  order-events-inventory, order-events-notification
```

Each consumer gets its own subscription = its own delivery cursor = at-least-once independently.

### Event payload design

- Define typed structs in the **publishing** service's `domain/events.go`
- Each consuming service **re-declares** the struct locally (separate Go modules can't share)
- Embed all data the consumer needs to act — avoid making consumers call back to get more info
  - Example: `order.cancelled` includes `line_items` (sku_id + quantity) so inventory
    doesn't need to call the order service to know what to release
- Field names (JSON tags) are a **public contract** — renaming is a breaking change

### Idempotency requirement

Every subscriber must be idempotent. Pub/Sub guarantees at-least-once delivery.
Typical guard: insert a unique row (e.g., `inventory_movements` keyed on `order_id`)
inside the same transaction as the state change. On duplicate, the insert fails and
the handler returns nil (ack without processing again).

### Error handling in subscribers

```
decode error   → return error (Nack → redelivery; poisonous messages need a dead-letter config)
unknown type   → return nil (Ack; silently discard unrecognized events)
business error → return error (Nack) if retrying makes sense, else log + Ack
```

---

## Bounded Contexts Within a Service

Not every feature belongs at the top level of a service. A **bounded context** is appropriate when:

- The feature has its own domain model (state machine, specialized entities)
- It has a multi-step workflow with rollback semantics (e.g., Stripe calls + DB writes)
- It should be testable in isolation without depending on the parent service's wiring

**Example: Order Cancellation** (`services/order/internal/cancellation/`)

```
cancellation/
  domain.go          # CancellationRequest entity + Status type
  errors.go          # sentinel errors specific to cancellation
  events.go          # typed event structs + publish helpers
  repository.go      # persistence interface + postgres implementation
  service.go         # orchestration: request → Stripe refund → DB write → events
  handler.go         # HTTP handler
```

The cancellation package has its own narrow interfaces (`OrderReader`, `PayoutReader`, `StripeClient`)
rather than depending on the parent service's full port interfaces. This makes unit testing trivial.

**When NOT to make a bounded context**: simple CRUD features, minor additions to existing entities.
Use bounded contexts only for genuinely complex, self-contained workflows.

---

## Batch & Scheduled Work

### Current state

There are currently **no batch jobs** in this codebase. All processing is event-driven via Pub/Sub.

### Patterns for future batch work

**Option 1: Event-driven trigger (recommended for most cases)**
A Cloud Scheduler job publishes a trigger event (`"inventory.stock_check_requested"`) to a topic.
A subscriber picks it up and performs the sweep. This keeps the batch logic inside the service
boundary and uses the same at-least-once + idempotency guarantees as other events.

**Option 2: Standalone task runner**
For heavy ETL or data migrations, add a `cmd/task/main.go` alongside `cmd/server/main.go`.
It wires the same repositories and service layer, runs the job, then exits.

```
services/{name}/
  cmd/
    server/main.go   # long-running HTTP+gRPC server
    task/main.go     # one-shot batch job: go run ./cmd/task -- --job=backfill-foo
```

The task binary imports the same `app/` and `adapter/postgres/` packages — no code duplication.

**Option 3: Periodic in-process job**
Use `time.Ticker` in `cmd/server/main.go` for low-frequency, low-stakes sweeps
(e.g., `recommend.RefreshPopularProducts()` every hour).

```go
go func() {
    ticker := time.NewTicker(1 * time.Hour)
    for range ticker.C {
        if err := recommendSvc.RefreshPopularProducts(ctx); err != nil {
            slog.Warn("popular products refresh failed", "error", err)
        }
    }
}()
```

Only appropriate when: the job is cheap, failure is non-critical, and horizontal scaling
won't cause duplicate runs (or duplicate runs are idempotent).

### Batch design rules

1. Always idempotent — batches can be re-run safely
2. Use the existing `TxRunner` for atomic DB updates
3. Use `domain/errors.go` sentinel errors for business rule violations inside the batch
4. Log progress with `slog` using structured fields (`"processed"`, `"skipped"`, `"failed"`)
5. Don't hold DB connections while calling external APIs (Stripe, GCP, etc.)

---

## Adding a New Service Dependency

When service A needs to call service B for the first time:

1. **Decide the mechanism** (see table above)
2. **Define the port interface** in A's `port/store.go`:
   ```go
   type OrderClient interface {
       CheckPurchase(ctx context.Context, tenantID uuid.UUID, ...) (*PurchaseCheckResult, error)
   }
   ```
3. **Implement the client** in A's `adapter/httpclient/order_client.go` (or gRPC client)
4. **Wire in main.go**: `orderClient := httpclient.NewOrderClient(cfg.OrderServiceURL, cfg.OrderInternalToken)`
5. **Add env vars** to A's `config/config.go`:
   ```go
   OrderServiceURL    string `env:"ORDER_SERVICE_URL"`
   OrderInternalToken string `env:"ORDER_INTERNAL_TOKEN"`
   ```
6. On the **called service** B, add an internal endpoint under `/internal/` with token validation

Never import package B's Go code directly into package A. Services communicate over the network,
not via Go function calls (they are separate binaries/modules).

---

## Current Service Communication Map

```
                      ┌─────────┐
                      │ gateway │  (8080)
                      └────┬────┘
            gRPC ──────────┼───────────── gRPC
             ┌─────────────┼─────────────┐
             ▼             ▼             ▼
         catalog        order        inventory
         (8082)         (8083)        (8084)
             │             │
     HTTP internal  HTTP internal
             │             │
             ▼             ▼
           cart ────► order (checkout)
           (8088)
             │
     HTTP internal
             │
             ▼
          inquiry
          (8090) ────► order (purchase-check)

Pub/Sub fan-out (order-events topic):
  order ──► inventory  (release stock on order.cancelled)
  order ──► notification (emails on order lifecycle events)

Pub/Sub fan-out (product-events topic):
  catalog ──► search  (index on product created/updated)
  catalog ──► recommend (update recommendation model)

Cart events:
  cart ──► recommend (cart.checked_out → record purchase behaviour)
```

**Internal token matrix** (who calls whom with X-Internal-Token):

| Caller  | Called  | Endpoint                                      |
| ------- | ------- | --------------------------------------------- |
| cart    | catalog | `GET /internal/skus/{id}`                     |
| cart    | order   | `POST /internal/checkouts`                    |
| inquiry | order   | `GET /internal/purchase-check`                |
| order   | auth    | `GET /internal/buyer-subscriptions/{auth0id}` |
| gateway | all     | healthz + authz endpoints                     |
