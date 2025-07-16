package twitch

import (
	"context"
	"encoding/base64"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/rmoriz/itsjustintv/internal/config"
	"github.com/rmoriz/itsjustintv/internal/webhook"
)

// Enricher handles metadata enrichment for stream events
type Enricher struct {
	config     *config.Config
	logger     *slog.Logger
	client     *Client
	httpClient *http.Client
	cacheDir   string
}

// NewEnricher creates a new metadata enricher
func NewEnricher(cfg *config.Config, logger *slog.Logger, client *Client) *Enricher {
	return &Enricher{
		config: cfg,
		logger: logger,
		client: client,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cacheDir: "data/image_cache",
	}
}

// Start initializes the enricher
func (e *Enricher) Start() error {
	// Ensure cache directory exists
	if err := os.MkdirAll(e.cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create image cache directory: %w", err)
	}

	// Start cleanup routine
	go e.cleanupRoutine()

	e.logger.Info("Metadata enricher started", "cache_dir", e.cacheDir)
	return nil
}

// EnrichPayload enriches a webhook payload with metadata from Twitch API
func (e *Enricher) EnrichPayload(ctx context.Context, payload *webhook.WebhookPayload, streamerConfig config.StreamerConfig) error {
	e.logger.Debug("Enriching payload", "streamer_id", payload.StreamerID)

	// Get user info for view count and profile image
	userInfo, err := e.client.GetUserInfo(ctx, payload.StreamerID)
	if err != nil {
		e.logger.Warn("Failed to get user info", "error", err, "streamer_id", payload.StreamerID)
		// Continue with partial enrichment
	} else {
		payload.ViewCount = userInfo.ViewCount
		payload.Description = userInfo.Description

		// Get profile image
		if userInfo.ProfileImageURL != "" {
			imageData, err := e.getProfileImage(ctx, userInfo.ProfileImageURL, payload.StreamerID)
			if err != nil {
				e.logger.Warn("Failed to get profile image", "error", err, "streamer_id", payload.StreamerID)
			} else {
				payload.Image = imageData
			}
		}
	}

	// Get channel info for tags and language
	channelInfo, err := e.client.GetChannelInfo(ctx, payload.StreamerID)
	if err != nil {
		e.logger.Warn("Failed to get channel info", "error", err, "streamer_id", payload.StreamerID)
		// Continue with basic data, tag filtering will be skipped
	} else {
		// Apply tag filtering according to PRD requirements
		if len(streamerConfig.TagFilter) > 0 {
			if !e.checkTagFilter(channelInfo.Tags, streamerConfig.TagFilter) {
				e.logger.Info("Stream blocked by tag filter",
					"streamer_login", payload.StreamerLogin,
					"twitch_tags", channelInfo.Tags,
					"tag_filter", streamerConfig.TagFilter)
				return fmt.Errorf("stream blocked by tag filter")
			}
		}

		// Merge dynamic tags (Twitch-provided) with static additional tags
		allTags := make([]string, 0, len(channelInfo.Tags)+len(payload.AdditionalTags))
		allTags = append(allTags, channelInfo.Tags...)       // Twitch-provided tags
		allTags = append(allTags, payload.AdditionalTags...) // Static additional tags
		payload.Tags = allTags

		// Set language from channel info
		payload.Language = e.detectLanguage(channelInfo.Tags, channelInfo.BroadcasterLanguage)
	}

	// Get followers count
	followersCount, err := e.client.GetFollowersCount(ctx, payload.StreamerID)
	if err != nil {
		e.logger.Warn("Failed to get followers count", "error", err, "streamer_id", payload.StreamerID)
	} else {
		payload.FollowersCount = followersCount
	}

	e.logger.Debug("Payload enrichment completed",
		"streamer_id", payload.StreamerID,
		"view_count", payload.ViewCount,
		"followers_count", payload.FollowersCount,
		"tags_count", len(payload.Tags),
		"has_image", payload.Image != nil)

	return nil
}

