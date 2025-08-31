package database

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/agentscan/agentscan/internal/domain/repositories"
	"github.com/agentscan/agentscan/pkg/errors"
	"github.com/agentscan/agentscan/pkg/types"
)

// UserRepositoryImpl implements the UserRepository interface
type UserRepositoryImpl struct {
	*StandardizedRepositoryImpl[types.User, uuid.UUID]
}

// GetByEmail retrieves a user by email
func (u *UserRepositoryImpl) GetByEmail(ctx context.Context, email string) (*types.User, error) {
	return u.FindOneBy(ctx, "email", email)
}

// GetBySupabaseID retrieves a user by Supabase ID
func (u *UserRepositoryImpl) GetBySupabaseID(ctx context.Context, supabaseID string) (*types.User, error) {
	return u.FindOneBy(ctx, "supabase_id", supabaseID)
}

// GetByGitHubID retrieves a user by GitHub ID
func (u *UserRepositoryImpl) GetByGitHubID(ctx context.Context, githubID int) (*types.User, error) {
	return u.FindOneBy(ctx, "github_id", githubID)
}

// GetByGitLabID retrieves a user by GitLab ID
func (u *UserRepositoryImpl) GetByGitLabID(ctx context.Context, gitlabID int) (*types.User, error) {
	return u.FindOneBy(ctx, "gitlab_id", gitlabID)
}

// UpdateLastLoginAt updates the last login timestamp
func (u *UserRepositoryImpl) UpdateLastLoginAt(ctx context.Context, id uuid.UUID, loginTime time.Time) error {
	updates := map[string]interface{}{
		"last_login_at": loginTime,
	}
	return u.Update(ctx, id, updates)
}

// GetActiveUsers retrieves all active users
func (u *UserRepositoryImpl) GetActiveUsers(ctx context.Context) ([]*types.User, error) {
	filters := map[string]interface{}{
		"is_active": true,
	}
	users, _, err := u.List(ctx, filters, 0, 0)
	return users, err
}

// DeactivateUser deactivates a user
func (u *UserRepositoryImpl) DeactivateUser(ctx context.Context, id uuid.UUID) error {
	updates := map[string]interface{}{
		"is_active": false,
	}
	return u.Update(ctx, id, updates)
}

// ActivateUser activates a user
func (u *UserRepositoryImpl) ActivateUser(ctx context.Context, id uuid.UUID) error {
	updates := map[string]interface{}{
		"is_active": true,
	}
	return u.Update(ctx, id, updates)
}

// SearchUsers searches users by query
func (u *UserRepositoryImpl) SearchUsers(ctx context.Context, query string, limit, offset int) ([]*types.User, int, error) {
	filters := map[string]interface{}{
		"name":  "%" + query + "%",
		"email": "%" + query + "%",
	}
	return u.List(ctx, filters, limit, offset)
}

// OrganizationRepositoryImpl implements the OrganizationRepository interface
type OrganizationRepositoryImpl struct {
	*StandardizedRepositoryImpl[types.Organization, uuid.UUID]
}

// GetByName retrieves an organization by name
func (o *OrganizationRepositoryImpl) GetByName(ctx context.Context, name string) (*types.Organization, error) {
	return o.FindOneBy(ctx, "name", name)
}

// GetBySlug retrieves an organization by slug
func (o *OrganizationRepositoryImpl) GetBySlug(ctx context.Context, slug string) (*types.Organization, error) {
	return o.FindOneBy(ctx, "slug", slug)
}

// GetUserOrganizations retrieves organizations for a user
func (o *OrganizationRepositoryImpl) GetUserOrganizations(ctx context.Context, userID uuid.UUID) ([]*types.Organization, error) {
	query := `
		SELECT o.* 
		FROM organizations o
		INNER JOIN organization_members om ON o.id = om.organization_id
		WHERE om.user_id = $1 AND om.is_active = true
		ORDER BY o.name
	`
	
	var orgs []*types.Organization
	err := o.GetDB().SelectContext(ctx, &orgs, query, userID)
	if err != nil {
		return nil, errors.NewDatabaseError("organization", "get_user_organizations").WithCause(err)
	}
	
	return orgs, nil
}

