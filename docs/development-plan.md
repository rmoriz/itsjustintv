# Development Plan - itsjustintv

**Project:** `itsjustintv`  
**Version:** 1.6  
**Author:** Moriz GmbH  
**Date:** July 13, 2025

---

## Development Task Breakdown

### Milestone 1: Project Scaffold + Config Loader (T+2 days)

**Tag:** `milestone-1-project-scaffold`

#### Tasks:

1. **Project Initialization**

   - [ ] Initialize Go module (`go mod init github.com/rmoriz/itsjustintv`)
   - [ ] Create directory structure according to architecture
   - [ ] Set up `.gitignore` for Go projects
   - [ ] Create initial `README.md` with logo (jpeg) and basic project info
   - [ ] Add all documentation files to git (PRD, architecture, development plan)
   - [ ] Create initial git commit with project foundation

2. **Configuration System**

   - [ ] Define configuration structs in `internal/config/`
   - [ ] Implement TOML parsing with `github.com/BurntSushi/toml`
   - [ ] Add environment variable override support
   - [ ] Create example `config.toml` with all options documented
   - [ ] Add configuration validation logic

3. **CLI Interface**

   - [ ] Implement command-line argument parsing
   - [ ] Add `--help`, `--version`, `--config` flags
   - [ ] Version embedding with Git commit info
   - [ ] Basic application entry point (`main.go`)

4. **Testing Foundation**
   - [ ] Set up test structure and utilities
   - [ ] Write unit tests for config loading
   - [ ] Add test fixtures and mock configurations

**Deliverables:**

- Working config loader with TOML + env vars
- CLI interface with basic flags
- Test suite for configuration system
- Project structure ready for next milestones

---

### Milestone 2: HTTP Server with Optional HTTPS (T+4 days)

**Tag:** `milestone-2-https-server`

#### Tasks:

1. **Basic HTTP Server**

   - [ ] Implement HTTP server in `internal/server/`
   - [ ] Add `/twitch` webhook endpoint (stub)
   - [ ] Health check endpoint (`/health`)
   - [ ] Graceful shutdown handling

2. **Optional Let's Encrypt Integration**

   - [ ] Implement optional autocert manager in `internal/tls/`
   - [ ] Certificate persistence to `data/acme_certs/` (when enabled)
   - [ ] Certificate validation on startup (when enabled)
   - [ ] Automatic renewal scheduling (when enabled)
   - [ ] TLS verification before proceeding (when enabled)

3. **Server Configuration**

   - [ ] Configurable listen address and ports
   - [ ] Optional ACME domain configuration
   - [ ] Certificate storage path configuration (when ACME enabled)
   - [ ] HTTP-only mode for reverse proxy deployments
   - [ ] TLS mode selection (autocert, disabled)

4. **Testing & Validation**
   - [ ] Unit tests for server components (both HTTP and HTTPS modes)
   - [ ] Integration tests with test certificates
   - [ ] Manual testing with real Let's Encrypt staging
   - [ ] Reverse proxy deployment testing

**Deliverables:**

- HTTP server with optional HTTPS/Let's Encrypt integration
- Support for reverse proxy deployments (HTTP-only mode)
- Certificate persistence and renewal (when ACME enabled)
- Flexible TLS configuration options
- Comprehensive test coverage for both deployment modes

---

### Milestone 3: Webhook Receipt & Validation (T+6 days)

**Tag:** `milestone-3-webhook-receipt`

#### Tasks:

1. **Twitch EventSub Handler**

   - [ ] Implement `/twitch` endpoint in `internal/handlers/`
   - [ ] Parse Twitch EventSub webhook payloads
   - [ ] Handle challenge verification for subscription setup
   - [ ] Extract `stream.online` event data

2. **HMAC Signature Validation**

   - [ ] Implement HMAC-SHA256 verification
   - [ ] Validate against Twitch webhook secret
   - [ ] Reject invalid signatures with proper HTTP codes
   - [ ] Add signature validation tests

3. **Streamer Matching Logic**

   - [ ] Match events to configured streamers by login/ID
   - [ ] Implement tag filtering logic
   - [ ] Handle unwanted subscriptions (410 Gone response)
   - [ ] Add comprehensive matching tests

4. **Request Processing Pipeline**
   - [ ] Request logging and metrics
   - [ ] Error handling and HTTP status codes
   - [ ] Request timeout handling
   - [ ] Structured logging for debugging

**Deliverables:**

- Secure webhook receipt with HMAC validation
- Streamer matching and tag filtering
- Proper HTTP responses for all scenarios
- Comprehensive webhook processing tests

---

### Milestone 4: Webhook Dispatch Logic w/ Retry Queue (T+8 days)

