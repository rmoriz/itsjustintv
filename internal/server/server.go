package server

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/rmoriz/itsjustintv/internal/cache"
	"github.com/rmoriz/itsjustintv/internal/config"
	"github.com/rmoriz/itsjustintv/internal/output"
	"github.com/rmoriz/itsjustintv/internal/retry"
	"github.com/rmoriz/itsjustintv/internal/telemetry"
	"github.com/rmoriz/itsjustintv/internal/twitch"
	"github.com/rmoriz/itsjustintv/internal/webhook"
	"go.opentelemetry.io/otel/attribute"
	"golang.org/x/crypto/acme/autocert"
)

// Server represents the HTTP server with optional HTTPS support
type Server struct {
	config              *config.Config
	httpServer          *http.Server
	logger              *slog.Logger
	certManager         *autocert.Manager
	webhookValidator    *webhook.Validator
	twitchProcessor     *twitch.Processor
	webhookDispatcher   *webhook.Dispatcher
	retryManager        *retry.Manager
	cacheManager        *cache.Manager
	twitchClient        *twitch.Client
	enricher            *twitch.Enricher
	outputWriter        *output.Writer
	subscriptionManager *twitch.SubscriptionManager
	telemetryManager    *telemetry.Manager
	configWatcher       *config.Watcher
}

// New creates a new server instance
func New(cfg *config.Config, logger *slog.Logger) *Server {
	webhookDispatcher := webhook.NewDispatcher(cfg, logger)
	cacheManager := cache.NewManager(logger, "data/cache.json", 2*time.Hour)
	retryManager := retry.NewManager(cfg, logger, webhookDispatcher)
	twitchClient := twitch.NewClient(cfg, logger)
	enricher := twitch.NewEnricher(cfg, logger, twitchClient)
	outputWriter := output.NewWriter(cfg, logger)
	subscriptionManager := twitch.NewSubscriptionManager(cfg, logger, twitchClient)
	telemetryManager := telemetry.NewManager(cfg, logger)

	return &Server{
		config:              cfg,
		logger:              logger,
		webhookValidator:    webhook.NewValidator(cfg.Twitch.WebhookSecret),
		twitchProcessor:     twitch.NewProcessor(cfg, logger),
		webhookDispatcher:   webhookDispatcher,
		retryManager:        retryManager,
		cacheManager:        cacheManager,
		twitchClient:        twitchClient,
		enricher:            enricher,
		outputWriter:        outputWriter,
		subscriptionManager: subscriptionManager,
		telemetryManager:    telemetryManager,
		configWatcher:       nil, // Will be initialized in Start
	}
}

