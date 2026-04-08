FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy go.work and module files for dependency caching
COPY go.work go.work
COPY backend/pkg/go.mod backend/pkg/go.sum ./backend/pkg/
COPY backend/services/gateway/go.mod backend/services/gateway/go.sum* ./backend/services/gateway/

# Download dependencies
RUN cd backend/services/gateway && go mod download

# Copy source
COPY backend/pkg/ ./backend/pkg/
COPY backend/services/gateway/ ./backend/services/gateway/

# Build
RUN cd backend/services/gateway && CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /bin/gateway ./cmd/server

# --- Runtime ---
FROM alpine:3.21

RUN apk add --no-cache ca-certificates tzdata \
    && addgroup -S app && adduser -S app -G app

COPY --from=builder /bin/gateway /bin/gateway

USER app

EXPOSE 8080

ENTRYPOINT ["/bin/gateway"]