// AddUserToOrganization adds a user to an organization
func (o *OrganizationRepositoryImpl) AddUserToOrganization(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	query := `
		INSERT INTO organization_members (id, organization_id, user_id, role, created_at)
		VALUES ($1, $2, $3, $4, NOW())
		ON CONFLICT (organization_id, user_id) 
		DO UPDATE SET role = $4, updated_at = NOW()
	`
	
	_, err := o.GetDB().ExecContext(ctx, query, uuid.New(), orgID, userID, role)
	if err != nil {
		return errors.NewDatabaseError("organization", "add_user").WithCause(err)
	}
	
	return nil
}

// RemoveUserFromOrganization removes a user from an organization
func (o *OrganizationRepositoryImpl) RemoveUserFromOrganization(ctx context.Context, orgID, userID uuid.UUID) error {
	query := "DELETE FROM organization_members WHERE organization_id = $1 AND user_id = $2"
	
	_, err := o.GetDB().ExecContext(ctx, query, orgID, userID)
	if err != nil {
		return errors.NewDatabaseError("organization", "remove_user").WithCause(err)
	}
	
	return nil
}

// GetOrganizationUsers retrieves users in an organization
func (o *OrganizationRepositoryImpl) GetOrganizationUsers(ctx context.Context, orgID uuid.UUID) ([]*types.User, error) {
	query := `
		SELECT u.* 
		FROM users u
		INNER JOIN organization_members om ON u.id = om.user_id
		WHERE om.organization_id = $1
		ORDER BY u.email
	`
	
	var users []*types.User
	err := o.GetDB().SelectContext(ctx, &users, query, orgID)
	if err != nil {
		return nil, errors.NewDatabaseError("organization", "get_users").WithCause(err)
	}
	
	return users, nil
}

// UpdateUserRole updates a user's role in an organization
func (o *OrganizationRepositoryImpl) UpdateUserRole(ctx context.Context, orgID, userID uuid.UUID, role string) error {
	query := `
		UPDATE organization_members 
		SET role = $3, updated_at = NOW()
		WHERE organization_id = $1 AND user_id = $2
	`
	
	result, err := o.GetDB().ExecContext(ctx, query, orgID, userID, role)
	if err != nil {
		return errors.NewDatabaseError("organization", "update_user_role").WithCause(err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewDatabaseError("organization", "update_user_role").WithCause(err)
	}
	
	if rowsAffected == 0 {
		return repositories.NewNotFoundError("organization_member", "")
	}
	
	return nil
}

// GetUserRole gets a user's role in an organization
func (o *OrganizationRepositoryImpl) GetUserRole(ctx context.Context, orgID, userID uuid.UUID) (string, error) {
	query := "SELECT role FROM organization_members WHERE organization_id = $1 AND user_id = $2"
	
	var role string
	err := o.GetDB().GetContext(ctx, &role, query, orgID, userID)
	if err != nil {
		return "", errors.NewDatabaseError("organization", "get_user_role").WithCause(err)
	}
	
	return role, nil
}

// IsUserMember checks if a user is a member of an organization
func (o *OrganizationRepositoryImpl) IsUserMember(ctx context.Context, orgID, userID uuid.UUID) (bool, error) {
	query := "SELECT EXISTS(SELECT 1 FROM organization_members WHERE organization_id = $1 AND user_id = $2)"
	
	var exists bool
	err := o.GetDB().GetContext(ctx, &exists, query, orgID, userID)
	if err != nil {
		return false, errors.NewDatabaseError("organization", "is_user_member").WithCause(err)
	}
	
	return exists, nil
}

// RepositoryRepositoryImpl implements the RepositoryRepository interface
type RepositoryRepositoryImpl struct {
	*StandardizedRepositoryImpl[types.Repository, uuid.UUID]
}

// GetByURL retrieves a repository by URL
func (r *RepositoryRepositoryImpl) GetByURL(ctx context.Context, url string) (*types.Repository, error) {
	return r.FindOneBy(ctx, "url", url)
}

// GetByProviderID retrieves a repository by provider and provider ID
func (r *RepositoryRepositoryImpl) GetByProviderID(ctx context.Context, provider, providerID string) (*types.Repository, error) {
	filters := map[string]interface{}{
		"provider":    provider,
		"provider_id": providerID,
	}
	
	repos, _, err := r.List(ctx, filters, 1, 0)
	if err != nil {
		return nil, err
	}
	
	if len(repos) == 0 {
		return nil, repositories.NewNotFoundError("repository", "")
	}
	
	return repos[0], nil
}

// GetByOrganization retrieves repositories by organization
func (r *RepositoryRepositoryImpl) GetByOrganization(ctx context.Context, orgID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]*types.Repository, int, error) {
	if filters == nil {
		filters = make(map[string]interface{})
	}
	filters["organization_id"] = orgID
	
	return r.List(ctx, filters, limit, offset)
}

