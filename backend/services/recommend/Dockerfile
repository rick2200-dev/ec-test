FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and all module files for dependency resolution.
COPY go.work go.work
COPY backend/pkg/go.mod backend/pkg/go.sum ./backend/pkg/
COPY backend/services/recommend/go.mod backend/services/recommend/go.sum ./backend/services/recommend/

# Download dependencies.
WORKDIR /src/backend/services/recommend
RUN go mod download

# Copy source code.
WORKDIR /src
COPY backend/pkg/ ./backend/pkg/
COPY backend/services/recommend/ ./backend/services/recommend/

# Build the binary.
WORKDIR /src/backend/services/recommend
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/recommend-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/recommend-server /bin/recommend-server

EXPOSE 8086

ENTRYPOINT ["/bin/recommend-server"]
