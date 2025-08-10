package findings

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/lib/pq"
)

// Repository handles database operations for findings
type Repository struct {
	db *sqlx.DB
}

// NewRepository creates a new findings repository
func NewRepository(db *sqlx.DB) *Repository {
	return &Repository{db: db}
}

// Create creates a new finding
func (r *Repository) Create(ctx context.Context, finding *Finding) error {
	query := `
		INSERT INTO findings (
			id, scan_result_id, scan_job_id, tool, rule_id, severity, category,
			title, description, file_path, line_number, column_number, code_snippet,
			confidence, consensus_score, status, fix_suggestion, references,
			created_at, updated_at
		) VALUES (
			:id, :scan_result_id, :scan_job_id, :tool, :rule_id, :severity, :category,
			:title, :description, :file_path, :line_number, :column_number, :code_snippet,
			:confidence, :consensus_score, :status, :fix_suggestion, :references,
			:created_at, :updated_at
		)`

	if finding.ID == uuid.Nil {
		finding.ID = uuid.New()
	}
	finding.CreatedAt = time.Now()
	finding.UpdatedAt = time.Now()

	_, err := r.db.NamedExecContext(ctx, query, finding)
	return err
}

// GetByID retrieves a finding by ID
func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (*Finding, error) {
	query := `
		SELECT id, scan_result_id, scan_job_id, tool, rule_id, severity, category,
			   title, description, file_path, line_number, column_number, code_snippet,
			   confidence, consensus_score, status, fix_suggestion, references,
			   created_at, updated_at
		FROM findings
		WHERE id = $1`

	var finding Finding
	err := r.db.GetContext(ctx, &finding, query, id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("finding not found")
		}
		return nil, err
	}

	return &finding, nil
}

// List retrieves findings with filtering and pagination
func (r *Repository) List(ctx context.Context, filter FindingFilter, limit, offset int) ([]*Finding, error) {
	query := `
		SELECT id, scan_result_id, scan_job_id, tool, rule_id, severity, category,
			   title, description, file_path, line_number, column_number, code_snippet,
			   confidence, consensus_score, status, fix_suggestion, references,
			   created_at, updated_at
		FROM findings`

	var conditions []string
	var args []interface{}
	argIndex := 1

	// Build WHERE conditions
	if filter.ScanJobID != nil {
		conditions = append(conditions, fmt.Sprintf("scan_job_id = $%d", argIndex))
		args = append(args, *filter.ScanJobID)
		argIndex++
	}

	if len(filter.Severity) > 0 {
		conditions = append(conditions, fmt.Sprintf("severity = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.Severity))
		argIndex++
	}

	if len(filter.Status) > 0 {
		statusStrings := make([]string, len(filter.Status))
		for i, status := range filter.Status {
			statusStrings[i] = string(status)
		}
		conditions = append(conditions, fmt.Sprintf("status = ANY($%d)", argIndex))
		args = append(args, pq.Array(statusStrings))
		argIndex++
	}

	if len(filter.Tool) > 0 {
		conditions = append(conditions, fmt.Sprintf("tool = ANY($%d)", argIndex))
		args = append(args, pq.Array(filter.Tool))
		argIndex++
	}

	if filter.FilePath != nil {
		conditions = append(conditions, fmt.Sprintf("file_path ILIKE $%d", argIndex))
		args = append(args, "%"+*filter.FilePath+"%")
		argIndex++
	}

	if filter.MinConfidence != nil {
		conditions = append(conditions, fmt.Sprintf("confidence >= $%d", argIndex))
		args = append(args, *filter.MinConfidence)
		argIndex++
	}

	if filter.Search != nil {
		conditions = append(conditions, fmt.Sprintf("(title ILIKE $%d OR description ILIKE $%d)", argIndex, argIndex))
		args = append(args, "%"+*filter.Search+"%")
		argIndex++
	}

	if len(conditions) > 0 {
		query += " WHERE " + strings.Join(conditions, " AND ")
	}

	query += " ORDER BY created_at DESC"

	if limit > 0 {
		query += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, limit)
		argIndex++
	}

	if offset > 0 {
		query += fmt.Sprintf(" OFFSET $%d", argIndex)
		args = append(args, offset)
	}

	var findings []*Finding
	err := r.db.SelectContext(ctx, &findings, query, args...)
	return findings, err
}

// Update updates a finding
func (r *Repository) Update(ctx context.Context, id uuid.UUID, update FindingUpdate) error {
	var setParts []string
	var args []interface{}
	argIndex := 1

	if update.Status != nil {
		setParts = append(setParts, fmt.Sprintf("status = $%d", argIndex))
		args = append(args, string(*update.Status))
		argIndex++
	}

	if update.FixSuggestion != nil {
		setParts = append(setParts, fmt.Sprintf("fix_suggestion = $%d", argIndex))
		args = append(args, *update.FixSuggestion)
		argIndex++
	}

	if len(setParts) == 0 {
		return fmt.Errorf("no fields to update")
	}

	setParts = append(setParts, fmt.Sprintf("updated_at = $%d", argIndex))
	args = append(args, time.Now())
	argIndex++

	query := fmt.Sprintf("UPDATE findings SET %s WHERE id = $%d", strings.Join(setParts, ", "), argIndex)
	args = append(args, id)

	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("finding not found")
	}

	return nil
}

