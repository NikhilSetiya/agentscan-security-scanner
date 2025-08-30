package database

import (
	"context"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/agentscan/agentscan/internal/domain/repositories"
	"github.com/agentscan/agentscan/pkg/errors"
	"github.com/agentscan/agentscan/pkg/types"
)

// FindingRepositoryImpl implements the FindingRepository interface
type FindingRepositoryImpl struct {
	*BaseRepositoryImpl[types.Finding, uuid.UUID]
}

// NewFindingRepository creates a new finding repository
func NewFindingRepository(db *sqlx.DB) repositories.FindingRepository {
	return &FindingRepositoryImpl{
		BaseRepositoryImpl: NewBaseRepository[types.Finding, uuid.UUID](db, "findings"),
	}
}

// GetByScanJob retrieves findings by scan job
func (f *FindingRepositoryImpl) GetByScanJob(ctx context.Context, scanJobID uuid.UUID, filters map[string]interface{}, limit, offset int) ([]*types.Finding, int, error) {
	filters["scan_job_id"] = scanJobID
	return f.List(ctx, filters, limit, offset)
}

// GetBySeverity retrieves findings by severity
func (f *FindingRepositoryImpl) GetBySeverity(ctx context.Context, severity string, filters map[string]interface{}, limit, offset int) ([]*types.Finding, int, error) {
	filters["severity"] = severity
	return f.List(ctx, filters, limit, offset)
}

// GetByAgent retrieves findings by agent name
func (f *FindingRepositoryImpl) GetByAgent(ctx context.Context, agentName string, filters map[string]interface{}, limit, offset int) ([]*types.Finding, int, error) {
	filters["agent_name"] = agentName
	return f.List(ctx, filters, limit, offset)
}

// UpdateStatus updates the finding status
func (f *FindingRepositoryImpl) UpdateStatus(ctx context.Context, id uuid.UUID, status string) error {
	updates := map[string]interface{}{
		"status": status,
	}
	return f.Update(ctx, id, updates)
}

// SuppressFinding suppresses a finding with reason
func (f *FindingRepositoryImpl) SuppressFinding(ctx context.Context, id uuid.UUID, reason string) error {
	updates := map[string]interface{}{
		"status":           "suppressed",
		"suppression_reason": reason,
	}
	return f.Update(ctx, id, updates)
}

