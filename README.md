# itsjustintv

![itsjustintv Logo](docs/itsjustintv-logo.png)

A configurable, self-hosted Go-based service that receives Twitch EventSub HTTP webhooks and notifies via downstream webhooks or file output when specified streamers go live.

## Features

- **Real-time Stream Notifications**: Receive instant notifications when your favorite streamers go live
- **Flexible Webhook Dispatching**: Send notifications to multiple endpoints with custom payloads
- **Metadata Enrichment**: Automatically fetch and include streamer metadata (follower count, profile images, etc.)
- **Tag Filtering**: Filter notifications based on stream tags (language, category, etc.)
- **Retry Mechanism**: Robust retry logic with exponential backoff for failed webhook deliveries
- **HTTPS Support**: Optional Let's Encrypt integration for secure webhook endpoints
- **File Output**: Optionally save webhook payloads to JSON files for debugging or archival
- **OpenTelemetry**: Built-in observability with metrics and tracing support

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

### Configuration

Generate an example configuration file:

```bash
./itsjustintv config example
```

Edit `config.example.toml` with your Twitch application credentials and streamer configurations:

```toml
[twitch]
client_id = "your_twitch_client_id"
client_secret = "your_twitch_client_secret"
webhook_secret = "your_webhook_secret"

[streamers.example_streamer]
user_id = "123456789"
login = "example_streamer"
webhook_url = "https://your-webhook-endpoint.com/webhook"
tag_filter = ["English", "Gaming"]
additional_tags = ["custom_tag"]
```

### Running

Start the service:

```bash
./itsjustintv --config config.toml
```

Or with environment variables:

```bash
export ITSJUSTINTV_TWITCH_CLIENT_ID="your_client_id"
export ITSJUSTINTV_TWITCH_CLIENT_SECRET="your_client_secret"
export ITSJUSTINTV_TWITCH_WEBHOOK_SECRET="your_webhook_secret"
./itsjustintv
```

## Configuration

### Server Configuration

```toml
[server]
listen_addr = "0.0.0.0"
port = 8080

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
```

### Streamer Configuration

```toml
[streamers.streamer_name]
user_id = "123456789"              # Twitch user ID
login = "streamer_login"           # Twitch login name
webhook_url = "https://example.com/webhook"
tag_filter = ["English", "Gaming"] # Optional: filter by stream tags
additional_tags = ["vip"]          # Optional: add custom tags to payload
hmac_secret = "optional_secret"    # Optional: HMAC sign this webhook
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

**Note**: OpenTelemetry support requires additional setup. When using single binary releases, telemetry features are included but optional.

## Webhook Payload

When a streamer goes live, the service sends a JSON payload to the configured webhook URL:

```json
{
  "streamer_login": "example_streamer",
  "streamer_name": "Example Streamer",
  "streamer_id": "123456789",
  "url": "https://twitch.tv/example_streamer",
  "view_count": 1337,
  "followers_count": 50000,
  "tags": ["English", "Gaming", "Just Chatting"],
  "language": "en",
  "description": "Playing some games and chatting!",
  "image": {
    "url": "https://static-cdn.jtvnw.net/jtv_user_pictures/...",
    "width": 300,
    "height": 300
  },
  "timestamp": "2025-07-13T12:00:00Z",
  "additional_tags": ["vip", "custom_tag"]
}
```

## CLI Commands

```bash
# Start the server
./itsjustintv

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

### Building

```bash
# Install dependencies
go mod tidy

# Run tests
go test ./...

# Run tests with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Build binary
go build -o itsjustintv ./cmd/itsjustintv

# Build with version information
go build -ldflags "-X github.com/rmoriz/itsjustintv/internal/cli.Version=1.6.0 -X github.com/rmoriz/itsjustintv/internal/cli.GitCommit=$(git rev-parse HEAD) -X github.com/rmoriz/itsjustintv/internal/cli.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ)" -o itsjustintv ./cmd/itsjustintv
```

### Multi-Platform Builds

The project supports building for multiple platforms:

```bash
# Linux amd64
GOOS=linux GOARCH=amd64 go build -o itsjustintv-linux-amd64 ./cmd/itsjustintv

# Linux arm64
GOOS=linux GOARCH=arm64 go build -o itsjustintv-linux-arm64 ./cmd/itsjustintv

# macOS aarch64 (Apple Silicon)
GOOS=darwin GOARCH=arm64 go build -o itsjustintv-darwin-aarch64 ./cmd/itsjustintv
```

**Note**: All binary releases include OpenTelemetry support and are built as static binaries for maximum compatibility.

## Docker

```bash
# Build Docker image
docker build -t itsjustintv .

# Run with Docker
docker run -d \
  --name itsjustintv \
  -p 8080:8080 \
  -v $(pwd)/config.toml:/app/config.toml \
  -v $(pwd)/data:/app/data \
  itsjustintv
```

## Architecture

The service is built with a modular architecture:

- **HTTP Server**: Handles incoming Twitch EventSub webhooks
- **Config Manager**: TOML configuration with hot-reloading support
- **Twitch Client**: Manages Twitch API interactions and token lifecycle
- **Webhook Dispatcher**: Sends notifications to configured endpoints
- **Metadata Enricher**: Fetches and caches streamer metadata
- **Retry Manager**: Handles failed webhook deliveries with exponential backoff
- **Cache Manager**: Provides deduplication and metadata caching
- **File Output**: Optional JSON file output for debugging

For detailed architecture information, see [docs/architecture.md](docs/architecture.md).

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

## License

This project is licensed under the MIT License - see the [LICENSE](LICENSE) file for details.

## Support

- **Issues**: [GitHub Issues](https://github.com/rmoriz/itsjustintv/issues)
- **Documentation**: [docs/](docs/)
- **Examples**: [examples/](examples/)

---

**itsjustintv** - Making Twitch stream notifications simple and reliable.