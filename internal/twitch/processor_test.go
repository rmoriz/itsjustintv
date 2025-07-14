package twitch

import (
	"encoding/json"
	"log/slog"
	"os"
	"testing"
	"time"

	"github.com/rmoriz/itsjustintv/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewProcessor(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))

	processor := NewProcessor(cfg, logger)

	assert.NotNil(t, processor)
	assert.Equal(t, cfg, processor.config)
	assert.Equal(t, logger, processor.logger)
}

func TestProcessNotificationVerification(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	processor := NewProcessor(cfg, logger)

	headers := EventSubHeaders{
		MessageType: MessageTypeWebhookCallbackVerification,
	}

	notification := EventSubNotification{
		Challenge: "test_challenge_123",
		Subscription: EventSubSubscription{
			ID:   "sub_123",
			Type: "stream.online",
		},
	}

	payload, err := json.Marshal(notification)
	require.NoError(t, err)

	result, err := processor.ProcessNotification(headers, payload)
	require.NoError(t, err)

	assert.Equal(t, "verification", result.Type)
	assert.Equal(t, "respond", result.Action)
	assert.Equal(t, "test_challenge_123", result.Challenge)
}

func TestProcessNotificationStreamOnlineConfiguredStreamer(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Streamers = map[string]config.StreamerConfig{
		"test_streamer": {
			UserID:           "123456789",
			Login:            "teststreamer",
			TargetWebhookURL: "https://example.com/webhook",
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	processor := NewProcessor(cfg, logger)

	headers := EventSubHeaders{
		MessageType:      MessageTypeNotification,
		SubscriptionType: "stream.online",
	}

	streamEvent := StreamOnlineEvent{
		ID:                   "stream_123",
		BroadcasterUserID:    "123456789",
		BroadcasterUserLogin: "teststreamer",
		BroadcasterUserName:  "Test Streamer",
		Type:                 "live",
		StartedAt:            time.Now(),
	}

	notification := EventSubNotification{
		Event: streamEvent,
		Subscription: EventSubSubscription{
			ID:   "sub_123",
			Type: "stream.online",
		},
	}

	payload, err := json.Marshal(notification)
	require.NoError(t, err)

	result, err := processor.ProcessNotification(headers, payload)
	require.NoError(t, err)

	assert.Equal(t, "stream.online", result.Type)
	assert.Equal(t, "process", result.Action)
	assert.NotNil(t, result.Event)
}

func TestProcessNotificationStreamOnlineUnconfiguredStreamer(t *testing.T) {
	cfg := config.DefaultConfig()
	// No streamers configured

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	processor := NewProcessor(cfg, logger)

	headers := EventSubHeaders{
		MessageType:      MessageTypeNotification,
		SubscriptionType: "stream.online",
	}

	streamEvent := StreamOnlineEvent{
		ID:                   "stream_123",
		BroadcasterUserID:    "999999999",
		BroadcasterUserLogin: "unknownstreamer",
		BroadcasterUserName:  "Unknown Streamer",
		Type:                 "live",
		StartedAt:            time.Now(),
	}

	notification := EventSubNotification{
		Event: streamEvent,
		Subscription: EventSubSubscription{
			ID:   "sub_123",
			Type: "stream.online",
		},
	}

	payload, err := json.Marshal(notification)
	require.NoError(t, err)

	result, err := processor.ProcessNotification(headers, payload)
	require.NoError(t, err)

	assert.Equal(t, "unconfigured_streamer", result.Type)
	assert.Equal(t, "revoke", result.Action)
}

func TestProcessNotificationRevocation(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	processor := NewProcessor(cfg, logger)

	headers := EventSubHeaders{
		MessageType: MessageTypeRevocation,
	}

	notification := EventSubNotification{
		Subscription: EventSubSubscription{
			ID:     "sub_123",
			Type:   "stream.online",
			Status: SubscriptionStatusAuthorizationRevoked,
		},
	}

	payload, err := json.Marshal(notification)
	require.NoError(t, err)

	result, err := processor.ProcessNotification(headers, payload)
	require.NoError(t, err)

	assert.Equal(t, "revocation", result.Type)
	assert.Equal(t, "ignore", result.Action)
}

func TestProcessNotificationUnsupportedSubscriptionType(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	processor := NewProcessor(cfg, logger)

	headers := EventSubHeaders{
		MessageType:      MessageTypeNotification,
		SubscriptionType: "unsupported.type",
	}

	notification := EventSubNotification{
		Event: map[string]interface{}{"test": "data"},
		Subscription: EventSubSubscription{
			ID:   "sub_123",
			Type: "unsupported.type",
		},
	}

	payload, err := json.Marshal(notification)
	require.NoError(t, err)

	result, err := processor.ProcessNotification(headers, payload)
	require.NoError(t, err)

	assert.Equal(t, "unsupported", result.Type)
	assert.Equal(t, "ignore", result.Action)
}

func TestProcessNotificationUnknownMessageType(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	processor := NewProcessor(cfg, logger)

	headers := EventSubHeaders{
		MessageType: "unknown_message_type",
	}

	notification := EventSubNotification{}
	payload, err := json.Marshal(notification)
	require.NoError(t, err)

	_, err = processor.ProcessNotification(headers, payload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "unknown message type")
}

func TestProcessNotificationInvalidJSON(t *testing.T) {
	cfg := config.DefaultConfig()
	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	processor := NewProcessor(cfg, logger)

	headers := EventSubHeaders{
		MessageType: MessageTypeNotification,
	}

	invalidPayload := []byte(`{"invalid": json}`)

	_, err := processor.ProcessNotification(headers, invalidPayload)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal notification")
}

func TestFindStreamerConfig(t *testing.T) {
	cfg := config.DefaultConfig()
	cfg.Streamers = map[string]config.StreamerConfig{
		"streamer1": {
			UserID: "123456789",
			Login:  "teststreamer1",
		},
		"streamer2": {
			UserID: "987654321",
			Login:  "teststreamer2",
		},
	}

	logger := slog.New(slog.NewTextHandler(os.Stdout, nil))
	processor := NewProcessor(cfg, logger)

	tests := []struct {
		name     string
		userID   string
		login    string
		expected bool
	}{
		{
			name:     "find by user ID",
			userID:   "123456789",
			login:    "different_login",
			expected: true,
		},
		{
			name:     "find by login",
			userID:   "different_id",
			login:    "teststreamer2",
			expected: true,
		},
		{
			name:     "find by login case insensitive",
			userID:   "different_id",
			login:    "TESTSTREAMER1",
			expected: true,
		},
		{
			name:     "not found",
			userID:   "unknown_id",
			login:    "unknown_login",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := processor.findStreamerConfig(tt.userID, tt.login)
			if tt.expected {
				assert.NotNil(t, result)
			} else {
				assert.Nil(t, result)
			}
		})
	}
}

func TestFindStreamerConfigKey(t *testing.T) {
	streamers := map[string]config.StreamerConfig{
		"streamer1": {
			UserID: "123456789",
			Login:  "teststreamer1",
		},
		"streamer2": {
			UserID: "987654321",
			Login:  "teststreamer2",
		},
	}

	tests := []struct {
		name     string
		userID   string
		login    string
		expected string
	}{
		{
			name:     "find by user ID",
			userID:   "123456789",
			login:    "different_login",
			expected: "streamer1",
		},
		{
			name:     "find by login",
			userID:   "different_id",
			login:    "teststreamer2",
			expected: "streamer2",
		},
		{
			name:     "not found",
			userID:   "unknown_id",
			login:    "unknown_login",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := findStreamerConfigKey(streamers, tt.userID, tt.login)
			assert.Equal(t, tt.expected, result)
		})
	}
}
