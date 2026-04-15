# Adding Features & New Services

## Table of Contents
1. [Adding a Use-Case Method to an Existing Service](#adding-a-use-case-method-to-an-existing-service)
2. [Adding a Domain Entity](#adding-a-domain-entity)
3. [Adding a Pub/Sub Event](#adding-a-pubsub-event)
4. [Adding a gRPC Endpoint](#adding-a-grpc-endpoint)
5. [Adding a New Service](#adding-a-new-service)
6. [Checklist: Dependency Rule Compliance](#checklist-dependency-rule-compliance)

---

## Adding a Use-Case Method to an Existing Service

Example: adding `ArchiveProduct` to the catalog service.

### Step 1 — Domain (`internal/domain/`)
Add the domain error if the business rule can fail:
```go
// domain/errors.go
var ErrProductAlreadyArchived = errors.New("product is already archived")
```

Add behavior to the entity if it encapsulates a state change:
```go
// domain/product.go
func (p *Product) Archive() error {
    if p.Status == StatusArchived {
        return ErrProductAlreadyArchived
    }
    p.Status = StatusArchived
    p.UpdatedAt = time.Now().UTC()
    return nil
}
```

### Step 2 — Port (`internal/port/`)
Add the method to the driving port (`service.go`) and any new repo method to the driven port (`store.go`):
```go
// port/service.go — add to CatalogUseCase interface
ArchiveProduct(ctx context.Context, tenantID, productID uuid.UUID) error

// port/store.go — if the repo needs a new method
type ProductStore interface {
    // existing methods...
    SetStatus(ctx context.Context, tenantID, productID uuid.UUID, status string) error
}
```

### Step 3 — App (`internal/app/`)
Implement the use-case method. Use only `domain/` + `port/` imports:
```go
// app/catalog_service.go
func (s *CatalogService) ArchiveProduct(ctx context.Context, tenantID, productID uuid.UUID) error {
    product, err := s.products.GetByID(ctx, tenantID, productID)
    if err != nil {
        return apperrors.Internal("failed to load product", err)
    }
    if product == nil {
        return domain.ErrProductNotFound
    }
    if err := product.Archive(); err != nil {
        return err  // domain error — don't wrap, let handler map it
    }
    if err := s.products.SetStatus(ctx, tenantID, productID, product.Status); err != nil {
        return apperrors.Internal("failed to archive product", err)
    }
    pubsub.PublishEvent(ctx, s.publisher, tenantID,
        domain.EventTypeProductArchived, "product-events",
        domain.ProductArchivedEvent{ProductID: productID.String()})
    return nil
}
```

### Step 4 — Adapter: postgres (`internal/adapter/postgres/`)
Implement the new repo method:
```go
// adapter/postgres/product_repo.go
func (r *ProductRepo) SetStatus(ctx context.Context, tenantID, productID uuid.UUID, status string) error {
    q := database.QueryerFromContext(ctx, r.pool)
    _, err := q.Exec(ctx,
        `UPDATE products SET status=$1, updated_at=NOW() WHERE tenant_id=$2 AND id=$3`,
        status, tenantID, productID)
    return err
}
```

### Step 5 — Adapter: HTTP handler (`internal/adapter/http/`)
Add the route. Map domain errors in the same file or in `error_mapper.go`:
```go
// adapter/http/catalog_handler.go
func (h *CatalogHandler) handleArchiveProduct(w http.ResponseWriter, r *http.Request) {
    tc, _ := tenant.FromContext(r.Context())
    productID := uuid.MustParse(chi.URLParam(r, "productID"))

    if err := h.svc.ArchiveProduct(r.Context(), tc.TenantID, productID); err != nil {
        httputil.Error(w, mapError(err))
        return
    }
    w.WriteHeader(http.StatusNoContent)
}
```

Add to `error_mapper.go` if a new domain error needs mapping:
```go
case errors.Is(err, domain.ErrProductAlreadyArchived):
    return apperrors.Conflict(err.Error())
```

### Step 6 — Verify
```bash
cd backend/services/catalog
go build ./...
go vet ./...
go test ./...
```

---

## Adding a Domain Entity

Example: adding a `Review` entity to the catalog service.

1. Create `domain/review.go` with the struct, constants, and behavior methods
2. Create or extend `domain/errors.go` with entity-specific errors
3. Add `ReviewStore` interface to `port/store.go`
4. Add review methods to `CatalogUseCase` in `port/service.go`
5. Implement `adapter/postgres/review_repo.go`
6. Implement the use-case methods in `app/catalog_service.go`
7. Add HTTP handler in `adapter/http/review_handler.go`
8. Register routes and wire dependencies in `cmd/server/main.go`

---

## Adding a Pub/Sub Event

### Publishing side (the service that owns the fact)

1. Add constant + struct to `domain/events.go`:
```go
const EventTypeReviewPublished = "review.published"

type ReviewPublishedEvent struct {
    ReviewID  string `json:"review_id"`
    ProductID string `json:"product_id"`
    Rating    int    `json:"rating"`
    TenantID  string `json:"tenant_id"`
}
```

2. Call `pubsub.PublishEvent` from the `app/` layer after the state change commits.

3. Ensure `publisher pubsub.Publisher` is a field on the service struct (injected in main.go).

### Subscribing side (a different service)

1. Create `adapter/pubsub/review_subscriber.go`
2. Define a narrow local interface for the use-case method needed:
   ```go
   type ReviewIndexer interface {
       IndexReview(ctx context.Context, tenantID uuid.UUID, reviewID string, rating int) error
   }
   ```
3. Re-declare the event struct locally (services can't share Go types across module boundaries):
   ```go
   type reviewPublishedData struct {
       ReviewID  string `json:"review_id"`
       ProductID string `json:"product_id"`
       Rating    int    `json:"rating"`
       TenantID  string `json:"tenant_id"`
   }
   ```
4. Implement `Start(ctx) error` that calls `subscriber.Subscribe(ctx, subscriptionName, handler)`
5. In `handleEvent`: switch on `event.Type`, use `decodeEventData` helper to JSON round-trip
6. Wire in `cmd/server/main.go`: `go reviewSub.Start(ctx)`
7. Provision the subscription in GCP (or add to the Pub/Sub emulator setup)

---

## Adding a gRPC Endpoint

See the `grpc-integration` skill for the full proto-first workflow.
Quick summary for adding an RPC to an **existing** service:

1. Add the RPC to the `.proto` file in `backend/proto/{service}/v1/`
2. Run `make proto-gen` to regenerate `backend/gen/go/`
3. Implement the new method on the gRPC server in `adapter/grpc/server.go`
4. Add the use-case method to `port/service.go` if it doesn't exist
5. Implement in `app/{name}_service.go`
6. Add type conversions to `adapter/grpc/convert.go`

---

## Adding a New Service

Use this checklist when the feature genuinely warrants a new service
(see `service-boundaries.md` for when NOT to add a new service).

### 1. Scaffold the Go module
```
services/{name}/
  go.mod                         # module github.com/Riku-KANO/ec-test/services/{name}
  cmd/server/main.go
  internal/
    domain/{entity}.go
    domain/errors.go
    domain/events.go
    port/store.go
    port/service.go
    app/{name}_service.go
    adapter/http/{name}_handler.go
    adapter/http/error_mapper.go
    adapter/postgres/{entity}_repo.go
    config/config.go
```

### 2. go.mod with replace directives
```
module github.com/Riku-KANO/ec-test/services/{name}

go 1.23

require (
    github.com/Riku-KANO/ec-test/pkg v0.0.0
    github.com/go-chi/chi/v5 v5.x.x
    github.com/google/uuid v1.x.x
    github.com/jackc/pgx/v5 v5.x.x
)

replace github.com/Riku-KANO/ec-test/pkg => ../../pkg
```

### 3. config/config.go pattern
```go
type Config struct {
    Addr        string `env:"PORT,default=:8091"`
    DatabaseURL string `env:"DATABASE_URL,required"`
    InternalToken string `env:"INTERNAL_TOKEN,required"`
    // Pub/Sub — optional (service still works without events)
    PubSubProjectID string `env:"PUBSUB_PROJECT_ID"`
}

func Load() Config {
    var cfg Config
    envconfig.MustProcess("", &cfg)
    return cfg
}
```

### 4. main.go wiring order
```
config.Load()
→ database.NewPool()
→ pubsub.NewGCPPublisher() (optional)
→ adapter/postgres repos
→ adapter/httpclient outbound clients (if needed)
→ app.NewXxxService(repos, clients, publisher)
→ adapter/http handlers
→ chi router + middleware
→ Start HTTP server (+ gRPC server if applicable)
→ Start Pub/Sub subscribers
→ Graceful shutdown on SIGTERM
```

### 5. Register in the gateway
If the new service is accessed by buyers/sellers via the gateway:
- Add a reverse proxy route in `gateway/internal/proxy/`
- Or add a gRPC client in `gateway/internal/grpcclient/` if gRPC

### 6. Infra checklist
- [ ] PostgreSQL schema migration in `db/migrations/`
- [ ] Pub/Sub topics + subscriptions provisioned (or added to emulator setup)
- [ ] Environment variables documented in `.env.example`
- [ ] Service added to `docker-compose.yml` (development)
- [ ] Service added to `k8s/` or `terraform/` (production)

---

## Checklist: Dependency Rule Compliance

Run these checks after any new feature:

```bash
# No pgx in port/ or app/
grep -r "jackc/pgx" internal/port/ internal/app/   # should be empty

# No net/http in app/
grep -r '"net/http"' internal/app/                  # should be empty

# No apperrors in app/ except for Internal() wrapping infra errors
grep -n "apperrors\." internal/app/*.go | grep -v "Internal("

# Build + test
go build ./...
go vet ./...
go test ./...
```

The golden rule: if you can explain a dependency in the sentence
"the domain needs to know about X to do its job" and X is an infrastructure detail,
the dependency is wrong — push X outward to `adapter/` or abstract it behind a port interface.
