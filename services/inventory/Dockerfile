FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and all module files for dependency resolution.
COPY go.work go.work
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/inventory/go.mod services/inventory/go.sum ./services/inventory/

# Download dependencies.
WORKDIR /src/services/inventory
RUN go mod download

# Copy source code.
WORKDIR /src
COPY pkg/ ./pkg/
COPY services/inventory/ ./services/inventory/

# Build the binary.
WORKDIR /src/services/inventory
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/inventory-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/inventory-server /bin/inventory-server

EXPOSE 8084 50054

ENTRYPOINT ["/bin/inventory-server"]
