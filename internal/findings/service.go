package findings

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// Service handles business logic for findings management
type Service struct {
	repo     *Repository
	exporter *ExportService
}

// NewService creates a new findings service
func NewService(db *sqlx.DB, exporter *ExportService) *Service {
	return &Service{
		repo:     NewRepository(db),
		exporter: exporter,
	}
}

// GetFinding retrieves a finding by ID
func (s *Service) GetFinding(ctx context.Context, id uuid.UUID) (*Finding, error) {
	return s.repo.GetByID(ctx, id)
}

// ListFindings retrieves findings with filtering and pagination
func (s *Service) ListFindings(ctx context.Context, filter FindingFilter, limit, offset int) ([]*Finding, error) {
	return s.repo.List(ctx, filter, limit, offset)
}

// UpdateFindingStatus updates the status of a finding
func (s *Service) UpdateFindingStatus(ctx context.Context, id uuid.UUID, userID uuid.UUID, status FindingStatus, comment *string) error {
	// Update the finding status
	update := FindingUpdate{
		Status: &status,
	}
	
	err := s.repo.Update(ctx, id, update)
	if err != nil {
		return fmt.Errorf("failed to update finding status: %w", err)
	}

	// Create user feedback for ML training
	feedback := &UserFeedback{
		FindingID: id,
		UserID:    userID,
		Action:    status,
		Comment:   comment,
	}

	err = s.repo.CreateUserFeedback(ctx, feedback)
	if err != nil {
		// Log error but don't fail the status update
		// In a real implementation, you'd use a proper logger
		fmt.Printf("Failed to create user feedback: %v\n", err)
	}

	return nil
}

// SuppressFinding creates a suppression rule for a finding
func (s *Service) SuppressFinding(ctx context.Context, findingID uuid.UUID, userID uuid.UUID, reason string, expiresAt *time.Time) error {
	// Get the finding to extract rule information
	finding, err := s.repo.GetByID(ctx, findingID)
	if err != nil {
		return fmt.Errorf("failed to get finding: %w", err)
	}

	// Create suppression rule
	suppression := &FindingSuppression{
		UserID:     userID,
		RuleID:     finding.RuleID,
		FilePath:   &finding.FilePath,
		LineNumber: finding.LineNumber,
		Reason:     reason,
		ExpiresAt:  expiresAt,
	}

	err = s.repo.CreateSuppression(ctx, suppression)
	if err != nil {
		return fmt.Errorf("failed to create suppression: %w", err)
	}

	// Update the finding status to ignored
	return s.UpdateFindingStatus(ctx, findingID, userID, FindingStatusIgnored, &reason)
}

// GetSuppressions retrieves suppression rules for a user
func (s *Service) GetSuppressions(ctx context.Context, userID uuid.UUID) ([]*FindingSuppression, error) {
	return s.repo.GetSuppressions(ctx, userID)
}

// DeleteSuppression removes a suppression rule
func (s *Service) DeleteSuppression(ctx context.Context, suppressionID uuid.UUID, userID uuid.UUID) error {
	return s.repo.DeleteSuppression(ctx, suppressionID, userID)
}

// GetFindingStats retrieves statistics about findings for a scan job
func (s *Service) GetFindingStats(ctx context.Context, scanJobID uuid.UUID) (*FindingStats, error) {
	return s.repo.GetStats(ctx, scanJobID)
}

// ExportFindings exports findings in the specified format
func (s *Service) ExportFindings(ctx context.Context, request ExportRequest) (*ExportResult, error) {
	// Get findings based on filter
	findings, err := s.repo.List(ctx, request.Filter, 0, 0) // No limit for export
	if err != nil {
		return nil, fmt.Errorf("failed to get findings for export: %w", err)
	}

	// Export using the export service
	return s.exporter.Export(ctx, findings, request.Format)
}

// GetUserFeedback retrieves user feedback for ML training
func (s *Service) GetUserFeedback(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*UserFeedback, error) {
	return s.repo.GetUserFeedback(ctx, userID, limit, offset)
}

// BulkUpdateFindings updates multiple findings at once
func (s *Service) BulkUpdateFindings(ctx context.Context, findingIDs []uuid.UUID, userID uuid.UUID, status FindingStatus, comment *string) error {
	for _, id := range findingIDs {
		err := s.UpdateFindingStatus(ctx, id, userID, status, comment)
		if err != nil {
			return fmt.Errorf("failed to update finding %s: %w", id, err)
		}
	}
	return nil
}

// IsSuppressed checks if a finding should be suppressed based on existing rules
func (s *Service) IsSuppressed(ctx context.Context, finding *Finding, userID uuid.UUID) (bool, error) {
	suppressions, err := s.repo.GetSuppressions(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, suppression := range suppressions {
		// Check if suppression matches the finding
		if suppression.RuleID == finding.RuleID {
			// Check file path match (if specified)
			if suppression.FilePath != nil && *suppression.FilePath != finding.FilePath {
				continue
			}

			// Check line number match (if specified)
			if suppression.LineNumber != nil && finding.LineNumber != nil && *suppression.LineNumber != *finding.LineNumber {
				continue
			}

			// Check if suppression has expired
			if suppression.ExpiresAt != nil && time.Now().After(*suppression.ExpiresAt) {
				continue
			}

			return true, nil
		}
	}

	return false, nil
}

// ApplySuppressions applies suppression rules to a list of findings
func (s *Service) ApplySuppressions(ctx context.Context, findings []*Finding, userID uuid.UUID) ([]*Finding, error) {
	var filteredFindings []*Finding

	for _, finding := range findings {
		suppressed, err := s.IsSuppressed(ctx, finding, userID)
		if err != nil {
			return nil, fmt.Errorf("failed to check suppression for finding %s: %w", finding.ID, err)
		}

		if !suppressed {
			filteredFindings = append(filteredFindings, finding)
		}
	}

	return filteredFindings, nil
}