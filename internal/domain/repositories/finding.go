package repositories

import (
	"context"

	"github.com/google/uuid"
	"github.com/NikhilSetiya/agentscan-security-scanner/internal/domain/entities"
)

// FindingRepository defines the interface for finding data operations
type FindingRepository interface {
	BaseRepository[*entities.Finding, uuid.UUID]
	
	// ListByScanJob retrieves findings for a specific scan job
	ListByScanJob(ctx context.Context, scanJobID uuid.UUID, filter Filter, pagination Pagination) ([]*entities.Finding, int64, error)
	
	// ListByRepository retrieves findings for a specific repository
	ListByRepository(ctx context.Context, repoID uuid.UUID, filter Filter, pagination Pagination) ([]*entities.Finding, int64, error)
	
	// ListBySeverity retrieves findings by severity level
	ListBySeverity(ctx context.Context, severity entities.FindingSeverity, filter Filter, pagination Pagination) ([]*entities.Finding, int64, error)
	
	// ListByStatus retrieves findings by status
	ListByStatus(ctx context.Context, status entities.FindingStatus, filter Filter, pagination Pagination) ([]*entities.Finding, int64, error)
	
	// UpdateStatus updates the status of a finding
	UpdateStatus(ctx context.Context, id uuid.UUID, status entities.FindingStatus) error
	
	// SetConsensusScore sets the consensus score for a finding
	SetConsensusScore(ctx context.Context, id uuid.UUID, score float64) error
	
	// AddFixSuggestion adds a fix suggestion to a finding
	AddFixSuggestion(ctx context.Context, id uuid.UUID, suggestion map[string]interface{}) error
	
	// GetStatsByScanJob retrieves finding statistics for a scan job
	GetStatsByScanJob(ctx context.Context, scanJobID uuid.UUID) (map[string]int64, error)
	
	// GetStatsByRepository retrieves finding statistics for a repository
	GetStatsByRepository(ctx context.Context, repoID uuid.UUID) (map[string]int64, error)
	
	// GetStatsBySeverity retrieves finding statistics by severity
	GetStatsBySeverity(ctx context.Context, filter Filter) (map[string]int64, error)
	
	// GetTrendData retrieves finding trend data over time
	GetTrendData(ctx context.Context, days int, filter Filter) ([]map[string]interface{}, error)
	
	// BulkUpdateStatus updates status for multiple findings
	BulkUpdateStatus(ctx context.Context, ids []uuid.UUID, status entities.FindingStatus) error
	
	// DeleteByScanJob deletes all findings for a scan job
	DeleteByScanJob(ctx context.Context, scanJobID uuid.UUID) error
	
	// GetDuplicates finds duplicate findings based on rule and file path
	GetDuplicates(ctx context.Context, ruleID, filePath string, lineNumber int) ([]*entities.Finding, error)
}