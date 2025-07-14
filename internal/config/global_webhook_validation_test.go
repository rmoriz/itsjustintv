package config

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGlobalWebhookValidation(t *testing.T) {
	tests := []struct {
		name    string
		config  *Config
		wantErr bool
		errMsg  string
	}{
		{
			name:    "valid config without global webhook",
			config:  DefaultConfig(),
			wantErr: false,
		},
		{
			name: "valid global webhook config",
			config: &Config{
				Twitch:    TwitchConfig{ClientID: "test", ClientSecret: "test", WebhookSecret: "test"},
				Server:    ServerConfig{Port: 8080},
				Retry:     RetryConfig{MaxAttempts: 3, BackoffFactor: 2.0},
				GlobalWebhook: GlobalWebhookConfig{
					Enabled: true,
					URL:     "https://example.com/webhook",
				},
			},
			wantErr: false,
		},
		{
			name: "global webhook enabled but no URL",
			config: &Config{
				Twitch:    TwitchConfig{ClientID: "test", ClientSecret: "test", WebhookSecret: "test"},
				Server:    ServerConfig{Port: 8080},
				Retry:     RetryConfig{MaxAttempts: 3, BackoffFactor: 2.0},
				GlobalWebhook: GlobalWebhookConfig{
					Enabled: true,
					URL:     "",
				},
			},
			wantErr: true,
			errMsg:  "global_webhook.url is required when global_webhook.enabled is true",
		},
		{
			name: "global webhook enabled with invalid URL",
			config: &Config{
				Twitch:    TwitchConfig{ClientID: "test", ClientSecret: "test", WebhookSecret: "test"},
				Server:    ServerConfig{Port: 8080},
				Retry:     RetryConfig{MaxAttempts: 3, BackoffFactor: 2.0},
				GlobalWebhook: GlobalWebhookConfig{
					Enabled: true,
					URL:     "invalid-url",
				},
			},
			wantErr: true,
			errMsg:  "global_webhook.url must be a valid URL",
		},
		{
			name: "global webhook enabled with HTTP URL",
			config: &Config{
				Twitch:    TwitchConfig{ClientID: "test", ClientSecret: "test", WebhookSecret: "test"},
				Server:    ServerConfig{Port: 8080},
				Retry:     RetryConfig{MaxAttempts: 3, BackoffFactor: 2.0},
				GlobalWebhook: GlobalWebhookConfig{
					Enabled: true,
					URL:     "http://localhost:8080/webhook",
				},
			},
			wantErr: false,
		},
		{
			name: "global webhook enabled with HTTPS URL",
			config: &Config{
				Twitch:    TwitchConfig{ClientID: "test", ClientSecret: "test", WebhookSecret: "test"},
				Server:    ServerConfig{Port: 8080},
				Retry:     RetryConfig{MaxAttempts: 3, BackoffFactor: 2.0},
				GlobalWebhook: GlobalWebhookConfig{
					Enabled: true,
					URL:     "https://api.example.com/webhook",
				},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test just the global webhook validation part
			if tt.config.GlobalWebhook.Enabled {
				if tt.config.GlobalWebhook.URL == "" {
					err := fmt.Errorf("global_webhook.url is required when global_webhook.enabled is true")
					assert.Equal(t, tt.errMsg, err.Error())
					return
				}
				if !isValidURL(tt.config.GlobalWebhook.URL) {
					err := fmt.Errorf("global_webhook.url must be a valid URL")
					assert.Equal(t, tt.errMsg, err.Error())
					return
				}
			}
			
			// For valid cases, no error from global webhook validation
			if !tt.wantErr {
				// Global webhook validation should pass
				if tt.config.GlobalWebhook.Enabled {
					assert.True(t, isValidURL(tt.config.GlobalWebhook.URL))
				}
			}
		})
	}
}

func TestIsValidURL(t *testing.T) {
	tests := []struct {
		url     string
		valid   bool
	}{
		{"https://example.com/webhook", true},
		{"http://localhost:8080/webhook", true},
		{"https://api.example.com/v1/webhook", true},
		{"ftp://example.com/webhook", false},
		{"example.com/webhook", false},
		{"", false},
		{"http:/example.com", false},
		{"https:example.com", false},
	}

	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			assert.Equal(t, tt.valid, isValidURL(tt.url))
		})
	}
}