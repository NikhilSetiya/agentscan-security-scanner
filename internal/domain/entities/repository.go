package entities

import (
	"time"

	"github.com/google/uuid"
)

// Repository represents a code repository entity in the domain
type Repository struct {
	ID             uuid.UUID              `json:"id"`
	OrganizationID uuid.UUID              `json:"organization_id"`
	Name           string                 `json:"name" validate:"required,min=1,max=255"`
	URL            string                 `json:"url" validate:"required,url"`
	Provider       string                 `json:"provider" validate:"required"`
	ProviderID     string                 `json:"provider_id" validate:"required"`
	DefaultBranch  string                 `json:"default_branch" validate:"required"`
	Language       string                 `json:"language"`
	Description    string                 `json:"description"`
	Languages      []string               `json:"languages"`
	Settings       map[string]interface{} `json:"settings"`
	IsActive       bool                   `json:"is_active"`
	LastScanAt     *time.Time             `json:"last_scan_at,omitempty"`
	CreatedAt      time.Time              `json:"created_at"`
	UpdatedAt      time.Time              `json:"updated_at"`
}

// NewRepository creates a new repository entity
func NewRepository(orgID uuid.UUID, name, url, provider, providerID, defaultBranch string) *Repository {
	return &Repository{
		ID:             uuid.New(),
		OrganizationID: orgID,
		Name:           name,
		URL:            url,
		Provider:       provider,
		ProviderID:     providerID,
		DefaultBranch:  defaultBranch,
		Languages:      []string{},
		Settings:       make(map[string]interface{}),
		IsActive:       true,
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}
}

// UpdateDetails updates repository details
func (r *Repository) UpdateDetails(name, description, language string) {
	r.Name = name
	r.Description = description
	r.Language = language
	r.UpdatedAt = time.Now()
}

// SetDefaultBranch sets the default branch
func (r *Repository) SetDefaultBranch(branch string) {
	r.DefaultBranch = branch
	r.UpdatedAt = time.Now()
}

// AddLanguage adds a programming language to the repository
func (r *Repository) AddLanguage(language string) {
	for _, lang := range r.Languages {
		if lang == language {
			return // Already exists
		}
	}
	r.Languages = append(r.Languages, language)
	r.UpdatedAt = time.Now()
}

// SetLanguages sets the programming languages for the repository
func (r *Repository) SetLanguages(languages []string) {
	r.Languages = languages
	r.UpdatedAt = time.Now()
}

// UpdateLastScanTime updates the last scan timestamp
func (r *Repository) UpdateLastScanTime() {
	now := time.Now()
	r.LastScanAt = &now
	r.UpdatedAt = now
}

// Deactivate marks the repository as inactive
func (r *Repository) Deactivate() {
	r.IsActive = false
	r.UpdatedAt = time.Now()
}

// Activate marks the repository as active
func (r *Repository) Activate() {
	r.IsActive = true
	r.UpdatedAt = time.Now()
}

// UpdateSetting updates a repository setting
func (r *Repository) UpdateSetting(key string, value interface{}) {
	if r.Settings == nil {
		r.Settings = make(map[string]interface{})
	}
	r.Settings[key] = value
	r.UpdatedAt = time.Now()
}

// GetSetting retrieves a repository setting
func (r *Repository) GetSetting(key string) (interface{}, bool) {
	if r.Settings == nil {
		return nil, false
	}
	value, exists := r.Settings[key]
	return value, exists
}

// Validate validates the repository entity
func (r *Repository) Validate() error {
	if r.Name == "" {
		return NewValidationError("name is required")
	}
	if r.URL == "" {
		return NewValidationError("url is required")
	}
	if r.Provider == "" {
		return NewValidationError("provider is required")
	}
	if r.ProviderID == "" {
		return NewValidationError("provider_id is required")
	}
	if r.DefaultBranch == "" {
		return NewValidationError("default_branch is required")
	}
	return nil
}