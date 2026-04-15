FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src

# Copy go.work and module files for dependency caching
COPY go.work go.work
COPY backend/pkg/go.mod backend/pkg/go.sum ./backend/pkg/
COPY backend/shared/api/gen/go/go.mod ./backend/shared/api/gen/go/
COPY backend/services/cart/api/gen/go/go.mod ./backend/services/cart/api/gen/go/
COPY backend/services/catalog/api/gen/go/go.mod ./backend/services/catalog/api/gen/go/
COPY backend/services/inquiry/api/gen/go/go.mod ./backend/services/inquiry/api/gen/go/
COPY backend/services/inventory/api/gen/go/go.mod ./backend/services/inventory/api/gen/go/
COPY backend/services/order/api/gen/go/go.mod ./backend/services/order/api/gen/go/
COPY backend/services/review/api/gen/go/go.mod ./backend/services/review/api/gen/go/
COPY backend/services/shipping/api/gen/go/go.mod ./backend/services/shipping/api/gen/go/
COPY backend/services/subscription/api/gen/go/go.mod ./backend/services/subscription/api/gen/go/
COPY backend/services/auth/go.mod backend/services/auth/go.sum ./backend/services/auth/
COPY backend/services/cart/go.mod backend/services/cart/go.sum ./backend/services/cart/
COPY backend/services/catalog/go.mod backend/services/catalog/go.sum ./backend/services/catalog/
COPY backend/services/gateway/go.mod backend/services/gateway/go.sum* ./backend/services/gateway/
COPY backend/services/inquiry/go.mod backend/services/inquiry/go.sum ./backend/services/inquiry/
COPY backend/services/inventory/go.mod backend/services/inventory/go.sum ./backend/services/inventory/
COPY backend/services/notification/go.mod backend/services/notification/go.sum ./backend/services/notification/
COPY backend/services/order/go.mod backend/services/order/go.sum ./backend/services/order/
COPY backend/services/recommend/go.mod backend/services/recommend/go.sum ./backend/services/recommend/
COPY backend/services/review/go.mod backend/services/review/go.sum ./backend/services/review/
COPY backend/services/search/go.mod backend/services/search/go.sum ./backend/services/search/
COPY backend/services/subscription/go.mod backend/services/subscription/go.sum ./backend/services/subscription/

# Download dependencies
RUN cd backend/services/gateway && go mod download

# Copy source
COPY backend/pkg/ ./backend/pkg/
COPY backend/shared/api/gen/go/ ./backend/shared/api/gen/go/
COPY backend/services/cart/api/gen/go/ ./backend/services/cart/api/gen/go/
COPY backend/services/catalog/api/gen/go/ ./backend/services/catalog/api/gen/go/
COPY backend/services/inquiry/api/gen/go/ ./backend/services/inquiry/api/gen/go/
COPY backend/services/inventory/api/gen/go/ ./backend/services/inventory/api/gen/go/
COPY backend/services/order/api/gen/go/ ./backend/services/order/api/gen/go/
COPY backend/services/review/api/gen/go/ ./backend/services/review/api/gen/go/
COPY backend/services/shipping/api/gen/go/ ./backend/services/shipping/api/gen/go/
COPY backend/services/subscription/api/gen/go/ ./backend/services/subscription/api/gen/go/
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
