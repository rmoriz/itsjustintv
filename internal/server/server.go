package server

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"golang.org/x/crypto/acme/autocert"
	"github.com/rmoriz/itsjustintv/internal/config"
)

// Server represents the HTTP server with optional HTTPS support
type Server struct {
	config     *config.Config
	httpServer *http.Server
	logger     *slog.Logger
	certManager *autocert.Manager
}

// New creates a new server instance
func New(cfg *config.Config, logger *slog.Logger) *Server {
	return &Server{
		config: cfg,
		logger: logger,
	}
}

// Start starts the HTTP server with optional HTTPS
func (s *Server) Start(ctx context.Context) error {
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
	w.Write([]byte(response))
	
	s.logger.Debug("Health check requested", "remote_addr", r.RemoteAddr)
}

// handleTwitchWebhook handles Twitch EventSub webhooks (stub for now)
func (s *Server) handleTwitchWebhook(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}

	// TODO: Implement actual webhook processing in Milestone 3
	s.logger.Info("Twitch webhook received (stub)", 
		"remote_addr", r.RemoteAddr,
		"content_length", r.ContentLength)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(`{"status":"received"}`))
}

// handleRoot handles requests to the root path
func (s *Server) handleRoot(w http.ResponseWriter, r *http.Request) {
	if r.URL.Path != "/" {
		http.NotFound(w, r)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("itsjustintv - Twitch EventSub webhook bridge\n"))
}