FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and all module files for dependency resolution.
COPY go.work go.work
COPY backend/pkg/go.mod backend/pkg/go.sum ./backend/pkg/
COPY backend/services/order/go.mod backend/services/order/go.sum ./backend/services/order/

# Download dependencies.
WORKDIR /src/backend/services/order
RUN go mod download

# Copy source code.
WORKDIR /src
COPY backend/pkg/ ./backend/pkg/
COPY backend/services/order/ ./backend/services/order/

# Build the binary.
WORKDIR /src/backend/services/order
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/order-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/order-server /bin/order-server

EXPOSE 8083

ENTRYPOINT ["/bin/order-server"]
