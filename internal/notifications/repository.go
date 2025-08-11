package notifications

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
)

// PostgreSQLRepository implements NotificationRepository using PostgreSQL
type PostgreSQLRepository struct {
	db *sqlx.DB
}

// NewPostgreSQLRepository creates a new PostgreSQL notification repository
func NewPostgreSQLRepository(db *sqlx.DB) *PostgreSQLRepository {
	return &PostgreSQLRepository{db: db}
}

// CreateChannel creates a new notification channel
func (r *PostgreSQLRepository) CreateChannel(ctx context.Context, channel *NotificationChannel) error {
	if channel.ID == uuid.Nil {
		channel.ID = uuid.New()
	}
	channel.CreatedAt = time.Now()
	channel.UpdatedAt = time.Now()

	query := `
		INSERT INTO notification_channels (
			id, user_id, org_id, type, name, config, enabled, preferences, created_at, updated_at
		) VALUES (
			:id, :user_id, :org_id, :type, :name, :config, :enabled, :preferences, :created_at, :updated_at
		)`

	_, err := r.db.NamedExecContext(ctx, query, channel)
	if err != nil {
		return fmt.Errorf("failed to create notification channel: %w", err)
	}

	return nil
}

// GetChannel retrieves a notification channel by ID
func (r *PostgreSQLRepository) GetChannel(ctx context.Context, id uuid.UUID) (*NotificationChannel, error) {
	var channel NotificationChannel
	query := `SELECT * FROM notification_channels WHERE id = $1`

	err := r.db.GetContext(ctx, &channel, query, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification channel: %w", err)
	}

	return &channel, nil
}

