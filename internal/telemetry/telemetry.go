package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/rmoriz/itsjustintv/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/trace"
)

// Manager handles OpenTelemetry setup and metrics
type Manager struct {
	config         *config.Config
	logger         *slog.Logger
	tracerProvider *sdktrace.TracerProvider
	meterProvider  *sdkmetric.MeterProvider
	tracer         trace.Tracer
	meter          metric.Meter
	
	// Metrics
	webhookCounter       metric.Int64Counter
	webhookDuration      metric.Float64Histogram
	webhookActive        metric.Int64UpDownCounter
	retryCounter         metric.Int64Counter
	retryQueueSize       metric.Int64ObservableGauge
	cacheOperations      metric.Int64Counter
	cacheSize            metric.Int64ObservableGauge
	twitchAPICalls       metric.Int64Counter
	twitchAPIDuration    metric.Float64Histogram
	configReloads        metric.Int64Counter
	configReloadErrors   metric.Int64Counter
}

// NewManager creates a new telemetry manager
func NewManager(cfg *config.Config, logger *slog.Logger) *Manager {
	return &Manager{
		config: cfg,
		logger: logger,
	}
}

// Start initializes OpenTelemetry
func (m *Manager) Start(ctx context.Context) error {
	if !m.config.Telemetry.Enabled {
		m.logger.Info("OpenTelemetry disabled")
		return nil
	}

	// Create resource with service information
	resource := resource.NewWithAttributes(
		"github.com/rmoriz/itsjustintv",
		attribute.String("service.name", m.config.Telemetry.ServiceName),
		attribute.String("service.version", m.config.Telemetry.ServiceVersion),
		attribute.String("service.instance.id", m.config.Telemetry.ServiceName),
	)

	// Initialize trace provider
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpointURL(m.config.Telemetry.Endpoint),
		otlptracehttp.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	m.tracerProvider = sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(traceExporter),
		sdktrace.WithResource(resource),
	)

	// Initialize meter provider
	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpointURL(m.config.Telemetry.Endpoint),
		otlpmetrichttp.WithInsecure(),
	)
	if err != nil {
		return fmt.Errorf("failed to create metric exporter: %w", err)
	}

	m.meterProvider = sdkmetric.NewMeterProvider(
		sdkmetric.WithReader(sdkmetric.NewPeriodicReader(metricExporter)),
		sdkmetric.WithResource(resource),
	)

	// Set global providers
	otel.SetTracerProvider(m.tracerProvider)
	otel.SetMeterProvider(m.meterProvider)

	// Initialize tracer and meter
	m.tracer = m.tracerProvider.Tracer("github.com/rmoriz/itsjustintv")
	m.meter = m.meterProvider.Meter("github.com/rmoriz/itsjustintv")

	// Initialize metrics
	if err := m.initMetrics(); err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}

	m.logger.Info("OpenTelemetry started",
		"endpoint", m.config.Telemetry.Endpoint,
		"service_name", m.config.Telemetry.ServiceName,
		"service_version", m.config.Telemetry.ServiceVersion)

	return nil
}

// initMetrics initializes all metrics
func (m *Manager) initMetrics() error {
	var err error

	// Webhook metrics
	m.webhookCounter, err = m.meter.Int64Counter("webhook_dispatched_total",
		metric.WithDescription("Total number of webhooks dispatched"),
		metric.WithUnit("{count}"))
	if err != nil {
		return err
	}

	m.webhookDuration, err = m.meter.Float64Histogram("webhook_dispatch_duration_seconds",
		metric.WithDescription("Duration of webhook dispatch operations"),
		metric.WithUnit("s"))
	if err != nil {
		return err
	}

	m.webhookActive, err = m.meter.Int64UpDownCounter("webhook_active_requests",
		metric.WithDescription("Number of active webhook requests"),
		metric.WithUnit("{count}"))
	if err != nil {
		return err
	}

	// Retry metrics
	m.retryCounter, err = m.meter.Int64Counter("retry_attempts_total",
		metric.WithDescription("Total number of retry attempts"),
		metric.WithUnit("{count}"))
	if err != nil {
		return err
	}

	m.retryQueueSize, err = m.meter.Int64ObservableGauge("retry_queue_size",
		metric.WithDescription("Current size of the retry queue"),
		metric.WithUnit("{count}"))
	if err != nil {
		return err
	}

	// Cache metrics
	m.cacheOperations, err = m.meter.Int64Counter("cache_operations_total",
		metric.WithDescription("Total number of cache operations"),
		metric.WithUnit("{count}"))
	if err != nil {
		return err
	}

	m.cacheSize, err = m.meter.Int64ObservableGauge("cache_size",
		metric.WithDescription("Current size of the cache"),
		metric.WithUnit("{count}"))
	if err != nil {
		return err
	}

	// Twitch API metrics
	m.twitchAPICalls, err = m.meter.Int64Counter("twitch_api_calls_total",
		metric.WithDescription("Total number of Twitch API calls"),
		metric.WithUnit("{count}"))
	if err != nil {
		return err
	}

	m.twitchAPIDuration, err = m.meter.Float64Histogram("twitch_api_duration_seconds",
		metric.WithDescription("Duration of Twitch API calls"),
		metric.WithUnit("s"))
	if err != nil {
		return err
	}

	// Config metrics
	m.configReloads, err = m.meter.Int64Counter("config_reloads_total",
		metric.WithDescription("Total number of config reloads"),
		metric.WithUnit("{count}"))
	if err != nil {
		return err
	}

	m.configReloadErrors, err = m.meter.Int64Counter("config_reload_errors_total",
		metric.WithDescription("Total number of config reload errors"),
		metric.WithUnit("{count}"))
	if err != nil {
		return err
	}

	return nil
}

