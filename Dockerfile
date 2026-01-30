# Builder stage
FROM golang:1.25-alpine@sha256:ac09a5f469f307e5da71e766b0bd59c9c49ea460a528cc3e6686513d64a6f1fb AS builder

LABEL io.modelcontextprotocol.server.name="io.github.brunogc-cit/flow-microstrategy-mcp"

WORKDIR /build

# Install CA certificates
RUN apk add --no-cache ca-certificates

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build -C cmd/flow-microstrategy-mcp -a -installsuffix cgo \
    -o ../../flow-microstrategy-mcp

# Runtime stage
FROM scratch

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/flow-microstrategy-mcp /app/flow-microstrategy-mcp

# Copy CA certificates for HTTPS connections
COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/ca-certificates.crt

# Run as non-root user (UID 65532 is a standard non-root user ID)
USER 65532:65532

# Set entrypoint
ENTRYPOINT ["/app/flow-microstrategy-mcp"]
