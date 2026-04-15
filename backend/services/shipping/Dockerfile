FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and module files for dependency resolution.
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
COPY backend/services/gateway/go.mod backend/services/gateway/go.sum ./backend/services/gateway/
COPY backend/services/inquiry/go.mod backend/services/inquiry/go.sum ./backend/services/inquiry/
COPY backend/services/inventory/go.mod backend/services/inventory/go.sum ./backend/services/inventory/
COPY backend/services/notification/go.mod backend/services/notification/go.sum ./backend/services/notification/
COPY backend/services/order/go.mod backend/services/order/go.sum ./backend/services/order/
COPY backend/services/recommend/go.mod backend/services/recommend/go.sum ./backend/services/recommend/
COPY backend/services/review/go.mod backend/services/review/go.sum ./backend/services/review/
COPY backend/services/search/go.mod backend/services/search/go.sum ./backend/services/search/
COPY backend/services/shipping/go.mod backend/services/shipping/go.sum ./backend/services/shipping/
COPY backend/services/subscription/go.mod backend/services/subscription/go.sum ./backend/services/subscription/

WORKDIR /src/backend/services/shipping
RUN go mod download

WORKDIR /src
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
COPY backend/services/shipping/ ./backend/services/shipping/

WORKDIR /src/backend/services/shipping
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/shipping-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/shipping-server /bin/shipping-server

EXPOSE 8092

ENTRYPOINT ["/bin/shipping-server"]
