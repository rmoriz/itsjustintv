package telemetry

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlpmetric/otlpmetrichttp"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/sdk/metric"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/semconv/v1.4.0"
	oteltrace "go.opentelemetry.io/otel/trace"

	"github.com/rmoriz/itsjustintv/internal/config"
)

// Manager handles OpenTelemetry setup and metrics
type Manager struct {
	config         *config.Config
	logger         *slog.Logger
	tracerProvider *trace.TracerProvider
	meterProvider  *metric.MeterProvider
	tracer         oteltrace.Tracer
	meter          metric.Meter
	
	// Metrics
	webhookCounter    metric.Int64Counter
	webhookDuration   metric.Float64Histogram
	eventCounter      metric.Int64Counter
	retryCounter      metric.Int64Counter
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

	// Create resource
	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(m.config.Telemetry.ServiceName),
			semconv.ServiceVersionKey.String(m.config.Telemetry.ServiceVersion),
		),
	)
	if err != nil {
		return fmt.Errorf("failed to create resource: %w", err)
	}

	// Setup tracing
	if err := m.setupTracing(ctx, res); err != nil {
		return fmt.Errorf("failed to setup tracing: %w", err)
	}

	// Setup metrics
	if err := m.setupMetrics(ctx, res); err != nil {
		return fmt.Errorf("failed to setup metrics: %w", err)
	}

	// Initialize metrics
	if err := m.initMetrics(); err != nil {
		return fmt.Errorf("failed to initialize metrics: %w", err)
	}

	m.logger.Info("OpenTelemetry started", 
		"endpoint", m.config.Telemetry.Endpoint,
		"service_name", m.config.Telemetry.ServiceName)

	return nil
}

// Stop shuts down OpenTelemetry
func (m *Manager) Stop(ctx context.Context) error {
	if !m.config.Telemetry.Enabled {
		return nil
	}

	if m.tracerProvider != nil {
		if err := m.tracerProvider.Shutdown(ctx); err != nil {
			m.logger.Error("Failed to shutdown tracer provider", "error", err)
		}
	}

	if m.meterProvider != nil {
		if err := m.meterProvider.Shutdown(ctx); err != nil {
			m.logger.Error("Failed to shutdown meter provider", "error", err)
		}
	}

	m.logger.Info("OpenTelemetry stopped")
	return nil
}

// setupTracing configures OpenTelemetry tracing
func (m *Manager) setupTracing(ctx context.Context, res *resource.Resource) error {
	traceExporter, err := otlptracehttp.New(ctx,
		otlptracehttp.WithEndpoint(m.config.Telemetry.Endpoint),
		otlptracehttp.WithInsecure(), // Use HTTPS in production
	)
	if err != nil {
		return fmt.Errorf("failed to create trace exporter: %w", err)
	}

	m.tracerProvider = trace.NewTracerProvider(
		trace.WithBatcher(traceExporter),
		trace.WithResource(res),
	)

	otel.SetTracerProvider(m.tracerProvider)
	m.tracer = otel.Tracer("itsjustintv")

	return nil
}

// setupMetrics configures OpenTelemetry metrics
func (m *Manager) setupMetrics(ctx context.Context, res *resource.Resource) error {
	metricExporter, err := otlpmetrichttp.New(ctx,
		otlpmetrichttp.WithEndpoint(m.config.Telemetry.Endpoint),
		otlpmetrichttp.WithInsecure(), // Use HTTPS in production
	)
	if err != nil {
		return fmt.Errorf("failed to create metric exporter: %w", err)
	}

	m.meterProvider = metric.NewMeterProvider(
		metric.WithReader(metric.NewPeriodicReader(metricExporter,
			metric.WithInterval(30*time.Second))),
		metric.WithResource(res),
	)

	otel.SetMeterProvider(m.meterProvider)
	m.meter = otel.Meter("itsjustintv")

	return nil
}

// initMetrics initializes metric instruments
func (m *Manager) initMetrics() error {
	var err error

	m.webhookCounter, err = m.meter.Int64Counter(
		"itsjustintv_webhooks_total",
		metric.WithDescription("Total number of webhook dispatches"),
	)
	if err != nil {
		return fmt.Errorf("failed to create webhook counter: %w", err)
	}

	m.webhookDuration, err = m.meter.Float64Histogram(
		"itsjustintv_webhook_duration_seconds",
		metric.WithDescription("Webhook dispatch duration in seconds"),
	)
	if err != nil {
		return fmt.Errorf("failed to create webhook duration histogram: %w", err)
	}

	m.eventCounter, err = m.meter.Int64Counter(
		"itsjustintv_events_total",
		metric.WithDescription("Total number of events processed"),
	)
	if err != nil {
		return fmt.Errorf("failed to create event counter: %w", err)
	}

	m.retryCounter, err = m.meter.Int64Counter(
		"itsjustintv_retries_total",
		metric.WithDescription("Total number of webhook retries"),
	)
	if err != nil {
		return fmt.Errorf("failed to create retry counter: %w", err)
	}

	return nil
}

// RecordWebhook records webhook metrics
func (m *Manager) RecordWebhook(ctx context.Context, success bool, duration time.Duration, streamerKey string) {
	if !m.config.Telemetry.Enabled {
		return
	}

	status := "success"
	if !success {
		status = "failure"
	}

	m.webhookCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("status", status),
		attribute.String("streamer", streamerKey),
	))

	m.webhookDuration.Record(ctx, duration.Seconds(), metric.WithAttributes(
		attribute.String("status", status),
		attribute.String("streamer", streamerKey),
	))
}

// RecordEvent records event processing metrics
func (m *Manager) RecordEvent(ctx context.Context, eventType string, streamerKey string) {
	if !m.config.Telemetry.Enabled {
		return
	}

	m.eventCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.String("event_type", eventType),
		attribute.String("streamer", streamerKey),
	))
}

// RecordRetry records retry metrics
func (m *Manager) RecordRetry(ctx context.Context, attempt int, streamerKey string) {
	if !m.config.Telemetry.Enabled {
		return
	}

	m.retryCounter.Add(ctx, 1, metric.WithAttributes(
		attribute.Int("attempt", attempt),
		attribute.String("streamer", streamerKey),
	))
}

// StartSpan starts a new trace span
func (m *Manager) StartSpan(ctx context.Context, name string) (context.Context, oteltrace.Span) {
	if !m.config.Telemetry.Enabled {
		return ctx, oteltrace.SpanFromContext(ctx)
	}

	return m.tracer.Start(ctx, name)
}