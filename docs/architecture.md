# Architecture Document - itsjustintv

**Project:** `itsjustintv`  
**Version:** 1.6  
**Author:** Moriz GmbH  
**Date:** July 13, 2025  

---

## 1. System Overview

`itsjustintv` is a Go-based microservice that acts as a bridge between Twitch's EventSub webhook system and downstream notification systems. It receives real-time stream events, enriches them with metadata, and dispatches notifications through configurable webhooks or file output.

### 1.1 High-Level Architecture

```
┌─────────────────┐    ┌──────────────────┐    ┌─────────────────┐
│                 │    │                  │    │                 │
│  Twitch API     │───▶│   itsjustintv    │───▶│  Downstream     │
│  EventSub       │    │   Service        │    │  Webhooks       │
│                 │    │                  │    │                 │
└─────────────────┘    └──────────────────┘    └─────────────────┘
                              │
                              ▼
                       ┌──────────────┐
                       │              │
                       │ File Output  │
                       │ (output.json)│
                       │              │
                       └──────────────┘
```

---

## 2. Component Architecture

### 2.1 Core Components

```
┌─────────────────────────────────────────────────────────────────┐
│                        itsjustintv Service                      │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   HTTP      │  │   Config    │  │  Webhook    │             │
│  │   Server    │  │   Manager   │  │  Dispatcher │             │
│  │             │  │             │  │             │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Twitch    │  │  Metadata   │  │    Cache    │             │
│  │   Client    │  │  Enricher   │  │   Manager   │             │
│  │             │  │             │  │             │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
├─────────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐             │
│  │   Retry     │  │    File     │  │ OpenTelemetry│             │
│  │   Manager   │  │   Output    │  │   Exporter  │             │
│  │             │  │   Writer    │  │             │             │
│  └─────────────┘  └─────────────┘  └─────────────┘             │
└─────────────────────────────────────────────────────────────────┘
```

### 2.2 Component Responsibilities

#### HTTP Server
- **Purpose**: Handles incoming Twitch EventSub webhooks
- **Endpoints**: `/twitch` (primary webhook endpoint)
- **Features**: 
  - HTTPS with Let's Encrypt autocert
  - HMAC signature validation
  - TLS certificate management and renewal
- **Dependencies**: Config Manager, Webhook Dispatcher

#### Config Manager
- **Purpose**: Manages configuration loading and hot-reloading
- **Features**:
  - TOML file parsing
  - Environment variable override
  - File system watching (fsnotify)
  - Live configuration updates
- **Dependencies**: None (foundational component)

#### Twitch Client
- **Purpose**: Manages Twitch API interactions
- **Features**:
  - App access token management (client_credentials flow)
  - EventSub subscription management
  - Token persistence and renewal
  - Subscription validation and cleanup
- **Dependencies**: Config Manager, Cache Manager

#### Metadata Enricher
- **Purpose**: Enriches stream events with additional data
- **Features**:
  - Profile image fetching and caching
  - View count and follower count retrieval
  - Tag merging (static + dynamic)
  - Language detection from tags
- **Dependencies**: Twitch Client, Cache Manager

#### Webhook Dispatcher
- **Purpose**: Sends notifications to configured endpoints
- **Features**:
  - JSON payload construction
  - HMAC signing (optional)
  - Retry logic with exponential backoff
  - Deduplication
- **Dependencies**: Retry Manager, Cache Manager, Metadata Enricher

#### Cache Manager
- **Purpose**: Handles in-memory and persistent caching
- **Features**:
  - Deduplication cache (2h TTL)
  - Profile image cache (7 days)
  - Token persistence
  - File-based persistence (JSON)
- **Dependencies**: None

#### Retry Manager
- **Purpose**: Manages failed webhook delivery retries
- **Features**:
  - Exponential backoff (30s initial, 30min max)
  - Persistent retry state
  - Background retry processing
- **Dependencies**: Cache Manager

#### File Output Writer
- **Purpose**: Writes webhook payloads to output file
- **Features**:
  - FIFO queue (default 25 entries)
  - JSON format
  - Atomic file operations
- **Dependencies**: None

#### OpenTelemetry Exporter
- **Purpose**: Provides observability and monitoring
- **Features**:
  - Span tracing
  - Metrics collection
  - OTLP export (gRPC/HTTP)
- **Dependencies**: All components (cross-cutting)

---

## 3. Data Flow

### 3.1 Webhook Receipt Flow