// GetChannelsByUser retrieves all notification channels for a user
func (r *PostgreSQLRepository) GetChannelsByUser(ctx context.Context, userID uuid.UUID) ([]NotificationChannel, error) {
	var channels []NotificationChannel
	query := `SELECT * FROM notification_channels WHERE user_id = $1 ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &channels, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user notification channels: %w", err)
	}

	return channels, nil
}

// GetChannelsByOrg retrieves all notification channels for an organization
func (r *PostgreSQLRepository) GetChannelsByOrg(ctx context.Context, orgID uuid.UUID) ([]NotificationChannel, error) {
	var channels []NotificationChannel
	query := `SELECT * FROM notification_channels WHERE org_id = $1 ORDER BY created_at DESC`

	err := r.db.SelectContext(ctx, &channels, query, orgID)
	if err != nil {
		return nil, fmt.Errorf("failed to get org notification channels: %w", err)
	}

	return channels, nil
}

// UpdateChannel updates a notification channel
func (r *PostgreSQLRepository) UpdateChannel(ctx context.Context, channel *NotificationChannel) error {
	channel.UpdatedAt = time.Now()

	query := `
		UPDATE notification_channels 
		SET name = :name, config = :config, enabled = :enabled, preferences = :preferences, updated_at = :updated_at
		WHERE id = :id`

	result, err := r.db.NamedExecContext(ctx, query, channel)
	if err != nil {
		return fmt.Errorf("failed to update notification channel: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification channel not found")
	}

	return nil
}

// DeleteChannel deletes a notification channel
func (r *PostgreSQLRepository) DeleteChannel(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM notification_channels WHERE id = $1`

	result, err := r.db.ExecContext(ctx, query, id)
	if err != nil {
		return fmt.Errorf("failed to delete notification channel: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("notification channel not found")
	}

	return nil
}

// LogEvent logs a notification event
func (r *PostgreSQLRepository) LogEvent(ctx context.Context, event *NotificationEvent) error {
	if event.ID == uuid.Nil {
		event.ID = uuid.New()
	}
	if event.CreatedAt.IsZero() {
		event.CreatedAt = time.Now()
	}

	query := `
		INSERT INTO notification_events (
			id, channel_id, type, status, message, error, metadata, created_at
		) VALUES (
			:id, :channel_id, :type, :status, :message, :error, :metadata, :created_at
		)
		ON CONFLICT (id) DO UPDATE SET
			status = EXCLUDED.status,
			error = EXCLUDED.error,
			metadata = EXCLUDED.metadata`

	_, err := r.db.NamedExecContext(ctx, query, event)
	if err != nil {
		return fmt.Errorf("failed to log notification event: %w", err)
	}

	return nil
}

// GetEvents retrieves notification events with filtering and pagination
func (r *PostgreSQLRepository) GetEvents(ctx context.Context, filter NotificationFilter, limit, offset int) ([]NotificationEvent, int64, error) {
	var events []NotificationEvent
	var total int64

	// Build WHERE clause
	whereClause, args := r.buildEventWhereClause(filter)

	// Get total count
	countQuery := `SELECT COUNT(*) FROM notification_events ne JOIN notification_channels nc ON ne.channel_id = nc.id ` + whereClause
	err := r.db.GetContext(ctx, &total, countQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to count notification events: %w", err)
	}

	// Get events
	query := `
		SELECT ne.* FROM notification_events ne 
		JOIN notification_channels nc ON ne.channel_id = nc.id 
		` + whereClause + `
		ORDER BY ne.created_at DESC 
		LIMIT $` + fmt.Sprintf("%d", len(args)+1) + ` OFFSET $` + fmt.Sprintf("%d", len(args)+2)

	args = append(args, limit, offset)
	err = r.db.SelectContext(ctx, &events, query, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to get notification events: %w", err)
	}

	return events, total, nil
}

// GetStats retrieves notification statistics
func (r *PostgreSQLRepository) GetStats(ctx context.Context, filter NotificationFilter) (*NotificationStats, error) {
	whereClause, args := r.buildEventWhereClause(filter)

	stats := &NotificationStats{
		ByChannel:   make(map[NotificationChannelType]int64),
		ByEventType: make(map[NotificationEventType]int64),
		LastUpdated: time.Now(),
	}

	// Get total counts
	query := `
		SELECT 
			COUNT(*) FILTER (WHERE ne.status = 'sent') as total_sent,
			COUNT(*) FILTER (WHERE ne.status = 'failed') as total_failed
		FROM notification_events ne 
		JOIN notification_channels nc ON ne.channel_id = nc.id 
		` + whereClause

	var counts struct {
		TotalSent   int64 `db:"total_sent"`
		TotalFailed int64 `db:"total_failed"`
	}

	err := r.db.GetContext(ctx, &counts, query, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get notification counts: %w", err)
	}

	stats.TotalSent = counts.TotalSent
	stats.TotalFailed = counts.TotalFailed

	// Get counts by channel type
	channelQuery := `
		SELECT nc.type, COUNT(*) as count
		FROM notification_events ne 
		JOIN notification_channels nc ON ne.channel_id = nc.id 
		` + whereClause + ` AND ne.status = 'sent'
		GROUP BY nc.type`

	var channelCounts []struct {
		Type  NotificationChannelType `db:"type"`
		Count int64                   `db:"count"`
	}

	err = r.db.SelectContext(ctx, &channelCounts, channelQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get channel counts: %w", err)
	}

	for _, count := range channelCounts {
		stats.ByChannel[count.Type] = count.Count
	}

	// Get counts by event type
	eventQuery := `
		SELECT ne.type, COUNT(*) as count
		FROM notification_events ne 
		JOIN notification_channels nc ON ne.channel_id = nc.id 
		` + whereClause + ` AND ne.status = 'sent'
		GROUP BY ne.type`

	var eventCounts []struct {
		Type  NotificationEventType `db:"type"`
		Count int64                 `db:"count"`
	}

	err = r.db.SelectContext(ctx, &eventCounts, eventQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get event type counts: %w", err)
	}

	for _, count := range eventCounts {
		stats.ByEventType[count.Type] = count.Count
	}

	// Get recent events
	recentQuery := `
		SELECT ne.* FROM notification_events ne 
		JOIN notification_channels nc ON ne.channel_id = nc.id 
		` + whereClause + `
		ORDER BY ne.created_at DESC 
		LIMIT 10`

	err = r.db.SelectContext(ctx, &stats.RecentEvents, recentQuery, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to get recent events: %w", err)
	}

	return stats, nil
}

// buildEventWhereClause builds WHERE clause for event queries
func (r *PostgreSQLRepository) buildEventWhereClause(filter NotificationFilter) (string, []interface{}) {
	var conditions []string
	var args []interface{}
	argIndex := 1

	if filter.UserID != nil {
		conditions = append(conditions, fmt.Sprintf("nc.user_id = $%d", argIndex))
		args = append(args, *filter.UserID)
		argIndex++
	}

	if filter.OrgID != nil {
		conditions = append(conditions, fmt.Sprintf("nc.org_id = $%d", argIndex))
		args = append(args, *filter.OrgID)
		argIndex++
	}

	if filter.ChannelType != nil {
		conditions = append(conditions, fmt.Sprintf("nc.type = $%d", argIndex))
		args = append(args, *filter.ChannelType)
		argIndex++
	}

	if filter.EventType != nil {
		conditions = append(conditions, fmt.Sprintf("ne.type = $%d", argIndex))
		args = append(args, *filter.EventType)
		argIndex++
	}

	if filter.Status != nil {
		conditions = append(conditions, fmt.Sprintf("ne.status = $%d", argIndex))
		args = append(args, *filter.Status)
		argIndex++
	}

	if filter.DateFrom != nil {
		conditions = append(conditions, fmt.Sprintf("ne.created_at >= $%d", argIndex))
		args = append(args, *filter.DateFrom)
		argIndex++
	}

	if filter.DateTo != nil {
		conditions = append(conditions, fmt.Sprintf("ne.created_at <= $%d", argIndex))
		args = append(args, *filter.DateTo)
		argIndex++
	}

	whereClause := ""
	if len(conditions) > 0 {
		whereClause = "WHERE " + fmt.Sprintf("(%s)", fmt.Sprintf("%s", conditions[0]))
		for _, condition := range conditions[1:] {
			whereClause += " AND " + condition
		}
	}

	return whereClause, args
}

// Custom types for JSON serialization

// Value implements the driver.Valuer interface for ChannelConfig
func (c ChannelConfig) Value() (driver.Value, error) {
	return json.Marshal(c)
}

// Scan implements the sql.Scanner interface for ChannelConfig
func (c *ChannelConfig) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into ChannelConfig", value)
	}

	return json.Unmarshal(bytes, c)
}

// Value implements the driver.Valuer interface for NotificationPreferences
func (p NotificationPreferences) Value() (driver.Value, error) {
	return json.Marshal(p)
}

// Scan implements the sql.Scanner interface for NotificationPreferences
func (p *NotificationPreferences) Scan(value interface{}) error {
	if value == nil {
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into NotificationPreferences", value)
	}

	return json.Unmarshal(bytes, p)
}

// Value implements the driver.Valuer interface for metadata
func (m MetadataMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	return json.Marshal(m)
}

// Scan implements the sql.Scanner interface for metadata
func (m *MetadataMap) Scan(value interface{}) error {
	if value == nil {
		*m = nil
		return nil
	}

	bytes, ok := value.([]byte)
	if !ok {
		return fmt.Errorf("cannot scan %T into MetadataMap", value)
	}

	return json.Unmarshal(bytes, m)
}