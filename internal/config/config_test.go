package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "0.0.0.0", cfg.Server.ListenAddr)
	assert.Equal(t, 8080, cfg.Server.Port)
	assert.False(t, cfg.Server.TLS.Enabled)
	assert.Equal(t, "data/acme_certs", cfg.Server.TLS.CertDir)

	assert.Equal(t, "data/tokens.json", cfg.Twitch.TokenFile)

	assert.Equal(t, 3, cfg.Retry.MaxAttempts)
	assert.Equal(t, time.Second, cfg.Retry.InitialDelay)
	assert.Equal(t, time.Minute*5, cfg.Retry.MaxDelay)
	assert.Equal(t, 2.0, cfg.Retry.BackoffFactor)

	assert.True(t, cfg.Output.Enabled)
	assert.Equal(t, "data/output.json", cfg.Output.FilePath)
	assert.Equal(t, 1000, cfg.Output.MaxLines)

	assert.False(t, cfg.Telemetry.Enabled)
	assert.Equal(t, "itsjustintv", cfg.Telemetry.ServiceName)
	assert.Equal(t, "0.1.0", cfg.Telemetry.ServiceVersion)

	assert.NotNil(t, cfg.Streamers)
	assert.Empty(t, cfg.Streamers)
}

func TestLoadConfigFromFile(t *testing.T) {
	// Create a temporary config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.toml")

	configContent := `
[server]
listen_addr = "127.0.0.1"
port = 9090

[server.tls]
enabled = true
domains = ["test.example.com"]

[twitch]
client_id = "test_client_id"
client_secret = "test_client_secret"
webhook_secret = "test_webhook_secret"

[streamers.test_streamer]
user_id = "123456"
login = "test_streamer"
webhook_url = "https://test.com/webhook"
tag_filter = ["English"]
additional_tags = ["test"]
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	// Load the config
	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	// Verify loaded values
	assert.Equal(t, "127.0.0.1", cfg.Server.ListenAddr)
	assert.Equal(t, 9090, cfg.Server.Port)
	assert.True(t, cfg.Server.TLS.Enabled)
	assert.Equal(t, []string{"test.example.com"}, cfg.Server.TLS.Domains)

	assert.Equal(t, "test_client_id", cfg.Twitch.ClientID)
	assert.Equal(t, "test_client_secret", cfg.Twitch.ClientSecret)
	assert.Equal(t, "test_webhook_secret", cfg.Twitch.WebhookSecret)

	// Verify streamer config
	require.Contains(t, cfg.Streamers, "test_streamer")
	streamer := cfg.Streamers["test_streamer"]
	assert.Equal(t, "123456", streamer.UserID)
	assert.Equal(t, "test_streamer", streamer.Login)
	assert.Equal(t, "https://test.com/webhook", streamer.WebhookURL)
	assert.Equal(t, []string{"English"}, streamer.TagFilter)
	assert.Equal(t, []string{"test"}, streamer.AdditionalTags)
}

func TestLoadConfigNonExistentFile(t *testing.T) {
	// Loading a non-existent file should fail validation due to missing required fields
	_, err := LoadConfig("non_existent_file.toml")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "client_id is required")
}

func TestEnvironmentOverrides(t *testing.T) {
	// Set environment variables
	os.Setenv("ITSJUSTINTV_SERVER_LISTEN_ADDR", "192.168.1.1")
	os.Setenv("ITSJUSTINTV_SERVER_PORT", "3000")
	os.Setenv("ITSJUSTINTV_TWITCH_CLIENT_ID", "env_client_id")
	os.Setenv("ITSJUSTINTV_TLS_ENABLED", "true")

	defer func() {
		os.Unsetenv("ITSJUSTINTV_SERVER_LISTEN_ADDR")
		os.Unsetenv("ITSJUSTINTV_SERVER_PORT")
		os.Unsetenv("ITSJUSTINTV_TWITCH_CLIENT_ID")
		os.Unsetenv("ITSJUSTINTV_TLS_ENABLED")
	}()

	// Create minimal config file
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "test_config.toml")
	configContent := `
[twitch]
client_secret = "test_secret"
webhook_secret = "test_webhook_secret"

[server.tls]
domains = ["test.com"]
`
	err := os.WriteFile(configPath, []byte(configContent), 0644)
	require.NoError(t, err)

	cfg, err := LoadConfig(configPath)
	require.NoError(t, err)

	// Verify environment overrides
	assert.Equal(t, "192.168.1.1", cfg.Server.ListenAddr)
	assert.Equal(t, 3000, cfg.Server.Port)
	assert.Equal(t, "env_client_id", cfg.Twitch.ClientID)
	assert.True(t, cfg.Server.TLS.Enabled)

	// Verify file values are still present
	assert.Equal(t, "test_secret", cfg.Twitch.ClientSecret)
	assert.Equal(t, "test_webhook_secret", cfg.Twitch.WebhookSecret)
}

func TestConfigValidation(t *testing.T) {
	tests := []struct {
		name          string
		modifyConfig  func(*Config)
		expectError   bool
		errorContains string
	}{
		{
			name: "valid config",
			modifyConfig: func(cfg *Config) {
				cfg.Twitch.ClientID = "test_id"
				cfg.Twitch.ClientSecret = "test_secret"
				cfg.Twitch.WebhookSecret = "test_webhook_secret"
			},
			expectError: false,
		},
		{
			name: "missing client_id",
			modifyConfig: func(cfg *Config) {
				cfg.Twitch.ClientSecret = "test_secret"
				cfg.Twitch.WebhookSecret = "test_webhook_secret"
			},
			expectError:   true,
			errorContains: "client_id is required",
		},
		{
			name: "missing client_secret",
			modifyConfig: func(cfg *Config) {
				cfg.Twitch.ClientID = "test_id"
				cfg.Twitch.WebhookSecret = "test_webhook_secret"
			},
			expectError:   true,
			errorContains: "client_secret is required",
		},
		{
			name: "missing webhook_secret",
			modifyConfig: func(cfg *Config) {
				cfg.Twitch.ClientID = "test_id"
				cfg.Twitch.ClientSecret = "test_secret"
			},
			expectError:   true,
			errorContains: "webhook_secret is required",
		},
		{
			name: "invalid port",
			modifyConfig: func(cfg *Config) {
				cfg.Twitch.ClientID = "test_id"
				cfg.Twitch.ClientSecret = "test_secret"
				cfg.Twitch.WebhookSecret = "test_webhook_secret"
				cfg.Server.Port = 0
			},
			expectError:   true,
			errorContains: "port must be between 1 and 65535",
		},
		{
			name: "TLS enabled without domains",
			modifyConfig: func(cfg *Config) {
				cfg.Twitch.ClientID = "test_id"
				cfg.Twitch.ClientSecret = "test_secret"
				cfg.Twitch.WebhookSecret = "test_webhook_secret"
				cfg.Server.TLS.Enabled = true
			},
			expectError:   true,
			errorContains: "domains is required when TLS is enabled",
		},
		{
			name: "invalid retry attempts",
			modifyConfig: func(cfg *Config) {
				cfg.Twitch.ClientID = "test_id"
				cfg.Twitch.ClientSecret = "test_secret"
				cfg.Twitch.WebhookSecret = "test_webhook_secret"
				cfg.Retry.MaxAttempts = 0
			},
			expectError:   true,
			errorContains: "max_attempts must be greater than 0",
		},
		{
			name: "invalid backoff factor",
			modifyConfig: func(cfg *Config) {
				cfg.Twitch.ClientID = "test_id"
				cfg.Twitch.ClientSecret = "test_secret"
				cfg.Twitch.WebhookSecret = "test_webhook_secret"
				cfg.Retry.BackoffFactor = 1.0
			},
			expectError:   true,
			errorContains: "backoff_factor must be greater than 1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := DefaultConfig()
			tt.modifyConfig(cfg)

			err := validateConfig(cfg)

			if tt.expectError {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.errorContains)
			} else {
				require.NoError(t, err)
			}
		})
	}
}