// UpdateLastScanAt updates the last scan timestamp
func (r *RepositoryRepositoryImpl) UpdateLastScanAt(ctx context.Context, id uuid.UUID, scanTime time.Time) error {
	updates := map[string]interface{}{
		"last_scan_at": scanTime,
	}
	return r.Update(ctx, id, updates)
}

// GetActiveRepositories retrieves all active repositories
func (r *RepositoryRepositoryImpl) GetActiveRepositories(ctx context.Context) ([]*types.Repository, error) {
	filters := map[string]interface{}{
		"is_active": true,
	}
	
	repos, _, err := r.List(ctx, filters, 0, 0)
	return repos, err
}

// GetRepositoriesByLanguage retrieves repositories by programming language
func (r *RepositoryRepositoryImpl) GetRepositoriesByLanguage(ctx context.Context, language string) ([]*types.Repository, error) {
	// Use JSON query for languages array
	query := "SELECT * FROM repositories WHERE languages ? $1"
	
	var repos []*types.Repository
	err := r.GetDB().SelectContext(ctx, &repos, query, language)
	if err != nil {
		return nil, errors.NewDatabaseError("repository", "get_by_language").WithCause(err)
	}
	
	return repos, nil
}

// GetRepositoriesByProvider retrieves repositories by provider
func (r *RepositoryRepositoryImpl) GetRepositoriesByProvider(ctx context.Context, provider string) ([]*types.Repository, error) {
	return r.FindBy(ctx, "provider", provider)
}

// SearchRepositories searches repositories by query
func (r *RepositoryRepositoryImpl) SearchRepositories(ctx context.Context, query string, orgID *uuid.UUID, limit, offset int) ([]*types.Repository, int, error) {
	filters := map[string]interface{}{
		"name": "%" + query + "%",
	}
	
	if orgID != nil {
		filters["organization_id"] = *orgID
	}
	
	return r.List(ctx, filters, limit, offset)
}

// GetRepositoryStatistics retrieves repository statistics
func (r *RepositoryRepositoryImpl) GetRepositoryStatistics(ctx context.Context, id uuid.UUID) (*repositories.RepositoryStatistics, error) {
	// This would use the optimized queries or materialized views
	// For now, return a placeholder
	return &repositories.RepositoryStatistics{
		TotalScans:     0,
		CompletedScans: 0,
		FailedScans:    0,
		TotalFindings:  0,
		HealthScore:    100.0,
	}, nil
}

// ScanJobRepositoryImpl implements the ScanJobRepository interface
type ScanJobRepositoryImpl struct {
	*StandardizedRepositoryImpl[types.ScanJob, uuid.UUID]
}

// GetByRepository retrieves scan jobs by repository
func (s *ScanJobRepositoryImpl) GetByRepository(ctx context.Context, repoID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]*types.ScanJob, int, error) {
	if filters == nil {
		filters = make(map[string]interface{})
	}
	filters["repository_id"] = repoID
	
	return s.List(ctx, filters, limit, offset)
}

