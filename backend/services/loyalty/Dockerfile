FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

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
COPY backend/services/coupon/api/gen/go/go.mod ./backend/services/coupon/api/gen/go/
COPY backend/services/loyalty/api/gen/go/go.mod ./backend/services/loyalty/api/gen/go/
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
COPY backend/services/subscription/go.mod backend/services/subscription/go.sum ./backend/services/subscription/
COPY backend/services/shipping/go.mod backend/services/shipping/go.sum ./backend/services/shipping/
COPY backend/services/loyalty/go.mod ./backend/services/loyalty/
COPY backend/services/coupon/go.mod ./backend/services/coupon/

WORKDIR /src/backend/services/loyalty
RUN go mod download

WORKDIR /src
COPY backend/pkg/ ./backend/pkg/
COPY backend/shared/api/gen/go/ ./backend/shared/api/gen/go/
COPY backend/services/loyalty/api/gen/go/ ./backend/services/loyalty/api/gen/go/
COPY backend/services/loyalty/ ./backend/services/loyalty/

WORKDIR /src/backend/services/loyalty
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/loyalty-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/loyalty-server /bin/loyalty-server

EXPOSE 8094 50061

ENTRYPOINT ["/bin/loyalty-server"]
