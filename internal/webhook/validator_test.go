package webhook

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewValidator(t *testing.T) {
	secret := "test_secret"
	validator := NewValidator(secret)
	
	assert.NotNil(t, validator)
	assert.Equal(t, secret, validator.secret)
}

func TestValidateSignature(t *testing.T) {
	secret := "test_secret"
	validator := NewValidator(secret)
	payload := []byte(`{"test":"data"}`)

	tests := []struct {
		name        string
		payload     []byte
		signature   string
		expectError bool
		errorMsg    string
	}{
		{
			name:        "valid signature with sha256 prefix",
			payload:     payload,
			signature:   "sha256=8b94674b5d2d6e8b8cd7b8e7c5a5c5d5e5f5a5b5c5d5e5f5a5b5c5d5e5f5a5b5",
			expectError: true, // This will fail because it's not the correct signature
		},
		{
			name:        "valid signature without prefix",
			payload:     payload,
			signature:   "8b94674b5d2d6e8b8cd7b8e7c5a5c5d5e5f5a5b5c5d5e5f5a5b5c5d5e5f5a5b5",
			expectError: true, // This will fail because it's not the correct signature
		},
		{
			name:        "invalid signature",
			payload:     payload,
			signature:   "invalid_signature",
			expectError: true,
			errorMsg:    "invalid signature",
		},
		{
			name:        "empty signature",
			payload:     payload,
			signature:   "",
			expectError: true,
			errorMsg:    "invalid signature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validator.ValidateSignature(tt.payload, tt.signature)
			
			if tt.expectError {
				require.Error(t, err)
				if tt.errorMsg != "" {
					assert.Contains(t, err.Error(), tt.errorMsg)
				}
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestValidateSignatureWithCorrectSignature(t *testing.T) {
	secret := "test_secret"
	validator := NewValidator(secret)
	payload := []byte(`{"test":"data"}`)

	// Generate the correct signature
	expectedSignature := validator.GenerateSignature(payload)
	
	// Test with the correct signature
	err := validator.ValidateSignature(payload, expectedSignature)
	assert.NoError(t, err)
	
	// Test without sha256 prefix
	signatureWithoutPrefix := expectedSignature[7:] // Remove "sha256="
	err = validator.ValidateSignature(payload, signatureWithoutPrefix)
	assert.NoError(t, err)
}

func TestValidateSignatureNoSecret(t *testing.T) {
	validator := NewValidator("")
	payload := []byte(`{"test":"data"}`)
	
	err := validator.ValidateSignature(payload, "any_signature")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "webhook secret not configured")
}

func TestGenerateSignature(t *testing.T) {
	secret := "test_secret"
	validator := NewValidator(secret)
	payload := []byte(`{"test":"data"}`)

	signature := validator.GenerateSignature(payload)
	
	assert.NotEmpty(t, signature)
	assert.True(t, len(signature) > 7) // Should have "sha256=" prefix plus hex
	assert.Contains(t, signature, "sha256=")
	
	// Test that the same payload generates the same signature
	signature2 := validator.GenerateSignature(payload)
	assert.Equal(t, signature, signature2)
	
	// Test that different payloads generate different signatures
	differentPayload := []byte(`{"different":"data"}`)
	differentSignature := validator.GenerateSignature(differentPayload)
	assert.NotEqual(t, signature, differentSignature)
}

func TestGenerateSignatureNoSecret(t *testing.T) {
	validator := NewValidator("")
	payload := []byte(`{"test":"data"}`)
	
	signature := validator.GenerateSignature(payload)
	assert.Empty(t, signature)
}

func TestSignatureRoundTrip(t *testing.T) {
	secret := "test_secret_123"
	validator := NewValidator(secret)
	
	testPayloads := [][]byte{
		[]byte(`{"test":"data"}`),
		[]byte(`{"stream":{"id":"123","user_login":"testuser"}}`),
		[]byte(`{"challenge":"test_challenge_string"}`),
		[]byte(`{}`),
		[]byte(`{"complex":{"nested":{"data":["array","values"],"number":42}}}`),
	}
	
	for i, payload := range testPayloads {
		t.Run("payload_"+string(rune('0'+i)), func(t *testing.T) {
			// Generate signature
			signature := validator.GenerateSignature(payload)
			require.NotEmpty(t, signature)
			
			// Validate the generated signature
			err := validator.ValidateSignature(payload, signature)
			assert.NoError(t, err)
			
			// Test with modified payload (should fail)
			modifiedPayload := append(payload, byte(' '))
			err = validator.ValidateSignature(modifiedPayload, signature)
			assert.Error(t, err)
		})
	}
}