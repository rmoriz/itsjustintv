# Build stage
FROM golang:1.24.5-alpine AS builder

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

# Build arguments
ARG VERSION=dev
ARG COMMIT=unknown
ARG BUILD_DATE=unknown

# Build the binary
RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags "-s -w -X github.com/rmoriz/itsjustintv/internal/cli.Version=${VERSION} -X github.com/rmoriz/itsjustintv/internal/cli.GitCommit=${COMMIT} -X github.com/rmoriz/itsjustintv/internal/cli.BuildDate=${BUILD_DATE}" \
    -o itsjustintv ./cmd/itsjustintv

# Final stage
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Create non-root user
RUN addgroup -g 1001 -S itsjustintv && \
    adduser -u 1001 -S itsjustintv -G itsjustintv

# Create data directory
RUN mkdir -p /app/data && \
    chown -R itsjustintv:itsjustintv /app

# Copy binary from builder
COPY --from=builder /app/itsjustintv /usr/local/bin/itsjustintv

# Set working directory
WORKDIR /app

# Switch to non-root user
USER itsjustintv

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["itsjustintv"]