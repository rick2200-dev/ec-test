FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and all module files for dependency resolution.
COPY go.work go.work
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/order/go.mod services/order/go.sum ./services/order/

# Download dependencies.
WORKDIR /src/services/order
RUN go mod download

# Copy source code.
WORKDIR /src
COPY pkg/ ./pkg/
COPY services/order/ ./services/order/

# Build the binary.
WORKDIR /src/services/order
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/order-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/order-server /bin/order-server

EXPOSE 8083

ENTRYPOINT ["/bin/order-server"]
