---
name: grpc-integration
description: |
  Streamlines adding gRPC to a Go microservice in this monorepo. Use this skill whenever the user wants to:
  add a new gRPC service, add RPCs to an existing service, create proto definitions, generate Go code from protos,
  implement gRPC servers or clients, or connect services via gRPC. Also trigger when the user mentions "proto",
  "buf generate", "grpc server", "grpc client", or "protobuf" in the context of this project.
---

# gRPC Integration Skill

This skill captures the end-to-end workflow for adding or extending gRPC in a Go microservices monorepo that uses buf, Go workspaces, and a gateway BFF pattern.

The monorepo has this relevant structure:

```
proto/                          # Proto definitions (buf-managed)
  buf.yaml                     # Lint: STANDARD, Breaking: FILE
  buf.gen.yaml                 # Local plugins → ../gen/go
  common/v1/common.proto       # Shared types (Money, Pagination, TenantContext)
  {service}/v1/{service}.proto # Per-service definitions
gen/go/                        # Generated Go code (its own module)
  {service}/v1/*.pb.go, *_grpc.pb.go
services/{service}/
  cmd/server/main.go
  internal/grpcserver/         # gRPC server implementation
    server.go                  # RPC handlers wrapping service layer
    convert.go                 # Domain <-> Proto conversions
services/gateway/
  internal/grpcclient/         # gRPC client wrappers
    clients.go                 # Connection management
    {service}.go               # Per-service helper methods
```

## Workflow

When the user asks to add gRPC for a service, follow these steps in order. Do NOT skip steps or parallelize steps that depend on earlier ones.

### Step 1: Define or Update Proto

Create/edit `proto/{service}/v1/{service}.proto`. Follow these conventions:

- `syntax = "proto3";`
- `package {service}.v1;`
- `option go_package = "github.com/Riku-KANO/ec-test/gen/go/{service}/v1;{service}v1";`
- Import shared types: `import "common/v1/common.proto";`
- Import timestamps if needed: `import "google/protobuf/timestamp.proto";`
- Group the file into sections: service definition, domain messages, then request/response pairs per RPC
- Use `common.v1.Money` for monetary amounts, `common.v1.PaginationRequest/Response` for pagination
- Each RPC gets its own `{RPC}Request` and `{RPC}Response` message — never reuse response messages across RPCs

### Step 2: Lint Protos

Run `buf lint` from the `proto/` directory. Fix any issues before proceeding:

```bash
cd proto && buf lint
```

Common fixes:

