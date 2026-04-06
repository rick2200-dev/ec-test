FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and all module files for dependency resolution.
COPY go.work go.work
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/auth/go.mod services/auth/go.sum ./services/auth/

# Download dependencies.
WORKDIR /src/services/auth
RUN go mod download

# Copy source code.
WORKDIR /src
COPY pkg/ ./pkg/
COPY services/auth/ ./services/auth/

# Build the binary.
WORKDIR /src/services/auth
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/auth-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/auth-server /bin/auth-server

EXPOSE 8081 50051

ENTRYPOINT ["/bin/auth-server"]
