FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and all module files for dependency resolution.
COPY go.work go.work
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/search/go.mod services/search/go.sum ./services/search/

# Download dependencies.
WORKDIR /src/services/search
RUN go mod download

# Copy source code.
WORKDIR /src
COPY pkg/ ./pkg/
COPY services/search/ ./services/search/

# Build the binary.
WORKDIR /src/services/search
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/search-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/search-server /bin/search-server

EXPOSE 8085

ENTRYPOINT ["/bin/search-server"]
