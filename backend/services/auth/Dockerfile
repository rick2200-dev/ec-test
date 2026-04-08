FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and all module files for dependency resolution.
COPY go.work go.work
COPY backend/pkg/go.mod backend/pkg/go.sum ./backend/pkg/
COPY backend/services/auth/go.mod backend/services/auth/go.sum ./backend/services/auth/

# Download dependencies.
WORKDIR /src/backend/services/auth
RUN go mod download

# Copy source code.
WORKDIR /src
COPY backend/pkg/ ./backend/pkg/
COPY backend/services/auth/ ./backend/services/auth/

# Build the binary.
WORKDIR /src/backend/services/auth
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/auth-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/auth-server /bin/auth-server

EXPOSE 8081 50051

ENTRYPOINT ["/bin/auth-server"]
