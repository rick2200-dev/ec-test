FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and module files for dependency resolution.
COPY go.work go.work
COPY backend/pkg/go.mod backend/pkg/go.sum ./backend/pkg/
COPY backend/services/inquiry/go.mod backend/services/inquiry/go.sum ./backend/services/inquiry/

WORKDIR /src/backend/services/inquiry
RUN go mod download

WORKDIR /src
COPY backend/pkg/ ./backend/pkg/
COPY backend/services/inquiry/ ./backend/services/inquiry/

WORKDIR /src/backend/services/inquiry
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/inquiry-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/inquiry-server /bin/inquiry-server

EXPOSE 8090

ENTRYPOINT ["/bin/inquiry-server"]
