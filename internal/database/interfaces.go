package database

import (
	"context"
	"time"

	"github.com/google/uuid"

	"github.com/NikhilSetiya/agentscan-security-scanner/pkg/types"
)

// Repository represents the database interface for the orchestrator
type Repository interface {
	// Health checks database connectivity
	Health(ctx context.Context) error

	// Scan job operations
	CreateScanJob(ctx context.Context, job *types.ScanJob) error
	GetScanJob(ctx context.Context, id uuid.UUID) (*types.ScanJob, error)
	UpdateScanJob(ctx context.Context, job *types.ScanJob) error
	DeleteScanJob(ctx context.Context, id uuid.UUID) error
	ListScanJobs(ctx context.Context, filter *ScanJobFilter, pagination *Pagination) ([]*types.ScanJob, int64, error)

	// Scan result operations
	CreateScanResult(ctx context.Context, result *types.ScanResult) error
	GetScanResult(ctx context.Context, id uuid.UUID) (*types.ScanResult, error)
	GetScanResults(ctx context.Context, scanJobID uuid.UUID) ([]*types.ScanResult, error)

	// Finding operations
	CreateFinding(ctx context.Context, finding *types.Finding) error
	GetFinding(ctx context.Context, id uuid.UUID) (*types.Finding, error)
	GetFindings(ctx context.Context, scanJobID uuid.UUID, filter *FindingFilter) ([]*types.Finding, error)
	UpdateFinding(ctx context.Context, finding *types.Finding) error

	// User feedback operations
	CreateUserFeedback(ctx context.Context, feedback *types.UserFeedback) error
	GetUserFeedback(ctx context.Context, findingID uuid.UUID) ([]*types.UserFeedback, error)

	// Repository operations
	CreateRepository(ctx context.Context, repo *types.Repository) error
	GetRepository(ctx context.Context, id uuid.UUID) (*types.Repository, error)
	UpdateRepository(ctx context.Context, repo *types.Repository) error
	ListRepositories(ctx context.Context, orgID uuid.UUID) ([]*types.Repository, error)

	// Organization operations
	CreateOrganization(ctx context.Context, org *types.Organization) error
	GetOrganization(ctx context.Context, id uuid.UUID) (*types.Organization, error)
	UpdateOrganization(ctx context.Context, org *types.Organization) error

	// User operations
	CreateUser(ctx context.Context, user *types.User) error
	GetUser(ctx context.Context, id uuid.UUID) (*types.User, error)
	GetUserByEmail(ctx context.Context, email string) (*types.User, error)
	UpdateUser(ctx context.Context, user *types.User) error
}

// ScanJobFilter represents filters for scan job queries
type ScanJobFilter struct {
	RepositoryID *uuid.UUID `json:"repository_id,omitempty"`
	UserID       *uuid.UUID `json:"user_id,omitempty"`
	Status       string     `json:"status,omitempty"`
	ScanType     string     `json:"scan_type,omitempty"`
	Since        time.Time  `json:"since,omitempty"`
	Until        time.Time  `json:"until,omitempty"`
}

// FindingFilter represents filters for finding queries
type FindingFilter struct {
	Severity string `json:"severity,omitempty"`
	Tool     string `json:"tool,omitempty"`
	Category string `json:"category,omitempty"`
	Status   string `json:"status,omitempty"`
	File     string `json:"file,omitempty"`
}

// Pagination represents pagination parameters
type Pagination struct {
	Page     int `json:"page"`
	PageSize int `json:"page_size"`
}