FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and all module files for dependency resolution.
COPY go.work go.work
COPY backend/pkg/go.mod backend/pkg/go.sum ./backend/pkg/
COPY backend/services/search/go.mod backend/services/search/go.sum ./backend/services/search/

# Download dependencies.
WORKDIR /src/backend/services/search
RUN go mod download

# Copy source code.
WORKDIR /src
COPY backend/pkg/ ./backend/pkg/
COPY backend/services/search/ ./backend/services/search/

# Build the binary.
WORKDIR /src/backend/services/search
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/search-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/search-server /bin/search-server

EXPOSE 8085

ENTRYPOINT ["/bin/search-server"]
