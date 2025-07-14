package retry

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"math"
	"os"
	"sync"
	"time"

	"github.com/rmoriz/itsjustintv/internal/config"
	"github.com/rmoriz/itsjustintv/internal/webhook"
)

// Manager handles retry logic for failed webhook dispatches
type Manager struct {
	config     *config.Config
	logger     *slog.Logger
	dispatcher *webhook.Dispatcher
	queue      []*webhook.DispatchRequest
	mutex      sync.RWMutex
	stopCh     chan struct{}
	wg         sync.WaitGroup
}

// NewManager creates a new retry manager
func NewManager(cfg *config.Config, logger *slog.Logger, dispatcher *webhook.Dispatcher) *Manager {
	return &Manager{
		config:     cfg,
		logger:     logger,
		dispatcher: dispatcher,
		queue:      make([]*webhook.DispatchRequest, 0),
		stopCh:     make(chan struct{}),
	}
}

// Start starts the retry manager background processing
func (m *Manager) Start(ctx context.Context) error {
	// Load existing retry state
	if err := m.loadState(); err != nil {
		m.logger.Warn("Failed to load retry state", "error", err)
	}

	// Start background processor
	m.wg.Add(1)
	go m.processRetries(ctx)

	m.logger.Info("Retry manager started")
	return nil
}

// Stop stops the retry manager
func (m *Manager) Stop() error {
	close(m.stopCh)
	m.wg.Wait()

	// Save current state
	if err := m.saveState(); err != nil {
		m.logger.Error("Failed to save retry state", "error", err)
		return err
	}

	m.logger.Info("Retry manager stopped")
	return nil
}

// AddRequest adds a failed request to the retry queue
func (m *Manager) AddRequest(req *webhook.DispatchRequest) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Calculate next retry time
	req.Attempt++
	req.NextRetry = m.calculateNextRetry(req.Attempt)

	m.queue = append(m.queue, req)

	m.logger.Info("Added request to retry queue",
		"webhook_url", req.WebhookURL,
		"streamer_key", req.StreamerKey,
		"attempt", req.Attempt,
		"next_retry", req.NextRetry)
}

// GetQueueSize returns the current size of the retry queue
func (m *Manager) GetQueueSize() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.queue)
}

// processRetries runs the background retry processing loop
func (m *Manager) processRetries(ctx context.Context) {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second) // Check every 30 seconds
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.processReadyRetries(ctx)
		}
	}
}

// processReadyRetries processes requests that are ready for retry
func (m *Manager) processReadyRetries(ctx context.Context) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	readyRequests := make([]*webhook.DispatchRequest, 0)
	remainingRequests := make([]*webhook.DispatchRequest, 0)

	// Separate ready requests from remaining ones
	for _, req := range m.queue {
		if now.After(req.NextRetry) && req.Attempt <= m.config.Retry.MaxAttempts {
			readyRequests = append(readyRequests, req)
		} else if req.Attempt <= m.config.Retry.MaxAttempts {
			remainingRequests = append(remainingRequests, req)
		} else {
			// Max attempts reached, drop the request
			m.logger.Warn("Dropping request after max attempts",
				"webhook_url", req.WebhookURL,
				"streamer_key", req.StreamerKey,
				"attempts", req.Attempt)
		}
	}

	m.queue = remainingRequests

	// Process ready requests
	for _, req := range readyRequests {
		go m.retryRequest(ctx, req)
	}

	if len(readyRequests) > 0 {
		m.logger.Info("Processing retry requests",
			"ready_count", len(readyRequests),
			"remaining_count", len(remainingRequests))
	}
}

// retryRequest attempts to retry a single request
func (m *Manager) retryRequest(ctx context.Context, req *webhook.DispatchRequest) {
	result := m.dispatcher.Dispatch(ctx, req)

	if !result.Success {
		// Add back to queue for another retry
		m.AddRequest(req)
	} else {
		m.logger.Info("Retry successful",
			"webhook_url", req.WebhookURL,
			"streamer_key", req.StreamerKey,
			"attempt", req.Attempt)
	}
}

// calculateNextRetry calculates the next retry time using exponential backoff
func (m *Manager) calculateNextRetry(attempt int) time.Time {
	// Start with initial delay
	delay := m.config.Retry.InitialDelay

	// Apply exponential backoff
	backoffMultiplier := math.Pow(m.config.Retry.BackoffFactor, float64(attempt-1))
	delay = time.Duration(float64(delay) * backoffMultiplier)

	// Cap at max delay
	if delay > m.config.Retry.MaxDelay {
		delay = m.config.Retry.MaxDelay
	}

	return time.Now().Add(delay)
}

// loadState loads retry state from disk
func (m *Manager) loadState() error {
	if _, err := os.Stat(m.config.Retry.StateFile); os.IsNotExist(err) {
		return nil // No state file exists yet
	}

	data, err := os.ReadFile(m.config.Retry.StateFile)
	if err != nil {
		return fmt.Errorf("failed to read state file: %w", err)
	}

	var state struct {
		Queue []*webhook.DispatchRequest `json:"queue"`
	}

	if err := json.Unmarshal(data, &state); err != nil {
		return fmt.Errorf("failed to unmarshal state: %w", err)
	}

	m.mutex.Lock()
	m.queue = state.Queue
	m.mutex.Unlock()

	m.logger.Info("Loaded retry state", "queue_size", len(state.Queue))
	return nil
}

// saveState saves retry state to disk
func (m *Manager) saveState() error {
	m.mutex.RLock()
	state := struct {
		Queue []*webhook.DispatchRequest `json:"queue"`
	}{
		Queue: m.queue,
	}
	m.mutex.RUnlock()

	data, err := json.MarshalIndent(state, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal state: %w", err)
	}

	if err := os.WriteFile(m.config.Retry.StateFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write state file: %w", err)
	}

	return nil
}

// UpdateConfig updates the retry manager configuration
func (m *Manager) UpdateConfig(newConfig *config.Config) {
	m.config = newConfig
}
