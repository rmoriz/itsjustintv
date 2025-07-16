# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.3.0] - 2025-07-16

### Added
- Multi-platform binary builds (Linux amd64/arm64, macOS arm64, Windows amd64)
- Enhanced error handling in subscription management
- Proper resource cleanup in CLI commands

### Changed
- Improved code quality and linting compliance
- Updated test expectations for version changes
- Enhanced error logging for Twitch client operations

### Fixed
- Fixed unchecked error returns in subscription management
- Improved resource cleanup in CLI commands

### Removed
- Unused `deleteSubscription` function from subscription manager

## [0.1.0] - 2025-07-13

### Added
- Initial release
- Twitch EventSub webhook integration
- Real-time stream notifications
- Automatic user ID resolution
- Smart webhook dispatching
- Rich metadata enrichment
- Robust retry logic with exponential backoff
- Duplicate detection
- HMAC signature validation
- HTTPS support with Let's Encrypt
- OpenTelemetry integration
- File output for debugging
- Health check endpoints
- Tag-based filtering
- Custom tag injection
- Per-streamer configuration
- CLI interface with subscription management
- Docker support
- Comprehensive documentation

[0.3.0]: https://github.com/rmoriz/itsjustintv/compare/v0.1.0...v0.3.0
[0.1.0]: https://github.com/rmoriz/itsjustintv/releases/tag/v0.1.0