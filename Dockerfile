# Final stage
FROM alpine:3.20

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata wget

# Create non-root user
RUN addgroup -g 1001 -S itsjustintv && 
    adduser -u 1001 -S itsjustintv -G itsjustintv

# Create data directory
RUN mkdir -p /app/data && 
    chown -R itsjustintv:itsjustintv /app

# Build arguments
ARG TARGETARCH

# Copy binary from build context
COPY itsjustintv-linux-${TARGETARCH} /usr/local/bin/itsjustintv

# Set working directory
WORKDIR /app

# Switch to non-root user
USER itsjustintv

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["itsjustintv"]

# Default command
CMD ["server"]