**Tag:** `milestone-4-webhook-dispatch`

#### Tasks:

1. **Webhook Dispatcher**

   - [ ] Implement webhook dispatcher in `internal/dispatcher/`
   - [ ] JSON payload construction
   - [ ] HTTP client with timeouts and retries
   - [ ] Optional HMAC signing for outbound webhooks

2. **Retry Queue System**

   - [ ] Implement retry manager in `internal/retry/`
   - [ ] Exponential backoff algorithm (30s initial, 30min max)
   - [ ] Persistent retry state in `data/retry_state.json`
   - [ ] Background retry processing

3. **Deduplication Cache**

   - [ ] Implement cache manager in `internal/cache/`
   - [ ] In-memory deduplication with 2h TTL
   - [ ] Optional persistent cache in `data/cache.json`
   - [ ] Cache cleanup and expiration

4. **Webhook Configuration**
   - [ ] Global default webhook support
   - [ ] Per-streamer webhook overrides
   - [ ] Webhook URL validation
   - [ ] HMAC secret configuration

**Deliverables:**

- Reliable webhook dispatch with retry logic
- Persistent retry queue surviving restarts
- Deduplication preventing duplicate notifications
- Configurable webhook targets per streamer

---

### Milestone 5: Metadata Enrichment (T+10 days)

**Tag:** `milestone-5-metadata-enrichment`

#### Tasks:

1. **Twitch API Client**

   - [ ] Implement Twitch client in `internal/twitch/`
   - [ ] App access token management (client_credentials)
   - [ ] Token persistence in `data/tokens.json`
   - [ ] Automatic token renewal before expiration

2. **Channel Data Enrichment**

   - [ ] Fetch view count and follower count
   - [ ] Retrieve dynamic tags from Twitch API
   - [ ] Language detection from tags (German/English)
   - [ ] URL construction (`https://twitch.tv/<login>`)

3. **Profile Image Caching**

   - [ ] Image fetching and caching in `data/image_cache/`
   - [ ] Base64 encoding for payload embedding
   - [ ] 7-day cache TTL with cleanup
   - [ ] MIME type detection

4. **Payload Construction**
   - [ ] Merge static and dynamic tags
   - [ ] Construct enriched webhook payload
   - [ ] Handle enrichment failures gracefully
   - [ ] Add enrichment metrics and logging

**Deliverables:**

- Twitch API integration with token management
- Rich webhook payloads with metadata
- Profile image caching and embedding
- Graceful degradation on enrichment failures

---

### Milestone 6: Output + Retry Persistence (JSON Files) (T+11 days)

**Tag:** `milestone-6-persistence`

#### Tasks:

1. **File Output Writer**

   - [ ] Implement output writer in `internal/output/`
   - [ ] FIFO queue for last N payloads (default 25)
   - [ ] Atomic file operations for `data/output.json`
   - [ ] Configurable output file location and size

2. **Persistence Layer**

   - [ ] Standardize JSON file operations
   - [ ] Atomic write operations with temp files
   - [ ] File locking for concurrent access
   - [ ] Backup and recovery mechanisms

3. **Data Directory Management**

   - [ ] Automatic `data/` directory creation
   - [ ] Permission handling and validation
   - [ ] Disk space monitoring and cleanup
   - [ ] Configuration for data directory location

4. **State Recovery**
   - [ ] Load retry state on startup
   - [ ] Resume failed webhook deliveries
   - [ ] Validate and repair corrupted state files
   - [ ] Migration support for state file formats

**Deliverables:**

- Persistent output file with webhook history
- Robust retry state persistence
- Reliable file operations with error handling
- State recovery and migration support

---

### Milestone 7: OpenTelemetry Integration (T+12 days)

**Tag:** `milestone-7-observability`

#### Tasks:

1. **OpenTelemetry Setup**

   - [ ] Add OpenTelemetry dependencies
   - [ ] Configure OTLP exporters (gRPC and HTTP)
   - [ ] Implement telemetry initialization
   - [ ] Add configuration for telemetry endpoints

2. **Distributed Tracing**

   - [ ] Add tracing to webhook receipt flow
   - [ ] Trace webhook dispatch and retries
   - [ ] Trace Twitch API calls and enrichment
   - [ ] Add correlation IDs for request tracking

3. **Metrics Collection**

   - [ ] Webhook receipt and dispatch counters
   - [ ] Retry attempt and success/failure metrics
   - [ ] Response time histograms
   - [ ] Cache hit/miss ratios

4. **Logging Integration**
   - [ ] Structured logging with trace correlation
   - [ ] Log levels and filtering
   - [ ] Sensitive data redaction
   - [ ] Log sampling for high-volume events

