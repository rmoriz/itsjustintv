package server

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/rmoriz/itsjustintv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerIntegrationHTTP(t *testing.T) {
	// Skip this test as it requires external dependencies and real network calls
	t.Skip("Skipping integration test that requires Twitch API and external dependencies")

	// Create test configuration
	cfg := config.DefaultConfig()
	cfg.Server.ListenAddr = "127.0.0.1"
	cfg.Server.Port = 18080 // Use fixed port for testing
	cfg.Server.TLS.Enabled = false
	cfg.Twitch.ClientID = "test_client_id"
	cfg.Twitch.ClientSecret = "test_client_secret"
	cfg.Twitch.WebhookSecret = "test_webhook_secret"

	logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
	server := New(cfg, logger)

	// Start server in background
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	serverDone := make(chan error, 1)
	go func() {
		serverDone <- server.Start(ctx)
	}()

	// Give server time to start
	time.Sleep(200 * time.Millisecond)

	baseURL := fmt.Sprintf("http://127.0.0.1:%d", cfg.Server.Port)

	// Test health endpoint
	t.Run("health endpoint", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/health")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "healthy")
		assert.Contains(t, string(body), "itsjustintv")
	})

	// Test root endpoint
	t.Run("root endpoint", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "text/plain", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "itsjustintv")
	})

	// Test Twitch webhook endpoint
	t.Run("twitch webhook endpoint", func(t *testing.T) {
		// Test without signature (should fail)
		resp, err := http.Post(baseURL+"/twitch", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusUnauthorized, resp.StatusCode)

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "Unauthorized")
	})

	// Test 404 for unknown paths
	t.Run("404 for unknown paths", func(t *testing.T) {
		resp, err := http.Get(baseURL + "/unknown")
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusNotFound, resp.StatusCode)
	})

	// Stop server
	cancel()

	// Wait for server to stop
	select {
	case err := <-serverDone:
		assert.NoError(t, err)
	case <-time.After(3 * time.Second):
		t.Fatal("Server did not stop within timeout")
	}
}
