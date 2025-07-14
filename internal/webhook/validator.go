package webhook

import (
	"crypto/hmac"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
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

// ValidateSignature validates the HMAC signature of a webhook payload
func (v *Validator) ValidateSignature(payload []byte, signature string) error {
	if v.secret == "" {
		return fmt.Errorf("webhook secret not configured")
	}

	// Determine algorithm from signature prefix
	var hashFunc func() hash.Hash
	var prefix string
	
	if strings.HasPrefix(signature, "sha1=") {
		prefix = "sha1="
		hashFunc = sha1.New
	} else if strings.HasPrefix(signature, "sha256=") {
		prefix = "sha256="
		hashFunc = sha256.New
	} else if strings.HasPrefix(signature, "sha512=") {
		prefix = "sha512="
		hashFunc = sha512.New
	} else {
		// Default to SHA-256 if no prefix found
		hashFunc = sha256.New
	}

	// Remove prefix if present
	signature = strings.TrimPrefix(signature, prefix)

	// Calculate expected signature
	mac := hmac.New(hashFunc, []byte(v.secret))
	mac.Write(payload)
	expectedSignature := hex.EncodeToString(mac.Sum(nil))

	// Compare signatures using constant time comparison
	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		return fmt.Errorf("invalid signature")
	}

	return nil
}

// GenerateSignature generates an HMAC signature for a payload using the specified algorithm
func (v *Validator) GenerateSignature(payload []byte, algorithm string) string {
	if v.secret == "" {
		return ""
	}

	var hashFunc func() hash.Hash
	var prefix string

	// Default to SHA-256 if not specified
	if algorithm == "" {
		algorithm = "SHA-256"
	}

	switch strings.ToUpper(algorithm) {
	case "SHA-1":
		hashFunc = sha1.New
		prefix = "sha1"
	case "SHA-256":
		hashFunc = sha256.New
		prefix = "sha256"
	case "SHA-512":
		hashFunc = sha512.New
		prefix = "sha512"
	default:
		hashFunc = sha256.New
		prefix = "sha256"
	}

	mac := hmac.New(hashFunc, []byte(v.secret))
	mac.Write(payload)
	return prefix + "=" + hex.EncodeToString(mac.Sum(nil))
}
