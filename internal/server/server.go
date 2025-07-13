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

	"golang.org/x/crypto/acme/autocert"
	"github.com/rmoriz/itsjustintv/internal/cache"
	"github.com/rmoriz/itsjustintv/internal/config"
	"github.com/rmoriz/itsjustintv/internal/output"
	"github.com/rmoriz/itsjustintv/internal/retry"
	"github.com/rmoriz/itsjustintv/internal/twitch"
	"github.com/rmoriz/itsjustintv/internal/webhook"
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
	}
}

// Start starts the HTTP server with optional HTTPS
func (s *Server) Start(ctx context.Context) error {
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

	// Start subscription manager
	if err := s.subscriptionManager.Start(ctx); err != nil {
		return fmt.Errorf("failed to start subscription manager: %w", err)
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

	s.logger.Info("Server stopped gracefully")
	return nil
}

// setupRoutes configures the HTTP routes
func (s *Server) setupRoutes(mux *http.ServeMux) {
	// Health check endpoint
	mux.HandleFunc("/health", s.handleHealth)
	
	// Twitch webhook endpoint (stub for now)
	mux.HandleFunc("/twitch", s.handleTwitchWebhook)
	
	// Root endpoint
	mux.HandleFunc("/", s.handleRoot)
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
	// Extract stream event data
	streamEvent, ok := processedEvent.Event.(twitch.StreamOnlineEvent)
	if !ok {
		return fmt.Errorf("invalid stream event type")
	}

	// Check for duplicates
	eventKey := s.cacheManager.GenerateEventKey(streamEvent.BroadcasterUserID, streamEvent.ID, streamEvent.StartedAt)
	if s.cacheManager.IsDuplicate(eventKey) {
		s.logger.Info("Duplicate event detected, skipping",
			"event_key", eventKey,
			"broadcaster_login", streamEvent.BroadcasterUserLogin,
			"message_id", messageID)
		return nil
	}

	// Add to cache to prevent future duplicates
	eventData, _ := json.Marshal(streamEvent)
	s.cacheManager.AddEvent(eventKey, eventData)

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

	// Enrich payload with metadata
	enrichCtx, enrichCancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer enrichCancel()

	if err := s.enricher.EnrichPayload(enrichCtx, payload, streamerConfig); err != nil {
		s.logger.Warn("Failed to enrich payload, continuing with basic data",
			"error", err,
			"streamer_key", streamerKey)
	}

	// Create dispatch request
	dispatchReq := &webhook.DispatchRequest{
		WebhookURL:  streamerConfig.WebhookURL,
		Payload:     *payload,
		HMACSecret:  streamerConfig.HMACSecret,
		StreamerKey: streamerKey,
		Attempt:     1,
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