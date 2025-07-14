package cli

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/rmoriz/itsjustintv/internal/config"
	"github.com/rmoriz/itsjustintv/internal/server"
	"github.com/spf13/cobra"
)

var (
	// Version information - will be set at build time
	Version   = "0.1.0"
	GitCommit = "unknown"
	BuildDate = "unknown"

	// Global flags
	configFile string
	verbose    bool
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "itsjustintv",
	Short: "A Twitch EventSub webhook bridge service",
	Long: `itsjustintv is a Go-based microservice that acts as a bridge between 
Twitch's EventSub webhook system and downstream notification systems. 

It receives real-time stream events, enriches them with metadata, and dispatches 
notifications through configurable webhooks or file output.`,
	RunE:          runServer,
	SilenceErrors: true,
	SilenceUsage:  true,
}

// Execute adds all child commands to the root command and sets flags appropriately.
func Execute() error {
	return rootCmd.Execute()
}

func init() {
	// Global flags
	rootCmd.PersistentFlags().StringVarP(&configFile, "config", "c", "config.toml", "config file path")
	rootCmd.PersistentFlags().BoolVarP(&verbose, "verbose", "v", false, "verbose output")

	// Add subcommands
	rootCmd.AddCommand(versionCmd)
	rootCmd.AddCommand(configCmd)
}

// determineConfigPath determines the configuration file path using priority order
func determineConfigPath(flagValue string) string {
	// 1. Use value from --config flag (if not default)
	if flagValue != "config.toml" {
		return flagValue
	}

	// 2. Use ITSJUSTINTV_CONFIG environment variable
	if envConfig := os.Getenv("ITSJUSTINTV_CONFIG"); envConfig != "" {
		return envConfig
	}

	// 3. Try to load config.toml from working directory
	return "config.toml"
}

// runServer is the main server command
func runServer(cmd *cobra.Command, args []string) error {
	// Determine configuration file path using priority order
	configPath := determineConfigPath(configFile)

	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return fmt.Errorf("configuration file not found: %s\n\nConfiguration file loading priority:\n1. Use value from --config flag\n2. Use ITSJUSTINTV_CONFIG environment variable\n3. Try to load config.toml from working directory", configPath)
	}

	// Load configuration
	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Always print which config file was loaded
	fmt.Printf("Loaded configuration from: %s\n", configPath)

	if verbose {
		fmt.Printf("Server will listen on: %s:%d\n", cfg.Server.ListenAddr, cfg.Server.Port)
		fmt.Printf("TLS enabled: %t\n", cfg.Server.TLS.Enabled)
	}

	// Setup logger
	logger := setupLogger(verbose)

	// Create and start server
	server := server.New(cfg, logger)

	ctx := cmd.Context()
	if err := server.Start(ctx); err != nil {
		return fmt.Errorf("server error: %w", err)
	}

	return nil
}

// versionCmd represents the version command
var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version information",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Printf("itsjustintv %s\n", Version)
		fmt.Printf("Git commit: %s\n", GitCommit)
		fmt.Printf("Build date: %s\n", BuildDate)
	},
}

// configCmd represents the config command
var configCmd = &cobra.Command{
	Use:   "config",
	Short: "Configuration management commands",
}

func init() {
	configCmd.AddCommand(configValidateCmd)
	configCmd.AddCommand(configExampleCmd)
}

// configValidateCmd validates the configuration file
var configValidateCmd = &cobra.Command{
	Use:           "validate",
	Short:         "Validate configuration file",
	SilenceErrors: true,
	SilenceUsage:  true,
	RunE: func(cmd *cobra.Command, args []string) error {
		configPath := determineConfigPath(configFile)

		// Check if config file exists
		if _, err := os.Stat(configPath); os.IsNotExist(err) {
			return fmt.Errorf("configuration file not found: %s\n\nConfiguration file loading priority:\n1. Use value from --config flag\n2. Use ITSJUSTINTV_CONFIG environment variable\n3. Try to load config.toml from working directory", configPath)
		}

		// Load configuration
		cfg, err := config.LoadConfig(configPath)
		if err != nil {
			return fmt.Errorf("configuration validation failed: %w", err)
		}

		fmt.Printf("Configuration file '%s' is valid\n", configPath)
		fmt.Printf("Found %d configured streamers\n", len(cfg.Streamers))

		return nil
	},
}

// configExampleCmd generates an example configuration file
var configExampleCmd = &cobra.Command{
	Use:   "example",
	Short: "Generate example configuration file",
	RunE: func(cmd *cobra.Command, args []string) error {
		examplePath := "config.example.toml"
		if len(args) > 0 {
			examplePath = args[0]
		}

		if err := generateExampleConfig(examplePath); err != nil {
			return fmt.Errorf("failed to generate example config: %w", err)
		}

		fmt.Printf("Example configuration written to: %s\n", examplePath)
		return nil
	},
}

// generateExampleConfig creates an example configuration file
func generateExampleConfig(path string) error {
	example := `# itsjustintv Configuration File
# This is an example configuration with all available options documented

[server]
# Server listen address and port
listen_addr = "0.0.0.0"
port = 8080

# TLS/HTTPS configuration (optional)
[server.tls]
enabled = false
domains = ["example.com"]  # Required if TLS is enabled
cert_dir = "data/acme_certs"

[twitch]
# Twitch application credentials (required)
client_id = "your_twitch_client_id"
client_secret = "your_twitch_client_secret"
webhook_secret = "your_webhook_secret_for_hmac_validation"
token_file = "data/tokens.json"

# Retry configuration for failed webhook deliveries
[retry]
max_attempts = 3
initial_delay = "1s"
max_delay = "5m"
backoff_factor = 2.0
state_file = "data/retry_state.json"

# File output configuration
[output]
enabled = true
file_path = "data/output.json"
max_lines = 1000

# OpenTelemetry configuration (optional)
[telemetry]
enabled = false
endpoint = "http://localhost:4318"
service_name = "itsjustintv"
service_version = "1.6.0"

# Streamer configurations
# Each streamer can have their own webhook URL and settings
[streamers.example_streamer]
user_id = "123456789"
login = "example_streamer"
webhook_url = "https://your-webhook-endpoint.com/webhook"
tag_filter = ["English", "Gaming"]  # Optional: only notify for streams with these tags
additional_tags = ["custom_tag"]    # Optional: add custom tags to webhook payload
hmac_secret = "optional_hmac_secret_for_this_webhook"

[streamers.another_streamer]
user_id = "987654321"
login = "another_streamer"
webhook_url = "https://another-endpoint.com/webhook"
# No tag_filter means all streams will trigger notifications
additional_tags = ["vip_streamer"]
`

	return os.WriteFile(path, []byte(example), 0644)
}

// setupLogger creates a structured logger
func setupLogger(verbose bool) *slog.Logger {
	level := slog.LevelInfo
	if verbose {
		level = slog.LevelDebug
	}

	opts := &slog.HandlerOptions{
		Level: level,
	}

	handler := slog.NewTextHandler(os.Stdout, opts)
	return slog.New(handler)
}
