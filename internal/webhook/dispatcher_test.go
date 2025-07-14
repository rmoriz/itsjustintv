package webhook

import (
	"context"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/rmoriz/itsjustintv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDispatcher(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	dispatcher := NewDispatcher(cfg, logger)

	assert.NotNil(t, dispatcher)
	assert.Equal(t, cfg, dispatcher.config)
	assert.Equal(t, logger, dispatcher.logger)
	assert.NotNil(t, dispatcher.httpClient)
}

func TestDispatchSuccess(t *testing.T) {
	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
		assert.Equal(t, "itsjustintv/1.6", r.Header.Get("User-Agent"))

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	dispatcher := NewDispatcher(cfg, logger)

	payload := WebhookPayload{
		StreamerLogin: "teststreamer",
		StreamerName:  "Test Streamer",
		StreamerID:    "123456789",
		URL:           "https://twitch.tv/teststreamer",
		Timestamp:     time.Now(),
	}

	req := &DispatchRequest{
		WebhookURL:  server.URL,
		Payload:     payload,
		StreamerKey: "test_streamer",
		Attempt:     1,
	}

	ctx := context.Background()
	result := dispatcher.Dispatch(ctx, req)

	assert.True(t, result.Success)
	assert.Equal(t, http.StatusOK, result.StatusCode)
	assert.Equal(t, 1, result.Attempt)
	assert.Empty(t, result.Error)
	assert.Greater(t, result.ResponseTime, time.Duration(0))
}

func TestDispatchWithHMAC(t *testing.T) {
	// Create test server that validates HMAC
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		signature := r.Header.Get("X-Signature-256")
		assert.NotEmpty(t, signature)
		assert.Contains(t, signature, "sha256=")

		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"status":"ok"}`))
	}))
	defer server.Close()

	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	dispatcher := NewDispatcher(cfg, logger)

	payload := WebhookPayload{
		StreamerLogin: "teststreamer",
		StreamerName:  "Test Streamer",
		StreamerID:    "123456789",
		URL:           "https://twitch.tv/teststreamer",
		Timestamp:     time.Now(),
	}

	req := &DispatchRequest{
		WebhookURL:  server.URL,
		Payload:     payload,
		HMACSecret:  "test_secret",
		StreamerKey: "test_streamer",
		Attempt:     1,
	}

	ctx := context.Background()
	result := dispatcher.Dispatch(ctx, req)

	assert.True(t, result.Success)
	assert.Equal(t, http.StatusOK, result.StatusCode)
}

func TestDispatchFailure(t *testing.T) {
	// Create test server that returns error
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(`{"error":"internal error"}`))
	}))
	defer server.Close()

	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	dispatcher := NewDispatcher(cfg, logger)

	payload := WebhookPayload{
		StreamerLogin: "teststreamer",
		StreamerName:  "Test Streamer",
		StreamerID:    "123456789",
		URL:           "https://twitch.tv/teststreamer",
		Timestamp:     time.Now(),
	}

	req := &DispatchRequest{
		WebhookURL:  server.URL,
		Payload:     payload,
		StreamerKey: "test_streamer",
		Attempt:     1,
	}

	ctx := context.Background()
	result := dispatcher.Dispatch(ctx, req)

	assert.False(t, result.Success)
	assert.Equal(t, http.StatusInternalServerError, result.StatusCode)
	assert.Equal(t, 1, result.Attempt)
	assert.Contains(t, result.Error, "HTTP 500")
}

func TestDispatchInvalidURL(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	dispatcher := NewDispatcher(cfg, logger)

	payload := WebhookPayload{
		StreamerLogin: "teststreamer",
		StreamerName:  "Test Streamer",
		StreamerID:    "123456789",
		URL:           "https://twitch.tv/teststreamer",
		Timestamp:     time.Now(),
	}

	req := &DispatchRequest{
		WebhookURL:  "invalid-url",
		Payload:     payload,
		StreamerKey: "test_streamer",
		Attempt:     1,
	}

	ctx := context.Background()
	result := dispatcher.Dispatch(ctx, req)

	assert.False(t, result.Success)
	assert.Equal(t, 1, result.Attempt)
	assert.Contains(t, result.Error, "request failed")
}

func TestCreatePayload(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	dispatcher := NewDispatcher(cfg, logger)

	streamerConfig := config.StreamerConfig{
		UserID:         "123456789",
		Login:          "teststreamer",
		WebhookURL:     "https://example.com/webhook",
		AdditionalTags: []string{"vip", "partner"},
	}

	eventData := map[string]interface{}{
		"broadcaster_user_id":    "123456789",
		"broadcaster_user_login": "teststreamer",
		"broadcaster_user_name":  "Test Streamer",
		"id":                     "stream_123",
		"type":                   "live",
	}

	payload := dispatcher.CreatePayload("test_streamer", streamerConfig, eventData)

	assert.Equal(t, "teststreamer", payload.StreamerLogin)
	assert.Equal(t, "Test Streamer", payload.StreamerName)
	assert.Equal(t, "123456789", payload.StreamerID)
	assert.Equal(t, "https://twitch.tv/teststreamer", payload.URL)
	assert.Equal(t, []string{"vip", "partner"}, payload.AdditionalTags)
	assert.False(t, payload.Timestamp.IsZero())
}

func TestCreatePayloadFallbacks(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	dispatcher := NewDispatcher(cfg, logger)

	streamerConfig := config.StreamerConfig{
		UserID:     "123456789",
		Login:      "teststreamer",
		WebhookURL: "https://example.com/webhook",
	}

	// Event data with missing fields
	eventData := map[string]interface{}{
		"broadcaster_user_id": "123456789",
	}

	payload := dispatcher.CreatePayload("test_streamer", streamerConfig, eventData)

	assert.Equal(t, "teststreamer", payload.StreamerLogin)
	assert.Equal(t, "teststreamer", payload.StreamerName) // Fallback to login
	assert.Equal(t, "123456789", payload.StreamerID)
	assert.Equal(t, "https://twitch.tv/teststreamer", payload.URL)
}

func TestWebhookPayloadJSON(t *testing.T) {
	payload := WebhookPayload{
		StreamerLogin:  "teststreamer",
		StreamerName:   "Test Streamer",
		StreamerID:     "123456789",
		URL:            "https://twitch.tv/teststreamer",
		ViewCount:      1337,
		FollowersCount: 50000,
		Tags:           []string{"English", "Gaming"},
		Language:       "en",
		Description:    "Playing games",
		Timestamp:      time.Date(2025, 7, 13, 12, 0, 0, 0, time.UTC),
		AdditionalTags: []string{"vip"},
	}

	data, err := json.Marshal(payload)
	require.NoError(t, err)

	var unmarshaled WebhookPayload
	err = json.Unmarshal(data, &unmarshaled)
	require.NoError(t, err)

	assert.Equal(t, payload.StreamerLogin, unmarshaled.StreamerLogin)
	assert.Equal(t, payload.StreamerName, unmarshaled.StreamerName)
	assert.Equal(t, payload.StreamerID, unmarshaled.StreamerID)
	assert.Equal(t, payload.ViewCount, unmarshaled.ViewCount)
	assert.Equal(t, payload.Tags, unmarshaled.Tags)
	assert.Equal(t, payload.AdditionalTags, unmarshaled.AdditionalTags)
}