// Start starts the HTTP server with optional HTTPS
func (s *Server) Start(ctx context.Context) error {
	// Start telemetry
	if err := s.telemetryManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start telemetry: %w", err)
	}

	// Start config watcher
	if err := s.startConfigWatcher(ctx); err != nil {
		return fmt.Errorf("failed to start config watcher: %w", err)
	}

	// Start Twitch client
	if err := s.twitchClient.Start(ctx); err != nil {
		return fmt.Errorf("failed to start Twitch client: %w", err)
	}

	// Resolve missing user IDs for streamers
	if err := config.ResolveStreamerUserIDs(ctx, s.config, s.twitchClient); err != nil {
		s.logger.Warn("Failed to resolve some streamer user IDs", "error", err)
		// Don't fail startup, just log the warning
	}

	// Start enricher
	if err := s.enricher.Start(); err != nil {
		return fmt.Errorf("failed to start enricher: %w", err)
	}

	// Start cache manager
	if err := s.cacheManager.Start(); err != nil {
		return fmt.Errorf("failed to start cache manager: %w", err)
	}

	// Start retry manager
	if err := s.retryManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start retry manager: %w", err)
	}

	// Start output writer
	if err := s.outputWriter.Start(); err != nil {
		return fmt.Errorf("failed to start output writer: %w", err)
	}

	// Setup routes
	mux := http.NewServeMux()
	s.setupRoutes(mux)

	// Configure server
	s.httpServer = &http.Server{
		Addr:         fmt.Sprintf("%s:%d", s.config.Server.ListenAddr, s.config.Server.Port),
		Handler:      mux,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 30 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Setup TLS if enabled
	if s.config.Server.TLS.Enabled {
		if err := s.setupTLS(); err != nil {
			return fmt.Errorf("failed to setup TLS: %w", err)
		}
	}

	// Start server in a goroutine
	serverErrors := make(chan error, 1)
	go func() {
		s.logger.Info("Starting HTTP server",
			"addr", s.httpServer.Addr,
			"tls_enabled", s.config.Server.TLS.Enabled)

		if s.config.Server.TLS.Enabled {
			serverErrors <- s.httpServer.ListenAndServeTLS("", "")
		} else {
			serverErrors <- s.httpServer.ListenAndServe()
		}
	}()

	// Wait a moment for the server to start listening
	time.Sleep(100 * time.Millisecond)

	// Start subscription manager AFTER HTTP server is running
	if err := s.subscriptionManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start subscription manager: %w", err)
	}

	// Wait for shutdown signal or server error
	shutdown := make(chan os.Signal, 1)
	signal.Notify(shutdown, syscall.SIGINT, syscall.SIGTERM)

	select {
	case err := <-serverErrors:
		if err != http.ErrServerClosed {
			return fmt.Errorf("server error: %w", err)
		}
	case sig := <-shutdown:
		s.logger.Info("Shutdown signal received", "signal", sig)

		// Graceful shutdown
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Server shutdown error", "error", err)
			return fmt.Errorf("server shutdown error: %w", err)
		}
	case <-ctx.Done():
		s.logger.Info("Context cancelled, shutting down server")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := s.httpServer.Shutdown(shutdownCtx); err != nil {
			s.logger.Error("Server shutdown error", "error", err)
			return fmt.Errorf("server shutdown error: %w", err)
		}
	}

	// Stop managers
	if err := s.retryManager.Stop(); err != nil {
		s.logger.Error("Retry manager stop error", "error", err)
	}
	if err := s.cacheManager.Stop(); err != nil {
		s.logger.Error("Cache manager stop error", "error", err)
	}
	if err := s.twitchClient.Stop(); err != nil {
		s.logger.Error("Twitch client stop error", "error", err)
	}
	if err := s.outputWriter.Stop(); err != nil {
		s.logger.Error("Output writer stop error", "error", err)
	}

	// Stop telemetry
	if err := s.telemetryManager.Stop(ctx); err != nil {
		s.logger.Error("Telemetry stop error", "error", err)
	}

	// Stop config watcher
	if s.configWatcher != nil {
		if err := s.configWatcher.Stop(); err != nil {
			s.logger.Error("Config watcher stop error", "error", err)
		}
	}

	s.logger.Info("Server stopped gracefully")
	return nil
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// Health check endpoint
	mux.HandleFunc("/health", s.instrumentHandler(s.handleHealth, "health"))

	// Twitch webhook endpoint
	mux.HandleFunc("/twitch", s.instrumentHandler(s.handleTwitchWebhook, "twitch_webhook"))

	// Root endpoint
	mux.HandleFunc("/", s.instrumentHandler(s.handleRoot, "root"))
}

