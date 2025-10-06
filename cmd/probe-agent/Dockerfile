# Multi-stage build for probe-agent
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod files
COPY go.mod go.sum ./

# Download dependencies
RUN go mod download

# Copy source code
COPY . .

# Build the probe-agent
RUN CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o probe-agent ./cmd/probe-agent

# Final stage
FROM alpine:latest

# Install runtime dependencies
RUN apk --no-cache add ca-certificates tzdata procps

# Create non-root user
RUN addgroup -g 1001 -S probe && \
    adduser -u 1001 -S probe -G probe

# Set working directory
WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/probe-agent .

# Change ownership to non-root user
USER probe

# Expose port (if needed for health checks)
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD pgrep probe-agent || exit 1

# Run the probe-agent
CMD ["./probe-agent"]
