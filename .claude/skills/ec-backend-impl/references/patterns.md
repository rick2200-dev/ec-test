# Implementation Patterns Reference

## Table of Contents
1. [Multi-Tenancy](#multi-tenancy)
2. [Transaction Handling](#transaction-handling)
3. [gRPC Adapter Pattern](#grpc-adapter-pattern)
4. [Outbound HTTP Clients](#outbound-http-clients)
5. [Pub/Sub Subscriber Pattern](#pubsub-subscriber-pattern)
6. [Domain Entity Behavior](#domain-entity-behavior)
7. [Port Interface Design](#port-interface-design)
8. [Composition Root (main.go)](#composition-root)

---

## Multi-Tenancy

Every database query is scoped to a `tenant_id` via PostgreSQL Row-Level Security (RLS).
The tenant ID flows:

```
JWT claim → pkg/middleware/auth.go → tenant.Context → ctx → repo.Method(ctx, tenantID, ...)
```

**Never** pass `tenantID` as a plain string — always use `uuid.UUID`.
**Always** pass `tenantID` as the first parameter after `ctx` in service and repo methods.

Extracting from context in a handler:
```go
tc, ok := tenant.FromContext(r.Context())
if !ok {
    httputil.Error(w, apperrors.Unauthorized("missing tenant context"))
    return
}
tenantID := tc.TenantID
```

The PostgreSQL `SET LOCAL app.current_tenant_id` is handled inside `pkg/database/tenant_scope.go`
before executing any query in a tenant-scoped connection or transaction.

---

## Transaction Handling

### The contract
- `pgx.Tx` **never** appears in `port/` or `app/` signatures.
- Transactions flow as context values.
- Repository methods use `database.QueryerFromContext(ctx, r.pool)` to get either the
  active transaction or the pool, making them work both inside and outside a transaction.

### TxRunner port
```go
// port/store.go
type TxRunner interface {
    RunTenantTx(ctx context.Context, tenantID uuid.UUID, fn func(ctx context.Context) error) error
}
```

### App layer usage
```go
// app/auth_service.go
func (s *AuthService) AddSellerUser(ctx context.Context, tenantID uuid.UUID, ...) error {
    return s.db.RunTenantTx(ctx, tenantID, func(txCtx context.Context) error {
        if err := s.sellerUsers.Create(txCtx, &user); err != nil {
            return err
        }
        return s.rbacAudit.Append(txCtx, &auditEntry)
        // txCtx carries the transaction; both repos use the same tx automatically
    })
}
```

### Adapter/postgres repository
```go
// adapter/postgres/user_repo.go
func (r *UserRepo) Create(ctx context.Context, u *domain.User) error {
    q := database.QueryerFromContext(ctx, r.pool)  // tx if in tx, pool otherwise
    return q.QueryRow(ctx, `INSERT INTO ...`, ...).Scan(&u.ID, ...)
}
```

---

## gRPC Adapter Pattern

### File layout
```
adapter/grpc/
  server.go      # implements the generated gRPC service interface
  convert.go     # proto ↔ domain type conversions
```

### server.go skeleton
```go
package grpcserver

type Server struct {
    catalogpb.UnimplementedCatalogServiceServer
    svc port.CatalogUseCase
}

func NewServer(svc port.CatalogUseCase) *Server {
    return &Server{svc: svc}
}

func (s *Server) GetProduct(ctx context.Context, req *catalogpb.GetProductRequest) (*catalogpb.GetProductResponse, error) {
    product, err := s.svc.GetProduct(ctx, uuid.MustParse(req.TenantId), uuid.MustParse(req.ProductId))
    if err != nil {
        return nil, toGRPCError(err)
    }
    return &catalogpb.GetProductResponse{Product: productToProto(product)}, nil
}

func toGRPCError(err error) error {
    var ae *apperrors.AppError
    if errors.As(err, &ae) {
        switch ae.Status {
        case http.StatusNotFound:      return status.Error(codes.NotFound, ae.Message)
        case http.StatusBadRequest:    return status.Error(codes.InvalidArgument, ae.Message)
        case http.StatusConflict:      return status.Error(codes.AlreadyExists, ae.Message)
        case http.StatusForbidden:     return status.Error(codes.PermissionDenied, ae.Message)
        case http.StatusUnauthorized:  return status.Error(codes.Unauthenticated, ae.Message)
        }
    }
    return status.Error(codes.Internal, "internal error")
}
```

### convert.go — keep conversions separate
```go
func productToProto(p *domain.Product) *catalogpb.Product { ... }
func protoToProductFilter(f *catalogpb.ProductFilter) domain.ProductFilter { ... }
```

### Starting gRPC server in main.go
```go
grpcSrv := grpc.NewServer(grpc.ChainUnaryInterceptor(
    middleware.TenantUnaryInterceptor(tenantJWTVerifier),
    middleware.LoggingUnaryInterceptor(),
))
catalogpb.RegisterCatalogServiceServer(grpcSrv, catalogGRPCServer)
go grpcSrv.Serve(grpcListener)
```

---

## Outbound HTTP Clients

Used for service-to-service calls when gRPC is not available (e.g., cart → catalog, inquiry → order).

### Location: `adapter/httpclient/{target}_client.go`
### Interface: defined in `port/store.go`

```go
// port/store.go
type SKULookupClient interface {
    LookupSKU(ctx context.Context, tenantID uuid.UUID, skuID uuid.UUID) (*SKULookup, error)
}

// SKULookup is a DTO in port/ to avoid circular imports between app/ and httpclient/
type SKULookup struct {
    SKUID         uuid.UUID
    SellerID      uuid.UUID
    PriceAmount   int64
    PriceCurrency string
    ProductName   string
    SKUCode       string
}
```

```go
// adapter/httpclient/catalog_client.go
type CatalogClient struct {
    baseURL       string
    internalToken string
    httpClient    *http.Client
}

func NewCatalogClient(baseURL, internalToken string) *CatalogClient {
    return &CatalogClient{
        baseURL:       baseURL,
        internalToken: internalToken,
        httpClient:    &http.Client{Timeout: 5 * time.Second},
    }
}

func (c *CatalogClient) LookupSKU(ctx context.Context, tenantID, skuID uuid.UUID) (*port.SKULookup, error) {
    req, _ := http.NewRequestWithContext(ctx, http.MethodGet,
        c.baseURL+"/internal/skus/"+skuID.String(), nil)
    req.Header.Set("X-Internal-Token", c.internalToken)
    req.Header.Set("X-Tenant-ID", tenantID.String())

    resp, err := c.httpClient.Do(req)
    if err != nil {
        return nil, apperrors.Internal("catalog lookup failed", err)
    }
    defer resp.Body.Close()

    if resp.StatusCode == http.StatusNotFound {
        return nil, apperrors.NotFound("sku not found")
    }
    if resp.StatusCode != http.StatusOK {
        return nil, apperrors.Internal("unexpected catalog status: "+resp.Status, nil)
    }
    var result port.SKULookup
    if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
        return nil, apperrors.Internal("decode catalog response", err)
    }
    return &result, nil
}
```

**Internal token validation** on the receiving service side:
```go
// Handler or middleware on the catalog service
if r.Header.Get("X-Internal-Token") != expectedToken {
    httputil.Error(w, apperrors.Unauthorized("invalid internal token"))
    return
}
```

---

## Pub/Sub Subscriber Pattern

### Location: `adapter/pubsub/{source}_subscriber.go`

```go
package subscriber

const orderEventsSubscription = "order-events-inventory"

type OrderSubscriber struct {
    subscriber pubsub.Subscriber
    svc        CancellationReleaser  // narrow interface, defined locally
}

func (s *OrderSubscriber) Start(ctx context.Context) error {
    return s.subscriber.Subscribe(ctx, orderEventsSubscription, s.handleEvent)
}

func (s *OrderSubscriber) handleEvent(ctx context.Context, event pubsub.Event) error {
    switch event.Type {
    case "order.cancelled":
        return s.handleOrderCancelled(ctx, event)
    default:
        return nil  // unknown events: ack silently (don't nack — prevents infinite redelivery)
    }
}
```

### Decoding event data
`event.Data` comes back as `map[string]any` from JSON. Round-trip through JSON to decode
into a typed struct:

```go
type orderCancelledData struct {
    OrderID   string          `json:"order_id"`
    TenantID  string          `json:"tenant_id"`
    LineItems []cancelledLine `json:"line_items"`
}

func decodeEventData(eventData any, target any) error {
    raw, _ := json.Marshal(eventData)
    return json.Unmarshal(raw, target)
}
```

### Dependency injection in main.go
```go
orderSub := ordersubscriber.NewOrderSubscriber(gcpSubscriber, inventorySvc)
go func() {
    if err := orderSub.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
        slog.Error("order subscriber stopped", "error", err)
    }
}()
```

**Idempotency**: Subscribers must be idempotent. Use a unique guard row (e.g., a `movement` row
keyed on `order_id`) to skip already-processed events. Nack on decode errors (triggers redelivery),
ack on successfully ignored events (e.g., unknown event type, or a line with invalid UUID after logging).

---

## Domain Entity Behavior

Move mutation logic into the domain when it reduces app-layer boilerplate and the operation
is a pure business rule with no infrastructure dependencies.

Good candidates:
- Status guard predicates: `order.CanBeCancelled() bool`
- In-place mutations: `cart.AddItem()`, `cart.RemoveItem()`, `cart.SetItemQuantity()`
- Derived values: `cart.Total() int64`, `cart.IsEmpty() bool`

**Do not** move anything into the domain that needs a repo call, HTTP call, or clock.
The domain is pure — it receives already-loaded data and returns new state or errors.

```go
// domain/cart.go
func (c *Cart) AddItem(item CartItem) {
    if idx := c.FindItem(item.SKUID); idx >= 0 {
        c.Items[idx].Quantity += item.Quantity
    } else {
        c.Items = append(c.Items, item)
    }
    c.UpdatedAt = time.Now().UTC()
}

func (c *Cart) SetItemQuantity(skuID uuid.UUID, quantity int) error {
    idx := c.FindItem(skuID)
    if idx < 0 {
        return ErrSKUNotInCart
    }
    c.Items[idx].Quantity = quantity
    c.UpdatedAt = time.Now().UTC()
    return nil
}
```

---

## Port Interface Design

### Driving port (`port/service.go`) — what handlers call
- One `XxxUseCase` interface per service
- All methods take `ctx context.Context` + `tenantID uuid.UUID` as first two params
- Return domain types, not proto types or HTTP response structs

### Driven ports (`port/store.go`) — what app needs from infrastructure
- Separate interfaces per resource: `OrderStore`, `PayoutStore`, `StripePayments`
- Each interface is narrow: only the methods actually used
- `TxRunner` always goes here if transactions are needed

### DTO placement rule
If a type is returned by an httpclient adapter AND consumed by the app layer, put it in `port/`
to avoid circular imports. Example: `port.SKULookup`, `port.PurchaseCheckResult`.

---

## Composition Root (main.go)

`cmd/server/main.go` is the **only** file that knows about all layers. Pattern:

```go
func main() {
    cfg := config.Load()

    // 1. Infrastructure
    pool := database.NewPool(cfg.DatabaseURL)
    publisher, _ := pubsub.NewGCPPublisher(ctx, cfg.ProjectID)
    stripeClient := stripe.NewClient(cfg.StripeKey)
    redisClient := redis.NewClient(cfg.RedisURL)

    // 2. Adapters (postgres)
    orderRepo := postgres.NewOrderRepo(pool)
    payoutRepo := postgres.NewPayoutRepo(pool)

    // 3. HTTP clients
    buyerSubClient := httpclient.NewBuyerSubscriptionClient(cfg.AuthServiceURL, cfg.AuthInternalToken)

    // 4. App layer
    orderSvc := app.NewOrderService(orderRepo, payoutRepo, stripeClient, publisher, buyerSubClient, cfg.DefaultShippingFee)

    // 5. Adapters (http + gRPC)
    orderHandler := handler.NewOrderHandler(orderSvc)
    grpcServer := grpcserver.NewServer(orderSvc)

    // 6. Start servers
    r := chi.NewRouter()
    r.Use(middleware.Auth(jwksURL, audience))
    orderHandler.RegisterRoutes(r)
    go grpcSrv.Serve(grpcListener)
    http.ListenAndServe(cfg.Addr, r)
}
```
