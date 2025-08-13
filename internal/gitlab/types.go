package gitlab

import (
	"time"

	"github.com/google/uuid"
)

// WebhookEvent represents a GitLab webhook event
type WebhookEvent struct {
	ID           uuid.UUID  `json:"id" db:"id"`
	EventType    string     `json:"event_type" db:"event_type"`
	Action       string     `json:"action" db:"action"`
	Repository   string     `json:"repository" db:"repository"`
	MergeRequest *int       `json:"merge_request,omitempty" db:"merge_request"`
	Branch       string     `json:"branch" db:"branch"`
	Commit       string     `json:"commit" db:"commit"`
	Payload      []byte     `json:"payload" db:"payload"`
	ProcessedAt  *time.Time `json:"processed_at,omitempty" db:"processed_at"`
	CreatedAt    time.Time  `json:"created_at" db:"created_at"`
}

// MergeRequestEvent represents a GitLab merge request webhook event
type MergeRequestEvent struct {
	ObjectKind       string `json:"object_kind"`
	EventType        string `json:"event_type"`
	User             User   `json:"user"`
	Project          Project `json:"project"`
	ObjectAttributes struct {
		ID           int    `json:"id"`
		IID          int    `json:"iid"`
		Title        string `json:"title"`
		Description  string `json:"description"`
		State        string `json:"state"`
		Action       string `json:"action"`
		SourceBranch string `json:"source_branch"`
		TargetBranch string `json:"target_branch"`
		LastCommit   struct {
			ID        string    `json:"id"`
			Message   string    `json:"message"`
			Timestamp time.Time `json:"timestamp"`
			URL       string    `json:"url"`
			Author    struct {
				Name  string `json:"name"`
				Email string `json:"email"`
			} `json:"author"`
		} `json:"last_commit"`
		URL         string    `json:"url"`
		Source      Project   `json:"source"`
		Target      Project   `json:"target"`
		CreatedAt   time.Time `json:"created_at"`
		UpdatedAt   time.Time `json:"updated_at"`
	} `json:"object_attributes"`
	Labels  []Label `json:"labels"`
	Changes struct {
		UpdatedAt struct {
			Previous time.Time `json:"previous"`
			Current  time.Time `json:"current"`
		} `json:"updated_at"`
	} `json:"changes"`
	Repository Repository `json:"repository"`
}

// PushEvent represents a GitLab push webhook event
type PushEvent struct {
	ObjectKind        string     `json:"object_kind"`
	EventName         string     `json:"event_name"`
	Before            string     `json:"before"`
	After             string     `json:"after"`
	Ref               string     `json:"ref"`
	CheckoutSHA       string     `json:"checkout_sha"`
	Message           string     `json:"message"`
	UserID            int        `json:"user_id"`
	UserName          string     `json:"user_name"`
	UserUsername      string     `json:"user_username"`
	UserEmail         string     `json:"user_email"`
	UserAvatar        string     `json:"user_avatar"`
	ProjectID         int        `json:"project_id"`
	Project           Project    `json:"project"`
	Commits           []Commit   `json:"commits"`
	TotalCommitsCount int        `json:"total_commits_count"`
	Repository        Repository `json:"repository"`
}

