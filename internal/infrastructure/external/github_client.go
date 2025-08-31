package external

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
)

// GitHubClient wraps the GitHub API client
type GitHubClient struct {
	client *github.Client
	token  string
}

// NewGitHubClient creates a new GitHub client
func NewGitHubClient(token string) *GitHubClient {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(context.Background(), ts)
	
	return &GitHubClient{
		client: github.NewClient(tc),
		token:  token,
	}
}

// Repository represents a GitHub repository
type Repository struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	FullName    string    `json:"full_name"`
	URL         string    `json:"html_url"`
	CloneURL    string    `json:"clone_url"`
	Language    string    `json:"language"`
	Description string    `json:"description"`
	Private     bool      `json:"private"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// Branch represents a GitHub branch
type Branch struct {
	Name      string `json:"name"`
	SHA       string `json:"sha"`
	Protected bool   `json:"protected"`
}

// Commit represents a GitHub commit
type Commit struct {
	SHA     string    `json:"sha"`
	Message string    `json:"message"`
	Author  string    `json:"author"`
	Date    time.Time `json:"date"`
	URL     string    `json:"url"`
}

// PullRequest represents a GitHub pull request
type PullRequest struct {
	Number    int       `json:"number"`
	Title     string    `json:"title"`
	State     string    `json:"state"`
	Author    string    `json:"author"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	HeadSHA   string    `json:"head_sha"`
	BaseSHA   string    `json:"base_sha"`
}

// GetRepository retrieves repository information
func (c *GitHubClient) GetRepository(ctx context.Context, owner, repo string) (*Repository, error) {
	ghRepo, _, err := c.client.Repositories.Get(ctx, owner, repo)
	if err != nil {
		return nil, fmt.Errorf("failed to get repository: %w", err)
	}
	
	return &Repository{
		ID:          ghRepo.GetID(),
		Name:        ghRepo.GetName(),
		FullName:    ghRepo.GetFullName(),
		URL:         ghRepo.GetHTMLURL(),
		CloneURL:    ghRepo.GetCloneURL(),
		Language:    ghRepo.GetLanguage(),
		Description: ghRepo.GetDescription(),
		Private:     ghRepo.GetPrivate(),
		CreatedAt:   ghRepo.GetCreatedAt().Time,
		UpdatedAt:   ghRepo.GetUpdatedAt().Time,
	}, nil
}

// ListRepositories lists repositories for the authenticated user
func (c *GitHubClient) ListRepositories(ctx context.Context, opts *github.RepositoryListOptions) ([]*Repository, error) {
	ghRepos, _, err := c.client.Repositories.List(ctx, "", opts)
	if err != nil {
		return nil, fmt.Errorf("failed to list repositories: %w", err)
	}
	
	repos := make([]*Repository, len(ghRepos))
	for i, ghRepo := range ghRepos {
		repos[i] = &Repository{
			ID:          ghRepo.GetID(),
			Name:        ghRepo.GetName(),
			FullName:    ghRepo.GetFullName(),
			URL:         ghRepo.GetHTMLURL(),
			CloneURL:    ghRepo.GetCloneURL(),
			Language:    ghRepo.GetLanguage(),
			Description: ghRepo.GetDescription(),
			Private:     ghRepo.GetPrivate(),
			CreatedAt:   ghRepo.GetCreatedAt().Time,
			UpdatedAt:   ghRepo.GetUpdatedAt().Time,
		}
	}
	
	return repos, nil
}

// GetBranches retrieves branches for a repository
func (c *GitHubClient) GetBranches(ctx context.Context, owner, repo string) ([]*Branch, error) {
	ghBranches, _, err := c.client.Repositories.ListBranches(ctx, owner, repo, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}
	
	branches := make([]*Branch, len(ghBranches))
	for i, ghBranch := range ghBranches {
		branches[i] = &Branch{
			Name:      ghBranch.GetName(),
			SHA:       ghBranch.GetCommit().GetSHA(),
			Protected: ghBranch.GetProtected(),
		}
	}
	
	return branches, nil
}

// GetCommits retrieves commits for a repository
func (c *GitHubClient) GetCommits(ctx context.Context, owner, repo string, opts *github.CommitsListOptions) ([]*Commit, error) {
	ghCommits, _, err := c.client.Repositories.ListCommits(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get commits: %w", err)
	}
	
	commits := make([]*Commit, len(ghCommits))
	for i, ghCommit := range ghCommits {
		commits[i] = &Commit{
			SHA:     ghCommit.GetSHA(),
			Message: ghCommit.GetCommit().GetMessage(),
			Author:  ghCommit.GetCommit().GetAuthor().GetName(),
			Date:    ghCommit.GetCommit().GetAuthor().GetDate().Time,
			URL:     ghCommit.GetHTMLURL(),
		}
	}
	
	return commits, nil
}

