package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGlobalWebhookConfig(t *testing.T) {
	tests := []struct {
		name           string
		config         *Config
		expectedGlobal GlobalWebhookConfig
	}{
		{
			name:   "default config has disabled global webhook",
			config: DefaultConfig(),
			expectedGlobal: GlobalWebhookConfig{
				Enabled:    false,
				URL:        "",
				HMACSecret: "",
			},
		},
		{
			name: "config with global webhook enabled",
			config: &Config{
				GlobalWebhook: GlobalWebhookConfig{
					Enabled:    true,
					URL:        "https://example.com/webhook",
					HMACSecret: "test-secret",
				},
			},
			expectedGlobal: GlobalWebhookConfig{
				Enabled:    true,
				URL:        "https://example.com/webhook",
				HMACSecret: "test-secret",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expectedGlobal.Enabled, tt.config.GlobalWebhook.Enabled)
			assert.Equal(t, tt.expectedGlobal.URL, tt.config.GlobalWebhook.URL)
			assert.Equal(t, tt.expectedGlobal.HMACSecret, tt.config.GlobalWebhook.HMACSecret)
		})
	}
}