```
┌─────────────┐
│   Twitch    │
│  EventSub   │
└──────┬──────┘
       │ POST /twitch
       ▼
┌─────────────┐
│ HTTP Server │
│ - Validate  │
│   HMAC      │
│ - Parse     │
│   payload   │
└──────┬──────┘
       │
       ▼
┌─────────────┐     ┌─────────────┐
│   Config    │────▶│  Streamer   │
│  Matching   │     │   Match?    │
└─────────────┘     └──────┬──────┘
                           │ Yes
                           ▼
                    ┌─────────────┐
                    │ Tag Filter  │
                    │   Check     │
                    └──────┬──────┘
                           │ Pass
                           ▼
                    ┌─────────────┐
                    │ Metadata    │
                    │ Enrichment  │
                    └──────┬──────┘
                           │
                           ▼
                    ┌─────────────┐
                    │  Webhook    │
                    │ Dispatch    │
                    └─────────────┘
```

### 3.2 Subscription Management Flow

```
┌─────────────┐
│  Service    │
│  Startup    │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Load Config │
│ & Validate  │
│ Twitch Auth │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Fetch       │
│ Current     │
│ Subs        │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Compare     │
│ with Config │
│ & Register  │
│ Missing     │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Start       │
│ Background  │
│ Validation  │
│ (1h + splay)│
└─────────────┘
```

### 3.3 Configuration Hot-Reload Flow

```
┌─────────────┐
│ File System │
│   Watcher   │
│ (fsnotify)  │
└──────┬──────┘
       │ config.toml changed
       ▼
┌─────────────┐
│ Parse &     │
│ Validate    │
│ New Config  │
└──────┬──────┘
       │ Valid?
       ▼
┌─────────────┐
│ Apply New   │
│ Config      │
│ Live        │
└──────┬──────┘
       │
       ▼
┌─────────────┐
│ Update      │
│ Twitch      │
│ Subs        │
└─────────────┘
```

---

## 4. Data Models

### 4.1 Core Structures

```go
// Configuration
type Config struct {
    Server    ServerConfig
    Twitch    TwitchConfig
    Streamers map[string]StreamerConfig
    Retry     RetryConfig
    Output    OutputConfig
    Telemetry TelemetryConfig
}

// Webhook Payload
type WebhookPayload struct {
    StreamerLogin   string            `json:"streamer_login"`
    StreamerName    string            `json:"streamer_name"`
    StreamerID      string            `json:"streamer_id"`
    URL             string            `json:"url"`
    ViewCount       int               `json:"view_count"`
    FollowersCount  int               `json:"followers_count"`
    Tags            []string          `json:"tags"`
    Language        string            `json:"language"`
    Description     string            `json:"description"`
    Image           *ImageData        `json:"image,omitempty"`
    Timestamp       time.Time         `json:"timestamp"`
    AdditionalTags  []string          `json:"additional_tags"`
}

// Cache Entry
type CacheEntry struct {
    Key       string    `json:"key"`
    Data      []byte    `json:"data"`
    ExpiresAt time.Time `json:"expires_at"`
}
```

### 4.2 Persistence Files

```
├── data/
│   ├── cache.json          # Deduplication cache
│   ├── retry_state.json    # Retry queue state
│   ├── tokens.json         # Twitch access tokens
│   ├── acme_certs/         # Let's Encrypt certificates and keys
│   │   ├── cert.pem        # TLS certificate
│   │   ├── key.pem         # Private key
│   │   └── acme_cache.json # ACME client state and metadata
│   ├── image_cache/        # Profile image cache
│   └── output.json         # Last N webhook payloads
```

---

## 5. Security Considerations

### 5.1 Authentication & Authorization
- **Twitch API**: Client credentials flow with secure token storage
- **Webhook Validation**: HMAC-SHA256 signature verification
- **Outbound Webhooks**: Optional HMAC signing

### 5.2 TLS/HTTPS
- **Let's Encrypt Integration**: Automatic certificate provisioning and renewal
- **Certificate Persistence**: Disk-based storage for restart resilience
- **Pre-validation**: TLS verification before Twitch subscription registration

### 5.3 Data Protection
- **No Credential Logging**: Sensitive data excluded from logs
- **Secure Token Storage**: Encrypted persistence of access tokens
- **Input Validation**: All webhook payloads validated before processing

---

## 6. Performance & Scalability

### 6.1 Performance Targets
- **Response Time**: <1s for webhook processing
- **Throughput**: Support 100+ concurrent streamers
- **Memory Usage**: Efficient caching with configurable TTLs

