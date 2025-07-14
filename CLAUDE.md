# CLAUDE.md - itsjustintv Development Guidelines

## Overview
This file contains development guidelines and project-specific information for working on itsjustintv - a Go-based Twitch EventSub webhook bridge service.

## Project Context
- **Name**: itsjustintv
- **Purpose**: Receive Twitch EventSub webhooks and dispatch notifications when configured streamers go live
- **Language**: Go 1.24.5
- **License**: MIT (c) 2025 Moriz GmbH
- **Repository**: rmoriz/itsjustintv

## Quick Start
- **Build**: `go build ./cmd/itsjustintv`
- **Test**: `go test -v ./...`
- **Lint**: `golangci-lint run`
- **Run**: `./itsjustintv --config config.toml`

## Architecture Overview
- **Core Service**: HTTP server receiving Twitch webhooks at `/twitch`
- **Key Components**: Webhook validation, streamer matching, metadata enrichment, webhook dispatching
- **Caching**: Deduplication (2h TTL), profile images (7 days), persistent JSON storage
- **TLS**: Optional Let's Encrypt autocert integration
- **Observability**: OpenTelemetry with OTLP exporters

## Configuration
- **File**: `config.toml` (TOML format)
- **Env Vars**: `ITSJUSTINTV_*` prefix overrides
- **Hot Reload**: fsnotify-based config watching (Phase 2)

## Development Workflow

### Code Standards
- **Language**: Modern Go (no deprecated APIs)
- **Linting**: golangci-lint with strict config
- **Testing**: Unit + integration tests required
- **Comments**: Godoc-style for public APIs
- **Error Handling**: Always handle errors explicitly

### Git Workflow
- **Branch**: `main` (trunk-based)
- **Commits**: Descriptive, scoped messages
- **Tags**: `v1.2.3` format for releases
- **Messages**: Always in English
- **AI Commits**: Include `AI-Generated-By: <model> via Claude`
- **Co-Authored-By**: Do not add Co-Authored-By to git commit messages

### Pre-Commit Checklist
1. ✅ Run all tests: `go test -v ./...`
2. ✅ Run linter: `golangci-lint run`
3. ✅ Check coverage: `go test -cover ./...`
4. ✅ Check file sizes (prevent 0-byte files)
5. ✅ Update version numbers for releases
6. ✅ Commit with appropriate message
7. ✅ Push when feature complete

## Commit Strategy
- **Small, atomic commits** per logical change
- **Feature branches** for significant work
- **Step-by-step commits** during implementation
- **Immediate push** after each completed task
- **No force pushes** to main branch

## Git Commands Template
```bash
# Before starting work
git pull origin main

# After each logical step
go test -v ./... && golangci-lint run && go test -cover ./...
git add .
git commit -m "feat: add telemetry metrics for webhook dispatch

- Added webhook counter and duration metrics
- Instrumented HTTP server with tracing middleware
- Added telemetry manager with OTLP exporters

AI-Generated-By: Claude via Claude Code"
git push origin main

# For feature work
git checkout -b feature/config-hot-reload
git push origin feature/config-hot-reload
```

### Testing Strategy
- **Unit Tests**: All public APIs
- **Integration Tests**: End-to-end webhook flows
- **Mock Services**: Twitch API simulation
- **Performance Tests**: Load validation

## File Structure
```
├── cmd/itsjustintv/           # Main binary
├── internal/
│   ├── cache/                 # Caching layer
│   ├── cli/                   # Command-line interface
│   ├── config/                # Configuration management
│   ├── output/                # File output writer
│   ├── retry/                 # Retry queue management
│   ├── telemetry/             # OpenTelemetry integration
│   ├── twitch/                # Twitch API client
│   ├── webhook/               # Webhook dispatching
│   └── server/                # HTTP server
├── docs/                      # Documentation
├── examples/                  # Sample configurations
└── data/                      # Runtime data (cache, tokens, etc.)
```

## Key Interfaces
- **StreamEventProvider**: Future platform extensibility
- **MetadataEnricher**: Pluggable enrichment pipeline
- **TwitchUserResolver**: User ID resolution for config

## Configuration Schema
```toml
[server]
listen_addr = "0.0.0.0"
port = 8080

[twitch]
client_id = "your-client-id"
client_secret = "your-client-secret"
webhook_secret = "your-webhook-secret"

[streamers.streamer1]
login = "streamername"
target_webhook_url = "https://example.com/webhook"
tag_filter = ["Science & Technology"]
additional_tags = ["selfhosted"]

[telemetry]
enabled = true
endpoint = "http://localhost:4318"
service_name = "itsjustintv"
```

## Deployment Options
- **Docker**: Alpine-based multi-arch images
- **Binary**: Static binaries for Linux/macOS
- **Systemd**: Linux service management
- **Reverse Proxy**: Behind nginx/traefik (HTTP mode)

## Monitoring & Observability
- **Metrics**: webhook_dispatched_total, retry_attempts_total, cache_operations_total
- **Traces**: HTTP server spans, webhook processing spans
- **Health**: `/health` endpoint for service status
- **Logs**: Structured logging with levels (DEBUG, INFO, WARN, ERROR)

## Release Process
- **CI**: GitHub Actions on every push
- **Docker**: Automatic `latest` tag on main
- **Releases**: Multi-arch binaries on Git tags
- **Platforms**: linux-amd64, linux-arm64, darwin-aarch64

## Common Commands
```bash
# Development
./itsjustintv --config dev.toml --log-level debug

# Production
docker run -v ./data:/app/data -v ./config.toml:/app/config.toml ghcr.io/rmoriz/itsjustintv:latest

# Testing
go test -v ./internal/server/ -run TestHandleTwitchWebhook

# Lint fixes
golangci-lint run --fix
```

## Troubleshooting
- **Config validation**: Check logs for validation errors
- **Webhook failures**: Check retry state file in data/
- **TLS issues**: Verify certificates in data/acme_certs/
- **Twitch auth**: Check token file in data/tokens.json

## Security Notes
- Never commit credentials to git
- Use environment variables for sensitive data
- Enable TLS in production
- Validate all webhook signatures
- Monitor for subscription expiration

## Contributing
1. Fork repository
2. Create feature branch
3. Follow testing/linting guidelines
4. Submit PR with clear description
5. Ensure CI passes

## Resources
- **PRD**: docs/product-requirements.md
- **Architecture**: docs/architecture.md
- **Execution Plan**: docs/EXECUTION_PLAN.md
- **Examples**: examples/ directory