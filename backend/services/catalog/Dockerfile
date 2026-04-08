FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and all module files for dependency resolution.
COPY go.work go.work
COPY backend/pkg/go.mod backend/pkg/go.sum ./backend/pkg/
COPY backend/services/catalog/go.mod backend/services/catalog/go.sum ./backend/services/catalog/

# Download dependencies.
WORKDIR /src/backend/services/catalog
RUN go mod download

# Copy source code.
WORKDIR /src
COPY backend/pkg/ ./backend/pkg/
COPY backend/services/catalog/ ./backend/services/catalog/

# Build the binary.
WORKDIR /src/backend/services/catalog
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/catalog-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/catalog-server /bin/catalog-server

EXPOSE 8082 50052

ENTRYPOINT ["/bin/catalog-server"]
