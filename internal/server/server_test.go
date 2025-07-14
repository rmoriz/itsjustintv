package server

import (
	"crypto/tls"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/rmoriz/itsjustintv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
	cfg.Twitch.WebhookSecret = "test_secret"
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	server := New(cfg, logger)

	// Create a valid webhook payload
	validPayload := `{"challenge":"test_challenge","subscription":{"id":"test","type":"stream.online"}}`

	// Generate valid signature
	signature := server.webhookValidator.GenerateSignature([]byte(validPayload))

	tests := []struct {
		name           string
		method         string
		payload        string
		signature      string
		headers        map[string]string
		expectedStatus int
		expectedBody   string
	}{
		{
			name:      "POST request with valid signature - verification",
			method:    http.MethodPost,
			payload:   validPayload,
			signature: signature,
			headers: map[string]string{
				"Twitch-Eventsub-Message-Type": "webhook_callback_verification",
			},
			expectedStatus: http.StatusOK,
			expectedBody:   "test_challenge",
		},
		{
			name:           "POST request with invalid signature",
			method:         http.MethodPost,
			payload:        validPayload,
			signature:      "invalid_signature",
			expectedStatus: http.StatusUnauthorized,
			expectedBody:   "Unauthorized",
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
			var req *http.Request
			if tt.payload != "" {
				req = httptest.NewRequest(tt.method, "/twitch", strings.NewReader(tt.payload))
			} else {
				req = httptest.NewRequest(tt.method, "/twitch", nil)
			}

			if tt.signature != "" {
				req.Header.Set("Twitch-Eventsub-Message-Signature", tt.signature)
			}

			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}

			w := httptest.NewRecorder()

			server.handleTwitchWebhook(w, req)

			resp := w.Result()
			body, _ := io.ReadAll(resp.Body)

			assert.Equal(t, tt.expectedStatus, resp.StatusCode)
			assert.Contains(t, string(body), tt.expectedBody)
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
	// Skip this test as it requires real Twitch API credentials
	t.Skip("Skipping integration test that requires Twitch API credentials")
}