- Remove unused imports (e.g., `google/protobuf/timestamp.proto` if no Timestamp fields)
- Ensure each RPC has a unique response message type (don't share responses between RPCs)
- Follow STANDARD lint rules (field names snake_case, enum values UPPER_SNAKE_CASE with type prefix)

### Step 3: Generate Go Code

```bash
cd proto && buf generate
```

This produces `gen/go/{service}/v1/{service}.pb.go` and `{service}_grpc.pb.go`.

After generation, ensure `gen/go/go.mod` exists. If this is the first time generating for a new service, run:

```bash
cd gen/go && go mod tidy
```

Also verify `gen/go` is listed in `go.work`. If not, add it.

### Step 4: Update Service go.mod

The service that will implement the gRPC server needs these replace directives in its `go.mod`:

```
replace github.com/Riku-KANO/ec-test/gen/go => ../../gen/go
replace github.com/Riku-KANO/ec-test/pkg => ../../pkg
```

Then run `go mod tidy` in the service directory. Also run `go mod tidy` in the gateway if adding a new client.

### Step 5: Implement gRPC Server

Create `services/{service}/internal/grpcserver/server.go`:

```go
package grpcserver

import (
    "context"

    {service}v1 "github.com/Riku-KANO/ec-test/gen/go/{service}/v1"
    "github.com/Riku-KANO/ec-test/services/{service}/internal/service"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type {Service}Server struct {
    {service}v1.Unimplemented{Service}ServiceServer
    svc *service.{Service}Service
}

func New{Service}Server(svc *service.{Service}Service) *{Service}Server {
    return &{Service}Server{svc: svc}
}

// Implement each RPC method:
// 1. Parse/validate request fields (UUIDs via uuid.Parse, pagination defaults)
// 2. Call service layer
// 3. Convert domain result to proto via conversion functions
// 4. Return proto response or toGRPCError(err)
```

Create `services/{service}/internal/grpcserver/convert.go` for domain-to-proto and proto-to-domain conversions:

- Use `timestamppb.New(t)` for `time.Time` -> `google.protobuf.Timestamp`
- Use `common.v1.Money{Amount: m.Amount, Currency: m.Currency}` for money fields
- Name functions `domain{Type}ToProto` and `proto{Type}ToDomain`

Error mapping function (`toGRPCError`):

```go
func toGRPCError(err error) error {
    var appErr *apperrors.AppError
    if errors.As(err, &appErr) {
        switch appErr.Code {
        case apperrors.ErrNotFound:
            return status.Error(codes.NotFound, appErr.Message)
        case apperrors.ErrBadRequest:
            return status.Error(codes.InvalidArgument, appErr.Message)
        case apperrors.ErrConflict:
            return status.Error(codes.AlreadyExists, appErr.Message)
        case apperrors.ErrForbidden:
            return status.Error(codes.PermissionDenied, appErr.Message)
        case apperrors.ErrUnauthorized:
            return status.Error(codes.Unauthenticated, appErr.Message)
        }
    }
    return status.Error(codes.Internal, "internal error")
}
```

### Step 6: Wire gRPC Server into main.go

Add to `services/{service}/cmd/server/main.go`:

```go
import (
    "net"
    "google.golang.org/grpc"
    {service}v1 "github.com/Riku-KANO/ec-test/gen/go/{service}/v1"
    "github.com/Riku-KANO/ec-test/services/{service}/internal/grpcserver"
)

// After creating the service layer instance:
grpcAddr := ":" + cfg.GRPCPort
grpcListener, err := net.Listen("tcp", grpcAddr)
if err != nil {
    slog.Error("failed to listen for gRPC", "error", err)
    os.Exit(1)
}

grpcSrv := grpc.NewServer()
{service}v1.Register{Service}ServiceServer(grpcSrv, grpcserver.New{Service}Server({service}Svc))

go func() {
    slog.Info("starting {service} gRPC server", "addr", grpcAddr)
    if err := grpcSrv.Serve(grpcListener); err != nil {
        slog.Error("gRPC server error", "error", err)
        os.Exit(1)
    }
}()

// In shutdown section:
grpcSrv.GracefulStop()
```

Ensure `GRPCPort` is added to the service's config struct if it doesn't exist.

### Step 7: Implement gRPC Client in Gateway (if needed)

If the gateway needs to call this service, update `services/gateway/internal/grpcclient/`:

1. **clients.go** — add connection and typed client fields to `GRPCClients`:

   ```go
   {service}Conn   *grpc.ClientConn
   {Service}Client {service}v1.{Service}ServiceClient
   ```

2. **{service}.go** — add wrapper methods that construct requests and call the client:

   ```go
   func (c *GRPCClients) {RPC}(ctx context.Context, ...) (*{service}v1.{RPC}Response, error) {
       return c.{Service}Client.{RPC}(ctx, &{service}v1.{RPC}Request{...})
   }
   ```

3. Add the service's gRPC address to gateway config (e.g., `{Service}GRPCAddr`).

### Step 8: Verify Build

```bash
go build ./services/{service}/...
go build ./services/gateway/...
```

If using Go workspaces, you can also run from the repo root:

```bash
go build ./...
```

## Port Convention

| Service      | HTTP | gRPC  |
| ------------ | ---- | ----- |
| gateway      | 8080 | —     |
| auth         | 8081 | —     |
| catalog      | 8082 | 50052 |
| order        | 8083 | 50053 |
| inventory    | 8084 | 50054 |
| search       | 8085 | 50055 |
| recommend    | 8086 | 50056 |
| notification | 8087 | 50057 |

When adding a new gRPC service, pick the next available gRPC port following this pattern.

## Checklist

Before marking gRPC integration complete, verify:

- [ ] Proto file lints cleanly (`buf lint`)
- [ ] Code generates without errors (`buf generate`)
- [ ] `gen/go/go.mod` is tidy
- [ ] Service `go.mod` has replace directives for gen/go and pkg
- [ ] gRPC server embeds `Unimplemented*Server` for forward compatibility
- [ ] Error mapping covers all AppError codes
- [ ] main.go starts gRPC on its own port in a goroutine
- [ ] main.go calls `GracefulStop()` on shutdown
- [ ] Gateway client (if needed) has connection, typed client, and wrapper methods
- [ ] `go build` passes for the service and gateway
