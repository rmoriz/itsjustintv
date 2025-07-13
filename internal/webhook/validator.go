package webhook

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

// Validator handles HMAC signature validation for webhooks
type Validator struct {
	secret string
}

// NewValidator creates a new webhook validator
func NewValidator(secret string) *Validator {
	return &Validator{
		secret: secret,
	}
}

// ValidateSignature validates the HMAC-SHA256 signature of a webhook payload
func (v *Validator) ValidateSignature(payload []byte, signature string) error {
	if v.secret == "" {
		return fmt.Errorf("webhook secret not configured")
	}

	// Remove "sha256=" prefix if present
	signature = strings.TrimPrefix(signature, "sha256=")

	// Calculate expected signature
	mac := hmac.New(sha256.New, []byte(v.secret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures using constant time comparison
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// GenerateSignature generates an HMAC-SHA256 signature for a payload
func (v *Validator) GenerateSignature(payload []byte) string {
	if v.secret == "" {
		return ""
	}

	mac := hmac.New(sha256.New, []byte(v.secret))
	mac.Write(payload)
	return "sha256=" + hex.EncodeToString(mac.Sum(nil))
}