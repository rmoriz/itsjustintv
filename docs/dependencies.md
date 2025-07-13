# Dependencies - itsjustintv

**Project:** `itsjustintv`  
**Go Version:** 1.24.5  
**Date:** July 13, 2025  

---

## Latest Dependency Versions

### Core Dependencies

| Package | Latest Version | Purpose |
|---------|----------------|---------|
| `github.com/BurntSushi/toml` | `v1.5.0` | TOML configuration parsing |
| `github.com/fsnotify/fsnotify` | `v1.9.0` | File system watching for config hot-reload |
| `github.com/spf13/cobra` | `v1.9.1` | CLI interface and command parsing |
| `golang.org/x/crypto` | `v0.40.0` | HMAC validation and cryptographic functions |
| `golang.org/x/net` | `v0.42.0` | HTTP client enhancements and autocert |

### OpenTelemetry Dependencies

| Package | Latest Version | Purpose |
|---------|----------------|---------|
| `go.opentelemetry.io/otel` | `v1.37.0` | Core OpenTelemetry SDK |
| `go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp` | `v1.37.0` | OTLP HTTP trace exporter |
| `go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp` | `v1.37.0` | OTLP HTTP metrics exporter |
| `go.opentelemetry.io/otel/sdk` | `v1.37.0` | OpenTelemetry SDK |
| `go.opentelemetry.io/otel/sdk/metric` | `v1.37.0` | Metrics SDK |
| `go.opentelemetry.io/otel/trace` | `v1.37.0` | Tracing API |
| `go.opentelemetry.io/otel/metric` | `v1.37.0` | Metrics API |

### Testing Dependencies

| Package | Latest Version | Purpose |
|---------|----------------|---------|
| `github.com/stretchr/testify` | `v1.10.0` | Testing assertions and mocking |

### Standard Library (Go 1.24.5)

| Package | Purpose |
|---------|---------|
| `net/http` | HTTP server and client |
| `crypto/hmac` | HMAC signature validation |
| `crypto/sha256` | SHA256 hashing |
| `encoding/json` | JSON marshaling/unmarshaling |
| `time` | Time handling and scheduling |
| `context` | Request context management |
| `log/slog` | Structured logging (Go 1.21+) |

---

## Dependency Rationale

### Configuration Management
- **TOML v1.5.0**: Latest stable version with full TOML spec support
- **fsnotify v1.9.0**: Most recent version with improved cross-platform support

### CLI Interface
- **Cobra v1.9.1**: Latest version with enhanced command parsing and help generation

### Security & Networking
- **golang.org/x/crypto v0.40.0**: Latest cryptographic functions and HMAC support
- **golang.org/x/net v0.42.0**: Enhanced HTTP client and autocert support

### Observability
- **OpenTelemetry v1.37.0**: Latest stable release across all OTel packages
- Consistent versioning across all OTel components for compatibility

### Testing
- **Testify v1.10.0**: Latest version with improved assertion methods

---

## Version Management Strategy

### Semantic Versioning
- Use exact versions for reproducible builds
- Pin major versions to avoid breaking changes
- Regular dependency updates with testing

### Security Updates
- Monitor for security advisories
- Automated dependency scanning in CI
- Prompt updates for security patches

### Compatibility Matrix
- All dependencies tested with Go 1.24.5
- OpenTelemetry packages use consistent v1.37.0
- No conflicting transitive dependencies

---

## go.mod Template

```go
module github.com/rmoriz/itsjustintv

go 1.24.5

require (
    github.com/BurntSushi/toml v1.5.0
    github.com/fsnotify/fsnotify v1.9.0
    github.com/spf13/cobra v1.9.1
    github.com/stretchr/testify v1.10.0
    go.opentelemetry.io/otel v1.37.0
    go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp v1.37.0
    go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp v1.37.0
    go.opentelemetry.io/otel/metric v1.37.0
    go.opentelemetry.io/otel/sdk v1.37.0
    go.opentelemetry.io/otel/sdk/metric v1.37.0
    go.opentelemetry.io/otel/trace v1.37.0
    golang.org/x/crypto v0.40.0
    golang.org/x/net v0.42.0
)
```

---

## Dependency Update Process

### Regular Updates
1. Check for new versions monthly
2. Test compatibility in development
3. Update documentation
4. Create PR with dependency updates

### Security Updates
1. Monitor security advisories
2. Apply patches immediately
3. Test critical paths
4. Deploy security updates quickly

### Breaking Changes
1. Review changelog and migration guides
2. Update code for breaking changes
3. Comprehensive testing
4. Document migration steps

This dependency list ensures we're using the latest, most secure, and performant versions of all required packages.