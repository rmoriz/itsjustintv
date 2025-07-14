package twitch

import (
	"time"
)

// EventSubNotification represents a Twitch EventSub notification
type EventSubNotification struct {
	Subscription EventSubSubscription `json:"subscription"`
	Challenge    string               `json:"challenge,omitempty"`
	Event        interface{}          `json:"event,omitempty"`
}

// EventSubSubscription represents the subscription metadata
type EventSubSubscription struct {
	ID        string                 `json:"id"`
	Status    string                 `json:"status"`
	Type      string                 `json:"type"`
	Version   string                 `json:"version"`
	Condition map[string]interface{} `json:"condition"`
	Transport EventSubTransport      `json:"transport"`
	CreatedAt time.Time              `json:"created_at"`
	Cost      int                    `json:"cost"`
}

// EventSubTransport represents the transport configuration
type EventSubTransport struct {
	Method   string `json:"method"`
	Callback string `json:"callback"`
}

// StreamOnlineEvent represents a stream.online event
type StreamOnlineEvent struct {
	ID                   string    `json:"id"`
	BroadcasterUserID    string    `json:"broadcaster_user_id"`
	BroadcasterUserLogin string    `json:"broadcaster_user_login"`
	BroadcasterUserName  string    `json:"broadcaster_user_name"`
	Type                 string    `json:"type"`
	StartedAt            time.Time `json:"started_at"`
}

// EventSubHeaders represents the headers sent with EventSub notifications
type EventSubHeaders struct {
	MessageID           string `json:"message_id"`
	MessageRetry        string `json:"message_retry"`
	MessageType         string `json:"message_type"`
	MessageSignature    string `json:"message_signature"`
	MessageTimestamp    string `json:"message_timestamp"`
	SubscriptionType    string `json:"subscription_type"`
	SubscriptionVersion string `json:"subscription_version"`
}

// MessageType constants
const (
	MessageTypeWebhookCallbackVerification = "webhook_callback_verification"
	MessageTypeNotification                = "notification"
	MessageTypeRevocation                  = "revocation"
)

// Subscription status constants
const (
	SubscriptionStatusEnabled                            = "enabled"
	SubscriptionStatusWebhookCallbackVerificationPending = "webhook_callback_verification_pending"
	SubscriptionStatusWebhookCallbackVerificationFailed  = "webhook_callback_verification_failed"
	SubscriptionStatusNotificationFailuresExceeded       = "notification_failures_exceeded"
	SubscriptionStatusAuthorizationRevoked               = "authorization_revoked"
	SubscriptionStatusUserRemoved                        = "user_removed"
)