// GetByUser retrieves scan jobs by user
func (s *ScanJobRepositoryImpl) GetByUser(ctx context.Context, userID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]*types.ScanJob, int, error) {
	if filters == nil {
		filters = make(map[string]interface{})
	}
	filters["user_id"] = userID
	
	return s.List(ctx, filters, limit, offset)
}

// GetByStatus retrieves scan jobs by status
func (s *ScanJobRepositoryImpl) GetByStatus(ctx context.Context, status string) ([]*types.ScanJob, error) {
	return s.FindBy(ctx, "status", status)
}

// GetByStatusAndPriority retrieves scan jobs by status and minimum priority
func (s *ScanJobRepositoryImpl) GetByStatusAndPriority(ctx context.Context, status string, minPriority int) ([]*types.ScanJob, error) {
	query := "SELECT * FROM scan_jobs WHERE status = $1 AND priority >= $2 ORDER BY priority DESC, created_at ASC"
	
	var jobs []*types.ScanJob
	err := s.GetDB().SelectContext(ctx, &jobs, query, status, minPriority)
	if err != nil {
		return nil, errors.NewDatabaseError("scan_job", "get_by_status_priority").WithCause(err)
	}
	
	return jobs, nil
}

// UpdateStatus updates the scan job status
func (s *ScanJobRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status string, message *string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	
	if message != nil {
		updates["error_message"] = *message
	}
	
	return s.Update(ctx, id, updates)
}

// GetRunningJobs retrieves all running scan jobs
func (s *ScanJobRepositoryImpl) GetRunningJobs(ctx context.Context) ([]*types.ScanJob, error) {
	return s.GetByStatus(ctx, "running")
}

// GetQueuedJobs retrieves queued scan jobs
func (s *ScanJobRepositoryImpl) GetQueuedJobs(ctx context.Context, limit int) ([]*types.ScanJob, error) {
	filters := map[string]interface{}{
		"status": "queued",
	}
	
	jobs, _, err := s.ListWithSort(ctx, filters, "priority", "DESC", limit, 0)
	return jobs, err
}

// GetJobsInQueue retrieves jobs in queue with priority filtering
func (s *ScanJobRepositoryImpl) GetJobsInQueue(ctx context.Context, maxPriority int, limit int) ([]*types.ScanJob, error) {
	query := `
		SELECT * FROM scan_jobs 
		WHERE status = 'queued' AND priority <= $1 
		ORDER BY priority DESC, created_at ASC 
		LIMIT $2
	`
	
	var jobs []*types.ScanJob
	err := s.GetDB().SelectContext(ctx, &jobs, query, maxPriority, limit)
	if err != nil {
		return nil, errors.NewDatabaseError("scan_job", "get_jobs_in_queue").WithCause(err)
	}
	
	return jobs, nil
}

// MarkAsStarted marks a scan job as started
func (s *ScanJobRepositoryImpl) MarkAsStarted(ctx context.Context, id uuid.UUID) error {
	updates := map[string]interface{}{
		"status":     "running",
		"started_at": time.Now(),
	}
	return s.Update(ctx, id, updates)
}

// MarkAsCompleted marks a scan job as completed
func (s *ScanJobRepositoryImpl) MarkAsCompleted(ctx context.Context, id uuid.UUID) error {
	updates := map[string]interface{}{
		"status":       "completed",
		"completed_at": time.Now(),
	}
	return s.Update(ctx, id, updates)
}

// MarkAsFailed marks a scan job as failed
func (s *ScanJobRepositoryImpl) MarkAsFailed(ctx context.Context, id uuid.UUID, errorMessage string) error {
	updates := map[string]interface{}{
		"status":        "failed",
		"completed_at":  time.Now(),
		"error_message": errorMessage,
	}
	return s.Update(ctx, id, updates)
}

// GetJobStatistics retrieves scan job statistics
func (s *ScanJobRepositoryImpl) GetJobStatistics(ctx context.Context, filters map[string]interface{}) (*repositories.ScanJobStatistics, error) {
	// This would use optimized queries or materialized views
	// For now, return a placeholder
	return &repositories.ScanJobStatistics{
		TotalJobs:   0,
		SuccessRate: 100.0,
		QueueLength: 0,
	}, nil
}

