package config

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

// Watcher watches configuration files for changes
type Watcher struct {
	logger       *slog.Logger
	configPath   string
	watcher      *fsnotify.Watcher
	config       *Config
	mu           sync.RWMutex
	reloadFunc   func(*Config) error
	debounceTime time.Duration
	done         chan struct{}
}

// NewWatcher creates a new configuration file watcher
func NewWatcher(configPath string, logger *slog.Logger, reloadFunc func(*Config) error) (*Watcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("failed to create file watcher: %w", err)
	}

	return &Watcher{
		logger:       logger,
		configPath:   configPath,
		watcher:      watcher,
		reloadFunc:   reloadFunc,
		debounceTime: 500 * time.Millisecond,
		done:         make(chan struct{}),
	}, nil
}

// Start begins watching the configuration file
func (w *Watcher) Start(ctx context.Context) error {
	if w.configPath == "" {
		w.logger.Debug("No config path provided, skipping file watcher")
		return nil
	}

	// Add the config file to watch list
	if err := w.watcher.Add(w.configPath); err != nil {
		return fmt.Errorf("failed to watch config file: %w", err)
	}

	// Also watch the directory for file creation events
	// This handles cases where the file is moved/renamed and recreated
	if err := w.watcher.Add(w.getConfigDir()); err != nil {
		w.logger.Warn("Failed to watch config directory", "error", err)
	}

	w.logger.Info("Configuration file watcher started", "path", w.configPath)

	go w.watchLoop(ctx)
	return nil
}

// watchLoop handles file system events
func (w *Watcher) watchLoop(ctx context.Context) {
	var debounceTimer *time.Timer
	var debounceChan <-chan time.Time

	for {
		select {
		case <-ctx.Done():
			return

		case <-w.done:
			return

		case event, ok := <-w.watcher.Events:
			if !ok {
				return
			}

			// Check if this is a relevant event for our config file
			if event.Name == w.configPath || event.Name == w.getConfigDir() {
				// Debounce rapid file changes
				if debounceTimer != nil {
					debounceTimer.Stop()
				}

				debounceTimer = time.NewTimer(w.debounceTime)
				debounceChan = debounceTimer.C
			}

		case <-debounceChan:
			// Reload configuration after debounce
			w.reloadConfig()
			debounceTimer = nil
			debounceChan = nil

		case err, ok := <-w.watcher.Errors:
			if !ok {
				return
			}
			w.logger.Error("File watcher error", "error", err)
		}
	}
}

// reloadConfig loads and validates the new configuration
func (w *Watcher) reloadConfig() {
	w.logger.Info("Configuration file changed, reloading...")

	newConfig, err := LoadConfig(w.configPath)
	if err != nil {
		w.logger.Error("Failed to load new configuration", "error", err)
		return
	}

	if err := newConfig.Validate(); err != nil {
		w.logger.Error("New configuration validation failed", "error", err)
		return
	}

	// Execute reload function
	if err := w.reloadFunc(newConfig); err != nil {
		w.logger.Error("Failed to apply new configuration", "error", err)
		return
	}

	w.mu.Lock()
	w.config = newConfig
	w.mu.Unlock()

	w.logger.Info("Configuration reloaded successfully")
}

// Stop stops watching configuration files
func (w *Watcher) Stop() error {
	close(w.done)
	if w.watcher != nil {
		return w.watcher.Close()
	}
	return nil
}

// getConfigDir returns the directory containing the config file
func (w *Watcher) getConfigDir() string {
	// Extract directory from config path
	// This is a simplified version - in production you might want to use filepath.Dir
	for i := len(w.configPath) - 1; i >= 0; i-- {
		if w.configPath[i] == '/' || w.configPath[i] == '\\' {
			return w.configPath[:i]
		}
	}
	return "."
}

// GetConfig returns the current configuration
func (w *Watcher) GetConfig() *Config {
	w.mu.RLock()
	defer w.mu.RUnlock()
	return w.config
}