### 6.2 Optimization Strategies
- **In-Memory Caching**: Fast access to frequently used data
- **Async Processing**: Non-blocking webhook dispatch
- **Connection Pooling**: Efficient HTTP client usage
- **Batch Operations**: Grouped Twitch API calls where possible

---

## 7. Error Handling & Resilience

### 7.1 Retry Mechanisms
- **Exponential Backoff**: 30s initial, 30min maximum window
- **Persistent State**: Retry queue survives service restarts
- **Circuit Breaking**: Prevent cascade failures

### 7.2 Graceful Degradation
- **Enrichment Failures**: Continue dispatch with basic payload
- **Partial Outages**: Individual streamer failures don't affect others
- **Configuration Errors**: Retain previous valid configuration

### 7.3 Monitoring & Alerting
- **Health Checks**: Service and dependency status
- **Metrics**: Success/failure rates, latency, queue depths
- **Distributed Tracing**: End-to-end request tracking

---

## 8. Deployment Architecture

### 8.1 Container Deployment
```
┌─────────────────────────────────────────────────────────────┐
│                     Docker Container                        │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────┐  ┌─────────────┐  ┌─────────────┐        │
│  │itsjustintv  │  │config.toml  │  │   data/     │        │
│  │  binary     │  │             │  │ (volumes)   │        │
│  │             │  │             │  │             │        │
│  └─────────────┘  └─────────────┘  └─────────────┘        │
├─────────────────────────────────────────────────────────────┤
│  Ports: 80, 443 (HTTPS)                                    │
│  Volumes: /app/data, /app/config.toml                      │
└─────────────────────────────────────────────────────────────┘
```

### 8.2 Binary Deployment
- **Static Binary**: Single executable with no dependencies
- **Systemd Service**: Linux service management
- **Configuration**: File-based with environment overrides

---

## 9. Development & Testing Strategy

### 9.1 Testing Approach
- **Unit Tests**: Individual component testing
- **Integration Tests**: End-to-end webhook flow
- **Mock Services**: Twitch API simulation
- **Load Testing**: Performance validation

### 9.2 CI/CD Pipeline

#### Continuous Integration (main branch)
```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│   Commit    │───▶│   Build &   │───▶│   Docker    │
│   to main   │    │    Test     │    │ Push :latest│
└─────────────┘    └─────────────┘    └─────────────┘
```

#### Tagged Releases
```
┌─────────────┐    ┌─────────────┐    ┌─────────────┐    ┌─────────────┐
│  Push Tag   │───▶│   Build &   │───▶│   Docker    │───▶│   GitHub    │
│  (v1.2.3)   │    │    Test     │    │Push :v1.2.3 │    │   Release   │
│             │    │             │    │& :latest    │    │             │
└─────────────┘    └─────────────┘    └─────────────┘    └─────────────┘
                                             │
                                             ▼
                                      ┌─────────────┐
                                      │   Static    │
                                      │   Binary    │
                                      │(multi-arch) │
                                      └─────────────┘
```

#### Pipeline Details

**On commits to `main`:**
- Run `go test` and linter
- Build Alpine-based Docker image
- Push to `ghcr.io/rmoriz/itsjustintv:latest`
- **No GitHub release created**

**On Git tag push (e.g., `v1.2.3`):**
- Run `go test` and linter
- Build static binaries for multiple platforms:
  - `linux-amd64` (musl)
  - `linux-arm64` (musl)
  - `darwin-aarch64` (macOS Apple Silicon)
- Build Alpine-based Docker image
- Push Docker image with tags: `v1.2.3` and `latest`
- Create GitHub release with multi-platform binary artifacts
- Generate release notes from commits since last tag

---

## 10. Future Extensibility

### 10.1 Plugin Architecture
The system is designed with interfaces to support future streaming platforms:

```go
type StreamEventProvider interface {
    Listen(handler func(StreamEvent)) error
}

type MetadataEnricher interface {
    Enrich(event StreamEvent) (EnrichedEvent, error)
}
```

### 10.2 Potential Extensions
- **YouTube Live**: YouTube streaming events
- **Kick.com**: Alternative streaming platform
- **Custom Providers**: Internal streaming systems
- **Advanced Filtering**: ML-based content classification
- **Multi-tenant**: Support multiple organizations

---

This architecture provides a solid foundation for implementing the `itsjustintv` service with clear separation of concerns, robust error handling, and extensibility for future enhancements.