// GetStats retrieves statistics about findings for a scan job
func (r *Repository) GetStats(ctx context.Context, scanJobID uuid.UUID) (*FindingStats, error) {
	query := `
		SELECT 
			COUNT(*) as total,
			COUNT(CASE WHEN severity = 'high' THEN 1 END) as high,
			COUNT(CASE WHEN severity = 'medium' THEN 1 END) as medium,
			COUNT(CASE WHEN severity = 'low' THEN 1 END) as low,
			COUNT(CASE WHEN status = 'open' THEN 1 END) as open,
			COUNT(CASE WHEN status = 'fixed' THEN 1 END) as fixed,
			COUNT(CASE WHEN status = 'ignored' THEN 1 END) as ignored,
			COUNT(CASE WHEN status = 'false_positive' THEN 1 END) as false_positives
		FROM findings
		WHERE scan_job_id = $1`

	var stats FindingStats
	err := r.db.GetContext(ctx, &stats, query, scanJobID)
	return &stats, err
}

// CreateSuppression creates a new finding suppression rule
func (r *Repository) CreateSuppression(ctx context.Context, suppression *FindingSuppression) error {
	query := `
		INSERT INTO finding_suppressions (
			id, user_id, rule_id, file_path, line_number, reason, expires_at, created_at, updated_at
		) VALUES (
			:id, :user_id, :rule_id, :file_path, :line_number, :reason, :expires_at, :created_at, :updated_at
		)`

	if suppression.ID == uuid.Nil {
		suppression.ID = uuid.New()
	}
	suppression.CreatedAt = time.Now()
	suppression.UpdatedAt = time.Now()

	_, err := r.db.NamedExecContext(ctx, query, suppression)
	return err
}

// GetSuppressions retrieves suppression rules for a user
func (r *Repository) GetSuppressions(ctx context.Context, userID uuid.UUID) ([]*FindingSuppression, error) {
	query := `
		SELECT id, user_id, rule_id, file_path, line_number, reason, expires_at, created_at, updated_at
		FROM finding_suppressions
		WHERE user_id = $1 AND (expires_at IS NULL OR expires_at > NOW())
		ORDER BY created_at DESC`

	var suppressions []*FindingSuppression
	err := r.db.SelectContext(ctx, &suppressions, query, userID)
	return suppressions, err
}

// DeleteSuppression deletes a suppression rule
func (r *Repository) DeleteSuppression(ctx context.Context, id uuid.UUID, userID uuid.UUID) error {
	query := `DELETE FROM finding_suppressions WHERE id = $1 AND user_id = $2`
	
	result, err := r.db.ExecContext(ctx, query, id, userID)
	if err != nil {
		return err
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if rowsAffected == 0 {
		return fmt.Errorf("suppression not found")
	}

	return nil
}

// CreateUserFeedback creates user feedback for a finding
func (r *Repository) CreateUserFeedback(ctx context.Context, feedback *UserFeedback) error {
	query := `
		INSERT INTO user_feedback (id, finding_id, user_id, action, comment, created_at)
		VALUES (:id, :finding_id, :user_id, :action, :comment, :created_at)
		ON CONFLICT (finding_id, user_id) 
		DO UPDATE SET action = EXCLUDED.action, comment = EXCLUDED.comment, created_at = EXCLUDED.created_at`

	if feedback.ID == uuid.Nil {
		feedback.ID = uuid.New()
	}
	feedback.CreatedAt = time.Now()

	_, err := r.db.NamedExecContext(ctx, query, feedback)
	return err
}

// GetUserFeedback retrieves user feedback for findings
func (r *Repository) GetUserFeedback(ctx context.Context, userID uuid.UUID, limit, offset int) ([]*UserFeedback, error) {
	query := `
		SELECT id, finding_id, user_id, action, comment, created_at
		FROM user_feedback
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3`

	var feedback []*UserFeedback
	err := r.db.SelectContext(ctx, &feedback, query, userID, limit, offset)
	return feedback, err
}