// GetRecentJobs retrieves recent scan jobs
func (s *ScanJobRepositoryImpl) GetRecentJobs(ctx context.Context, limit int) ([]*types.ScanJob, error) {
	jobs, _, err := s.ListWithSort(ctx, nil, "created_at", "DESC", limit, 0)
	return jobs, err
}

// CleanupOldJobs removes old scan jobs
func (s *ScanJobRepositoryImpl) CleanupOldJobs(ctx context.Context, olderThan time.Time) (int, error) {
	query := "DELETE FROM scan_jobs WHERE created_at < $1"
	
	result, err := s.GetDB().ExecContext(ctx, query, olderThan)
	if err != nil {
		return 0, errors.NewDatabaseError("scan_job", "cleanup").WithCause(err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return 0, errors.NewDatabaseError("scan_job", "cleanup").WithCause(err)
	}
	
	return int(rowsAffected), nil
}

// FindingRepositoryImpl implements the FindingRepository interface
type FindingRepositoryImpl struct {
	*StandardizedRepositoryImpl[types.Finding, uuid.UUID]
}

// GetByScanJob retrieves findings by scan job
func (f *FindingRepositoryImpl) GetByScanJob(ctx context.Context, scanJobID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]*types.Finding, int, error) {
	if filters == nil {
		filters = make(map[string]interface{})
	}
	filters["scan_job_id"] = scanJobID
	
	return f.List(ctx, filters, limit, offset)
}

// GetBySeverity retrieves findings by severity
func (f *FindingRepositoryImpl) GetBySeverity(ctx context.Context, severity string, filters map[string]interface{}, limit, offset int) ([]*types.Finding, int, error) {
	if filters == nil {
		filters = make(map[string]interface{})
	}
	filters["severity"] = severity
	
	return f.List(ctx, filters, limit, offset)
}

// GetByAgent retrieves findings by agent name
func (f *FindingRepositoryImpl) GetByAgent(ctx context.Context, agentName string, filters map[string]interface{}, limit, offset int) ([]*types.Finding, int, error) {
	if filters == nil {
		filters = make(map[string]interface{})
	}
	filters["tool"] = agentName
	
	return f.List(ctx, filters, limit, offset)
}

// GetByRepository retrieves findings by repository
func (f *FindingRepositoryImpl) GetByRepository(ctx context.Context, repoID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]*types.Finding, int, error) {
	query := `
		SELECT f.* FROM findings f
		INNER JOIN scan_jobs sj ON f.scan_job_id = sj.id
		WHERE sj.repository_id = $1
	`
	
	// Add additional filters if provided
	args := []interface{}{repoID}
	argIndex := 2
	
	if filters != nil {
		for field, value := range filters {
			query += fmt.Sprintf(" AND f.%s = $%d", field, argIndex)
			args = append(args, value)
			argIndex++
		}
	}
	
	query += " ORDER BY f.created_at DESC"
	
	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++
		
		if offset > 0 {
			query += fmt.Sprintf(" OFFSET $%d", argIndex)
			args = append(args, offset)
		}
	}
	
	var findings []*types.Finding
	err := f.GetDB().SelectContext(ctx, &findings, query, args...)
	if err != nil {
		return nil, 0, errors.NewDatabaseError("finding", "get_by_repository").WithCause(err)
	}
	
	// Get total count
	countQuery := `
		SELECT COUNT(*) FROM findings f
		INNER JOIN scan_jobs sj ON f.scan_job_id = sj.id
		WHERE sj.repository_id = $1
	`
	
	var total int
	err = f.GetDB().GetContext(ctx, &total, countQuery, repoID)
	if err != nil {
		return nil, 0, errors.NewDatabaseError("finding", "get_by_repository_count").WithCause(err)
	}
	
	return findings, total, nil
}

// Additional methods would be implemented similarly...
// For brevity, I'll include placeholders for the remaining methods

