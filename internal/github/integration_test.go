package github

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/NikhilSetiya/agentscan-security-scanner/internal/database"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/orchestrator"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/config"
	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

func TestGitHubIntegration(t *testing.T) {
	// Skip integration tests in short mode
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Setup test dependencies
	cfg := &config.Config{
		GitHub: config.GitHubConfig{
			AppID:      123456,
			PrivateKey: generateTestPrivateKey(t),
		},
	}

	// Mock database repositories
	repos := &database.Repositories{
		Users:    &mockUserRepository{},
		ScanJobs: nil, // Not needed for this test
		Findings: nil, // Not needed for this test
	}

	// Mock orchestrator
	mockOrch := &mockOrchestrator{}

	// Create GitHub service
	githubService := NewService(cfg, repos)
	webhookHandler := NewWebhookHandler(repos, mockOrch, githubService)

	t.Run("HandlePullRequestWebhook", func(t *testing.T) {
		// Create test PR webhook payload
		prEvent := PullRequestEvent{
			Action: "opened",
			Number: 123,
			PullRequest: struct {
				ID     int    `json:"id"`
				Number int    `json:"number"`
				State  string `json:"state"`
				Title  string `json:"title"`
				Body   string `json:"body"`
				Head   struct {
					SHA string `json:"sha"`
					Ref string `json:"ref"`
				} `json:"head"`
				Base struct {
					SHA string `json:"sha"`
					Ref string `json:"ref"`
				} `json:"base"`
				User struct {
					Login string `json:"login"`
				} `json:"user"`
			}{
				ID:     123,
				Number: 123,
				State:  "open",
				Title:  "Test PR",
				Body:   "Test PR body",
				Head: struct {
					SHA string `json:"sha"`
					Ref string `json:"ref"`
				}{
					SHA: "abc123",
					Ref: "feature-branch",
				},
				Base: struct {
					SHA string `json:"sha"`
					Ref string `json:"ref"`
				}{
					SHA: "def456",
					Ref: "main",
				},
				User: struct {
					Login string `json:"login"`
				}{
					Login: "testuser",
				},
			},
			Repository: struct {
				ID       int    `json:"id"`
				Name     string `json:"name"`
				FullName string `json:"full_name"`
				HTMLURL  string `json:"html_url"`
				CloneURL string `json:"clone_url"`
				Owner    struct {
					Login string `json:"login"`
				} `json:"owner"`
			}{
				ID:       456,
				Name:     "test-repo",
				FullName: "testorg/test-repo",
				HTMLURL:  "https://github.com/testorg/test-repo",
				CloneURL: "https://github.com/testorg/test-repo.git",
				Owner: struct {
					Login string `json:"login"`
				}{
					Login: "testorg",
				},
			},
			Installation: struct {
				ID int64 `json:"id"`
			}{
				ID: 789,
			},
		}

		payload, err := json.Marshal(prEvent)
		require.NoError(t, err)

		// Create test HTTP request
		req := httptest.NewRequest("POST", "/webhooks/github", strings.NewReader(string(payload)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "pull_request")
		req.Header.Set("X-GitHub-Delivery", "test-delivery-id")
		// Skip signature verification for test
		req.Header.Set("X-Hub-Signature-256", "")

		// Create response recorder
		w := httptest.NewRecorder()

		// Handle webhook
		webhookHandler.HandleWebhook(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "OK", w.Body.String())

		// Verify scan was submitted
		assert.True(t, mockOrch.scanSubmitted)
		assert.Equal(t, "feature-branch", mockOrch.lastScanRequest.Branch)
		assert.Equal(t, "abc123", mockOrch.lastScanRequest.CommitSHA)
	})

	t.Run("HandlePushWebhook", func(t *testing.T) {
		// Reset mock
		mockOrch.scanSubmitted = false

		// Create test push webhook payload
		pushEvent := PushEvent{
			Ref:    "refs/heads/main",
			Before: "def456",
			After:  "abc123",
			Repository: struct {
				ID       int    `json:"id"`
				Name     string `json:"name"`
				FullName string `json:"full_name"`
				HTMLURL  string `json:"html_url"`
				CloneURL string `json:"clone_url"`
				Owner    struct {
					Login string `json:"login"`
				} `json:"owner"`
			}{
				ID:       456,
				Name:     "test-repo",
				FullName: "testorg/test-repo",
				HTMLURL:  "https://github.com/testorg/test-repo",
				CloneURL: "https://github.com/testorg/test-repo.git",
				Owner: struct {
					Login string `json:"login"`
				}{
					Login: "testorg",
				},
			},
			Installation: struct {
				ID int64 `json:"id"`
			}{
				ID: 789,
			},
		}

		payload, err := json.Marshal(pushEvent)
		require.NoError(t, err)

		// Create test HTTP request
		req := httptest.NewRequest("POST", "/webhooks/github", strings.NewReader(string(payload)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Event", "push")
		req.Header.Set("X-GitHub-Delivery", "test-delivery-id")
		// Skip signature verification for test
		req.Header.Set("X-Hub-Signature-256", "")

		// Create response recorder
		w := httptest.NewRecorder()

		// Handle webhook
		webhookHandler.HandleWebhook(w, req)

		// Verify response
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "OK", w.Body.String())

		// Verify scan was submitted
		assert.True(t, mockOrch.scanSubmitted)
		assert.Equal(t, "main", mockOrch.lastScanRequest.Branch)
		assert.Equal(t, "abc123", mockOrch.lastScanRequest.CommitSHA)
	})

	t.Run("GenerateWorkflowYAML", func(t *testing.T) {
		options := DefaultWorkflowOptions()
		options.SetupNode = true
		options.SetupPython = true

		yaml := GenerateWorkflowYAML("test-repo", options)

		// Verify workflow contains expected elements
		assert.Contains(t, yaml, "name: AgentScan Security")
		assert.Contains(t, yaml, "on:")
		assert.Contains(t, yaml, "pull_request:")
		assert.Contains(t, yaml, "push:")
		assert.Contains(t, yaml, "uses: actions/setup-node@v4")
		assert.Contains(t, yaml, "uses: actions/setup-python@v4")
		assert.Contains(t, yaml, "uses: agentscan/agentscan-action@v1")
		assert.Contains(t, yaml, "fail-on-severity: high")
		assert.Contains(t, yaml, "upload-sarif@v2")
	})

	t.Run("GenerateActionYAML", func(t *testing.T) {
		yaml := GenerateActionYAML()

		// Verify action.yml contains expected elements
		assert.Contains(t, yaml, "name: 'AgentScan Security Scanner'")
		assert.Contains(t, yaml, "inputs:")
		assert.Contains(t, yaml, "api-url:")
		assert.Contains(t, yaml, "api-token:")
		assert.Contains(t, yaml, "outputs:")
		assert.Contains(t, yaml, "results-file:")
		assert.Contains(t, yaml, "runs:")
		assert.Contains(t, yaml, "using: 'docker'")
	})
}

func TestGitHubClient(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration tests in short mode")
	}

	// Generate test private key
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Create client
	client := NewClient(123456, privateKey, 789)

	t.Run("GenerateJWT", func(t *testing.T) {
		token, err := client.generateJWT()
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Verify token format (should have 3 parts separated by dots)
		parts := strings.Split(token, ".")
		assert.Len(t, parts, 3)
	})

	t.Run("CreateCheckRun", func(t *testing.T) {
		// Mock GitHub API server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.URL.Path, "/repos/testorg/test-repo/check-runs")
			assert.Contains(t, r.Header.Get("Authorization"), "token")
			assert.Equal(t, "application/vnd.github.v3+json", r.Header.Get("Accept"))

			// Return mock response
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id": 123, "status": "in_progress"}`))
		}))
		defer server.Close()

		// Update client base URL to use test server
		client.baseURL = server.URL

		checkRun := &CheckRun{
			Name:    "AgentScan Security",
			HeadSHA: "abc123",
			Status:  "in_progress",
		}

		err := client.CreateCheckRun(context.Background(), "testorg", "test-repo", checkRun)
		assert.NoError(t, err)
	})

	t.Run("CreatePRComment", func(t *testing.T) {
		// Mock GitHub API server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.URL.Path, "/repos/testorg/test-repo/issues/123/comments")

			// Return mock response
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"id": 456, "body": "Test comment"}`))
		}))
		defer server.Close()

		// Update client base URL to use test server
		client.baseURL = server.URL

		comment := &PRComment{
			Body: "Test comment",
		}

		err := client.CreatePRComment(context.Background(), "testorg", "test-repo", 123, comment)
		assert.NoError(t, err)
	})

	t.Run("CreateStatusCheck", func(t *testing.T) {
		// Mock GitHub API server
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			// Verify request
			assert.Equal(t, "POST", r.Method)
			assert.Contains(t, r.URL.Path, "/repos/testorg/test-repo/statuses/abc123")

			// Return mock response
			w.WriteHeader(http.StatusCreated)
			w.Write([]byte(`{"state": "success", "context": "agentscan/security"}`))
		}))
		defer server.Close()

		// Update client base URL to use test server
		client.baseURL = server.URL

		status := &StatusCheck{
			State:       "success",
			Description: "No security issues found",
			Context:     "agentscan/security",
		}

		err := client.CreateStatusCheck(context.Background(), "testorg", "test-repo", "abc123", status)
		assert.NoError(t, err)
	})
}

// Mock implementations for testing

type mockOrchestrator struct {
	scanSubmitted   bool
	lastScanRequest *orchestrator.ScanRequest
}

func (m *mockOrchestrator) SubmitScan(ctx context.Context, req *orchestrator.ScanRequest) (*types.ScanJob, error) {
	m.scanSubmitted = true
	m.lastScanRequest = req
	return &types.ScanJob{
		ID:        uuid.New(),
		Branch:    req.Branch,
		CommitSHA: req.CommitSHA,
	}, nil
}

func (m *mockOrchestrator) GetScanStatus(ctx context.Context, jobID string) (*orchestrator.ScanStatus, error) {
	return &orchestrator.ScanStatus{
		Status: "completed",
	}, nil
}

func (m *mockOrchestrator) GetScanResults(ctx context.Context, jobID string, filter *orchestrator.ResultFilter) (*orchestrator.ScanResults, error) {
	return &orchestrator.ScanResults{
		Status:   "completed",
		Findings: []orchestrator.Finding{},
	}, nil
}

func (m *mockOrchestrator) CancelScan(ctx context.Context, jobID string) error {
	return nil
}

func (m *mockOrchestrator) ListScans(ctx context.Context, filter *orchestrator.ScanFilter, pagination *orchestrator.Pagination) (*orchestrator.ScanList, error) {
	return &orchestrator.ScanList{}, nil
}

func (m *mockOrchestrator) Start(ctx context.Context) error {
	return nil
}

func (m *mockOrchestrator) Stop(ctx context.Context) error {
	return nil
}

func (m *mockOrchestrator) Health(ctx context.Context) error {
	return nil
}

type mockUserRepository struct{}

func (m *mockUserRepository) Create(ctx context.Context, user *types.User) error {
	return nil
}

func (m *mockUserRepository) GetByID(ctx context.Context, id uuid.UUID) (*types.User, error) {
	return &types.User{ID: id}, nil
}

func (m *mockUserRepository) GetByEmail(ctx context.Context, email string) (*types.User, error) {
	return &types.User{Email: email}, nil
}

func (m *mockUserRepository) GetBySupabaseID(ctx context.Context, supabaseID string) (*types.User, error) {
	return &types.User{SupabaseID: &supabaseID}, nil
}

func (m *mockUserRepository) Update(ctx context.Context, user *types.User) error {
	return nil
}

func (m *mockUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockUserRepository) List(ctx context.Context, pagination *database.Pagination) ([]*types.User, int64, error) {
	return []*types.User{}, 0, nil
}

type mockOrganizationRepository struct{}

func (m *mockOrganizationRepository) Create(ctx context.Context, org *types.Organization) error {
	return nil
}

func (m *mockOrganizationRepository) GetByID(ctx context.Context, id uuid.UUID) (*types.Organization, error) {
	return &types.Organization{ID: id}, nil
}

func (m *mockOrganizationRepository) GetBySlug(ctx context.Context, slug string) (*types.Organization, error) {
	return &types.Organization{Slug: slug}, nil
}

func (m *mockOrganizationRepository) Update(ctx context.Context, org *types.Organization) error {
	return nil
}

func (m *mockOrganizationRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockOrganizationRepository) ListByUser(ctx context.Context, userID uuid.UUID) ([]*types.Organization, error) {
	return []*types.Organization{}, nil
}

type mockRepositoryRepository struct{}

func (m *mockRepositoryRepository) Create(ctx context.Context, repo *types.Repository) error {
	return nil
}

func (m *mockRepositoryRepository) GetByID(ctx context.Context, id uuid.UUID) (*types.Repository, error) {
	return &types.Repository{
		ID:            id,
		DefaultBranch: "main",
	}, nil
}

func (m *mockRepositoryRepository) GetByProviderID(ctx context.Context, provider, providerID string) (*types.Repository, error) {
	return &types.Repository{
		ID:            uuid.New(),
		DefaultBranch: "main",
		Provider:      provider,
		ProviderID:    providerID,
	}, nil
}

func (m *mockRepositoryRepository) Update(ctx context.Context, repo *types.Repository) error {
	return nil
}

func (m *mockRepositoryRepository) Delete(ctx context.Context, id uuid.UUID) error {
	return nil
}

func (m *mockRepositoryRepository) ListByOrganization(ctx context.Context, orgID uuid.UUID) ([]*types.Repository, error) {
	return []*types.Repository{}, nil
}



// generateTestPrivateKey generates a test RSA private key in PEM format
func generateTestPrivateKey(t *testing.T) string {
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	require.NoError(t, err)

	// Convert to PEM format
	privateKeyBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	privateKeyPEM := string(pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privateKeyBytes,
	}))

	return privateKeyPEM
}