package database

import (
	"context"
	"encoding/json"

	"github.com/jmoiron/sqlx"
	"github.com/agentscan/agentscan/internal/domain/repositories"
	"github.com/agentscan/agentscan/pkg/errors"
)

// AuditTrail handles audit logging for repository operations
type AuditTrail struct {
	db        *sqlx.DB
	tableName string
}

// NewAuditTrail creates a new audit trail instance
func NewAuditTrail(db *sqlx.DB, tableName string) *AuditTrail {
	return &AuditTrail{
		db:        db,
		tableName: tableName,
	}
}

// Record records an audit entry
func (at *AuditTrail) Record(ctx context.Context, entry *repositories.AuditEntry) error {
	// Serialize changes and metadata to JSON
	changesJSON, err := json.Marshal(entry.Changes)
	if err != nil {
		return errors.NewDatabaseError("audit", "failed to serialize changes").WithCause(err)
	}

	metadataJSON, err := json.Marshal(entry.Metadata)
	if err != nil {
		return errors.NewDatabaseError("audit", "failed to serialize metadata").WithCause(err)
	}

	query := `
		INSERT INTO ` + at.tableName + ` 
		(id, entity_id, entity_type, action, user_id, changes, metadata, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
	`

	_, err = at.db.ExecContext(ctx, query,
		entry.ID,
		entry.EntityID,
		entry.EntityType,
		entry.Action,
		entry.UserID,
		changesJSON,
		metadataJSON,
		entry.CreatedAt,
	)

	if err != nil {
		return errors.NewDatabaseError("audit", "failed to insert audit entry").WithCause(err)
	}

	return nil
}

// GetAuditTrail retrieves audit entries for an entity
func (at *AuditTrail) GetAuditTrail(ctx context.Context, entityID string, limit, offset int) ([]*repositories.AuditEntry, error) {
	query := `
		SELECT id, entity_id, entity_type, action, user_id, changes, metadata, created_at
		FROM ` + at.tableName + `
		WHERE entity_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`

	rows, err := at.db.QueryContext(ctx, query, entityID, limit, offset)
	if err != nil {
		return nil, errors.NewDatabaseError("audit", "failed to query audit trail").WithCause(err)
	}
	defer rows.Close()

	var entries []*repositories.AuditEntry
	for rows.Next() {
		entry := &repositories.AuditEntry{}
		var changesJSON, metadataJSON []byte

		err := rows.Scan(
			&entry.ID,
			&entry.EntityID,
			&entry.EntityType,
			&entry.Action,
			&entry.UserID,
			&changesJSON,
			&metadataJSON,
			&entry.CreatedAt,
		)
		if err != nil {
			return nil, errors.NewDatabaseError("audit", "failed to scan audit entry").WithCause(err)
		}

		// Deserialize JSON fields
		if err := json.Unmarshal(changesJSON, &entry.Changes); err != nil {
			return nil, errors.NewDatabaseError("audit", "failed to deserialize changes").WithCause(err)
		}

		if err := json.Unmarshal(metadataJSON, &entry.Metadata); err != nil {
			return nil, errors.NewDatabaseError("audit", "failed to deserialize metadata").WithCause(err)
		}

		entries = append(entries, entry)
	}

	return entries, nil
}