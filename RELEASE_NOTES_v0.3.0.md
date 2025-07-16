# Release Notes - itsjustintv v0.3.0

**Release Date:** July 16, 2025  
**Git Tag:** `v0.3.0`

## 🎉 What's New in v0.3.0

### ✨ Features & Improvements
- **Enhanced Error Handling**: Improved error handling in subscription management with proper cleanup
- **Code Quality**: Removed unused functions and improved code maintainability
- **Better Logging**: Enhanced error logging for Twitch client operations

### 🔧 Technical Improvements
- **Linting Compliance**: Fixed all linting issues for better code quality
- **Test Coverage**: Updated test expectations to match new version
- **Build System**: Improved build process with proper version embedding

### 🐛 Bug Fixes
- Fixed unchecked error returns in subscription management
- Removed unused `deleteSubscription` function from subscription manager
- Improved resource cleanup in CLI commands

### 📦 Distribution
- Multi-platform binaries available:
  - Linux (amd64, arm64)
  - macOS (arm64)
  - Windows (amd64)

## 🔄 Breaking Changes
None in this release.

## 📋 Full Changelog
- Updated version numbers across codebase
- Fixed linting issues in `internal/cli/subscriptions.go`
- Removed unused code from `internal/twitch/subscription_manager.go`
- Updated test expectations for version changes
- Improved error handling and resource cleanup

## 🚀 Getting Started

### Download
Download the appropriate binary for your platform from the [releases page](https://github.com/rmoriz/itsjustintv/releases/tag/v0.3.0).

### Installation
```bash
# Linux/macOS
chmod +x itsjustintv-*
./itsjustintv-* --help

# Windows
itsjustintv-windows-amd64.exe --help
```

### Quick Start
1. Copy `config.example.toml` to `config.toml`
2. Configure your Twitch credentials
3. Add your streamers and webhook endpoints
4. Run: `./itsjustintv --config config.toml`

## 🔗 Links
- [Documentation](https://github.com/rmoriz/itsjustintv#readme)
- [Configuration Guide](https://github.com/rmoriz/itsjustintv#configuration)
- [Docker Images](https://github.com/rmoriz/itsjustintv/pkgs/container/itsjustintv)

---

**Full Changelog**: https://github.com/rmoriz/itsjustintv/compare/v0.1.0...v0.3.0