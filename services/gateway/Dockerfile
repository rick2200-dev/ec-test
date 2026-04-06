FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy go.work and module files for dependency caching
COPY go.work go.work
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/gateway/go.mod services/gateway/go.sum* ./services/gateway/

# Download dependencies
RUN cd services/gateway && go mod download

# Copy source
COPY pkg/ ./pkg/
COPY services/gateway/ ./services/gateway/

# Build
RUN cd services/gateway && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/gateway ./cmd/server

# --- Runtime ---
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app && adduser -S app -G app

COPY --from=builder /bin/gateway /bin/gateway

USER app

EXPOSE 8080

ENTRYPOINT ["/bin/gateway"]
