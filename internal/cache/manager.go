package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"sync"
	"time"
)

// Manager handles deduplication caching
type Manager struct {
	logger    *slog.Logger
	cache     map[string]*Entry
	mutex     sync.RWMutex
	cacheFile string
	ttl       time.Duration
}

// Entry represents a cache entry
type Entry struct {
	Key       string    `json:"key"`
	Data      []byte    `json:"data"`
	ExpiresAt time.Time `json:"expires_at"`
	CreatedAt time.Time `json:"created_at"`
}

// NewManager creates a new cache manager
func NewManager(logger *slog.Logger, cacheFile string, ttl time.Duration) *Manager {
	return &Manager{
		logger:    logger,
		cache:     make(map[string]*Entry),
		cacheFile: cacheFile,
		ttl:       ttl,
	}
}

// Start starts the cache manager and loads existing cache
func (m *Manager) Start() error {
	if err := m.loadCache(); err != nil {
		m.logger.Warn("Failed to load cache", "error", err)
	}

	// Start cleanup routine
	go m.cleanupRoutine()

	m.logger.Info("Cache manager started", "ttl", m.ttl)
	return nil
}

// Stop stops the cache manager and saves cache to disk
func (m *Manager) Stop() error {
	if err := m.saveCache(); err != nil {
		m.logger.Error("Failed to save cache", "error", err)
		return err
	}

	m.logger.Info("Cache manager stopped")
	return nil
}

// IsDuplicate checks if an event is a duplicate based on its key
func (m *Manager) IsDuplicate(eventKey string) bool {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	entry, exists := m.cache[eventKey]
	if !exists {
		return false
	}

	// Check if entry has expired
	if time.Now().After(entry.ExpiresAt) {
		// Entry expired, remove it
		delete(m.cache, eventKey)
		return false
	}

	return true
}

// AddEvent adds an event to the cache to prevent duplicates
func (m *Manager) AddEvent(eventKey string, eventData []byte) {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	entry := &Entry{
		Key:       eventKey,
		Data:      eventData,
		ExpiresAt: time.Now().Add(m.ttl),
		CreatedAt: time.Now(),
	}

	m.cache[eventKey] = entry

	m.logger.Debug("Added event to cache",
		"key", eventKey,
		"expires_at", entry.ExpiresAt)
}

// GenerateEventKey generates a unique key for an event
func (m *Manager) GenerateEventKey(streamerID, eventID string, timestamp time.Time) string {
	// Create a unique key based on streamer ID, event ID, and timestamp
	data := fmt.Sprintf("%s:%s:%d", streamerID, eventID, timestamp.Unix())
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

// GetCacheSize returns the current number of entries in the cache
func (m *Manager) GetCacheSize() int {
	m.mutex.RLock()
	defer m.mutex.RUnlock()
	return len(m.cache)
}

// GetCacheStats returns cache statistics
func (m *Manager) GetCacheStats() map[string]interface{} {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	now := time.Now()
	expired := 0
	active := 0

	for _, entry := range m.cache {
		if now.After(entry.ExpiresAt) {
			expired++
		} else {
			active++
		}
	}

	return map[string]interface{}{
		"total_entries":   len(m.cache),
		"active_entries":  active,
		"expired_entries": expired,
		"ttl_seconds":     int(m.ttl.Seconds()),
	}
}

// cleanupRoutine runs periodic cleanup of expired entries
func (m *Manager) cleanupRoutine() {
	ticker := time.NewTicker(10 * time.Minute) // Cleanup every 10 minutes
	defer ticker.Stop()

	for range ticker.C {
		m.cleanup()
	}
}

// cleanup removes expired entries from the cache
func (m *Manager) cleanup() {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	now := time.Now()
	removed := 0

	for key, entry := range m.cache {
		if now.After(entry.ExpiresAt) {
			delete(m.cache, key)
			removed++
		}
	}

	if removed > 0 {
		m.logger.Debug("Cache cleanup completed",
			"removed_entries", removed,
			"remaining_entries", len(m.cache))
	}
}

// loadCache loads cache from disk
func (m *Manager) loadCache() error {
	if _, err := os.Stat(m.cacheFile); os.IsNotExist(err) {
		return nil // No cache file exists yet
	}

	data, err := os.ReadFile(m.cacheFile)
	if err != nil {
		return fmt.Errorf("failed to read cache file: %w", err)
	}

	var entries []*Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return fmt.Errorf("failed to unmarshal cache: %w", err)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()

	// Load entries, filtering out expired ones
	now := time.Now()
	loaded := 0

	for _, entry := range entries {
		if now.Before(entry.ExpiresAt) {
			m.cache[entry.Key] = entry
			loaded++
		}
	}

	m.logger.Info("Loaded cache from disk",
		"total_entries", len(entries),
		"loaded_entries", loaded)

	return nil
}

// saveCache saves cache to disk
func (m *Manager) saveCache() error {
	m.mutex.RLock()
	defer m.mutex.RUnlock()

	// Convert cache map to slice
	entries := make([]*Entry, 0, len(m.cache))
	for _, entry := range m.cache {
		entries = append(entries, entry)
	}

	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal cache: %w", err)
	}

	if err := os.WriteFile(m.cacheFile, data, 0644); err != nil {
		return fmt.Errorf("failed to write cache file: %w", err)
	}

	m.logger.Debug("Saved cache to disk", "entries", len(entries))
	return nil
}