// UpdateStatus updates the finding status
func (f *FindingRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	return f.Update(ctx, id, updates)
}

// SuppressFinding suppresses a finding with reason
func (f *FindingRepositoryImpl) SuppressFinding(ctx context.Context, id uuid.UUID, reason string, userID uuid.UUID) error {
	updates := map[string]interface{}{
		"status":             "ignored",
		"suppression_reason": reason,
		"suppressed_by":      userID,
		"suppressed_at":      time.Now(),
	}
	return f.Update(ctx, id, updates)
}

// UnsuppressFinding unsuppresses a finding
func (f *FindingRepositoryImpl) UnsuppressFinding(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	updates := map[string]interface{}{
		"status":             "open",
		"suppression_reason": nil,
		"suppressed_by":      nil,
		"suppressed_at":      nil,
		"unsuppressed_by":    userID,
		"unsuppressed_at":    time.Now(),
	}
	return f.Update(ctx, id, updates)
}

// Placeholder implementations for remaining methods
func (f *FindingRepositoryImpl) GetStatistics(ctx context.Context, filters map[string]interface{}) (*repositories.FindingStatistics, error) {
	return &repositories.FindingStatistics{}, nil
}

func (f *FindingRepositoryImpl) BulkUpdateStatus(ctx context.Context, ids []uuid.UUID, status string, userID uuid.UUID) error {
	return nil
}

func (f *FindingRepositoryImpl) GetSimilarFindings(ctx context.Context, findingID uuid.UUID, threshold float64) ([]*types.Finding, error) {
	return nil, nil
}

func (f *FindingRepositoryImpl) GetFindingTrends(ctx context.Context, days int, filters map[string]interface{}) ([]*repositories.FindingTrend, error) {
	return nil, nil
}

func (f *FindingRepositoryImpl) SearchFindings(ctx context.Context, query string, filters map[string]interface{}, limit, offset int) ([]*types.Finding, int, error) {
	return nil, 0, nil
}

// ScanResultRepositoryImpl implements the ScanResultRepository interface
type ScanResultRepositoryImpl struct {
	*StandardizedRepositoryImpl[types.ScanResult, uuid.UUID]
}

// Placeholder implementations
func (s *ScanResultRepositoryImpl) GetByScanJob(ctx context.Context, scanJobID uuid.UUID) ([]*types.ScanResult, error) {
	return s.FindBy(ctx, "scan_job_id", scanJobID)
}

func (s *ScanResultRepositoryImpl) GetByAgent(ctx context.Context, agentName string, limit, offset int) ([]*types.ScanResult, int, error) {
	filters := map[string]interface{}{
		"agent_name": agentName,
	}
	return s.List(ctx, filters, limit, offset)
}

func (s *ScanResultRepositoryImpl) GetByStatus(ctx context.Context, status string) ([]*types.ScanResult, error) {
	return s.FindBy(ctx, "status", status)
}

func (s *ScanResultRepositoryImpl) GetAgentStatistics(ctx context.Context, agentName string, days int) (*repositories.AgentStatistics, error) {
	return &repositories.AgentStatistics{}, nil
}

func (s *ScanResultRepositoryImpl) GetPerformanceMetrics(ctx context.Context, filters map[string]interface{}) (*repositories.PerformanceMetrics, error) {
	return &repositories.PerformanceMetrics{}, nil
}

func (s *ScanResultRepositoryImpl) CleanupOldResults(ctx context.Context, olderThan time.Time) (int, error) {
	return 0, nil
}

// UserFeedbackRepositoryImpl implements the UserFeedbackRepository interface
type UserFeedbackRepositoryImpl struct {
	*StandardizedRepositoryImpl[types.UserFeedback, uuid.UUID]
}

// Placeholder implementations
func (u *UserFeedbackRepositoryImpl) GetByFinding(ctx context.Context, findingID uuid.UUID) ([]*types.UserFeedback, error) {
	return u.FindBy(ctx, "finding_id", findingID)
}

