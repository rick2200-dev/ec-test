FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and all module files for dependency resolution.
COPY go.work go.work
COPY backend/pkg/go.mod backend/pkg/go.sum ./backend/pkg/
COPY backend/services/notification/go.mod backend/services/notification/go.sum ./backend/services/notification/

# Download dependencies.
WORKDIR /src/backend/services/notification
RUN go mod download

# Copy source code.
WORKDIR /src
COPY backend/pkg/ ./backend/pkg/
COPY backend/services/notification/ ./backend/services/notification/

# Build the binary.
WORKDIR /src/backend/services/notification
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/notification-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/notification-server /bin/notification-server

EXPOSE 8087

ENTRYPOINT ["/bin/notification-server"]
