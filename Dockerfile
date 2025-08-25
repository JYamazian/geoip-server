# Multi-stage build for smaller final image
FROM golang:1.21-alpine AS builder

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Set working directory
WORKDIR /app

# Copy go mod file
COPY go.mod ./

# Initialize and download dependencies
RUN go mod tidy && go mod download

# Copy source code
COPY src/ ./

# Ensure dependencies are up to date and build
RUN go mod tidy && CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o geoip-server .

# Final stage with init container support
FROM alpine:latest

# Install curl and bash for health checks only
RUN apk --no-cache add ca-certificates curl

WORKDIR /app

# Copy the binary from builder stage
COPY --from=builder /app/geoip-server .

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD curl -f http://localhost:8080/health || exit 1

# Default command - start server
CMD ["./geoip-server"]
