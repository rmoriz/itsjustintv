package twitch

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"github.com/rmoriz/itsjustintv/internal/config"
)

// SubscriptionManager handles Twitch EventSub subscription lifecycle
type SubscriptionManager struct {
	config      *config.Config
	logger      *slog.Logger
	client      *Client
	httpClient  *http.Client
	callbackURL string
}

// SubscriptionRequest represents a request to create an EventSub subscription
type SubscriptionRequest struct {
	Type      string                 `json:"type"`
	Version   string                 `json:"version"`
	Condition map[string]interface{} `json:"condition"`
	Transport SubscriptionTransport  `json:"transport"`
}

// SubscriptionTransport represents the transport configuration for subscriptions
type SubscriptionTransport struct {
	Method   string `json:"method"`
	Callback string `json:"callback"`
	Secret   string `json:"secret"`
}

// SubscriptionResponse represents the response from creating a subscription
type SubscriptionResponse struct {
	Data         []EventSubSubscription `json:"data"`
	Total        int                    `json:"total"`
	TotalCost    int                    `json:"total_cost"`
	MaxTotalCost int                    `json:"max_total_cost"`
}

// NewSubscriptionManager creates a new subscription manager
func NewSubscriptionManager(cfg *config.Config, logger *slog.Logger, client *Client) *SubscriptionManager {
	// Use incoming_webhook_url if specified, otherwise build from server config
	callbackURL := cfg.Twitch.IncomingWebhookURL
	if callbackURL == "" {
		callbackURL = buildCallbackURL(cfg)
	}

	return &SubscriptionManager{
		config:      cfg,
		logger:      logger,
		client:      client,
		httpClient:  &http.Client{Timeout: 30 * time.Second},
		callbackURL: callbackURL,
	}
}

// Start initializes subscription management
func (sm *SubscriptionManager) Start(ctx context.Context) error {
	sm.logger.Info("Starting EventSub subscription manager", "callback_url", sm.callbackURL)

	// Initial subscription sync
	if err := sm.syncSubscriptions(ctx); err != nil {
		return fmt.Errorf("failed to sync subscriptions: %w", err)
	}

	// Start background sync task
	go sm.backgroundSync(ctx)

	return nil
}

// SyncSubscriptions fetches current subscriptions and creates missing ones
func (sm *SubscriptionManager) SyncSubscriptions(ctx context.Context) error {
	return sm.syncSubscriptions(ctx)
}

// syncSubscriptions fetches current subscriptions and creates missing ones
func (sm *SubscriptionManager) syncSubscriptions(ctx context.Context) error {
	sm.logger.Info("Syncing EventSub subscriptions")

	// Get current subscriptions
	currentSubs, err := sm.getSubscriptions(ctx)
	if err != nil {
		return fmt.Errorf("failed to get current subscriptions: %w", err)
	}

	sm.logger.Info("Current EventSub subscriptions",
		"count", len(currentSubs.Data),
		"total_cost", currentSubs.TotalCost,
		"max_total_cost", currentSubs.MaxTotalCost)

	// Build map of existing subscriptions by broadcaster_user_id
	existingSubs := make(map[string]*EventSubSubscription)
	for i := range currentSubs.Data {
		sub := &currentSubs.Data[i]
		if sub.Type == "stream.online" && sub.Status == SubscriptionStatusEnabled {
			if broadcasterID, ok := sub.Condition["broadcaster_user_id"].(string); ok {
				existingSubs[broadcasterID] = sub
			}
		}
	}

	// Check each configured streamer
	var created, existing int
	for streamerKey, streamerConfig := range sm.config.Streamers {
		if streamerConfig.UserID == "" {
			sm.logger.Warn("Skipping streamer with missing user_id", "streamer_key", streamerKey)
			continue
		}

		if _, exists := existingSubs[streamerConfig.UserID]; exists {
			existing++
			sm.logger.Debug("Subscription already exists",
				"streamer_key", streamerKey,
				"user_id", streamerConfig.UserID)
			continue
		}

		// Create subscription
		if err := sm.createSubscription(ctx, streamerConfig.UserID); err != nil {
			sm.logger.Error("Failed to create subscription",
				"error", err,
				"streamer_key", streamerKey,
				"user_id", streamerConfig.UserID)
			continue
		}

		created++
		sm.logger.Info("Created EventSub subscription",
			"streamer_key", streamerKey,
			"user_id", streamerConfig.UserID)
	}

	sm.logger.Info("Subscription sync complete",
		"existing", existing,
		"created", created)

	return nil
}

