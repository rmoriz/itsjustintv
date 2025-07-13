package twitch

import (
	"testing"

	"github.com/rmoriz/itsjustintv/internal/config"
)

func TestBuildCallbackURL(t *testing.T) {
	tests := []struct {
		name     string
		config   *config.Config
		expected string
	}{
		{
			name: "HTTPS with domain",
			config: &config.Config{
				Server: config.ServerConfig{
					ListenAddr: "0.0.0.0",
					Port:       443,
					TLS: struct {
						Enabled bool     `toml:"enabled"`
						Domains []string `toml:"domains"`
						CertDir string   `toml:"cert_dir"`
					}{
						Enabled: true,
						Domains: []string{"example.com", "www.example.com"},
					},
				},
			},
			expected: "https://example.com/twitch",
		},
		{
			name: "HTTP with custom port",
			config: &config.Config{
				Server: config.ServerConfig{
					ListenAddr: "0.0.0.0",
					Port:       8080,
					TLS: struct {
						Enabled bool     `toml:"enabled"`
						Domains []string `toml:"domains"`
						CertDir string   `toml:"cert_dir"`
					}{
						Enabled: false,
					},
				},
			},
			expected: "http://localhost:8080/twitch",
		},
		{
			name: "HTTPS with standard port",
			config: &config.Config{
				Server: config.ServerConfig{
					ListenAddr: "0.0.0.0",
					Port:       443,
					TLS: struct {
						Enabled bool     `toml:"enabled"`
						Domains []string `toml:"domains"`
						CertDir string   `toml:"cert_dir"`
					}{
						Enabled: true,
						Domains: []string{"api.example.com"},
					},
				},
			},
			expected: "https://api.example.com/twitch",
		},
		{
			name: "HTTP with standard port",
			config: &config.Config{
				Server: config.ServerConfig{
					ListenAddr: "localhost",
					Port:       80,
					TLS: struct {
						Enabled bool     `toml:"enabled"`
						Domains []string `toml:"domains"`
						CertDir string   `toml:"cert_dir"`
					}{
						Enabled: false,
					},
				},
			},
			expected: "http://localhost/twitch",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := buildCallbackURL(tt.config)
			if result != tt.expected {
				t.Errorf("buildCallbackURL() = %v, want %v", result, tt.expected)
			}
		})
	}
}