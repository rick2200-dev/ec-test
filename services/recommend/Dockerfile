FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git

WORKDIR /src

# Copy go.work and all module files for dependency resolution.
COPY go.work go.work
COPY pkg/go.mod pkg/go.sum ./pkg/
COPY services/recommend/go.mod services/recommend/go.sum ./services/recommend/

# Download dependencies.
WORKDIR /src/services/recommend
RUN go mod download

# Copy source code.
WORKDIR /src
COPY pkg/ ./pkg/
COPY services/recommend/ ./services/recommend/

# Build the binary.
WORKDIR /src/services/recommend
RUN CGO_ENABLED=0 GOOS=linux go build -o /bin/recommend-server ./cmd/server

# ---
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

COPY --from=builder /bin/recommend-server /bin/recommend-server

EXPOSE 8086

ENTRYPOINT ["/bin/recommend-server"]
