package secrets

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
)

// SupabaseSecretsManager manages secrets using Supabase Edge Functions
type SupabaseSecretsManager struct {
	supabaseURL    string
	serviceRoleKey string
	httpClient     *http.Client
	cache          map[string]secretCacheEntry
	cacheMutex     sync.RWMutex
	logger         *slog.Logger
}

type secretCacheEntry struct {
	value     string
	expiresAt time.Time
}

type supabaseSecretRequest struct {
	Name string `json:"name"`
}

type supabaseSecretResponse struct {
	Value string `json:"value"`
	Error string `json:"error,omitempty"`
}

type supabaseSecretsListResponse struct {
	Secrets []string `json:"secrets"`
	Error   string   `json:"error,omitempty"`
}

// NewSupabaseSecretsManager creates a new Supabase secrets manager
func NewSupabaseSecretsManager(supabaseURL, serviceRoleKey string, logger *slog.Logger) *SupabaseSecretsManager {
	return &SupabaseSecretsManager{
		supabaseURL:    strings.TrimSuffix(supabaseURL, "/"),
		serviceRoleKey: serviceRoleKey,
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
		cache:  make(map[string]secretCacheEntry),
		logger: logger,
	}
}

// GetSecret retrieves a secret by name
func (sm *SupabaseSecretsManager) GetSecret(ctx context.Context, name string) (string, error) {
	// Check cache first
	sm.cacheMutex.RLock()
	if entry, exists := sm.cache[name]; exists && time.Now().Before(entry.expiresAt) {
		sm.cacheMutex.RUnlock()
		sm.logger.Debug("Secret retrieved from cache", "name", name)
		return entry.value, nil
	}
	sm.cacheMutex.RUnlock()

	// Fetch from Supabase
	value, err := sm.fetchSecret(ctx, name)
	if err != nil {
		return "", fmt.Errorf("failed to fetch secret %s: %w", name, err)
	}

	// Cache the result for 5 minutes
	sm.cacheMutex.Lock()
	sm.cache[name] = secretCacheEntry{
		value:     value,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	sm.cacheMutex.Unlock()

	sm.logger.Debug("Secret retrieved from Supabase", "name", name)
	return value, nil
}

// SetSecret stores a secret
func (sm *SupabaseSecretsManager) SetSecret(ctx context.Context, name, value string) error {
	err := sm.storeSecret(ctx, name, value)
	if err != nil {
		return fmt.Errorf("failed to store secret %s: %w", name, err)
	}

	// Update cache
	sm.cacheMutex.Lock()
	sm.cache[name] = secretCacheEntry{
		value:     value,
		expiresAt: time.Now().Add(5 * time.Minute),
	}
	sm.cacheMutex.Unlock()

	sm.logger.Info("Secret stored successfully", "name", name)
	return nil
}

// ListSecrets returns a list of available secret names
func (sm *SupabaseSecretsManager) ListSecrets(ctx context.Context) ([]string, error) {
	secrets, err := sm.fetchSecretsList(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list secrets: %w", err)
	}

	sm.logger.Debug("Listed secrets", "count", len(secrets))
	return secrets, nil
}

// DeleteSecret removes a secret
func (sm *SupabaseSecretsManager) DeleteSecret(ctx context.Context, name string) error {
	err := sm.removeSecret(ctx, name)
	if err != nil {
		return fmt.Errorf("failed to delete secret %s: %w", name, err)
	}

	// Remove from cache
	sm.cacheMutex.Lock()
	delete(sm.cache, name)
	sm.cacheMutex.Unlock()

	sm.logger.Info("Secret deleted successfully", "name", name)
	return nil
}

// ClearCache clears the secrets cache
func (sm *SupabaseSecretsManager) ClearCache() {
	sm.cacheMutex.Lock()
	sm.cache = make(map[string]secretCacheEntry)
	sm.cacheMutex.Unlock()
	sm.logger.Debug("Secrets cache cleared")
}

// fetchSecret retrieves a secret from Supabase Edge Function
func (sm *SupabaseSecretsManager) fetchSecret(ctx context.Context, name string) (string, error) {
	reqBody := supabaseSecretRequest{Name: name}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/functions/v1/get-secret", sm.supabaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonBody)))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+sm.serviceRoleKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := sm.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var response supabaseSecretResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return "", fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != "" {
		return "", fmt.Errorf("supabase error: %s", response.Error)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return response.Value, nil
}

// storeSecret stores a secret using Supabase Edge Function
func (sm *SupabaseSecretsManager) storeSecret(ctx context.Context, name, value string) error {
	reqBody := map[string]string{
		"name":  name,
		"value": value,
	}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/functions/v1/set-secret", sm.supabaseURL)
	req, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(jsonBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+sm.serviceRoleKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := sm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// fetchSecretsList retrieves list of secrets from Supabase Edge Function
func (sm *SupabaseSecretsManager) fetchSecretsList(ctx context.Context) ([]string, error) {
	url := fmt.Sprintf("%s/functions/v1/list-secrets", sm.supabaseURL)
	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+sm.serviceRoleKey)

	resp, err := sm.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	var response supabaseSecretsListResponse
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if response.Error != "" {
		return nil, fmt.Errorf("supabase error: %s", response.Error)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return response.Secrets, nil
}

// removeSecret deletes a secret using Supabase Edge Function
func (sm *SupabaseSecretsManager) removeSecret(ctx context.Context, name string) error {
	reqBody := supabaseSecretRequest{Name: name}
	jsonBody, err := json.Marshal(reqBody)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	url := fmt.Sprintf("%s/functions/v1/delete-secret", sm.supabaseURL)
	req, err := http.NewRequestWithContext(ctx, "DELETE", url, strings.NewReader(string(jsonBody)))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+sm.serviceRoleKey)
	req.Header.Set("Content-Type", "application/json")

	resp, err := sm.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to make request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	return nil
}

// MigrateFromEnv migrates secrets from environment variables to Supabase
func (sm *SupabaseSecretsManager) MigrateFromEnv(ctx context.Context, cfg *config.Config) error {
	secretsToMigrate := map[string]string{
		"JWT_SECRET":        cfg.Auth.JWTSecret,
		"GITHUB_CLIENT_ID":  cfg.Auth.GitHubClientID,
		"GITHUB_SECRET":     cfg.Auth.GitHubSecret,
		"GITLAB_CLIENT_ID":  cfg.Auth.GitLabClientID,
		"GITLAB_SECRET":     cfg.Auth.GitLabSecret,
		"DB_PASSWORD":       cfg.Database.Password,
		"REDIS_PASSWORD":    cfg.Redis.Password,
	}

	for name, value := range secretsToMigrate {
		if value != "" {
			if err := sm.SetSecret(ctx, name, value); err != nil {
				sm.logger.Error("Failed to migrate secret", "name", name, "error", err)
				return fmt.Errorf("failed to migrate secret %s: %w", name, err)
			}
			sm.logger.Info("Successfully migrated secret", "name", name)
		}
	}

	sm.logger.Info("All secrets migrated successfully")
	return nil
}

// LoadSecretsIntoConfig loads secrets from Supabase into config
func (sm *SupabaseSecretsManager) LoadSecretsIntoConfig(ctx context.Context, cfg *config.Config) error {
	secretMappings := map[string]*string{
		"JWT_SECRET":        &cfg.Auth.JWTSecret,
		"GITHUB_CLIENT_ID":  &cfg.Auth.GitHubClientID,
		"GITHUB_SECRET":     &cfg.Auth.GitHubSecret,
		"GITLAB_CLIENT_ID":  &cfg.Auth.GitLabClientID,
		"GITLAB_SECRET":     &cfg.Auth.GitLabSecret,
		"DB_PASSWORD":       &cfg.Database.Password,
		"REDIS_PASSWORD":    &cfg.Redis.Password,
	}

	for secretName, configField := range secretMappings {
		if *configField == "" { // Only load if not already set
			value, err := sm.GetSecret(ctx, secretName)
			if err != nil {
				sm.logger.Warn("Failed to load secret, using environment value", "name", secretName, "error", err)
				continue
			}
			*configField = value
			sm.logger.Debug("Loaded secret into config", "name", secretName)
		}
	}

	return nil
}