// createSubscription creates a new EventSub subscription for a broadcaster
func (sm *SubscriptionManager) createSubscription(ctx context.Context, broadcasterUserID string) error {
	if err := sm.client.EnsureValidToken(ctx); err != nil {
		return fmt.Errorf("failed to ensure valid token: %w", err)
	}

	request := SubscriptionRequest{
		Type:    "stream.online",
		Version: "1",
		Condition: map[string]interface{}{
			"broadcaster_user_id": broadcasterUserID,
		},
		Transport: SubscriptionTransport{
			Method:   "webhook",
			Callback: sm.callbackURL,
			Secret:   sm.config.Twitch.WebhookSecret,
		},
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", "https://api.twitch.tv/helix/eventsub/subscriptions", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	sm.client.setAuthHeaders(req)
	req.Header.Set("Content-Type", "application/json")

	resp, err := sm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if resp.StatusCode != http.StatusAccepted {
		return fmt.Errorf("subscription creation failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response SubscriptionResponse
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(response.Data) == 0 {
		return fmt.Errorf("no subscription data in response")
	}

	sub := response.Data[0]
	sm.logger.Debug("Subscription created successfully",
		"subscription_id", sub.ID,
		"status", sub.Status,
		"broadcaster_user_id", broadcasterUserID)

	return nil
}

// GetSubscriptions retrieves current EventSub subscriptions
func (sm *SubscriptionManager) GetSubscriptions(ctx context.Context) (*SubscriptionResponse, error) {
	return sm.getSubscriptions(ctx)
}

// getSubscriptions retrieves current EventSub subscriptions
func (sm *SubscriptionManager) getSubscriptions(ctx context.Context) (*SubscriptionResponse, error) {
	if err := sm.client.EnsureValidToken(ctx); err != nil {
		return nil, fmt.Errorf("failed to ensure valid token: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "GET", "https://api.twitch.tv/helix/eventsub/subscriptions", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	sm.client.setAuthHeaders(req)

	resp, err := sm.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get subscriptions failed with status %d: %s", resp.StatusCode, string(body))
	}

	var response SubscriptionResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &response, nil
}


// backgroundSync runs periodic subscription validation and cleanup
func (sm *SubscriptionManager) backgroundSync(ctx context.Context) {
	// Initial delay with splay (0-15 minutes)
	initialDelay := time.Duration(15) * time.Minute
	timer := time.NewTimer(initialDelay)
	defer timer.Stop()

	for {
		select {
		case <-ctx.Done():
			sm.logger.Info("Background subscription sync stopped")
			return
		case <-timer.C:
			sm.logger.Debug("Running background subscription sync")

			if err := sm.syncSubscriptions(ctx); err != nil {
				sm.logger.Error("Background subscription sync failed", "error", err)
			}

			// Schedule next sync (1 hour + 0-15 minute splay)
			nextSync := time.Hour + time.Duration(15)*time.Minute
			timer.Reset(nextSync)
		}
	}
}

// UpdateConfig updates the subscription manager with new configuration
func (sm *SubscriptionManager) UpdateConfig(newConfig *config.Config) error {
	sm.config = newConfig
	sm.callbackURL = buildCallbackURL(newConfig)
	sm.logger.Info("Updated subscription manager configuration")
	return nil
}

// RefreshSubscriptions refreshes subscriptions based on the new configuration
func (sm *SubscriptionManager) RefreshSubscriptions(ctx context.Context) error {
	sm.logger.Info("Refreshing EventSub subscriptions due to configuration change")
	return sm.syncSubscriptions(ctx)
}

// buildCallbackURL constructs the callback URL for EventSub subscriptions
func buildCallbackURL(cfg *config.Config) string {
	// Use external_domain if specified (for reverse proxy scenarios)
	if cfg.Server.ExternalDomain != "" {
		// Use HTTPS by default for external domains, as they typically use reverse proxies with TLS
		return fmt.Sprintf("https://%s/twitch", cfg.Server.ExternalDomain)
	}

	scheme := "http"
	if cfg.Server.TLS.Enabled {
		scheme = "https"
	}

	// If TLS is enabled, use the first domain
	if cfg.Server.TLS.Enabled && len(cfg.Server.TLS.Domains) > 0 {
		return fmt.Sprintf("%s://%s/twitch", scheme, cfg.Server.TLS.Domains[0])
	}

	// Otherwise use listen address and port
	host := cfg.Server.ListenAddr
	if host == "0.0.0.0" || host == "" {
		host = "localhost" // This won't work for production, but helps with local testing
	}

	port := cfg.Server.Port
	if (scheme == "https" && port == 443) || (scheme == "http" && port == 80) {
		return fmt.Sprintf("%s://%s/twitch", scheme, host)
	}

	return fmt.Sprintf("%s://%s:%d/twitch", scheme, host, port)
}
