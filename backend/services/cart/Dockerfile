FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and all module files for dependency resolution.
COPY go.work go.work
COPY backend/pkg/go.mod backend/pkg/go.sum ./backend/pkg/
COPY backend/services/cart/go.mod backend/services/cart/go.sum ./backend/services/cart/

# Download dependencies.
WORKDIR /src/backend/services/cart
RUN go mod download

# Copy source code.
WORKDIR /src
COPY backend/pkg/ ./backend/pkg/
COPY backend/services/cart/ ./backend/services/cart/

# Build the binary.
WORKDIR /src/backend/services/cart
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/cart-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/cart-server /bin/cart-server

EXPOSE 8088

ENTRYPOINT ["/bin/cart-server"]
