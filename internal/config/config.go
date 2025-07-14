package config

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/BurntSushi/toml"
)

// Config represents the main configuration structure
type Config struct {
	Server    ServerConfig              `toml:"server"`
	Twitch    TwitchConfig              `toml:"twitch"`
	Streamers map[string]StreamerConfig `toml:"streamers"`
	Retry     RetryConfig               `toml:"retry"`
	Output    OutputConfig              `toml:"output"`
	Telemetry TelemetryConfig           `toml:"telemetry"`
	
	// Internal fields (not loaded from TOML)
	configPath string
}

// ServerConfig holds HTTP server configuration
type ServerConfig struct {
	ListenAddr string `toml:"listen_addr"`
	Port       int    `toml:"port"`
	TLS        struct {
		Enabled bool     `toml:"enabled"`
		Domains []string `toml:"domains"`
		CertDir string   `toml:"cert_dir"`
	} `toml:"tls"`
}

// TwitchConfig holds Twitch API configuration
type TwitchConfig struct {
	ClientID      string `toml:"client_id"`
	ClientSecret  string `toml:"client_secret"`
	WebhookSecret string `toml:"webhook_secret"`
	TokenFile     string `toml:"token_file"`
}

// StreamerConfig holds individual streamer configuration
type StreamerConfig struct {
	UserID         string   `toml:"user_id"`
	Login          string   `toml:"login"`
	WebhookURL     string   `toml:"webhook_url"`
	TagFilter      []string `toml:"tag_filter"`
	AdditionalTags []string `toml:"additional_tags"`
	HMACSecret     string   `toml:"hmac_secret"`
}

// RetryConfig holds retry mechanism configuration
type RetryConfig struct {
	MaxAttempts   int           `toml:"max_attempts"`
	InitialDelay  time.Duration `toml:"initial_delay"`
	MaxDelay      time.Duration `toml:"max_delay"`
	BackoffFactor float64       `toml:"backoff_factor"`
	StateFile     string        `toml:"state_file"`
}

// OutputConfig holds file output configuration
type OutputConfig struct {
	Enabled  bool   `toml:"enabled"`
	FilePath string `toml:"file_path"`
	MaxLines int    `toml:"max_lines"`
}

// TelemetryConfig holds OpenTelemetry configuration
type TelemetryConfig struct {
	Enabled        bool   `toml:"enabled"`
	Endpoint       string `toml:"endpoint"`
	ServiceName    string `toml:"service_name"`
	ServiceVersion string `toml:"service_version"`
}

// DefaultConfig returns a configuration with sensible defaults
func DefaultConfig() *Config {
	return &Config{
		Server: ServerConfig{
			ListenAddr: "0.0.0.0",
			Port:       8080,
			TLS: struct {
				Enabled bool     `toml:"enabled"`
				Domains []string `toml:"domains"`
				CertDir string   `toml:"cert_dir"`
			}{
				Enabled: false,
				Domains: []string{},
				CertDir: "data/acme_certs",
			},
		},
		Twitch: TwitchConfig{
			TokenFile: "data/tokens.json",
		},
		Retry: RetryConfig{
			MaxAttempts:   3,
			InitialDelay:  time.Second,
			MaxDelay:      time.Minute * 5,
			BackoffFactor: 2.0,
			StateFile:     "data/retry_state.json",
		},
		Output: OutputConfig{
			Enabled:  true,
			FilePath: "data/output.json",
			MaxLines: 1000,
		},
		Telemetry: TelemetryConfig{
			Enabled:        false,
			ServiceName:    "itsjustintv",
			ServiceVersion: "0.1.0",
		},
		Streamers: make(map[string]StreamerConfig),
	}
}

// LoadConfig loads configuration from a TOML file with environment variable overrides
func LoadConfig(configPath string) (*Config, error) {
	config := DefaultConfig()

	// Load from file if it exists
	if configPath != "" {
		if _, err := os.Stat(configPath); err == nil {
			if _, err := toml.DecodeFile(configPath, config); err != nil {
				return nil, fmt.Errorf("failed to decode config file %s: %w", configPath, err)
			}
		} else if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to check config file %s: %w", configPath, err)
		}
	}

	// Apply environment variable overrides
	if err := applyEnvOverrides(config); err != nil {
		return nil, fmt.Errorf("failed to apply environment overrides: %w", err)
	}

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("configuration validation failed: %w", err)
	}

	// Set config path for later reference
	config.configPath = configPath
	
	return config, nil
}

