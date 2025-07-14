# Release Notes - v0.2.0

## 🚀 Major Release: Enhanced Configuration & Reverse Proxy Support

**Release Date:** July 14, 2025  
**Version:** v0.2.0  
**Previous Version:** v0.1.0

## 📋 Summary

This major release introduces comprehensive configuration improvements, reverse proxy support, and enhanced webhook flexibility. All breaking changes have been implemented with clear migration paths.

## 🔥 Breaking Changes

### Configuration Field Renaming
- `webhook_url` → `target_webhook_url` (streamer-specific destination)
- `hmac_secret` → `target_webhook_secret` (streamer-specific signing)

**Migration:** Update your `config.toml` files:
```toml
# OLD (v0.1.0)
[streamers.example]
webhook_url = "https://example.com/webhook"
hmac_secret = "secret123"

# NEW (v0.2.0)
[streamers.example]
target_webhook_url = "https://example.com/webhook"
target_webhook_secret = "secret123"
```

## ✨ New Features

### 1. Reverse Proxy Support
- **New:** `external_domain` configuration for reverse proxy deployments
- **Usage:** Deploy behind nginx, traefik, cloud load balancers without TLS configuration issues
- **Example:**
```toml
[server]
listen_addr = "127.0.0.1"  # Bind to localhost
port = 8080               # Internal port
external_domain = "your-domain.com"  # External domain for webhook URLs
```

### 2. Configurable Webhook Headers & Algorithms
- **New:** `target_webhook_header` - Customize signature HTTP header (default: `X-Hub-Signature-256`)
- **New:** `target_webhook_hashing` - Choose hashing algorithm: `SHA-256`, `SHA-512`, or `SHA-1`
- **Usage:** Support different webhook endpoint requirements

### 3. Explicit Incoming Webhook URL
- **New:** `incoming_webhook_url` configuration for explicit webhook endpoint control
- **Usage:** Override automatic URL construction when needed
- **Priority Order:**
  1. `twitch.incoming_webhook_url` (highest priority)
  2. `server.external_domain` (reverse proxy HTTPS)
  3. `server.tls.domains[0]` (direct HTTPS)
  4. `server.listen_addr:port` (fallback)

### 4. Global Webhook Configuration
- **New:** Global webhook fallback for streamers without specific URLs
- **Usage:** Single webhook endpoint for all streamers when no streamer-specific URL is provided

## 🏗️ Architecture Improvements

### Enhanced URL Construction
The `buildCallbackURL()` function now properly handles:
- Reverse proxy scenarios with `external_domain`
- HTTPS by default for external domains
- Graceful fallback to direct server configuration
- Proper URL construction for production deployments

### Configuration Validation
- Added comprehensive validation for all new configuration options
- Environment variable overrides for all new fields
- Clear error messages for configuration issues

## 📖 Documentation Updates

- **README.md:** Updated with reverse proxy configuration examples
- **config.example.toml:** Added all new configuration options
- **examples/https-config.toml:** Added reverse proxy examples
- **Environment variables:** Documented all new environment variable overrides

## 🔧 Configuration Examples

### Reverse Proxy Setup (nginx/traefik)
```toml
[server]
listen_addr = "127.0.0.1"
port = 8080
external_domain = "twitch-alerts.example.com"

[twitch]
client_id = "your_client_id"
client_secret = "your_client_secret"
webhook_secret = "your_webhook_secret"
incoming_webhook_url = "https://twitch-alerts.example.com/twitch"
```

### Advanced Webhook Configuration
```toml
[streamers.example]
login = "shroud"
target_webhook_url = "https://discord.com/api/webhooks/..."
target_webhook_secret = "discord_webhook_secret"
target_webhook_header = "X-Signature-Ed25519"
target_webhook_hashing = "SHA-512"
tag_filter = ["English", "Gaming"]
```

## 🧪 Testing

- **68 tests passing** - All existing functionality preserved
- **New tests added** for all configuration changes
- **Integration tests** updated for reverse proxy scenarios
- **Configuration validation tests** for all new fields

## 🐛 Bug Fixes

- Fixed webhook URL construction for reverse proxy deployments
- Resolved configuration field naming inconsistencies
- Improved error handling for invalid webhook configurations

## 🚦 Migration Guide

### Step 1: Update Configuration Files
1. Rename all `webhook_url` → `target_webhook_url`
2. Rename all `hmac_secret` → `target_webhook_secret`
3. Add `external_domain` if using reverse proxy
4. Add `incoming_webhook_url` for explicit control

### Step 2: Environment Variables
Update any environment variables:
```bash
# OLD
export WEBHOOK_URL="..."
export HMAC_SECRET="..."

# NEW
export TARGET_WEBHOOK_URL="..."
export TARGET_WEBHOOK_SECRET="..."
```

### Step 3: Validate Configuration
```bash
./itsjustintv config validate
```

## 📊 Compatibility

- **Go Version:** 1.24.5+
- **Docker:** Full compatibility maintained
- **Environment Variables:** All new overrides supported
- **Backward Compatibility:** Intentionally broken for clarity (v0.1.0 configs need updates)

## 🎯 Production Readiness

This release is **production-ready** with:
- Comprehensive configuration validation
- Reverse proxy deployment support
- Production-grade webhook security
- Clear migration documentation
- Full test coverage

## 🔗 Quick Start

```bash
# Download latest release
wget https://github.com/rmoriz/itsjustintv/releases/download/v0.2.0/itsjustintv-linux-amd64

# Generate new config
./itsjustintv config example > config.toml

# Edit configuration and start
./itsjustintv --config config.toml
```

---

**Ready for production deployment!** 🚀

For support, please check the [GitHub Issues](https://github.com/rmoriz/itsjustintv/issues) or [GitHub Discussions](https://github.com/rmoriz/itsjustintv/discussions).