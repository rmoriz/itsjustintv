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

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/rmoriz/itsjustintv/internal/config"
)

func TestServerIntegrationHTTP(t *testing.T) {
	// Skip in short mode
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// Create test configuration
	cfg := config.DefaultConfig()
	cfg.Server.ListenAddr = "127.0.0.1"
	cfg.Server.Port = 0 // Use random available port
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

	// Get the actual port the server is listening on
	// Since we used port 0, we need to extract it from the server
	actualAddr := server.httpServer.Addr
	if actualAddr == "127.0.0.1:0" {
		// In real integration tests, we'd need a way to get the actual port
		// For now, we'll test with a fixed port
		cfg.Server.Port = 18080
		cancel() // Stop the current server
		
		// Wait for it to stop
		select {
		case <-serverDone:
		case <-time.After(2 * time.Second):
		}

		// Start new server with fixed port
		server = New(cfg, logger)
		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
		
		go func() {
			serverDone <- server.Start(ctx)
		}()
		
		time.Sleep(200 * time.Millisecond)
	}

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
		resp, err := http.Post(baseURL+"/twitch", "application/json", nil)
		require.NoError(t, err)
		defer resp.Body.Close()

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.Equal(t, "application/json", resp.Header.Get("Content-Type"))

		body, err := io.ReadAll(resp.Body)
		require.NoError(t, err)
		assert.Contains(t, string(body), "received")
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