// checkTagFilter checks if any Twitch-provided tag matches the filter
func (e *Enricher) checkTagFilter(twitchTags []string, tagFilter []string) bool {
	if len(tagFilter) == 0 {
		return true // No filter, allow all
	}

	// Check each Twitch-provided tag against the filter (case-insensitive exact match)
	for _, twitchTag := range twitchTags {
		for _, filterTag := range tagFilter {
			if strings.EqualFold(twitchTag, filterTag) {
				return true // Found a match
			}
		}
	}

	return false // No matching tags found
}

// getProfileImage fetches and caches a profile image
func (e *Enricher) getProfileImage(ctx context.Context, imageURL, streamerID string) (*webhook.ImageData, error) {
	// Check cache first
	cacheFile := filepath.Join(e.cacheDir, streamerID+".jpg")
	if imageData := e.loadCachedImage(cacheFile); imageData != nil {
		return imageData, nil
	}

	// Fetch image from URL
	req, err := http.NewRequestWithContext(ctx, "GET", imageURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := e.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch image: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("image request failed with status %d", resp.StatusCode)
	}

	// Read image data
	imageBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read image data: %w", err)
	}

	// Cache the image
	if err := os.WriteFile(cacheFile, imageBytes, 0644); err != nil {
		e.logger.Warn("Failed to cache image", "error", err, "streamer_id", streamerID)
	}

	// Create image data
	imageData := &webhook.ImageData{
		URL:    imageURL,
		Width:  300, // Twitch profile images are typically 300x300
		Height: 300,
		Data:   base64.StdEncoding.EncodeToString(imageBytes),
	}

	return imageData, nil
}

// loadCachedImage loads an image from cache if it exists and is not expired
func (e *Enricher) loadCachedImage(cacheFile string) *webhook.ImageData {
	// Check if file exists and is not too old (7 days)
	info, err := os.Stat(cacheFile)
	if err != nil {
		return nil // File doesn't exist
	}

	if time.Since(info.ModTime()) > 7*24*time.Hour {
		// Cache expired, remove file
		os.Remove(cacheFile)
		return nil
	}

	// Load cached image
	imageBytes, err := os.ReadFile(cacheFile)
	if err != nil {
		return nil
	}

	return &webhook.ImageData{
		URL:    "", // We don't store the original URL in cache
		Width:  300,
		Height: 300,
		Data:   base64.StdEncoding.EncodeToString(imageBytes),
	}
}

// detectLanguage detects the language from tags and broadcaster language
func (e *Enricher) detectLanguage(tags []string, broadcasterLanguage string) string {
	// Check tags for language indicators
	for _, tag := range tags {
		tagLower := strings.ToLower(tag)
		switch tagLower {
		case "english":
			return "en"
		case "german", "deutsch":
			return "de"
		case "spanish", "español":
			return "es"
		case "french", "français":
			return "fr"
		case "italian", "italiano":
			return "it"
		case "portuguese", "português":
			return "pt"
		case "russian", "русский":
			return "ru"
		case "japanese", "日本語":
			return "ja"
		case "korean", "한국어":
			return "ko"
		case "chinese", "中文":
			return "zh"
		}
	}

	// Fall back to broadcaster language
	if broadcasterLanguage != "" {
		return broadcasterLanguage
	}

	// Default to English
	return "en"
}

// cleanupRoutine runs periodic cleanup of expired cached images
func (e *Enricher) cleanupRoutine() {
	ticker := time.NewTicker(24 * time.Hour) // Cleanup daily
	defer ticker.Stop()

	for range ticker.C {
		e.cleanupCache()
	}
}

// cleanupCache removes expired cached images
func (e *Enricher) cleanupCache() {
	entries, err := os.ReadDir(e.cacheDir)
	if err != nil {
		e.logger.Warn("Failed to read cache directory", "error", err)
		return
	}

	removed := 0
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(e.cacheDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			continue
		}

		// Remove files older than 7 days
		if time.Since(info.ModTime()) > 7*24*time.Hour {
			if err := os.Remove(filePath); err == nil {
				removed++
			}
		}
	}

	if removed > 0 {
		e.logger.Debug("Image cache cleanup completed", "removed_files", removed)
	}
}

// UpdateConfig updates the enricher configuration
func (e *Enricher) UpdateConfig(newConfig *config.Config) {
	e.config = newConfig
}
