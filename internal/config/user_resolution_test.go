package config

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockTwitchUserResolver implements TwitchUserResolver for testing
type MockTwitchUserResolver struct {
	users map[string]*MockTwitchUserInfo
}

type MockTwitchUserInfo struct {
	id    string
	login string
}

func (u *MockTwitchUserInfo) GetID() string {
	return u.id
}

func (u *MockTwitchUserInfo) GetLogin() string {
	return u.login
}

func (m *MockTwitchUserResolver) GetUserInfoByLoginForConfig(ctx context.Context, login string) (TwitchUserInfo, error) {
	if user, exists := m.users[login]; exists {
		return user, nil
	}
	return nil, assert.AnError
}

func TestResolveStreamerUserIDs(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		mockUsers      map[string]*MockTwitchUserInfo
		expectedConfig *Config
		expectError    bool
	}{
		{
			name: "resolve missing user ID",
			config: &Config{
				Streamers: map[string]StreamerConfig{
					"test_streamer": {
						Login:      "testuser",
						WebhookURL: "https://example.com/webhook",
					},
				},
			},
			mockUsers: map[string]*MockTwitchUserInfo{
				"testuser": {id: "123456789", login: "testuser"},
			},
			expectedConfig: &Config{
				Streamers: map[string]StreamerConfig{
					"test_streamer": {
						UserID:     "123456789",
						Login:      "testuser",
						WebhookURL: "https://example.com/webhook",
					},
				},
			},
			expectError: false,
		},
		{
			name: "skip streamer with existing user ID",
			config: &Config{
				Streamers: map[string]StreamerConfig{
					"existing_streamer": {
						UserID:     "987654321",
						Login:      "existinguser",
						WebhookURL: "https://example.com/webhook",
					},
				},
			},
			mockUsers: map[string]*MockTwitchUserInfo{
				"existinguser": {id: "123456789", login: "existinguser"},
			},
			expectedConfig: &Config{
				Streamers: map[string]StreamerConfig{
					"existing_streamer": {
						UserID:     "987654321", // Should remain unchanged
						Login:      "existinguser",
						WebhookURL: "https://example.com/webhook",
					},
				},
			},
			expectError: false,
		},
		{
			name: "skip streamer without login",
			config: &Config{
				Streamers: map[string]StreamerConfig{
					"no_login_streamer": {
						WebhookURL: "https://example.com/webhook",
					},
				},
			},
			mockUsers: map[string]*MockTwitchUserInfo{},
			expectedConfig: &Config{
				Streamers: map[string]StreamerConfig{
					"no_login_streamer": {
						WebhookURL: "https://example.com/webhook",
					},
				},
			},
			expectError: false,
		},
		{
			name: "error when user not found",
			config: &Config{
				Streamers: map[string]StreamerConfig{
					"unknown_streamer": {
						Login:      "unknownuser",
						WebhookURL: "https://example.com/webhook",
					},
				},
			},
			mockUsers:   map[string]*MockTwitchUserInfo{},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockResolver := &MockTwitchUserResolver{users: tt.mockUsers}

			err := ResolveStreamerUserIDs(context.Background(), tt.config, mockResolver)

			if tt.expectError {
				require.Error(t, err)
				return
			}

			require.NoError(t, err)
			assert.Equal(t, tt.expectedConfig.Streamers, tt.config.Streamers)
		})
	}
}
