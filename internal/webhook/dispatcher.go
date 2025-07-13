package webhook

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

	"github.com/rmoriz/itsjustintv/internal/config"
)

// Dispatcher handles webhook dispatching with retry logic
type Dispatcher struct {
	config     *config.Config
	logger     *slog.Logger
	httpClient *http.Client
	validator  *Validator
}

// NewDispatcher creates a new webhook dispatcher
func NewDispatcher(cfg *config.Config, logger *slog.Logger) *Dispatcher {
	return &Dispatcher{
		config: cfg,
		logger: logger,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		validator: NewValidator(""), // Will be set per webhook
	}
}

// WebhookPayload represents the payload sent to webhooks
type WebhookPayload struct {
	StreamerLogin   string            `json:"streamer_login"`
	StreamerName    string            `json:"streamer_name"`
	StreamerID      string            `json:"streamer_id"`
	URL             string            `json:"url"`
	ViewCount       int               `json:"view_count,omitempty"`
	FollowersCount  int               `json:"followers_count,omitempty"`
	Tags            []string          `json:"tags,omitempty"`
	Language        string            `json:"language,omitempty"`
	Description     string            `json:"description,omitempty"`
	Image           *ImageData        `json:"image,omitempty"`
	Timestamp       time.Time         `json:"timestamp"`
	AdditionalTags  []string          `json:"additional_tags,omitempty"`
}

// ImageData represents profile image data
type ImageData struct {
	URL    string `json:"url"`
	Width  int    `json:"width"`
	Height int    `json:"height"`
	Data   string `json:"data,omitempty"` // Base64 encoded image data
}

// DispatchRequest represents a webhook dispatch request
type DispatchRequest struct {
	WebhookURL  string         `json:"webhook_url"`
	Payload     WebhookPayload `json:"payload"`
	HMACSecret  string         `json:"hmac_secret,omitempty"`
	StreamerKey string         `json:"streamer_key"`
	Attempt     int            `json:"attempt"`
	NextRetry   time.Time      `json:"next_retry,omitempty"`
}

// DispatchResult represents the result of a webhook dispatch
type DispatchResult struct {
	Success      bool          `json:"success"`
	StatusCode   int           `json:"status_code,omitempty"`
	Error        string        `json:"error,omitempty"`
	ResponseTime time.Duration `json:"response_time"`
	Attempt      int           `json:"attempt"`
}

// Dispatch sends a webhook with the given payload
func (d *Dispatcher) Dispatch(ctx context.Context, req *DispatchRequest) *DispatchResult {
	start := time.Now()
	
	d.logger.Info("Dispatching webhook",
		"webhook_url", req.WebhookURL,
		"streamer_key", req.StreamerKey,
		"attempt", req.Attempt)

	// Marshal payload to JSON
	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		return &DispatchResult{
			Success:      false,
			Error:        fmt.Sprintf("failed to marshal payload: %v", err),
			ResponseTime: time.Since(start),
			Attempt:      req.Attempt,
		}
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", req.WebhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		return &DispatchResult{
			Success:      false,
			Error:        fmt.Sprintf("failed to create request: %v", err),
			ResponseTime: time.Since(start),
			Attempt:      req.Attempt,
		}
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("User-Agent", "itsjustintv/1.6")

	// Add HMAC signature if secret is provided
	if req.HMACSecret != "" {
		validator := NewValidator(req.HMACSecret)
		signature := validator.GenerateSignature(payloadBytes)
		httpReq.Header.Set("X-Signature-256", signature)
	}

	// Send request
	resp, err := d.httpClient.Do(httpReq)
	if err != nil {
		return &DispatchResult{
			Success:      false,
			Error:        fmt.Sprintf("request failed: %v", err),
			ResponseTime: time.Since(start),
			Attempt:      req.Attempt,
		}
	}
	defer resp.Body.Close()

	responseTime := time.Since(start)
	success := resp.StatusCode >= 200 && resp.StatusCode < 300

	result := &DispatchResult{
		Success:      success,
		StatusCode:   resp.StatusCode,
		ResponseTime: responseTime,
		Attempt:      req.Attempt,
	}

	if !success {
		result.Error = fmt.Sprintf("HTTP %d", resp.StatusCode)
	}

	d.logger.Info("Webhook dispatch completed",
		"webhook_url", req.WebhookURL,
		"streamer_key", req.StreamerKey,
		"attempt", req.Attempt,
		"success", success,
		"status_code", resp.StatusCode,
		"response_time", responseTime)

	return result
}

// CreatePayload creates a webhook payload from stream event data
func (d *Dispatcher) CreatePayload(streamerKey string, streamerConfig config.StreamerConfig, eventData map[string]interface{}) *WebhookPayload {
	payload := &WebhookPayload{
		StreamerLogin:  streamerConfig.Login,
		StreamerName:   getStringFromEvent(eventData, "broadcaster_user_name"),
		StreamerID:     streamerConfig.UserID,
		URL:            fmt.Sprintf("https://twitch.tv/%s", streamerConfig.Login),
		Timestamp:      time.Now().UTC(),
		AdditionalTags: streamerConfig.AdditionalTags,
	}

	// Extract data from event if available
	if payload.StreamerName == "" {
		payload.StreamerName = streamerConfig.Login // Fallback to login
	}
	if payload.StreamerID == "" {
		payload.StreamerID = getStringFromEvent(eventData, "broadcaster_user_id")
	}

	return payload
}

// getStringFromEvent safely extracts a string value from event data
func getStringFromEvent(eventData map[string]interface{}, key string) string {
	if value, ok := eventData[key]; ok {
		if str, ok := value.(string); ok {
			return str
		}
	}
	return ""
}