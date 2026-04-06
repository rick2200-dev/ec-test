FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and all module files for dependency resolution.
COPY go.work go.work
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/notification/go.mod services/notification/go.sum ./services/notification/

# Download dependencies.
WORKDIR /src/services/notification
RUN go mod download

# Copy source code.
WORKDIR /src
COPY pkg/ ./pkg/
COPY services/notification/ ./services/notification/

# Build the binary.
WORKDIR /src/services/notification
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/notification-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/notification-server /bin/notification-server

EXPOSE 8087

ENTRYPOINT ["/bin/notification-server"]