// instrumentHandler wraps HTTP handlers with telemetry
func (s *Server) instrumentHandler(next http.HandlerFunc, operation string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		// Start span
		ctx, span := s.telemetryManager.StartSpan(ctx, fmt.Sprintf("http.%s", operation),
			attribute.String("http.method", r.Method),
			attribute.String("http.url", r.URL.String()),
			attribute.String("http.user_agent", r.UserAgent()),
		)
		defer span.End()

		// Track active requests
		s.telemetryManager.RecordWebhookActive(ctx, 1)
		defer s.telemetryManager.RecordWebhookActive(ctx, -1)

		// Create response writer to capture status code
		wrapped := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

		start := time.Now()
		next(wrapped, r.WithContext(ctx))
		duration := time.Since(start)

		// Record metrics
		s.telemetryManager.RecordWebhook(ctx, wrapped.statusCode < 400, duration, operation)

		// Add attributes to span
		span.SetAttributes(
			attribute.Int("http.status_code", wrapped.statusCode),
			attribute.Float64("http.duration_ms", float64(duration.Milliseconds())),
		)
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// startConfigWatcher initializes and starts the configuration file watcher
func (s *Server) startConfigWatcher(ctx context.Context) error {
	configPath := s.config.GetConfigPath()
	if configPath == "" {
		s.logger.Debug("No config path available, skipping file watcher")
		return nil
	}

	watcher, err := config.NewWatcher(configPath, s.logger, s.handleConfigReload)
	if err != nil {
		return fmt.Errorf("failed to create config watcher: %w", err)
	}

	s.configWatcher = watcher
	return s.configWatcher.Start(ctx)
}

// handleConfigReload handles configuration changes and updates subscriptions
func (s *Server) handleConfigReload(newConfig *config.Config) error {
	ctx := context.Background()

	// Record config reload metric
	if s.telemetryManager != nil {
		s.telemetryManager.RecordConfigReload(ctx, true)
	}

	s.logger.Info("Handling configuration reload")

	// Update config reference
	s.config = newConfig

	// Update subscription manager with new config
	if s.subscriptionManager != nil {
		if err := s.subscriptionManager.UpdateConfig(newConfig); err != nil {
			s.logger.Error("Failed to update subscription manager config", "error", err)
			if s.telemetryManager != nil {
				s.telemetryManager.RecordConfigReload(ctx, false)
			}
			return fmt.Errorf("failed to update subscription manager: %w", err)
		}

		// Refresh subscriptions based on new configuration
		if err := s.subscriptionManager.RefreshSubscriptions(ctx); err != nil {
			s.logger.Error("Failed to refresh subscriptions", "error", err)
			if s.telemetryManager != nil {
				s.telemetryManager.RecordConfigReload(ctx, false)
			}
			return fmt.Errorf("failed to refresh subscriptions: %w", err)
		}
	}

	// Update webhook dispatcher with new config
	if s.webhookDispatcher != nil {
		s.webhookDispatcher.UpdateConfig(newConfig)
	}

	// Update retry manager with new config
	if s.retryManager != nil {
		s.retryManager.UpdateConfig(newConfig)
	}

	// Update output writer with new config
	if s.outputWriter != nil {
		s.outputWriter.UpdateConfig(newConfig)
	}

	// Update enricher with new config
	if s.enricher != nil {
		s.enricher.UpdateConfig(newConfig)
	}

	s.logger.Info("Configuration reload completed successfully")
	return nil
}

// setupTLS configures TLS with Let's Encrypt autocert
func (s *Server) setupTLS() error {
	if len(s.config.Server.TLS.Domains) == 0 {
		return fmt.Errorf("TLS domains must be specified when TLS is enabled")
	}

	// Ensure cert directory exists
	if err := os.MkdirAll(s.config.Server.TLS.CertDir, 0700); err != nil {
		return fmt.Errorf("failed to create cert directory: %w", err)
	}

	// Setup autocert manager
	s.certManager = &autocert.Manager{
		Prompt:     autocert.AcceptTOS,
		HostPolicy: autocert.HostWhitelist(s.config.Server.TLS.Domains...),
		Cache:      autocert.DirCache(s.config.Server.TLS.CertDir),
	}

	// Configure TLS
	s.httpServer.TLSConfig = &tls.Config{
		GetCertificate: s.certManager.GetCertificate,
		NextProtos:     []string{"h2", "http/1.1"},
		MinVersion:     tls.VersionTLS12,
	}

	// Start HTTP-01 challenge server on port 80 if we're listening on 443
	if s.config.Server.Port == 443 {
		go func() {
			s.logger.Info("Starting HTTP-01 challenge server on :80")
			challengeServer := &http.Server{
				Addr:    ":80",
				Handler: s.certManager.HTTPHandler(nil),
			}
			if err := challengeServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				s.logger.Error("Challenge server error", "error", err)
			}
		}()
	}

	s.logger.Info("TLS configured with Let's Encrypt",
		"domains", s.config.Server.TLS.Domains,
		"cert_dir", s.config.Server.TLS.CertDir)

	return nil
}

// handleHealth handles health check requests
func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)

	response := `{"status":"healthy","service":"itsjustintv","timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `"}`
	_, _ = w.Write([]byte(response))

	s.logger.Debug("Health check requested", "remote_addr", r.RemoteAddr)
}

