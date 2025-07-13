package twitch

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	"github.com/rmoriz/itsjustintv/internal/config"
)

// Processor handles Twitch EventSub webhook processing
type Processor struct {
	config *config.Config
	logger *slog.Logger
}

// NewProcessor creates a new Twitch webhook processor
func NewProcessor(cfg *config.Config, logger *slog.Logger) *Processor {
	return &Processor{
		config: cfg,
		logger: logger,
	}
}

// ProcessNotification processes a Twitch EventSub notification
func (p *Processor) ProcessNotification(headers EventSubHeaders, payload []byte) (*ProcessedEvent, error) {
	var notification EventSubNotification
	if err := json.Unmarshal(payload, &notification); err != nil {
		return nil, fmt.Errorf("failed to unmarshal notification: %w", err)
	}

	p.logger.Debug("Processing EventSub notification",
		"message_type", headers.MessageType,
		"subscription_type", headers.SubscriptionType,
		"subscription_id", notification.Subscription.ID)

	switch headers.MessageType {
	case MessageTypeWebhookCallbackVerification:
		return p.handleVerification(notification)
	case MessageTypeNotification:
		return p.handleNotification(headers, notification)
	case MessageTypeRevocation:
		return p.handleRevocation(notification)
	default:
		return nil, fmt.Errorf("unknown message type: %s", headers.MessageType)
	}
}

// ProcessedEvent represents a processed Twitch event
type ProcessedEvent struct {
	Type      string      `json:"type"`
	Challenge string      `json:"challenge,omitempty"`
	Event     interface{} `json:"event,omitempty"`
	Action    string      `json:"action"` // "respond", "process", "ignore", "revoke"
}

// handleVerification handles webhook callback verification
func (p *Processor) handleVerification(notification EventSubNotification) (*ProcessedEvent, error) {
	p.logger.Info("Webhook verification challenge received",
		"subscription_id", notification.Subscription.ID,
		"subscription_type", notification.Subscription.Type)

	return &ProcessedEvent{
		Type:      "verification",
		Challenge: notification.Challenge,
		Action:    "respond",
	}, nil
}

// handleNotification handles actual event notifications
func (p *Processor) handleNotification(headers EventSubHeaders, notification EventSubNotification) (*ProcessedEvent, error) {
	switch notification.Subscription.Type {
	case "stream.online":
		return p.handleStreamOnline(notification)
	default:
		p.logger.Warn("Unsupported subscription type", "type", notification.Subscription.Type)
		return &ProcessedEvent{
			Type:   "unsupported",
			Action: "ignore",
		}, nil
	}
}

// handleStreamOnline handles stream.online events
func (p *Processor) handleStreamOnline(notification EventSubNotification) (*ProcessedEvent, error) {
	// Parse the event data
	eventData, err := json.Marshal(notification.Event)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	var streamEvent StreamOnlineEvent
	if err := json.Unmarshal(eventData, &streamEvent); err != nil {
		return nil, fmt.Errorf("failed to unmarshal stream event: %w", err)
	}

	p.logger.Info("Stream online event received",
		"broadcaster_id", streamEvent.BroadcasterUserID,
		"broadcaster_login", streamEvent.BroadcasterUserLogin,
		"broadcaster_name", streamEvent.BroadcasterUserName,
		"started_at", streamEvent.StartedAt)

	// Check if we have this streamer configured
	streamerConfig := p.findStreamerConfig(streamEvent.BroadcasterUserID, streamEvent.BroadcasterUserLogin)
	if streamerConfig == nil {
		p.logger.Info("Stream event for unconfigured streamer, responding with 410 Gone",
			"broadcaster_login", streamEvent.BroadcasterUserLogin)
		return &ProcessedEvent{
			Type:   "unconfigured_streamer",
			Action: "revoke",
		}, nil
	}

	p.logger.Info("Processing stream online event for configured streamer",
		"streamer_login", streamEvent.BroadcasterUserLogin,
		"config_key", findStreamerConfigKey(p.config.Streamers, streamEvent.BroadcasterUserID, streamEvent.BroadcasterUserLogin))

	return &ProcessedEvent{
		Type:   "stream.online",
		Event:  streamEvent,
		Action: "process",
	}, nil
}

// handleRevocation handles subscription revocation
func (p *Processor) handleRevocation(notification EventSubNotification) (*ProcessedEvent, error) {
	p.logger.Warn("Subscription revoked",
		"subscription_id", notification.Subscription.ID,
		"subscription_type", notification.Subscription.Type,
		"status", notification.Subscription.Status)

	return &ProcessedEvent{
		Type:   "revocation",
		Action: "ignore",
	}, nil
}

// findStreamerConfig finds a streamer configuration by user ID or login
func (p *Processor) findStreamerConfig(userID, login string) *config.StreamerConfig {
	for _, streamerConfig := range p.config.Streamers {
		if streamerConfig.UserID == userID || 
		   strings.EqualFold(streamerConfig.Login, login) {
			return &streamerConfig
		}
	}
	return nil
}

// findStreamerConfigKey finds the configuration key for a streamer
func findStreamerConfigKey(streamers map[string]config.StreamerConfig, userID, login string) string {
	for key, streamerConfig := range streamers {
		if streamerConfig.UserID == userID || 
		   strings.EqualFold(streamerConfig.Login, login) {
			return key
		}
	}
	return ""
}