// PipelineEvent represents a GitLab pipeline webhook event
type PipelineEvent struct {
	ObjectKind       string `json:"object_kind"`
	ObjectAttributes struct {
		ID         int       `json:"id"`
		Ref        string    `json:"ref"`
		Tag        bool      `json:"tag"`
		SHA        string    `json:"sha"`
		BeforeSHA  string    `json:"before_sha"`
		Source     string    `json:"source"`
		Status     string    `json:"status"`
		Stages     []string  `json:"stages"`
		CreatedAt  time.Time `json:"created_at"`
		FinishedAt time.Time `json:"finished_at"`
		Duration   int       `json:"duration"`
		Variables  []struct {
			Key   string `json:"key"`
			Value string `json:"value"`
		} `json:"variables"`
	} `json:"object_attributes"`
	MergeRequest struct {
		ID           int    `json:"id"`
		IID          int    `json:"iid"`
		Title        string `json:"title"`
		SourceBranch string `json:"source_branch"`
		TargetBranch string `json:"target_branch"`
		State        string `json:"state"`
		URL          string `json:"url"`
	} `json:"merge_request"`
	User    User    `json:"user"`
	Project Project `json:"project"`
	Commit  struct {
		ID        string    `json:"id"`
		Message   string    `json:"message"`
		Timestamp time.Time `json:"timestamp"`
		URL       string    `json:"url"`
		Author    struct {
			Name  string `json:"name"`
			Email string `json:"email"`
		} `json:"author"`
	} `json:"commit"`
	Builds []struct {
		ID         int       `json:"id"`
		Stage      string    `json:"stage"`
		Name       string    `json:"name"`
		Status     string    `json:"status"`
		CreatedAt  time.Time `json:"created_at"`
		StartedAt  time.Time `json:"started_at"`
		FinishedAt time.Time `json:"finished_at"`
		When       string    `json:"when"`
		Manual     bool      `json:"manual"`
		AllowFailure bool    `json:"allow_failure"`
		User       User      `json:"user"`
		Runner     struct {
			ID          int    `json:"id"`
			Description string `json:"description"`
			Active      bool   `json:"active"`
			IsShared    bool   `json:"is_shared"`
		} `json:"runner"`
		ArtifactsFile struct {
			Filename string `json:"filename"`
			Size     int    `json:"size"`
		} `json:"artifacts_file"`
	} `json:"builds"`
}

// User represents a GitLab user
type User struct {
	ID        int    `json:"id"`
	Name      string `json:"name"`
	Username  string `json:"username"`
	State     string `json:"state"`
	AvatarURL string `json:"avatar_url"`
	WebURL    string `json:"web_url"`
}

// Project represents a GitLab project
type Project struct {
	ID                int    `json:"id"`
	Name              string `json:"name"`
	Description       string `json:"description"`
	WebURL            string `json:"web_url"`
	AvatarURL         string `json:"avatar_url"`
	GitSSHURL         string `json:"git_ssh_url"`
	GitHTTPURL        string `json:"git_http_url"`
	Namespace         string `json:"namespace"`
	VisibilityLevel   int    `json:"visibility_level"`
	PathWithNamespace string `json:"path_with_namespace"`
	DefaultBranch     string `json:"default_branch"`
	CIConfigPath      string `json:"ci_config_path"`
	Homepage          string `json:"homepage"`
	URL               string `json:"url"`
	SSHURL            string `json:"ssh_url"`
	HTTPURL           string `json:"http_url"`
}

// Repository represents a GitLab repository
type Repository struct {
	Name            string `json:"name"`
	URL             string `json:"url"`
	Description     string `json:"description"`
	Homepage        string `json:"homepage"`
	GitHTTPURL      string `json:"git_http_url"`
	GitSSHURL       string `json:"git_ssh_url"`
	VisibilityLevel int    `json:"visibility_level"`
}

// Label represents a GitLab label
type Label struct {
	ID          int    `json:"id"`
	Title       string `json:"title"`
	Color       string `json:"color"`
	ProjectID   int    `json:"project_id"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Template    bool   `json:"template"`
	Description string `json:"description"`
	Type        string `json:"type"`
	GroupID     int    `json:"group_id"`
}

// Commit represents a GitLab commit
type Commit struct {
	ID        string    `json:"id"`
	Message   string    `json:"message"`
	Timestamp time.Time `json:"timestamp"`
	URL       string    `json:"url"`
	Author    struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	} `json:"author"`
	Added    []string `json:"added"`
	Modified []string `json:"modified"`
	Removed  []string `json:"removed"`
}

// CommitStatus represents a GitLab commit status
type CommitStatus struct {
	State       string `json:"state"`       // pending, running, success, failed, canceled
	Description string `json:"description"`
	Name        string `json:"name"`
	TargetURL   string `json:"target_url"`
	Coverage    float64 `json:"coverage,omitempty"`
}

// MRComment represents a GitLab merge request comment
type MRComment struct {
	Body string `json:"body"`
}