// handleTwitchWebhook handles Twitch EventSub webhooks
func (s *Server) handleTwitchWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// Read the request body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("Failed to read request body", "error", err)
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Extract EventSub headers
	headers := twitch.EventSubHeaders{
		MessageID:           r.Header.Get("Twitch-Eventsub-Message-Id"),
		MessageRetry:        r.Header.Get("Twitch-Eventsub-Message-Retry"),
		MessageType:         r.Header.Get("Twitch-Eventsub-Message-Type"),
		MessageSignature:    r.Header.Get("Twitch-Eventsub-Message-Signature"),
		MessageTimestamp:    r.Header.Get("Twitch-Eventsub-Message-Timestamp"),
		SubscriptionType:    r.Header.Get("Twitch-Eventsub-Subscription-Type"),
		SubscriptionVersion: r.Header.Get("Twitch-Eventsub-Subscription-Version"),
	}

	s.logger.Debug("Twitch webhook received",
		"remote_addr", r.RemoteAddr,
		"message_type", headers.MessageType,
		"subscription_type", headers.SubscriptionType,
		"message_id", headers.MessageID)

	// Validate HMAC signature
	if err := s.webhookValidator.ValidateSignature(body, headers.MessageSignature); err != nil {
		s.logger.Warn("Invalid webhook signature",
			"error", err,
			"remote_addr", r.RemoteAddr,
			"message_id", headers.MessageID)
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Process the notification
	processedEvent, err := s.twitchProcessor.ProcessNotification(headers, body)
	if err != nil {
		s.logger.Error("Failed to process notification",
			"error", err,
			"message_id", headers.MessageID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}

	// Handle the response based on the processed event
	switch processedEvent.Action {
	case "respond":
		// Webhook verification challenge
		w.Header().Set("Content-Type", "text/plain")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(processedEvent.Challenge))
		s.logger.Info("Webhook verification challenge responded",
			"message_id", headers.MessageID)

	case "process":
		// Process the event - dispatch webhooks
		if err := s.processStreamEvent(processedEvent, headers.MessageID); err != nil {
			s.logger.Error("Failed to process stream event",
				"error", err,
				"message_id", headers.MessageID)
			http.Error(w, "Internal server error", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"processed"}`))
		s.logger.Info("Event processed successfully",
			"message_id", headers.MessageID,
			"event_type", processedEvent.Type)

	case "revoke":
		// Unwanted subscription - respond with 410 Gone
		w.WriteHeader(http.StatusGone)
		s.logger.Info("Unwanted subscription, responded with 410 Gone",
			"message_id", headers.MessageID)

	case "ignore":
		// Ignore the event
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ignored"}`))
		s.logger.Debug("Event ignored",
			"message_id", headers.MessageID,
			"event_type", processedEvent.Type)

	default:
		s.logger.Error("Unknown action from processed event",
			"action", processedEvent.Action,
			"message_id", headers.MessageID)
		http.Error(w, "Internal server error", http.StatusInternalServerError)
	}
}

// handleRoot handles requests to the root path
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte("itsjustintv - Twitch EventSub webhook bridge\n"))
}

