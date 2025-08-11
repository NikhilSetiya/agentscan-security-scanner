package security

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEncryptionService_EncryptDecrypt(t *testing.T) {
	service := NewEncryptionService("test-key-123")

	tests := []struct {
		name      string
		plaintext string
	}{
		{"empty string", ""},
		{"simple text", "hello world"},
		{"special characters", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"unicode", "Hello 世界 🌍"},
		{"long text", "This is a very long text that should be encrypted and decrypted properly without any issues"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Encrypt
			ciphertext, err := service.Encrypt(tt.plaintext)
			require.NoError(t, err)

			if tt.plaintext == "" {
				assert.Equal(t, "", ciphertext)
				return
			}

			assert.NotEqual(t, tt.plaintext, ciphertext)
			assert.NotEmpty(t, ciphertext)

			// Decrypt
			decrypted, err := service.Decrypt(ciphertext)
			require.NoError(t, err)
			assert.Equal(t, tt.plaintext, decrypted)
		})
	}
}

func TestEncryptionService_EncryptSensitiveFields(t *testing.T) {
	service := NewEncryptionService("test-key-123")

	data := map[string]interface{}{
		"username":         "testuser",
		"password":         "secret123",
		"api_key":          "key123",
		"normal_field":     "normal_value",
		"access_token":     "token123",
		"non_string_field": 42,
	}

	encrypted, err := service.EncryptSensitiveFields(data)
	require.NoError(t, err)

	// Check that sensitive fields are encrypted
	assert.NotEqual(t, "secret123", encrypted["password"])
	assert.NotEqual(t, "key123", encrypted["api_key"])
	assert.NotEqual(t, "token123", encrypted["access_token"])

	// Check that non-sensitive fields are unchanged
	assert.Equal(t, "testuser", encrypted["username"])
	assert.Equal(t, "normal_value", encrypted["normal_field"])
	assert.Equal(t, 42, encrypted["non_string_field"])

	// Decrypt and verify
	decrypted, err := service.DecryptSensitiveFields(encrypted)
	require.NoError(t, err)

	assert.Equal(t, "secret123", decrypted["password"])
	assert.Equal(t, "key123", decrypted["api_key"])
	assert.Equal(t, "token123", decrypted["access_token"])
}

func TestEncryptionService_HashVerifyPassword(t *testing.T) {
	service := NewEncryptionService("test-key-123")

	password := "mySecurePassword123!"

	// Hash password
	hash, err := service.HashPassword(password)
	require.NoError(t, err)
	assert.NotEmpty(t, hash)
	assert.NotEqual(t, password, hash)

	// Verify correct password
	valid, err := service.VerifyPassword(password, hash)
	require.NoError(t, err)
	assert.True(t, valid)

	// Verify incorrect password
	valid, err = service.VerifyPassword("wrongPassword", hash)
	require.NoError(t, err)
	assert.False(t, valid)
}

func TestEncryptionService_GenerateSecureToken(t *testing.T) {
	service := NewEncryptionService("test-key-123")

	// Test different token lengths
	lengths := []int{16, 32, 64}

	for _, length := range lengths {
		t.Run(fmt.Sprintf("length_%d", length), func(t *testing.T) {
			token, err := service.GenerateSecureToken(length)
			require.NoError(t, err)
			assert.NotEmpty(t, token)

			// Generate another token and ensure they're different
			token2, err := service.GenerateSecureToken(length)
			require.NoError(t, err)
			assert.NotEqual(t, token, token2)
		})
	}
}

func TestEncryptionService_DecryptInvalidData(t *testing.T) {
	service := NewEncryptionService("test-key-123")

	tests := []struct {
		name       string
		ciphertext string
	}{
		{"invalid base64", "invalid-base64!"},
		{"too short", "dGVzdA=="}, // "test" in base64, too short for nonce
		{"wrong key", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.name == "wrong key" {
				// Encrypt with one key, decrypt with another
				service1 := NewEncryptionService("key1")
				service2 := NewEncryptionService("key2")

				encrypted, err := service1.Encrypt("test")
				require.NoError(t, err)

				_, err = service2.Decrypt(encrypted)
				assert.Error(t, err)
			} else {
				_, err := service.Decrypt(tt.ciphertext)
				assert.Error(t, err)
			}
		})
	}
}

func TestEncryptionService_VerifyPasswordInvalidHash(t *testing.T) {
	service := NewEncryptionService("test-key-123")

	tests := []struct {
		name string
		hash string
	}{
		{"invalid base64", "invalid-base64!"},
		{"wrong length", "dGVzdA=="}, // "test" in base64, wrong length
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			valid, err := service.VerifyPassword("password", tt.hash)
			assert.Error(t, err)
			assert.False(t, valid)
		})
	}
}