// GetConfigPath returns the path to the configuration file
func (config *Config) GetConfigPath() string {
	return config.configPath
}

// ResolveStreamerUserIDs resolves missing user IDs for streamers using Twitch API
func ResolveStreamerUserIDs(ctx context.Context, config *Config, twitchClient TwitchUserResolver) error {
	for key, streamer := range config.Streamers {
		// Skip if user_id is already set
		if streamer.UserID != "" {
			continue
		}

		// Skip if login is not set
		if streamer.Login == "" {
			continue
		}

		// Resolve user ID using login
		userInfo, err := twitchClient.GetUserInfoByLoginForConfig(ctx, streamer.Login)
		if err != nil {
			return fmt.Errorf("failed to resolve user ID for streamer '%s' with login '%s': %w", key, streamer.Login, err)
		}

		// Update the streamer config with resolved user ID
		streamer.UserID = userInfo.GetID()
		config.Streamers[key] = streamer

		fmt.Printf("Resolved user ID for streamer '%s': login='%s' -> user_id='%s'\n", key, streamer.Login, userInfo.GetID())
	}

	return nil
}

// TwitchUserResolver interface for resolving user information
type TwitchUserResolver interface {
	GetUserInfoByLoginForConfig(ctx context.Context, login string) (TwitchUserInfo, error)
}

// TwitchUserInfo represents basic user information needed for resolution
type TwitchUserInfo interface {
	GetID() string
	GetLogin() string
}

// applyEnvOverrides applies environment variable overrides to the configuration
func applyEnvOverrides(config *Config) error {
	// Server configuration
	if val := os.Getenv("ITSJUSTINTV_SERVER_LISTEN_ADDR"); val != "" {
		config.Server.ListenAddr = val
	}
	if val := os.Getenv("ITSJUSTINTV_SERVER_PORT"); val != "" {
		var port int
		if _, err := fmt.Sscanf(val, "%d", &port); err == nil {
			config.Server.Port = port
		}
	}

	// Twitch configuration
	if val := os.Getenv("ITSJUSTINTV_TWITCH_CLIENT_ID"); val != "" {
		config.Twitch.ClientID = val
	}
	if val := os.Getenv("ITSJUSTINTV_TWITCH_CLIENT_SECRET"); val != "" {
		config.Twitch.ClientSecret = val
	}
	if val := os.Getenv("ITSJUSTINTV_TWITCH_WEBHOOK_SECRET"); val != "" {
		config.Twitch.WebhookSecret = val
	}

	// TLS configuration
	if val := os.Getenv("ITSJUSTINTV_TLS_ENABLED"); val == "true" {
		config.Server.TLS.Enabled = true
	}

	return nil
}

// validateConfig validates the configuration for required fields and logical consistency
func validateConfig(config *Config) error {
	// Validate Twitch configuration
	if config.Twitch.ClientID == "" {
		return fmt.Errorf("twitch.client_id is required")
	}
	if config.Twitch.ClientSecret == "" {
		return fmt.Errorf("twitch.client_secret is required")
	}
	if config.Twitch.WebhookSecret == "" {
		return fmt.Errorf("twitch.webhook_secret is required")
	}

	// Validate server configuration
	if config.Server.Port <= 0 || config.Server.Port > 65535 {
		return fmt.Errorf("server.port must be between 1 and 65535")
	}

	// Validate TLS configuration
	if config.Server.TLS.Enabled && len(config.Server.TLS.Domains) == 0 {
		return fmt.Errorf("server.tls.domains is required when TLS is enabled")
	}

	// Validate retry configuration
	if config.Retry.MaxAttempts <= 0 {
		return fmt.Errorf("retry.max_attempts must be greater than 0")
	}
	if config.Retry.BackoffFactor <= 1.0 {
		return fmt.Errorf("retry.backoff_factor must be greater than 1.0")
	}

	// Ensure data directories exist
	dataDirs := []string{
		filepath.Dir(config.Twitch.TokenFile),
		filepath.Dir(config.Retry.StateFile),
		filepath.Dir(config.Output.FilePath),
		config.Server.TLS.CertDir,
		"data/image_cache",
	}

	for _, dir := range dataDirs {
		if err := os.MkdirAll(dir, 0755); err != nil {
			return fmt.Errorf("failed to create data directory %s: %w", dir, err)
		}
	}

	return nil
}

// Validate validates the configuration
func (config *Config) Validate() error {
	return validateConfig(config)
}
