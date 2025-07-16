# Next Features for itsjustintv

Based on the roadmap and immediate development priorities, here are the next features to implement:

## Immediate Priority (v0.4.0) - Next 2-4 weeks

### 1. Enhanced Monitoring & Metrics 🔍
**Priority: High**
- **Prometheus Metrics Endpoint**: `/metrics` endpoint for Prometheus scraping
- **Key Metrics to Track**:
  - Webhook success/failure rates
  - Request latency percentiles
  - Active subscriptions count
  - Twitch API call rates
  - Memory and CPU usage
- **Implementation**: Add `github.com/prometheus/client_golang` dependency

### 2. Structured Logging 📊
**Priority: High**
- **JSON Logging**: Replace text logging with structured JSON
- **Log Levels**: Configurable log levels (debug, info, warn, error)
- **Request Correlation**: Add request IDs for tracing
- **Implementation**: Enhance existing slog usage

### 3. Configuration Hot Reload 🔧
**Priority: Medium**
- **File Watching**: Monitor config file changes
- **Graceful Reload**: Update configuration without restart
- **Validation**: Ensure new config is valid before applying
- **Implementation**: Use `github.com/fsnotify/fsnotify` (already in dependencies)

## Short Term (v0.4.1-v0.4.2) - Next 1-2 months

### 4. Enhanced Health Checks 🏥
**Priority: Medium**
- **Dependency Checks**: Check Twitch API connectivity
- **Detailed Status**: More granular health information
- **Readiness vs Liveness**: Separate endpoints for K8s
- **Database Health**: If/when database is added

### 5. Webhook Templates 🎯
**Priority: Medium**
- **Go Templates**: Customizable webhook payloads
- **Template Variables**: Access to all stream metadata
- **Multiple Formats**: Slack, Discord, Teams presets
- **Custom Fields**: User-defined payload fields

### 6. Stream Offline Events 🔄
**Priority: Medium**
- **Event Support**: Track when streams end
- **Configuration**: Per-streamer offline notification settings
- **Debouncing**: Prevent spam from brief disconnections
- **Duration Tracking**: Include stream duration in notifications

## Development Guidelines

### Implementation Order
1. **Metrics First**: Essential for production monitoring
2. **Logging Second**: Improves debugging and operations
3. **Hot Reload Third**: Reduces operational overhead
4. **Health Checks Fourth**: Improves reliability monitoring
5. **Templates Fifth**: Enhances user experience
6. **Offline Events Sixth**: Adds feature completeness

### Technical Considerations
- **Backwards Compatibility**: All changes must be backwards compatible
- **Configuration**: New features should be opt-in via configuration
- **Testing**: Each feature needs comprehensive unit and integration tests
- **Documentation**: Update README and add feature-specific docs
- **Performance**: Monitor impact on existing functionality

### Code Quality Standards
- **Linting**: All code must pass golangci-lint
- **Test Coverage**: Maintain >80% test coverage
- **Error Handling**: Proper error handling and logging
- **Documentation**: Godoc comments for all public functions

## Community Input

### Feature Requests to Consider
- **Multi-platform Notifications**: YouTube, Discord native integration
- **Web Dashboard**: Browser-based configuration interface
- **API Endpoints**: REST API for external integrations
- **Database Backend**: PostgreSQL/MySQL support for scaling
- **Rate Limiting**: Configurable rate limits for webhooks

### Feedback Channels
- GitHub Issues for bug reports and feature requests
- Discussions for general feedback and questions
- Pull Requests for community contributions

## Success Metrics

### v0.4.0 Goals
- **Monitoring**: Prometheus metrics endpoint functional
- **Observability**: JSON logging with correlation IDs
- **Operations**: Hot reload working reliably
- **Reliability**: Enhanced health checks providing detailed status
- **User Experience**: Webhook templates for common platforms

### Quality Gates
- All tests passing
- Linting clean
- Documentation updated
- Performance benchmarks maintained
- Memory usage stable
- No breaking changes

---

**Next Review**: After v0.4.0 release
**Target Release Date**: August 2025