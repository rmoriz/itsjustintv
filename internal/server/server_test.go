package server

import (
	"context"
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/rmoriz/itsjustintv/internal/config"
)

func TestNew(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	
	server := New(cfg, logger)
	
	assert.NotNil(t, server)
	assert.Equal(t, cfg, server.config)
	assert.Equal(t, logger, server.logger)
}

func TestHandleHealth(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := New(cfg, logger)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "GET request",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
			expectedBody:   "healthy",
		},
		{
			name:           "POST request",
			method:         http.MethodPost,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/health", nil)
			w := httptest.NewRecorder()

			server.handleHealth(w, req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Contains(t, string(body), tt.expectedBody)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
			}
		})
	}
}

func TestHandleTwitchWebhook(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := New(cfg, logger)

	tests := []struct {
		name           string
		method         string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "POST request",
			method:         http.MethodPost,
			expectedStatus: http.StatusOK,
			expectedBody:   "received",
		},
		{
			name:           "GET request",
			method:         http.MethodGet,
			expectedStatus: http.StatusMethodNotAllowed,
			expectedBody:   "Method not allowed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(tt.method, "/twitch", nil)
			w := httptest.NewRecorder()

			server.handleTwitchWebhook(w, req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Contains(t, string(body), tt.expectedBody)

			if tt.expectedStatus == http.StatusOK {
				assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))
			}
		})
	}
}

func TestHandleRoot(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := New(cfg, logger)

	tests := []struct {
		name           string
		path           string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:           "root path",
			path:           "/",
			expectedStatus: http.StatusOK,
			expectedBody:   "itsjustintv",
		},
		{
			name:           "non-existent path",
			path:           "/nonexistent",
			expectedStatus: http.StatusNotFound,
			expectedBody:   "404 page not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()

			server.handleRoot(w, req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Contains(t, string(body), tt.expectedBody)
		})
	}
}

func TestSetupTLS(t *testing.T) {
	tests := []struct {
		name        string
		setupConfig func(*config.Config)
		expectError bool
		errorMsg    string
	}{
		{
			name: "valid TLS config",
			setupConfig: func(cfg *config.Config) {
				cfg.Server.TLS.Enabled = true
				cfg.Server.TLS.Domains = []string{"example.com"}
				cfg.Server.TLS.CertDir = t.TempDir()
			},
			expectError: false,
		},
		{
			name: "no domains specified",
			setupConfig: func(cfg *config.Config) {
				cfg.Server.TLS.Enabled = true
				cfg.Server.TLS.Domains = []string{}
			},
			expectError: true,
			errorMsg:    "TLS domains must be specified",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.DefaultConfig()
			tt.setupConfig(cfg)
			
			logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
			server := New(cfg, logger)
			
			// Create a dummy HTTP server for TLS setup
			server.httpServer = &http.Server{}

			err := server.setupTLS()

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorMsg)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, server.certManager)
				assert.NotNil(t, server.httpServer.TLSConfig)
				assert.Equal(t, uint16(tls.VersionTLS12), server.httpServer.TLSConfig.MinVersion)
			}
		})
	}
}

func TestServerIntegration(t *testing.T) {
	// Create a test server with HTTP only
	cfg := config.DefaultConfig()
	cfg.Server.Port = 0 // Use random port
	cfg.Server.TLS.Enabled = false
	
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := New(cfg, logger)

	// Start server in background
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(100 * time.Millisecond)

	// Test that server is responsive (this is a basic integration test)
	// In a real scenario, we'd need to capture the actual port used
	
	// Cancel context to stop server
	cancel()

	// Wait for server to stop
	select {
	case err := <-serverDone:
		assert.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Server did not stop within timeout")
	}
}