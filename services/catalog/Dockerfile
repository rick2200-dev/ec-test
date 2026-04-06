FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and all module files for dependency resolution.
COPY go.work go.work
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/catalog/go.mod services/catalog/go.sum ./services/catalog/

# Download dependencies.
WORKDIR /src/services/catalog
RUN go mod download

# Copy source code.
WORKDIR /src
COPY pkg/ ./pkg/
COPY services/catalog/ ./services/catalog/

# Build the binary.
WORKDIR /src/services/catalog
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/catalog-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/catalog-server /bin/catalog-server

EXPOSE 8082 50052

ENTRYPOINT ["/bin/catalog-server"]