// processStreamEvent processes a stream.online event and dispatches webhooks
func (s *Server) processStreamEvent(processedEvent *twitch.ProcessedEvent, messageID string) error {
	ctx, span := s.telemetryManager.StartSpan(context.Background(), "process_stream_event",
		attribute.String("message_id", messageID),
		attribute.String("broadcaster_user_id", processedEvent.Event.(twitch.StreamOnlineEvent).BroadcasterUserID))
	defer span.End()

	// Extract stream event data
	streamEvent, ok := processedEvent.Event.(twitch.StreamOnlineEvent)
	if !ok {
		span.RecordError(fmt.Errorf("invalid stream event type"))
		return fmt.Errorf("invalid stream event type")
	}

	// Check for duplicates
	eventKey := s.cacheManager.GenerateEventKey(streamEvent.BroadcasterUserID, streamEvent.ID, streamEvent.StartedAt)
	if s.cacheManager.IsDuplicate(eventKey) {
		s.logger.Info("Duplicate event detected, skipping",
			"event_key", eventKey,
			"broadcaster_login", streamEvent.BroadcasterUserLogin,
			"message_id", messageID)
		span.SetAttributes(attribute.Bool("duplicate", true))
		return nil
	}

	// Add to cache to prevent future duplicates
	eventData, _ := json.Marshal(streamEvent)
	s.cacheManager.AddEvent(eventKey, eventData)
	s.telemetryManager.RecordCacheOperation(ctx, "add", true)

	// Find streamer configuration
	var streamerKey string
	var streamerConfig config.StreamerConfig
	found := false

	for key, cfg := range s.config.Streamers {
		if cfg.UserID == streamEvent.BroadcasterUserID || cfg.Login == streamEvent.BroadcasterUserLogin {
			streamerKey = key
			streamerConfig = cfg
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("streamer configuration not found")
	}

	// Create webhook payload
	eventDataMap := map[string]interface{}{
		"broadcaster_user_id":    streamEvent.BroadcasterUserID,
		"broadcaster_user_login": streamEvent.BroadcasterUserLogin,
		"broadcaster_user_name":  streamEvent.BroadcasterUserName,
		"id":                     streamEvent.ID,
		"type":                   streamEvent.Type,
		"started_at":             streamEvent.StartedAt,
	}

	payload := s.webhookDispatcher.CreatePayload(streamerKey, streamerConfig, eventDataMap)

	// Enrich payload with metadata and apply tag filtering
	enrichCtx, enrichCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer enrichCancel()

	if err := s.enricher.EnrichPayload(enrichCtx, payload, streamerConfig); err != nil {
		if err.Error() == "stream blocked by tag filter" {
			s.logger.Info("Stream blocked by tag filter, skipping webhook dispatch",
				"streamer_key", streamerKey,
				"streamer_login", streamEvent.BroadcasterUserLogin)
			return nil
		}

		s.logger.Warn("Failed to enrich payload, continuing with basic data",
			"error", err,
			"streamer_key", streamerKey)
	}

	// Determine webhook URL and secret
	webhookURL := streamerConfig.TargetWebhookURL
	webhookSecret := streamerConfig.TargetWebhookSecret
	webhookHeader := streamerConfig.TargetWebhookHeader
	webhookHashing := streamerConfig.TargetWebhookHashing

	// Use global webhook if streamer-specific URL is not provided and global is enabled
	if webhookURL == "" && s.config.GlobalWebhook.Enabled && s.config.GlobalWebhook.URL != "" {
		webhookURL = s.config.GlobalWebhook.URL
		webhookSecret = s.config.GlobalWebhook.TargetWebhookSecret
		webhookHeader = s.config.GlobalWebhook.TargetWebhookHeader
		webhookHashing = s.config.GlobalWebhook.TargetWebhookHashing
		s.logger.Debug("Using global webhook configuration",
			"streamer_key", streamerKey,
			"webhook_url", webhookURL)
	}

	// Validate webhook URL
	if webhookURL == "" {
		s.logger.Error("No webhook URL configured for streamer",
			"streamer_key", streamerKey,
			"has_global_webhook", s.config.GlobalWebhook.Enabled)
		return fmt.Errorf("no webhook URL configured for streamer: %s", streamerKey)
	}

	// Create dispatch request
	dispatchReq := &webhook.DispatchRequest{
		WebhookURL:     webhookURL,
		Payload:        *payload,
		WebhookSecret:  webhookSecret,
		WebhookHeader:  webhookHeader,
		WebhookHashing: webhookHashing,
		StreamerKey:    streamerKey,
		Attempt:        1,
	}

	// Attempt initial dispatch
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result := s.webhookDispatcher.Dispatch(ctx, dispatchReq)

	// Write to output file
	errorMsg := ""
	if !result.Success {
		errorMsg = result.Error
		// Add to retry queue
		s.retryManager.AddRequest(dispatchReq)
		s.logger.Warn("Initial webhook dispatch failed, added to retry queue",
			"webhook_url", dispatchReq.WebhookURL,
			"streamer_key", streamerKey,
			"error", result.Error,
			"status_code", result.StatusCode)
	} else {
		s.logger.Info("Webhook dispatched successfully",
			"webhook_url", dispatchReq.WebhookURL,
			"streamer_key", streamerKey,
			"response_time", result.ResponseTime)
	}

	// Write payload to output file
	if err := s.outputWriter.WritePayload(*payload, result.Success, errorMsg); err != nil {
		s.logger.Warn("Failed to write payload to output file", "error", err)
	}

	return nil
}
