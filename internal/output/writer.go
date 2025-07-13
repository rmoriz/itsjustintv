package output

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"

	"github.com/rmoriz/itsjustintv/internal/config"
	"github.com/rmoriz/itsjustintv/internal/webhook"
)

// Writer handles writing webhook payloads to JSON files
type Writer struct {
	config   *config.Config
	logger   *slog.Logger
	mutex    sync.Mutex
	payloads []OutputEntry
}

// OutputEntry represents a single output entry
type OutputEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Payload   webhook.WebhookPayload `json:"payload"`
	Success   bool                   `json:"success"`
	Error     string                 `json:"error,omitempty"`
}

// NewWriter creates a new output writer
func NewWriter(cfg *config.Config, logger *slog.Logger) *Writer {
	return &Writer{
		config:   cfg,
		logger:   logger,
		payloads: make([]OutputEntry, 0),
	}
}

// Start initializes the writer and loads existing data
func (w *Writer) Start() error {
	if !w.config.Output.Enabled {
		w.logger.Info("File output disabled")
		return nil
	}

	if err := w.loadExistingData(); err != nil {
		w.logger.Warn("Failed to load existing output data", "error", err)
	}

	w.logger.Info("Output writer started", "file_path", w.config.Output.FilePath)
	return nil
}

// Stop saves current data to disk
func (w *Writer) Stop() error {
	if !w.config.Output.Enabled {
		return nil
	}

	if err := w.saveData(); err != nil {
		w.logger.Error("Failed to save output data", "error", err)
		return err
	}

	w.logger.Info("Output writer stopped")
	return nil
}

// WritePayload writes a webhook payload to the output file
func (w *Writer) WritePayload(payload webhook.WebhookPayload, success bool, errorMsg string) error {
	if !w.config.Output.Enabled {
		return nil
	}

	w.mutex.Lock()
	defer w.mutex.Unlock()

	entry := OutputEntry{
		Timestamp: time.Now().UTC(),
		Payload:   payload,
		Success:   success,
		Error:     errorMsg,
	}

	// Add to in-memory list
	w.payloads = append(w.payloads, entry)

	// Trim to max lines if needed
	if len(w.payloads) > w.config.Output.MaxLines {
		w.payloads = w.payloads[len(w.payloads)-w.config.Output.MaxLines:]
	}

	// Save to disk
	if err := w.saveData(); err != nil {
		return fmt.Errorf("failed to save output data: %w", err)
	}

	w.logger.Debug("Wrote payload to output file",
		"streamer_login", payload.StreamerLogin,
		"success", success,
		"total_entries", len(w.payloads))

	return nil
}

// GetRecentPayloads returns the most recent payloads
func (w *Writer) GetRecentPayloads(limit int) []OutputEntry {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if limit <= 0 || limit > len(w.payloads) {
		limit = len(w.payloads)
	}

	// Return the last 'limit' entries
	start := len(w.payloads) - limit
	if start < 0 {
		start = 0
	}

	result := make([]OutputEntry, limit)
	copy(result, w.payloads[start:])
	return result
}

// GetStats returns statistics about the output writer
func (w *Writer) GetStats() map[string]interface{} {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	successful := 0
	failed := 0

	for _, entry := range w.payloads {
		if entry.Success {
			successful++
		} else {
			failed++
		}
	}

	return map[string]interface{}{
		"enabled":           w.config.Output.Enabled,
		"total_entries":     len(w.payloads),
		"successful_sends":  successful,
		"failed_sends":      failed,
		"max_lines":         w.config.Output.MaxLines,
		"file_path":         w.config.Output.FilePath,
	}
}

// loadExistingData loads existing output data from disk
func (w *Writer) loadExistingData() error {
	if _, err := os.Stat(w.config.Output.FilePath); os.IsNotExist(err) {
		return nil // No file exists yet
	}

	data, err := os.ReadFile(w.config.Output.FilePath)
	if err != nil {
		return fmt.Errorf("failed to read output file: %w", err)
	}

	var entries []OutputEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to unmarshal output data: %w", err)
	}

	w.payloads = entries

	// Trim to max lines if needed
	if len(w.payloads) > w.config.Output.MaxLines {
		w.payloads = w.payloads[len(w.payloads)-w.config.Output.MaxLines:]
	}

	w.logger.Info("Loaded existing output data",
		"entries", len(w.payloads),
		"file_path", w.config.Output.FilePath)

	return nil
}

// saveData saves current data to disk
func (w *Writer) saveData() error {
	data, err := json.MarshalIndent(w.payloads, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal output data: %w", err)
	}

	if err := os.WriteFile(w.config.Output.FilePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write output file: %w", err)
	}

	return nil
}