**Deliverables:**

- Complete OpenTelemetry integration
- Distributed tracing across all components
- Comprehensive metrics for monitoring
- Structured logging with trace correlation

---

### Milestone 8: CI Workflow, Binary & Image Release (T+13 days)

**Tag:** `milestone-8-ci-cd`

#### Tasks:

1. **GitHub Actions Workflow**

   - [ ] Create `.github/workflows/ci.yml`
   - [ ] Go test and linting on main commits
   - [ ] Docker build and push to GHCR
   - [ ] Separate workflow for tagged releases

2. **Docker Configuration**

   - [ ] Create multi-stage `Dockerfile` with Alpine base
   - [ ] Optimize image size and security
   - [ ] Add health check and proper entrypoint
   - [ ] Configure image labels and metadata

3. **Release Automation**

   - [ ] Static binary builds for multiple platforms:
     - [ ] linux-amd64 (musl)
     - [ ] linux-arm64 (musl) 
     - [ ] darwin-aarch64 (macOS Apple Silicon)
   - [ ] GitHub release creation on tags
   - [ ] Artifact upload and release notes
   - [ ] Version embedding in binaries

4. **Build Optimization**
   - [ ] Go build flags for static linking
   - [ ] Binary size optimization
   - [ ] Cross-compilation setup
   - [ ] Build caching and parallelization

**Deliverables:**

- Automated CI/CD pipeline
- Docker images on GHCR
- Multi-platform static binary releases (Linux amd64/arm64, macOS aarch64)
- Automated release management

---

### Milestone 9: Final QA, Docs, README (T+14 days)

**Tag:** `milestone-9-release-ready`

#### Tasks:

1. **Documentation Completion**

   - [ ] Complete `README.md` with usage examples
   - [ ] Document Twitch OAuth setup process
   - [ ] Add Docker deployment guide
   - [ ] Create configuration reference

2. **Example Configurations**

   - [ ] Production-ready config examples
   - [ ] Docker Compose setup
   - [ ] Systemd service file
   - [ ] Example webhook payloads in `examples/payloads/`

3. **Quality Assurance**

   - [ ] End-to-end testing with real Twitch webhooks
   - [ ] Load testing with multiple streamers
   - [ ] Security audit and penetration testing
   - [ ] Performance profiling and optimization

4. **Release Preparation**
   - [ ] Version 1.0.0 release preparation
   - [ ] Final documentation review
   - [ ] Security and dependency audit
   - [ ] Release announcement preparation

**Deliverables:**

- Production-ready service
- Complete documentation and examples
- Comprehensive test coverage
- Version 1.0.0 release

---

## Implementation Strategy

### Development Principles

1. **Test-Driven Development**: Write tests before implementation
2. **Incremental Progress**: Each milestone builds on the previous
3. **Early Integration**: Test components together frequently
4. **Documentation First**: Document interfaces before implementation
5. **Security by Design**: Security considerations in every component

### Testing Strategy

- **Unit Tests**: Individual component testing (>80% coverage)
- **Integration Tests**: Component interaction testing
- **End-to-End Tests**: Full webhook flow testing
- **Load Tests**: Performance validation with multiple streamers
- **Security Tests**: HMAC validation and TLS verification

### Risk Mitigation

- **Twitch API Changes**: Abstract API interactions behind interfaces
- **Certificate Renewal**: Comprehensive testing of Let's Encrypt flow
- **State Corruption**: Robust file operations with validation
- **Performance Issues**: Early profiling and optimization
- **Security Vulnerabilities**: Regular dependency updates and audits

### Dependencies Management

- **Latest Versions**: Use most recent stable versions (see `docs/dependencies.md`)
- **Vetted Libraries**: Only well-maintained, secure dependencies with active development
- **Version Pinning**: Lock exact dependency versions for reproducible builds
- **Security Scanning**: Regular vulnerability scanning of dependencies
- **Consistent Versioning**: OpenTelemetry packages use unified v1.37.0 across all components

---

## Development Environment Setup

### Required Tools

- Go 1.24.5 (latest) for development
- Docker for containerization
- Git for version control
- just for build automation and task running
- golangci-lint for code quality

### Development Workflow

1. Create feature branch from main
2. Implement with tests
3. Run full test suite and linting
4. Create pull request with description
5. Merge to main after review
6. Tag milestone when complete

### Local Testing

- Use Twitch EventSub CLI for webhook simulation
- ngrok for local HTTPS testing
- Docker Compose for integration testing
- Test configuration files for different scenarios

This development plan provides a clear roadmap for implementing all requirements while maintaining high quality and security standards.
