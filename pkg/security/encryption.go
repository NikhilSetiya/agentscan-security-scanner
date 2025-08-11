package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

// EncryptionService provides data encryption and decryption capabilities
type EncryptionService struct {
	key []byte
}

// NewEncryptionService creates a new encryption service with the provided key
func NewEncryptionService(key string) *EncryptionService {
	// Derive a 32-byte key using PBKDF2
	salt := []byte("agentscan-salt-2024") // In production, use a random salt per installation
	derivedKey := pbkdf2.Key([]byte(key), salt, 10000, 32, sha256.New)
	
	return &EncryptionService{
		key: derivedKey,
	}
}

// Encrypt encrypts plaintext data using AES-GCM
func (e *EncryptionService) Encrypt(plaintext string) (string, error) {
	if plaintext == "" {
		return "", nil
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// Decrypt decrypts ciphertext data using AES-GCM
func (e *EncryptionService) Decrypt(ciphertext string) (string, error) {
	if ciphertext == "" {
		return "", nil
	}

	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", errors.New("ciphertext too short")
	}

	nonce, ciphertext_bytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext_bytes, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

// EncryptSensitiveFields encrypts sensitive fields in a struct
func (e *EncryptionService) EncryptSensitiveFields(data map[string]interface{}) (map[string]interface{}, error) {
	sensitiveFields := []string{
		"password", "token", "secret", "key", "credential",
		"api_key", "access_token", "refresh_token", "private_key",
		"webhook_secret", "oauth_secret", "database_password",
	}

	result := make(map[string]interface{})
	for k, v := range data {
		if contains(sensitiveFields, k) {
			if str, ok := v.(string); ok && str != "" {
				encrypted, err := e.Encrypt(str)
				if err != nil {
					return nil, fmt.Errorf("failed to encrypt field %s: %w", k, err)
				}
				result[k] = encrypted
			} else {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result, nil
}

// DecryptSensitiveFields decrypts sensitive fields in a struct
func (e *EncryptionService) DecryptSensitiveFields(data map[string]interface{}) (map[string]interface{}, error) {
	sensitiveFields := []string{
		"password", "token", "secret", "key", "credential",
		"api_key", "access_token", "refresh_token", "private_key",
		"webhook_secret", "oauth_secret", "database_password",
	}

	result := make(map[string]interface{})
	for k, v := range data {
		if contains(sensitiveFields, k) {
			if str, ok := v.(string); ok && str != "" {
				decrypted, err := e.Decrypt(str)
				if err != nil {
					return nil, fmt.Errorf("failed to decrypt field %s: %w", k, err)
				}
				result[k] = decrypted
			} else {
				result[k] = v
			}
		} else {
			result[k] = v
		}
	}

	return result, nil
}

// HashPassword creates a secure hash of a password using PBKDF2
func (e *EncryptionService) HashPassword(password string) (string, error) {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		return "", fmt.Errorf("failed to generate salt: %w", err)
	}

	hash := pbkdf2.Key([]byte(password), salt, 10000, 32, sha256.New)
	
	// Combine salt and hash
	combined := append(salt, hash...)
	return base64.StdEncoding.EncodeToString(combined), nil
}

// VerifyPassword verifies a password against its hash
func (e *EncryptionService) VerifyPassword(password, hash string) (bool, error) {
	combined, err := base64.StdEncoding.DecodeString(hash)
	if err != nil {
		return false, fmt.Errorf("failed to decode hash: %w", err)
	}

	if len(combined) != 48 { // 16 bytes salt + 32 bytes hash
		return false, errors.New("invalid hash format")
	}

	salt := combined[:16]
	expectedHash := combined[16:]
	
	actualHash := pbkdf2.Key([]byte(password), salt, 10000, 32, sha256.New)
	
	// Constant time comparison
	if len(expectedHash) != len(actualHash) {
		return false, nil
	}
	
	var result byte
	for i := 0; i < len(expectedHash); i++ {
		result |= expectedHash[i] ^ actualHash[i]
	}
	
	return result == 0, nil
}

// GenerateSecureToken generates a cryptographically secure random token
func (e *EncryptionService) GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// contains checks if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}