// Stop shuts down OpenTelemetry
func (m *Manager) Stop(ctx context.Context) error {
	if !m.config.Telemetry.Enabled {
		return nil
	}

	var err error
	if m.tracerProvider != nil {
		err = m.tracerProvider.Shutdown(ctx)
	}
	if m.meterProvider != nil {
		if shutdownErr := m.meterProvider.Shutdown(ctx); err == nil {
			err = shutdownErr
		}
	}

	if err != nil {
		return fmt.Errorf("failed to shutdown OpenTelemetry: %w", err)
	}

	m.logger.Info("OpenTelemetry stopped")
	return nil
}

// StartSpan starts a new span
func (m *Manager) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, trace.Span) {
	if !m.config.Telemetry.Enabled {
		return ctx, trace.SpanFromContext(ctx)
	}
	return m.tracer.Start(ctx, name, trace.WithAttributes(attrs...))
}

// RecordWebhook records webhook metrics
func (m *Manager) RecordWebhook(ctx context.Context, success bool, duration time.Duration, streamerKey string) {
	if !m.config.Telemetry.Enabled {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("streamer_key", streamerKey),
	}
	if success {
		attrs = append(attrs, attribute.String("status", "success"))
	} else {
		attrs = append(attrs, attribute.String("status", "failure"))
	}

	m.webhookCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.webhookDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordWebhookActive increments/decrements active webhook counter
func (m *Manager) RecordWebhookActive(ctx context.Context, delta int64) {
	if !m.config.Telemetry.Enabled {
		return
	}
	m.webhookActive.Add(ctx, delta)
}

// RecordRetry records retry metrics
func (m *Manager) RecordRetry(ctx context.Context, attempt int, streamerKey string) {
	if !m.config.Telemetry.Enabled {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("streamer_key", streamerKey),
		attribute.Int("attempt", attempt),
	}
	m.retryCounter.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordTwitchAPICall records Twitch API metrics
func (m *Manager) RecordTwitchAPICall(ctx context.Context, endpoint string, duration time.Duration, success bool) {
	if !m.config.Telemetry.Enabled {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("endpoint", endpoint),
	}
	if success {
		attrs = append(attrs, attribute.String("status", "success"))
	} else {
		attrs = append(attrs, attribute.String("status", "failure"))
	}

	m.twitchAPICalls.Add(ctx, 1, metric.WithAttributes(attrs...))
	m.twitchAPIDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordCacheOperation records cache metrics
func (m *Manager) RecordCacheOperation(ctx context.Context, operation string, success bool) {
	if !m.config.Telemetry.Enabled {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("operation", operation),
	}
	if success {
		attrs = append(attrs, attribute.String("status", "success"))
	} else {
		attrs = append(attrs, attribute.String("status", "failure"))
	}

	m.cacheOperations.Add(ctx, 1, metric.WithAttributes(attrs...))
}

// RecordConfigReload records config reload metrics
func (m *Manager) RecordConfigReload(ctx context.Context, success bool) {
	if !m.config.Telemetry.Enabled {
		return
	}

	if success {
		m.configReloads.Add(ctx, 1)
	} else {
		m.configReloadErrors.Add(ctx, 1)
	}
}

// GetTracer returns the tracer instance
func (m *Manager) GetTracer() trace.Tracer {
	return m.tracer
}

// GetMeter returns the meter instance
func (m *Manager) GetMeter() metric.Meter {
	return m.meter
}