// GetStatistics retrieves finding statistics
func (f *FindingRepositoryImpl) GetStatistics(ctx context.Context, filters map[string]interface{}) (*repositories.FindingStatistics, error) {
	// Build base query with filters
	whereClauses := f.buildWhereClauses(filters)
	
	baseQuery := "SELECT COUNT(*) as total, severity, status, agent_name FROM findings"
	
	var conditions []string
	var args []interface{}
	argIndex := 1
	
	for _, whereClause := range whereClauses {
		condition := whereClause.Condition
		for i, arg := range whereClause.Args {
			placeholder := fmt.Sprintf("$%d", i+1)
			actualPlaceholder := fmt.Sprintf("$%d", argIndex)
			condition = strings.Replace(condition, placeholder, actualPlaceholder, 1)
			args = append(args, arg)
			argIndex++
		}
		conditions = append(conditions, condition)
	}
	
	if len(conditions) > 0 {
		baseQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	// Get total count
	var total int
	countQuery := strings.Replace(baseQuery, "COUNT(*) as total, severity, status, agent_name", "COUNT(*)", 1)
	countQuery = strings.Replace(countQuery, " GROUP BY severity, status, agent_name", "", 1)
	
	err := f.GetDB().GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, errors.NewDatabaseError("statistics", "failed to get total count").WithCause(err)
	}
	
	// Get statistics by severity
	severityQuery := baseQuery + " GROUP BY severity"
	var severityStats []struct {
		Severity string `db:"severity"`
		Count    int    `db:"total"`
	}
	
	err = f.GetDB().SelectContext(ctx, &severityStats, severityQuery, args...)
	if err != nil {
		return nil, errors.NewDatabaseError("statistics", "failed to get severity statistics").WithCause(err)
	}
	
	bySeverity := make(map[string]int)
	for _, stat := range severityStats {
		bySeverity[stat.Severity] = stat.Count
	}
	
	// Get statistics by status
	statusQuery := strings.Replace(baseQuery, "severity", "status", 1) + " GROUP BY status"
	var statusStats []struct {
		Status string `db:"status"`
		Count  int    `db:"total"`
	}
	
	err = f.GetDB().SelectContext(ctx, &statusStats, statusQuery, args...)
	if err != nil {
		return nil, errors.NewDatabaseError("statistics", "failed to get status statistics").WithCause(err)
	}
	
	byStatus := make(map[string]int)
	for _, stat := range statusStats {
		byStatus[stat.Status] = stat.Count
	}
	
	// Get statistics by agent
	agentQuery := strings.Replace(baseQuery, "severity", "agent_name", 1) + " GROUP BY agent_name"
	var agentStats []struct {
		AgentName string `db:"agent_name"`
		Count     int    `db:"total"`
	}
	
	err = f.GetDB().SelectContext(ctx, &agentStats, agentQuery, args...)
	if err != nil {
		return nil, errors.NewDatabaseError("statistics", "failed to get agent statistics").WithCause(err)
	}
	
	byAgent := make(map[string]int)
	for _, stat := range agentStats {
		byAgent[stat.AgentName] = stat.Count
	}
	
	// Get trends (last 30 days)
	trendsQuery := `
		SELECT DATE(created_at) as date, COUNT(*) as count 
		FROM findings 
		WHERE created_at >= NOW() - INTERVAL '30 days'
	`
	
	if len(conditions) > 0 {
		trendsQuery += " AND " + strings.Join(conditions, " AND ")
	}
	
	trendsQuery += " GROUP BY DATE(created_at) ORDER BY date"
	
	var trendsData []repositories.TrendData
	err = f.GetDB().SelectContext(ctx, &trendsData, trendsQuery, args...)
	if err != nil {
		return nil, errors.NewDatabaseError("statistics", "failed to get trends").WithCause(err)
	}
	
	// Get top files with most findings
	topFilesQuery := `
		SELECT file_path, COUNT(*) as count 
		FROM findings 
	`
	
	if len(conditions) > 0 {
		topFilesQuery += " WHERE " + strings.Join(conditions, " AND ")
	}
	
	topFilesQuery += " GROUP BY file_path ORDER BY count DESC LIMIT 10"
	
	var topFiles []repositories.FileStatistic
	err = f.GetDB().SelectContext(ctx, &topFiles, topFilesQuery, args...)
	if err != nil {
		return nil, errors.NewDatabaseError("statistics", "failed to get top files").WithCause(err)
	}
	
	return &repositories.FindingStatistics{
		Total:      total,
		BySeverity: bySeverity,
		ByStatus:   byStatus,
		ByAgent:    byAgent,
		Trends:     trendsData,
		TopFiles:   topFiles,
	}, nil
}

// BulkUpdateStatus updates status for multiple findings
func (f *FindingRepositoryImpl) BulkUpdateStatus(ctx context.Context, ids []uuid.UUID, status string) error {
	if len(ids) == 0 {
		return errors.NewValidationError("no finding IDs provided")
	}
	
	// Build placeholders for IN clause
	placeholders := make([]string, len(ids))
	args := make([]interface{}, len(ids)+1)
	
	for i, id := range ids {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = id
	}
	args[len(ids)] = status
	
	query := fmt.Sprintf(`
		UPDATE findings 
		SET status = $%d, updated_at = NOW() 
		WHERE id IN (%s)
	`, len(ids)+1, strings.Join(placeholders, ", "))
	
	result, err := f.GetDB().ExecContext(ctx, query, args...)
	if err != nil {
		return errors.NewDatabaseError("bulk_update", "failed to bulk update findings").WithCause(err)
	}
	
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return errors.NewDatabaseError("bulk_update", "failed to get rows affected").WithCause(err)
	}
	
	if rowsAffected == 0 {
		return errors.NewNotFoundError("findings")
	}
	
	return nil
}