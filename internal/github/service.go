package github

import (
	"context"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"sync"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
)

// Service manages GitHub App integrations
type Service struct {
	config  *config.Config
	repos   *database.Repositories
	clients map[int64]*Client // installationID -> Client
	mu      sync.RWMutex
}

// NewService creates a new GitHub service
func NewService(cfg *config.Config, repos *database.Repositories) *Service {
	return &Service{
		config:  cfg,
		repos:   repos,
		clients: make(map[int64]*Client),
	}
}

// GetClientForInstallation gets or creates a GitHub client for an installation
func (s *Service) GetClientForInstallation(ctx context.Context, installationID int64) (*Client, error) {
	s.mu.RLock()
	client, exists := s.clients[installationID]
	s.mu.RUnlock()

	if exists {
		return client, nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	// Double-check after acquiring write lock
	if client, exists := s.clients[installationID]; exists {
		return client, nil
	}

	// Get GitHub App configuration for this installation
	appConfig, err := s.getAppConfigForInstallation(ctx, installationID)
	if err != nil {
		return nil, fmt.Errorf("failed to get app config: %w", err)
	}

	// Parse private key
	privateKey, err := parsePrivateKey(appConfig.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}

	// Create client
	client = NewClient(appConfig.AppID, privateKey, installationID)
	s.clients[installationID] = client

	return client, nil
}

// getAppConfigForInstallation gets the GitHub App configuration for an installation
func (s *Service) getAppConfigForInstallation(ctx context.Context, installationID int64) (*GitHubApp, error) {
	// TODO: Implement database lookup for GitHub App configuration
	// For now, return a mock configuration
	return &GitHubApp{
		AppID:         123456, // This should come from environment or database
		InstallationID: installationID,
		PrivateKey:    s.config.GitHub.PrivateKey, // This should be stored encrypted in database
	}, nil
}

// parsePrivateKey parses a PEM-encoded RSA private key
func parsePrivateKey(privateKeyPEM string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(privateKeyPEM))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		// Try PKCS8 format
		key, err := x509.ParsePKCS8PrivateKey(block.Bytes)
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		
		rsaKey, ok := key.(*rsa.PrivateKey)
		if !ok {
			return nil, fmt.Errorf("private key is not RSA")
		}
		
		return rsaKey, nil
	}

	return privateKey, nil
}

// SyncRepositories syncs repositories from a GitHub installation
func (s *Service) SyncRepositories(ctx context.Context, installationID int64, organizationID string) error {
	client, err := s.GetClientForInstallation(ctx, installationID)
	if err != nil {
		return fmt.Errorf("failed to get GitHub client: %w", err)
	}

	repos, err := client.GetInstallationRepositories(ctx)
	if err != nil {
		return fmt.Errorf("failed to get installation repositories: %w", err)
	}

	// TODO: Implement repository synchronization with database
	// This would involve:
	// 1. Creating/updating repository records
	// 2. Setting up proper permissions
	// 3. Detecting languages and frameworks
	
	fmt.Printf("Found %d repositories for installation %d\n", len(repos), installationID)
	for _, repo := range repos {
		fmt.Printf("- %s (ID: %d, Private: %t)\n", repo.FullName, repo.ID, repo.Private)
	}

	return nil
}

// ValidateRepositoryAccess validates that a user has access to a repository
func (s *Service) ValidateRepositoryAccess(ctx context.Context, userID, repositoryID string) (bool, error) {
	// TODO: Implement repository access validation
	// This would check:
	// 1. User's GitHub permissions for the repository
	// 2. Organization membership
	// 3. Repository visibility settings
	
	return true, nil // For now, allow all access
}

// CreateCheckRun creates a GitHub check run for a scan
func (s *Service) CreateCheckRun(ctx context.Context, installationID int64, owner, repo, sha string, scanID string) error {
	client, err := s.GetClientForInstallation(ctx, installationID)
	if err != nil {
		return fmt.Errorf("failed to get GitHub client: %w", err)
	}

	checkRun := &CheckRun{
		Name:    "AgentScan Security",
		HeadSHA: sha,
		Status:  "in_progress",
		Output: &CheckRunOutput{
			Title:   "Security Scan in Progress",
			Summary: "AgentScan is analyzing your code for security vulnerabilities...",
		},
	}

	return client.CreateCheckRun(ctx, owner, repo, checkRun)
}

// UpdateCheckRunWithResults updates a GitHub check run with scan results
func (s *Service) UpdateCheckRunWithResults(ctx context.Context, installationID int64, owner, repo string, checkRunID int64, results interface{}) error {
	client, err := s.GetClientForInstallation(ctx, installationID)
	if err != nil {
		return fmt.Errorf("failed to get GitHub client: %w", err)
	}

	// TODO: Format results into check run format
	checkRun := &CheckRun{
		Status:     "completed",
		Conclusion: "success", // This should be determined by results
		Output: &CheckRunOutput{
			Title:   "Security Scan Complete",
			Summary: "AgentScan has completed analyzing your code.",
		},
	}

	return client.UpdateCheckRun(ctx, owner, repo, checkRunID, checkRun)
}