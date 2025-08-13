package github

import (
	"time"

	"github.com/google/uuid"
)

// GitHubApp represents a GitHub App installation
type GitHubApp struct {
	ID           uuid.UUID `json:"id" db:"id"`
	AppID        int64     `json:"app_id" db:"app_id"`
	InstallationID int64   `json:"installation_id" db:"installation_id"`
	OrganizationID uuid.UUID `json:"organization_id" db:"organization_id"`
	PrivateKey   string    `json:"-" db:"private_key"` // Encrypted
	WebhookSecret string   `json:"-" db:"webhook_secret"` // Encrypted
	Permissions  map[string]string `json:"permissions" db:"permissions"`
	Events       []string  `json:"events" db:"events"`
	CreatedAt    time.Time `json:"created_at" db:"created_at"`
	UpdatedAt    time.Time `json:"updated_at" db:"updated_at"`
}

// WebhookEvent represents a GitHub webhook event
type WebhookEvent struct {
	ID          uuid.UUID `json:"id" db:"id"`
	EventType   string    `json:"event_type" db:"event_type"`
	Action      string    `json:"action" db:"action"`
	Repository  string    `json:"repository" db:"repository"`
	PullRequest *int      `json:"pull_request,omitempty" db:"pull_request"`
	Branch      string    `json:"branch" db:"branch"`
	Commit      string    `json:"commit" db:"commit"`
	Payload     []byte    `json:"payload" db:"payload"`
	ProcessedAt *time.Time `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt   time.Time `json:"created_at" db:"created_at"`
}

// PullRequestEvent represents a GitHub pull request webhook event
type PullRequestEvent struct {
	Action      string `json:"action"`
	Number      int    `json:"number"`
	PullRequest struct {
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
	} `json:"pull_request"`
	Repository struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
		CloneURL string `json:"clone_url"`
		Owner    struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
}

// PushEvent represents a GitHub push webhook event
type PushEvent struct {
	Ref        string `json:"ref"`
	Before     string `json:"before"`
	After      string `json:"after"`
	Repository struct {
		ID       int    `json:"id"`
		Name     string `json:"name"`
		FullName string `json:"full_name"`
		HTMLURL  string `json:"html_url"`
		CloneURL string `json:"clone_url"`
		Owner    struct {
			Login string `json:"login"`
		} `json:"owner"`
	} `json:"repository"`
	Installation struct {
		ID int64 `json:"id"`
	} `json:"installation"`
	Commits []struct {
		ID      string `json:"id"`
		Message string `json:"message"`
		Author  struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"commits"`
}

// CheckRun represents a GitHub check run
type CheckRun struct {
	Name        string `json:"name"`
	HeadSHA     string `json:"head_sha"`
	Status      string `json:"status"` // queued, in_progress, completed
	Conclusion  string `json:"conclusion,omitempty"` // success, failure, neutral, cancelled, skipped, timed_out, action_required
	StartedAt   *time.Time `json:"started_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
	Output      *CheckRunOutput `json:"output,omitempty"`
}

// CheckRunOutput represents the output of a GitHub check run
type CheckRunOutput struct {
	Title       string                    `json:"title"`
	Summary     string                    `json:"summary"`
	Text        string                    `json:"text,omitempty"`
	Annotations []CheckRunAnnotation      `json:"annotations,omitempty"`
}

// CheckRunAnnotation represents an annotation in a GitHub check run
type CheckRunAnnotation struct {
	Path            string `json:"path"`
	StartLine       int    `json:"start_line"`
	EndLine         int    `json:"end_line"`
	StartColumn     *int   `json:"start_column,omitempty"`
	EndColumn       *int   `json:"end_column,omitempty"`
	AnnotationLevel string `json:"annotation_level"` // notice, warning, failure
	Message         string `json:"message"`
	Title           string `json:"title,omitempty"`
	RawDetails      string `json:"raw_details,omitempty"`
}

// PRComment represents a comment on a GitHub pull request
type PRComment struct {
	Body string `json:"body"`
}

// PRCommentResponse represents a GitHub pull request comment response
type PRCommentResponse struct {
	ID   int64  `json:"id"`
	Body string `json:"body"`
	User struct {
		Login string `json:"login"`
		Type  string `json:"type"`
	} `json:"user"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// StatusCheck represents a GitHub status check
type StatusCheck struct {
	State       string `json:"state"` // pending, success, error, failure
	TargetURL   string `json:"target_url,omitempty"`
	Description string `json:"description,omitempty"`
	Context     string `json:"context"`
}