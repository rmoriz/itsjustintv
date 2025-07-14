# itsjustintv Execution Plan

## Overview
This document outlines the step-by-step implementation plan to complete the missing features identified in the PRD analysis. Each phase has clear deliverables and can be implemented independently.

## Current Status
✅ **Phase 1 Complete**: OpenTelemetry integration with comprehensive metrics and tracing

## Execution Phases

### Phase 1: OpenTelemetry Integration ✅ COMPLETE
**Status**: ✅ **COMPLETED**

**Deliverables**:
- [x] OpenTelemetry dependencies added to go.mod
- [x] Core telemetry infrastructure in `internal/telemetry/telemetry.go`
- [x] HTTP server instrumentation with middleware
- [x] Webhook processing instrumentation with spans
- [x] Metrics for webhook dispatch, retry attempts, cache operations, and Twitch API calls
- [x] Configuration-based on/off toggle
- [x] Production-ready OTLP exporters

### Phase 2: Configuration Hot-Reloading
**Status**: 🔄 **READY TO START**

**Deliverables**:
- [ ] Add fsnotify dependency for file watching
- [ ] Implement config file watcher in `internal/config/watcher.go`
- [ ] Add configuration reload mechanism with debouncing
- [ ] Update subscription manager to handle config changes
- [ ] Add telemetry metrics for config reloads
- [ ] Graceful handling of config validation errors

**Files to Modify**:
- `go.mod`: Add fsnotify dependency
- `internal/config/config.go`: Add reload functionality
- `internal/server/server.go`: Integrate config watcher
- `internal/twitch/subscription_manager.go`: Handle config changes

### Phase 3: Tag Filtering Implementation Fix
**Status**: 📋 **PENDING**

**Deliverables**:
- [ ] Fix tag filtering logic in `internal/twitch/processor.go`
- [ ] Add comprehensive tag matching (exact, partial, case-insensitive)
- [ ] Add tag validation and normalization
- [ ] Update configuration schema for tag filtering
- [ ] Add tests for tag filtering scenarios

**Current Issue**: Tag filtering is implemented but not working correctly according to PRD requirements.

### Phase 4: Global Webhook Configuration
**Status**: 📋 **PENDING**

**Deliverables**:
- [ ] Add global webhook URL configuration option
- [ ] Implement fallback to global URL when streamer-specific URL is missing
- [ ] Update webhook dispatcher to use global configuration
- [ ] Add validation for global webhook configuration
- [ ] Update documentation with examples

**Configuration Schema**:
```toml
[webhook]
global_webhook_url = "https://example.com/webhook"
global_hmac_secret = "global-secret"
```

### Phase 5: Testing and Validation
**Status**: 📋 **PENDING**

**Deliverables**:
- [ ] Comprehensive integration tests for all new features
- [ ] End-to-end testing scenarios
- [ ] Performance testing with telemetry enabled
- [ ] Documentation updates
- [ ] Example configurations
- [ ] Grafana dashboard examples

## Implementation Order
1. **Phase 2**: Configuration hot-reloading (fsnotify)
2. **Phase 3**: Fix tag filtering implementation
3. **Phase 4**: Global webhook configuration
4. **Phase 5**: Testing and documentation

## Technical Decisions
- **fsnotify** for file watching (cross-platform, battle-tested)
- **debounced reloads** to prevent excessive config reloads
- **backward compatibility** maintained throughout all phases
- **feature flags** for gradual rollout

## Success Criteria
- All tests pass (unit, integration, e2e)
- Zero downtime during configuration changes
- Comprehensive observability via OpenTelemetry
- Clear documentation and examples
- Performance impact < 5% with telemetry enabled

## Next Steps
Begin **Phase 2** by implementing configuration hot-reloading with fsnotify.