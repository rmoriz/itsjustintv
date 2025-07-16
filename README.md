# itsjustintv

![itsjustintv Logo](docs/itsjustintv-logo.png)

[![Go Version](https://img.shields.io/badge/Go-1.24.5+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/github/v/release/rmoriz/itsjustintv)](https://github.com/rmoriz/itsjustintv/releases)
[![Docker](https://img.shields.io/badge/Docker-Available-2496ED?style=flat&logo=docker)](https://github.com/rmoriz/itsjustintv/pkgs/container/itsjustintv)

A **production-ready**, self-hosted Go service that bridges Twitch EventSub webhooks with your notification systems. Get instant, reliable notifications when your favorite streamers go live with rich metadata and flexible delivery options.

## Table of Contents

- [Key Features](#key-features)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Webhook Payload](#webhook-payload)
- [CLI Commands](#cli-commands)
- [Development](#development)
- [Docker](#docker)
- [Architecture](#architecture)
- [Contributing](#contributing)

## Key Features

### Core Functionality
- **Real-time Stream Notifications**: Instant notifications via Twitch EventSub webhooks
- **Automatic User ID Resolution**: Configure streamers with just their login name - user IDs are resolved automatically
- **Smart Webhook Dispatching**: Send notifications to multiple endpoints with custom payloads
- **Rich Metadata Enrichment**: Automatically fetch streamer info (follower count, profile images, descriptions)

### Reliability & Performance
- **Robust Retry Logic**: Exponential backoff for failed webhook deliveries with persistent state
- **Duplicate Detection**: Built-in deduplication prevents spam notifications
- **HMAC Signature Validation**: Secure webhook verification and optional payload signing
- **Graceful Error Handling**: Continues operation even when external services fail

### Operations & Monitoring
- **HTTPS Support**: Let's Encrypt integration for secure webhook endpoints
- **OpenTelemetry Integration**: Built-in observability with metrics and distributed tracing
- **File Output**: JSON logging for debugging, archival, and integration testing
- **Health Checks**: Built-in health endpoints for monitoring and load balancers

### Advanced Filtering
- **Tag-based Filtering**: Only notify for streams with specific tags (language, category, etc.)
- **Custom Tag Injection**: Add your own tags to webhook payloads for downstream processing
- **Per-streamer Configuration**: Different webhook URLs and settings for each streamer

## Quick Start

### Installation

#### Using Just (Recommended for Development)

This project includes a `justfile` for easy development workflow management. [Just](https://github.com/casey/just) is a command runner similar to `make` but with a more modern syntax.

**Install Just:**

```bash
# macOS
brew install just

# Linux/macOS (using installer script)
curl --proto '=https' --tlsv1.2 -sSf https://just.systems/install.sh | bash -s -- --to ~/.local/bin

# Or download from releases: https://github.com/casey/just/releases
```

**Common development commands:**

```bash
# Show all available commands
just

# Build the application
just build

# Run with example config
just run

# Run tests
just test

# Run tests with coverage
just test-coverage

# Format code and run linter
just check

# Quick development cycle (format, test, build, run)
just dev

# Build for multiple platforms
just build-all

# Create config.toml from example
just config
```

#### Manual Installation

Download the latest binary for your platform from the [releases page](https://github.com/rmoriz/itsjustintv/releases):

- **Linux amd64**: `itsjustintv-linux-amd64`
- **Linux arm64**: `itsjustintv-linux-arm64` 
- **macOS aarch64**: `itsjustintv-darwin-aarch64`

Or build from source:

```bash
git clone https://github.com/rmoriz/itsjustintv.git
cd itsjustintv
go build -o itsjustintv ./cmd/itsjustintv
```

### Getting Started

1. **Get Twitch Application Credentials**
   
   Create a Twitch application at [dev.twitch.tv/console](https://dev.twitch.tv/console):
   - Set OAuth Redirect URL to your webhook endpoint
   - Note down your Client ID and Client Secret
   - Generate a webhook secret for HMAC validation

2. **Generate Configuration**

   ```bash
   ./itsjustintv config example
   cp config.example.toml config.toml
   ```

3. **Configure Your Streamers**

   Edit `config.toml` with your credentials and streamers:

   ```toml
   [twitch]
   client_id = "your_twitch_client_id"
   client_secret = "your_twitch_client_secret"
   webhook_secret = "your_webhook_secret"

   # Simple configuration - user_id will be resolved automatically!
   [streamers.my_favorite_streamer]
   login = "shroud"  # Just the login name - user_id auto-resolved
   target_webhook_url = "https://your-webhook-endpoint.com/webhook"
   
   # Advanced configuration with filtering
   [streamers.another_streamer]
   login = "ninja"
   target_webhook_url = "https://another-endpoint.com/webhook"
   tag_filter = ["English", "Gaming"]  # Only notify for these tags
   additional_tags = ["vip_streamer"]  # Add custom tags to payload
   target_webhook_secret = "optional_hmac_secret"  # Sign this webhook
   ```

4. **Start the Service**

   ```bash
   ./itsjustintv --config config.toml
   ```

   The service will:
   - Automatically resolve user IDs for streamers configured with just `login`
   - Start listening for Twitch webhooks on the configured port
   - Begin dispatching notifications when streamers go live

## Configuration

### Automatic User ID Resolution

**NEW**: You can now configure streamers with just their login name! The service automatically resolves user IDs using Twitch's API during startup.

```toml
# Before (still supported)
[streamers.example_streamer]
user_id = "123456789"
login = "example_streamer"
target_webhook_url = "https://example.com/webhook"

# After (recommended - simpler!)
[streamers.example_streamer]
login = "example_streamer"  # user_id will be auto-resolved
target_webhook_url = "https://example.com/webhook"
```

**How it works:**
- On startup, the service checks each streamer configuration
- If `user_id` is missing but `login` is present, it queries Twitch's API
- The resolved `user_id` is used internally (not saved to config file)
- Logs show successful resolutions: `Resolved user ID for streamer 'example': login='shroud' -> user_id='37402112'`

### Server Configuration

```toml
[server]
listen_addr = "0.0.0.0"
port = 8080
# External domain for reverse proxy scenarios (nginx, traefik, cloud load balancers)
external_domain = "your-domain.com"

# Optional HTTPS with Let's Encrypt
[server.tls]
enabled = true
domains = ["your-domain.com"]
cert_dir = "data/acme_certs"
```

### Twitch Integration

```toml
[twitch]
client_id = "your_twitch_client_id"
client_secret = "your_twitch_client_secret"
webhook_secret = "your_webhook_secret_for_hmac_validation"
token_file = "data/tokens.json"

# Incoming webhook URL for Twitch EventSub subscriptions
# This is the URL Twitch will send webhook notifications to
# If not specified, it will be constructed from server configuration
incoming_webhook_url = "https://your-domain.com/twitch"
```

### Streamer Configuration

```toml
[streamers.streamer_name]
# Option 1: Automatic resolution (recommended)
login = "streamer_login"           # Twitch login name - user_id auto-resolved

# Option 2: Manual specification (still supported)
user_id = "123456789"              # Twitch user ID
login = "streamer_login"           # Twitch login name

# Common settings
target_webhook_url = "https://example.com/webhook"
tag_filter = ["English", "Gaming"] # Optional: filter by stream tags
additional_tags = ["vip"]          # Optional: add custom tags to payload
target_webhook_secret = "optional_secret"    # Optional: HMAC sign this webhook
target_webhook_header = "X-Hub-Signature-256" # Optional: signature header name
target_webhook_hashing = "SHA-256"            # Optional: hashing algorithm
```

### Retry Configuration

```toml
[retry]
max_attempts = 3
initial_delay = "1s"
max_delay = "5m"
backoff_factor = 2.0
state_file = "data/retry_state.json"
```

### File Output

```toml
[output]
enabled = true
file_path = "data/output.json"
max_lines = 1000
```

### OpenTelemetry (Optional)

```toml
[telemetry]
enabled = true
endpoint = "http://localhost:4318"
service_name = "itsjustintv"
service_version = "1.6.0"
```

### Reverse Proxy Configuration

For production deployments behind reverse proxies (nginx, traefik, cloud load balancers):

```toml
[server]
listen_addr = "127.0.0.1"  # Bind to localhost
port = 8080               # Internal port
external_domain = "your-domain.com"  # External domain for webhook URLs

[twitch]
# Optional: Use incoming_webhook_url for explicit control
incoming_webhook_url = "https://your-domain.com/twitch"
```

**Priority order for webhook URLs:**
1. `twitch.incoming_webhook_url` (highest priority)
2. `server.external_domain` (for reverse proxy HTTPS)
3. `server.tls.domains[0]` (for direct HTTPS)
4. `server.listen_addr:port` (fallback)

### Environment Variables

Override any configuration with environment variables:

```bash
export ITSJUSTINTV_TWITCH_CLIENT_ID="your_client_id"
export ITSJUSTINTV_TWITCH_CLIENT_SECRET="your_client_secret"
export ITSJUSTINTV_TWITCH_WEBHOOK_SECRET="your_webhook_secret"
export ITSJUSTINTV_SERVER_PORT="8080"
export ITSJUSTINTV_TLS_ENABLED="true"
export ITSJUSTINTV_SERVER_EXTERNAL_DOMAIN="your-domain.com"
```

## Webhook Payload

When a streamer goes live, the service sends a rich JSON payload to the configured webhook URL:

```json
{
  "streamer_login": "shroud",
  "streamer_name": "shroud",
  "streamer_id": "37402112",
  "url": "https://twitch.tv/shroud",
  "view_count": 1337,
  "followers_count": 50000,
  "tags": ["English", "Gaming", "FPS"],
  "language": "en",
  "description": "Professional gamer and content creator",
  "image": {
    "url": "https://static-cdn.jtvnw.net/jtv_user_pictures/...",
    "width": 300,
    "height": 300
  },
  "timestamp": "2025-07-13T12:00:00Z",
  "additional_tags": ["vip", "custom_tag"],
  "stream": {
    "id": "123456789",
    "type": "live",
    "started_at": "2025-07-13T12:00:00Z",
    "title": "Playing some FPS games!",
    "game_name": "Counter-Strike 2",
    "game_id": "32399"
  }
}
```

### HMAC Signature Verification

If you configure an `hmac_secret` for a streamer, webhooks will include an HMAC signature in the `X-Signature-256` header:

```bash
X-Signature-256: sha256=abc123def456...
```

Verify the signature in your webhook handler:

```python
import hmac
import hashlib

def verify_signature(payload, signature, secret):
    expected = hmac.new(
        secret.encode('utf-8'),
        payload,
        hashlib.sha256
    ).hexdigest()
    return hmac.compare_digest(f"sha256={expected}", signature)
```

## CLI Commands

```bash
# Start the server
./itsjustintv

# Start with specific config file
./itsjustintv --config /path/to/config.toml

# Enable verbose logging
./itsjustintv --verbose

# Show version information
./itsjustintv version

# Validate configuration
./itsjustintv config validate

# Generate example configuration
./itsjustintv config example [output_file]

# Show help
./itsjustintv --help
```

## Development

### Prerequisites

- Go 1.24.5 or later
- Git
- [Just](https://github.com/casey/just) (recommended)

### Development Workflow

```bash
# Clone the repository
git clone https://github.com/rmoriz/itsjustintv.git
cd itsjustintv

# Install development tools
just install-tools

# Run the development cycle
just dev  # formats, tests, builds, and runs

# Or run individual commands
just fmt      # Format code
just lint     # Run linter
just test     # Run tests
just build    # Build binary
just run      # Run with example config
```

### Testing

```bash
# Run all tests
just test

# Run tests with coverage
just test-coverage

# Run integration tests
just test-integration

# Watch for changes and re-run tests
just watch
```

### Building

```bash
# Build for current platform
just build

# Build with version information
just build-release v1.6.0

# Build for all platforms
just build-all v1.6.0

# Clean build artifacts
just clean
```

### Code Quality

```bash
# Run all quality checks
just check

# Individual checks
just fmt      # Format code
just lint     # Run golangci-lint
just test     # Run tests
```

## Docker

### Using Pre-built Images

```bash
# Pull the latest image
docker pull ghcr.io/rmoriz/itsjustintv:latest

# Run with Docker
docker run -d \
  --name itsjustintv \
  -p 8080:8080 \
  -v $(pwd)/config.toml:/app/config.toml \
  -v $(pwd)/data:/app/data \
  ghcr.io/rmoriz/itsjustintv:latest
```

### Building Your Own Image

```bash
# Build Docker image
just docker-build itsjustintv:latest

# Run your custom image
just docker-run itsjustintv:latest 8080
```

### Docker Compose

```yaml
version: '3.8'
services:
  itsjustintv:
    image: ghcr.io/rmoriz/itsjustintv:latest
    ports:
      - "8080:8080"
    volumes:
      - ./config.toml:/app/config.toml
      - ./data:/app/data
    environment:
      - ITSJUSTINTV_TWITCH_CLIENT_ID=your_client_id
      - ITSJUSTINTV_TWITCH_CLIENT_SECRET=your_client_secret
      - ITSJUSTINTV_TWITCH_WEBHOOK_SECRET=your_webhook_secret
    restart: unless-stopped
    healthcheck:
      test: ["CMD", "wget", "--quiet", "--tries=1", "--spider", "http://localhost:8080/health"]
      interval: 30s
      timeout: 10s
      retries: 3
```

## Architecture

The service is built with a modular, production-ready architecture:

### Core Components

- **HTTP Server**: Handles incoming Twitch EventSub webhooks with graceful shutdown
- **Config Manager**: TOML configuration with environment variable overrides
- **Twitch Client**: Manages API interactions, token lifecycle, and user resolution
- **Webhook Dispatcher**: Concurrent notification delivery with retry logic
- **Metadata Enricher**: Fetches and caches streamer metadata with fallbacks
- **Retry Manager**: Persistent retry queue with exponential backoff
- **Cache Manager**: Event deduplication and metadata caching
- **Output Writer**: Structured JSON logging for debugging and integration

### Data Flow

1. **Startup**: Load config → Resolve user IDs → Start Twitch client → Initialize services
2. **Webhook Receipt**: Validate signature → Process notification → Check for duplicates
3. **Event Processing**: Find streamer config → Enrich metadata → Create payload
4. **Delivery**: Dispatch webhook → Handle failures → Queue retries → Log results

### Security Features

- HMAC signature validation for incoming webhooks
- Optional HMAC signing for outgoing webhooks
- Let's Encrypt integration for HTTPS
- No sensitive data in logs
- Secure token storage and rotation

For detailed architecture information, see [docs/architecture.md](docs/architecture.md).

## Monitoring & Observability

### Health Checks

```bash
# Basic health check
curl http://localhost:8080/health

# Response
{
  "status": "healthy",
  "service": "itsjustintv",
  "timestamp": "2025-07-13T12:00:00Z"
}
```

### OpenTelemetry Integration

Enable comprehensive observability:

```toml
[telemetry]
enabled = true
endpoint = "http://localhost:4318"  # OTLP HTTP endpoint
service_name = "itsjustintv"
service_version = "1.6.0"
```

**Metrics collected:**
- Webhook processing latency
- Success/failure rates
- Retry queue depth
- API call performance

**Traces include:**
- End-to-end webhook processing
- Twitch API interactions
- Metadata enrichment
- Webhook delivery attempts

### Logging

Structured JSON logs with configurable levels:

```bash
# Enable verbose logging
./itsjustintv --verbose

# Example log output
{"time":"2025-07-13T12:00:00Z","level":"INFO","msg":"Resolved user ID for streamer","streamer_key":"shroud","login":"shroud","user_id":"37402112"}
{"time":"2025-07-13T12:00:00Z","level":"INFO","msg":"Webhook dispatched successfully","webhook_url":"https://example.com/webhook","streamer_key":"shroud","response_time":"150ms"}
```

## Troubleshooting

### Common Issues

**User ID Resolution Fails**
```bash
# Check Twitch credentials
./itsjustintv config validate

# Verify streamer login exists
curl -H "Client-ID: your_client_id" \
     -H "Authorization: Bearer your_token" \
     "https://api.twitch.tv/helix/users?login=streamer_name"
```

**Webhooks Not Received**
- Verify EventSub subscription is active in Twitch Developer Console
- Check webhook URL is publicly accessible
- Validate HMAC signature implementation
- Review server logs for signature validation errors

**High Memory Usage**
- Reduce cache retention period
- Lower `max_lines` in output configuration
- Check for webhook endpoint timeouts causing retry buildup

### Debug Mode

```bash
# Enable verbose logging
./itsjustintv --verbose

# Check configuration
./itsjustintv config validate

# Test webhook endpoint
curl -X POST your-webhook-url \
  -H "Content-Type: application/json" \
  -d '{"test": "payload"}'
```

## Contributing

We welcome contributions! Please see our [Contributing Guide](CONTRIBUTING.md) for details.

### Development Setup

1. Fork the repository
2. Clone your fork: `git clone https://github.com/yourusername/itsjustintv.git`
3. Create a feature branch: `git checkout -b feature/amazing-feature`
4. Install development tools: `just install-tools`
5. Make your changes and test: `just check`
6. Commit your changes: `git commit -m 'Add amazing feature'`
7. Push to the branch: `git push origin feature/amazing-feature`
8. Open a Pull Request

### Code Standards

- Follow Go best practices and idioms
- Add tests for new functionality
- Update documentation for user-facing changes
- Run `just check` before committing
- Use conventional commit messages

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support & Community

- **Issues**: [GitHub Issues](https://github.com/rmoriz/itsjustintv/issues)
- **Discussions**: [GitHub Discussions](https://github.com/rmoriz/itsjustintv/discussions)
- **Documentation**: [docs/](docs/)
- **Examples**: [examples/](examples/)

## Changelog

See [CHANGELOG.md](CHANGELOG.md) for a detailed history of changes.

---

**itsjustintv** - Making Twitch stream notifications simple, reliable, and production-ready.