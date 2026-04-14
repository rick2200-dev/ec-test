---
name: ec-backend-impl
description: |
  Implementation guide for the ec-test Go microservices backend. Use this skill whenever
  you are working on ANY backend code in C:\Users\81903\projects\ec-test\backend — adding
  features, new services, domain models, ports, adapters, events, gRPC endpoints, HTTP
  handlers, transactions, error handling, or service-to-service communication. Also use
  this skill when the user asks about service boundaries, batch patterns, or how different
  parts of the backend fit together. Invoke proactively even if the user just mentions a
  service name or asks "how should I implement X in the backend".
---

# EC Marketplace Backend — Implementation Guide

This codebase is a multi-tenant EC (e-commerce) marketplace with 10 Go microservices.
Each service is a **separate Go module** under `backend/services/{name}/`.
Shared infrastructure lives in `backend/pkg/` (module `github.com/Riku-KANO/ec-test/pkg`).
Proto definitions live in `backend/proto/`, generated code in `backend/gen/go/`.

## Services at a Glance

| Service | Port | Role | Communication |
|---------|------|------|---------------|
| gateway | 8080 | JWT validation, routing, rate limiting | REST in → gRPC/HTTP out |
| auth | 8081 | Tenants, sellers, buyers, RBAC, API tokens | HTTP |
| catalog | 8082 | Products, SKUs, categories | gRPC + HTTP |
| order | 8083 | Orders, Stripe payments, commissions | gRPC + HTTP + Pub/Sub |
| inventory | 8084 | Stock reservation / release | gRPC + Pub/Sub (subscriber) |
| search | 8085 | Vertex AI Search indexing | HTTP + Pub/Sub (subscriber) |
| recommend | 8086 | Personalized recommendations | HTTP + Pub/Sub (subscriber) |
| notification | 8087 | Email/push notifications | Pub/Sub only (subscriber) |
| cart | 8088 | Cart (Redis) + multi-seller checkout | HTTP |
| inquiry | 8090 | Buyer↔seller conversations | HTTP |

---

## Canonical Directory Layout (per service)

```
services/{name}/
  cmd/server/main.go          # composition root — wires everything, starts servers
  internal/
    domain/                   # INNERMOST — zero infrastructure imports
      {entity}.go             #   structs, value objects, behavior methods
      errors.go               #   sentinel errors (transport-agnostic)
      events.go               #   typed event structs (replaces map[string]any)
    port/                     # INTERFACES ONLY — no implementations
      store.go                #   driven ports: DB repos, external API clients
      service.go              #   driving port: use-case interface for handlers/gRPC
    app/                      # APPLICATION LAYER
      {name}_service.go       #   orchestration: imports domain/ + port/ only
    adapter/
      postgres/               #   pgx repository implementations
      http/                   #   chi handlers + error_mapper.go
      grpc/                   #   gRPC server + convert.go
      httpclient/             #   outbound HTTP clients (implements port interfaces)
      pubsub/                 #   Pub/Sub subscriber(s)
      redis/                  #   Redis store (cart only)
      stripe/                 #   Stripe client (order only)
    config/config.go          #   env-var config struct
```

---

## The One Rule: Dependency Direction

```
domain/ ←── port/ ←── app/ ←── adapter/*  ←── cmd/
```

- `domain/` imports **nothing** from this repo (only stdlib + uuid/json)
- `port/` imports `domain/` only
- `app/` imports `domain/` + `port/` only — **no pgx, no net/http, no Stripe**
- `adapter/*` imports everything it needs
- `cmd/` is the only place where all layers are wired together

**Quick check**: if you see `pgx` or `net/http` imported in `app/` or `port/`, that's a violation.

---

## Error Handling: Two Layers

### In `app/` (business logic)
Return **domain sentinel errors** for business rule violations:
```go
// domain/errors.go
var ErrOrderNotFound   = errors.New("order not found")
var ErrInvalidQuantity = errors.New("quantity must be positive")

// app/order_service.go — business rule violation
if order == nil {
    return nil, domain.ErrOrderNotFound   // ✓ domain error, no HTTP coupling
}

// app/order_service.go — infrastructure failure
if err := s.repo.Create(ctx, order); err != nil {
    return nil, apperrors.Internal("failed to create order", err)  // ✓ infra wrapping
}
```

### In `adapter/http/` (handler layer)
Map domain errors → HTTP status via a local `mapError()`:
```go
// adapter/http/error_mapper.go
func mapError(err error) *apperrors.AppError {
    switch {
    case errors.Is(err, domain.ErrOrderNotFound):
        return apperrors.NotFound(err.Error())
    case errors.Is(err, domain.ErrInvalidQuantity):
        return apperrors.BadRequest(err.Error())
    default:
        return apperrors.Internal("internal error", err)
    }
}
```

---

## Event Publishing: Typed Structs

Never use `map[string]any`. Always define typed structs in `domain/events.go`:
```go
// domain/events.go
const EventTypeOrderCreated = "order.created"

type OrderCreatedEvent struct {
    OrderID      string `json:"order_id"`
    SellerID     string `json:"seller_id"`
    BuyerAuth0ID string `json:"buyer_auth0_id"`
    TotalAmount  int64  `json:"total_amount"`
    Currency     string `json:"currency"`
}
```

Then publish from `app/`:
```go
pubsub.PublishEvent(ctx, s.publisher, tenantID,
    domain.EventTypeOrderCreated, "order-events",
    domain.OrderCreatedEvent{...})
```

Pub/Sub topics & subscriptions:
- One topic per domain area: `order-events`, `cart-events`, `product-events`
- One subscription **per consuming service**: `order-events-inventory`, `order-events-notification`
- Subscribers re-declare mirrored structs locally (services are separate modules)

---

## Transaction Handling: Context Propagation

Transactions flow through context — **`pgx.Tx` never appears in port or app signatures**.

```go
// app layer — calls TxRunner port
s.db.RunTenantTx(ctx, tenantID, func(txCtx context.Context) error {
    s.users.Create(txCtx, &user)    // repo reads tx from context
    s.audit.Append(txCtx, &entry)
})

// adapter/postgres layer — extracts tx from context
func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
    q := database.QueryerFromContext(ctx, r.pool)  // returns tx if present, pool otherwise
    return q.QueryRow(ctx, sql, ...).Scan(...)
}
```

`TxRunner` port:
```go
type TxRunner interface {
    RunTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(ctx context.Context) error) error
}
```

---

## Port Design: Driven vs. Driving

```
port/store.go     ← "driven" ports (what app needs from infrastructure)
port/service.go   ← "driving" port (use-case interface consumed by handlers/gRPC)
```

DTOs that would create circular imports (e.g., `SKULookup` returned by catalog's httpclient
and also used in the app layer) belong in `port/` — not in `domain/` and not in `adapter/`.

---

## Reference Files

For detailed patterns, read the relevant reference file when needed:

| Topic | File |
|-------|------|
| Transactions, gRPC, HTTP clients, multi-tenancy | `references/patterns.md` |
| Service boundaries, communication choices, batch patterns | `references/service-boundaries.md` |
| Adding a feature or new service step-by-step | `references/new-feature.md` |
| Transactional outbox pattern (reliable event publishing) | `references/outbox-pattern.md` |