// GetPullRequests retrieves pull requests for a repository
func (c *GitHubClient) GetPullRequests(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*PullRequest, error) {
	ghPRs, _, err := c.client.PullRequests.List(ctx, owner, repo, opts)
	if err != nil {
		return nil, fmt.Errorf("failed to get pull requests: %w", err)
	}
	
	prs := make([]*PullRequest, len(ghPRs))
	for i, ghPR := range ghPRs {
		prs[i] = &PullRequest{
			Number:    ghPR.GetNumber(),
			Title:     ghPR.GetTitle(),
			State:     ghPR.GetState(),
			Author:    ghPR.GetUser().GetLogin(),
			CreatedAt: ghPR.GetCreatedAt().Time,
			UpdatedAt: ghPR.GetUpdatedAt().Time,
			HeadSHA:   ghPR.GetHead().GetSHA(),
			BaseSHA:   ghPR.GetBase().GetSHA(),
		}
	}
	
	return prs, nil
}

// CreateWebhook creates a webhook for a repository
func (c *GitHubClient) CreateWebhook(ctx context.Context, owner, repo, url string, events []string) error {
	hook := &github.Hook{
		Name:   github.String("web"),
		Active: github.Bool(true),
		Events: events,
		Config: map[string]interface{}{
			"url":          url,
			"content_type": "json",
		},
	}
	
	_, _, err := c.client.Repositories.CreateHook(ctx, owner, repo, hook)
	if err != nil {
		return fmt.Errorf("failed to create webhook: %w", err)
	}
	
	return nil
}

// DeleteWebhook deletes a webhook from a repository
func (c *GitHubClient) DeleteWebhook(ctx context.Context, owner, repo string, hookID int64) error {
	_, err := c.client.Repositories.DeleteHook(ctx, owner, repo, hookID)
	if err != nil {
		return fmt.Errorf("failed to delete webhook: %w", err)
	}
	
	return nil
}

// CreatePullRequestComment creates a comment on a pull request
func (c *GitHubClient) CreatePullRequestComment(ctx context.Context, owner, repo string, number int, body string) error {
	comment := &github.IssueComment{
		Body: github.String(body),
	}
	
	_, _, err := c.client.Issues.CreateComment(ctx, owner, repo, number, comment)
	if err != nil {
		return fmt.Errorf("failed to create pull request comment: %w", err)
	}
	
	return nil
}

// CreateCommitStatus creates a status for a commit
func (c *GitHubClient) CreateCommitStatus(ctx context.Context, owner, repo, sha, state, description, targetURL string) error {
	status := &github.RepoStatus{
		State:       github.String(state),
		Description: github.String(description),
		TargetURL:   github.String(targetURL),
		Context:     github.String("agentscan/security-scan"),
	}
	
	_, _, err := c.client.Repositories.CreateStatus(ctx, owner, repo, sha, status)
	if err != nil {
		return fmt.Errorf("failed to create commit status: %w", err)
	}
	
	return nil
}

// ParseRepositoryURL parses a GitHub repository URL to extract owner and repo
func ParseRepositoryURL(url string) (owner, repo string, err error) {
	// Handle different URL formats
	url = strings.TrimSuffix(url, ".git")
	url = strings.TrimSuffix(url, "/")
	
	if strings.Contains(url, "github.com") {
		parts := strings.Split(url, "/")
		if len(parts) >= 2 {
			owner = parts[len(parts)-2]
			repo = parts[len(parts)-1]
			return owner, repo, nil
		}
	}
	
	return "", "", fmt.Errorf("invalid GitHub repository URL: %s", url)
}

// ValidateWebhookSignature validates a GitHub webhook signature
func ValidateWebhookSignature(payload []byte, signature, secret string) bool {
	return github.ValidateSignature(signature, payload, []byte(secret))
}

// ParseWebhookPayload parses a GitHub webhook payload
func ParseWebhookPayload(eventType string, payload []byte) (interface{}, error) {
	switch eventType {
	case "push":
		var event github.PushEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf("failed to parse push event: %w", err)
		}
		return &event, nil
		
	case "pull_request":
		var event github.PullRequestEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf("failed to parse pull request event: %w", err)
		}
		return &event, nil
		
	case "repository":
		var event github.RepositoryEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			return nil, fmt.Errorf("failed to parse repository event: %w", err)
		}
		return &event, nil
		
	default:
		return nil, fmt.Errorf("unsupported event type: %s", eventType)
	}
}