func (u *UserFeedbackRepositoryImpl) GetByUser(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*types.UserFeedback, int, error) {
	filters := map[string]interface{}{
		"user_id": userID,
	}
	return u.List(ctx, filters, limit, offset)
}

func (u *UserFeedbackRepositoryImpl) GetByAction(ctx context.Context, action string, limit, offset int) ([]*types.UserFeedback, int, error) {
	filters := map[string]interface{}{
		"action": action,
	}
	return u.List(ctx, filters, limit, offset)
}

func (u *UserFeedbackRepositoryImpl) GetFeedbackStatistics(ctx context.Context, filters map[string]interface{}) (*repositories.FeedbackStatistics, error) {
	return &repositories.FeedbackStatistics{}, nil
}

func (u *UserFeedbackRepositoryImpl) GetUserFeedbackSummary(ctx context.Context, userID uuid.UUID) (*repositories.UserFeedbackSummary, error) {
	return &repositories.UserFeedbackSummary{}, nil
}

// SoftDeleteRepositoryImpl provides soft delete functionality
type SoftDeleteRepositoryImpl[T any, ID comparable] struct {
	*StandardizedRepositoryImpl[T, ID]
}

// SoftDelete marks an entity as deleted without removing it
func (s *SoftDeleteRepositoryImpl[T, ID]) SoftDelete(ctx context.Context, id ID) error {
	updates := map[string]interface{}{
		"deleted_at": time.Now(),
		"is_deleted": true,
	}
	return s.Update(ctx, id, updates)
}

// SoftDeleteBatch marks multiple entities as deleted
func (s *SoftDeleteRepositoryImpl[T, ID]) SoftDeleteBatch(ctx context.Context, ids []ID) error {
	updates := make([]repositories.BatchUpdate[ID], len(ids))
	for i, id := range ids {
		updates[i] = repositories.BatchUpdate[ID]{
			ID: id,
			Updates: map[string]interface{}{
				"deleted_at": time.Now(),
				"is_deleted": true,
			},
		}
	}
	return s.UpdateBatch(ctx, updates)
}

// Restore restores a soft-deleted entity
func (s *SoftDeleteRepositoryImpl[T, ID]) Restore(ctx context.Context, id ID) error {
	updates := map[string]interface{}{
		"deleted_at": nil,
		"is_deleted": false,
	}
	return s.Update(ctx, id, updates)
}

// RestoreBatch restores multiple soft-deleted entities
func (s *SoftDeleteRepositoryImpl[T, ID]) RestoreBatch(ctx context.Context, ids []ID) error {
	updates := make([]repositories.BatchUpdate[ID], len(ids))
	for i, id := range ids {
		updates[i] = repositories.BatchUpdate[ID]{
			ID: id,
			Updates: map[string]interface{}{
				"deleted_at": nil,
				"is_deleted": false,
			},
		}
	}
	return s.UpdateBatch(ctx, updates)
}

// ListIncludingDeleted lists entities including soft-deleted ones
func (s *SoftDeleteRepositoryImpl[T, ID]) ListIncludingDeleted(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*T, int, error) {
	// Remove any is_deleted filter to include deleted entities
	if filters != nil {
		delete(filters, "is_deleted")
	}
	return s.List(ctx, filters, limit, offset)
}

// ListOnlyDeleted lists only soft-deleted entities
func (s *SoftDeleteRepositoryImpl[T, ID]) ListOnlyDeleted(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*T, int, error) {
	if filters == nil {
		filters = make(map[string]interface{})
	}
	filters["is_deleted"] = true
	return s.List(ctx, filters, limit, offset)
}

// HardDelete permanently deletes an entity
func (s *SoftDeleteRepositoryImpl[T, ID]) HardDelete(ctx context.Context, id ID) error {
	return s.StandardizedRepositoryImpl.Delete(ctx, id)
}

// HardDeleteBatch permanently deletes multiple entities
func (s *SoftDeleteRepositoryImpl[T, ID]) HardDeleteBatch(ctx context.Context, ids []ID) error {
	return s.StandardizedRepositoryImpl.DeleteBatch(ctx, ids)
}