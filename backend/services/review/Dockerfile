FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and module files for dependency resolution.
COPY go.work go.work
COPY backend/pkg/go.mod backend/pkg/go.sum ./backend/pkg/
COPY backend/services/review/go.mod backend/services/review/go.sum ./backend/services/review/

WORKDIR /src/backend/services/review
RUN go mod download

WORKDIR /src
COPY backend/pkg/ ./backend/pkg/
COPY backend/services/review/ ./backend/services/review/

WORKDIR /src/backend/services/review
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/review-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/review-server /bin/review-server

EXPOSE 8091

ENTRYPOINT ["/bin/review-server"]
