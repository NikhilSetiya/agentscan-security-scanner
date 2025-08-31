package database

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	"github.com/agentscan/agentscan/internal/domain/repositories"
	"github.com/agentscan/agentscan/pkg/errors"
	"github.com/agentscan/agentscan/pkg/types"
)

// StandardizedRepositoryImpl provides a standardized base implementation
type StandardizedRepositoryImpl[T any, ID comparable] struct {
	*BaseRepositoryImpl[T, ID]
	auditEnabled   bool
	cacheEnabled   bool
	metricsEnabled bool
	eventEnabled   bool
	
	// Additional components
	auditor    *AuditTrail
	cache      *RepositoryCache
	metrics    *RepositoryMetrics
	events     *EventPublisher
	validator  *EntityValidator[T]
}

// NewStandardizedRepository creates a new standardized repository
func NewStandardizedRepository[T any, ID comparable](
	db *sqlx.DB, 
	tableName string, 
	config *repositories.RepositoryConfig,
) *StandardizedRepositoryImpl[T, ID] {
	base := NewBaseRepository[T, ID](db, tableName)
	
	repo := &StandardizedRepositoryImpl[T, ID]{
		BaseRepositoryImpl: base,
		auditEnabled:       config.AuditEnabled,
		cacheEnabled:       config.CacheEnabled,
		metricsEnabled:     config.MetricsEnabled,
		eventEnabled:       config.EventsEnabled,
	}
	
	// Initialize components based on configuration
	if config.AuditEnabled {
		repo.auditor = NewAuditTrail(db, config.AuditTableName)
	}
	if config.CacheEnabled {
		repo.cache = NewRepositoryCache(config.CacheURL, config.DefaultCacheTTL)
	}
	if config.MetricsEnabled {
		repo.metrics = NewRepositoryMetrics(config.MetricsNamespace)
	}
	if config.EventsEnabled {
		repo.events = NewEventPublisher(config.EventBusURL)
	}
	if config.ValidationEnabled {
		repo.validator = NewEntityValidator[T](config.StrictValidation)
	}
	
	return repo
}

// Create with audit, cache, metrics, and events
func (r *StandardizedRepositoryImpl[T, ID]) Create(ctx context.Context, entity *T) error {
	startTime := time.Now()
	
	// Validate entity
	if r.validator != nil {
		if err := r.validator.Validate(ctx, entity); err != nil {
			r.recordMetrics("create", startTime, err)
			return err
		}
	}
	
	// Execute base create
	err := r.BaseRepositoryImpl.Create(ctx, entity)
	if err != nil {
		r.recordMetrics("create", startTime, err)
		return err
	}
	
	// Post-create operations
	r.recordAudit(ctx, "create", entity, nil, nil)
	r.invalidateCache(ctx, entity)
	r.publishEvent(ctx, "created", entity, nil)
	r.recordMetrics("create", startTime, nil)
	
	return nil
}

// Update with audit, cache, metrics, and events
func (r *StandardizedRepositoryImpl[T, ID]) Update(ctx context.Context, id ID, updates map[string]interface{}) error {
	startTime := time.Now()
	
	// Get old entity for audit trail
	var oldEntity *T
	if r.auditEnabled || r.eventEnabled {
		oldEntity, _ = r.BaseRepositoryImpl.GetByID(ctx, id)
	}
	
	// Execute base update
	err := r.BaseRepositoryImpl.Update(ctx, id, updates)
	if err != nil {
		r.recordMetrics("update", startTime, err)
		return err
	}
	
	// Get new entity for audit and events
	var newEntity *T
	if r.auditEnabled || r.eventEnabled {
		newEntity, _ = r.BaseRepositoryImpl.GetByID(ctx, id)
	}
	
	// Post-update operations
	r.recordAudit(ctx, "update", newEntity, oldEntity, updates)
	r.invalidateCacheByID(ctx, id)
	r.publishEvent(ctx, "updated", newEntity, oldEntity)
	r.recordMetrics("update", startTime, nil)
	
	return nil
}

// Delete with audit, cache, metrics, and events
func (r *StandardizedRepositoryImpl[T, ID]) Delete(ctx context.Context, id ID) error {
	startTime := time.Now()
	
	// Get entity for audit trail and events
	var entity *T
	if r.auditEnabled || r.eventEnabled {
		entity, _ = r.BaseRepositoryImpl.GetByID(ctx, id)
	}
	
	// Execute base delete
	err := r.BaseRepositoryImpl.Delete(ctx, id)
	if err != nil {
		r.recordMetrics("delete", startTime, err)
		return err
	}
	
	// Post-delete operations
	r.recordAudit(ctx, "delete", entity, nil, nil)
	r.invalidateCacheByID(ctx, id)
	r.publishEvent(ctx, "deleted", entity, nil)
	r.recordMetrics("delete", startTime, nil)
	
	return nil
}

// GetByID with caching
func (r *StandardizedRepositoryImpl[T, ID]) GetByID(ctx context.Context, id ID) (*T, error) {
	startTime := time.Now()
	
	// Try cache first
	if r.cacheEnabled && r.cache != nil {
		if entity, found := r.cache.Get(ctx, r.cacheKey(id)); found {
			r.recordMetrics("get", startTime, nil)
			return entity.(*T), nil
		}
	}
	
	// Get from database
	entity, err := r.BaseRepositoryImpl.GetByID(ctx, id)
	if err != nil {
		r.recordMetrics("get", startTime, err)
		return nil, err
	}
	
	// Cache the result
	if r.cacheEnabled && r.cache != nil && entity != nil {
		r.cache.Set(ctx, r.cacheKey(id), entity)
	}
	
	r.recordMetrics("get", startTime, nil)
	return entity, nil
}

// List with caching and metrics
func (r *StandardizedRepositoryImpl[T, ID]) List(ctx context.Context, filters map[string]interface{}, limit, offset int) ([]*T, int, error) {
	startTime := time.Now()
	
	// Try cache for common queries
	cacheKey := r.listCacheKey(filters, limit, offset)
	if r.cacheEnabled && r.cache != nil {
		if result, found := r.cache.GetList(ctx, cacheKey); found {
			r.recordMetrics("list", startTime, nil)
			return result.Data, result.Total, nil
		}
	}
	
	// Get from database
	entities, total, err := r.BaseRepositoryImpl.List(ctx, filters, limit, offset)
	if err != nil {
		r.recordMetrics("list", startTime, err)
		return nil, 0, err
	}
	
	// Cache the result
	if r.cacheEnabled && r.cache != nil {
		r.cache.SetList(ctx, cacheKey, &CachedListResult[T]{
			Data:  entities,
			Total: total,
		})
	}
	
	r.recordMetrics("list", startTime, nil)
	return entities, total, nil
}

// Batch operations with transaction support
func (r *StandardizedRepositoryImpl[T, ID]) CreateBatch(ctx context.Context, entities []*T) error {
	if len(entities) == 0 {
		return nil
	}
	
	startTime := time.Now()
	
	// Validate all entities first
	if r.validator != nil {
		for _, entity := range entities {
			if err := r.validator.Validate(ctx, entity); err != nil {
				r.recordMetrics("create_batch", startTime, err)
				return err
			}
		}
	}
	
	// Execute in transaction
	err := r.withTransaction(ctx, func(tx *sqlx.Tx) error {
		for _, entity := range entities {
			if err := r.createInTx(ctx, tx, entity); err != nil {
				return err
			}
		}
		return nil
	})
	
	if err != nil {
		r.recordMetrics("create_batch", startTime, err)
		return err
	}
	
	// Post-create operations
	for _, entity := range entities {
		r.recordAudit(ctx, "create", entity, nil, nil)
		r.invalidateCache(ctx, entity)
		r.publishEvent(ctx, "created", entity, nil)
	}
	
	r.recordMetrics("create_batch", startTime, nil)
	return nil
}

// UpdateBatch with transaction support
func (r *StandardizedRepositoryImpl[T, ID]) UpdateBatch(ctx context.Context, updates []repositories.BatchUpdate[ID]) error {
	if len(updates) == 0 {
		return nil
	}
	
	startTime := time.Now()
	
	// Execute in transaction
	err := r.withTransaction(ctx, func(tx *sqlx.Tx) error {
		for _, update := range updates {
			if err := r.updateInTx(ctx, tx, update.ID, update.Updates); err != nil {
				return err
			}
		}
		return nil
	})
	
	if err != nil {
		r.recordMetrics("update_batch", startTime, err)
		return err
	}
	
	// Post-update operations
	for _, update := range updates {
		r.invalidateCacheByID(ctx, update.ID)
		// Note: Audit and events would need individual entity fetching
	}
	
	r.recordMetrics("update_batch", startTime, nil)
	return nil
}

// DeleteBatch with transaction support
func (r *StandardizedRepositoryImpl[T, ID]) DeleteBatch(ctx context.Context, ids []ID) error {
	if len(ids) == 0 {
		return nil
	}
	
	startTime := time.Now()
	
	// Get entities for audit and events
	var entities []*T
	if r.auditEnabled || r.eventEnabled {
		for _, id := range ids {
			if entity, err := r.BaseRepositoryImpl.GetByID(ctx, id); err == nil {
				entities = append(entities, entity)
			}
		}
	}
	
	// Execute in transaction
	err := r.withTransaction(ctx, func(tx *sqlx.Tx) error {
		for _, id := range ids {
			if err := r.deleteInTx(ctx, tx, id); err != nil {
				return err
			}
		}
		return nil
	})
	
	if err != nil {
		r.recordMetrics("delete_batch", startTime, err)
		return err
	}
	
	// Post-delete operations
	for i, id := range ids {
		if i < len(entities) {
			r.recordAudit(ctx, "delete", entities[i], nil, nil)
			r.publishEvent(ctx, "deleted", entities[i], nil)
		}
		r.invalidateCacheByID(ctx, id)
	}
	
	r.recordMetrics("delete_batch", startTime, nil)
	return nil
}

// Advanced querying with caching
func (r *StandardizedRepositoryImpl[T, ID]) Query(ctx context.Context, options *repositories.QueryOptions) ([]*T, int, error) {
	startTime := time.Now()
	
	// Build query from options
	query, args := r.buildAdvancedQuery(options)
	
	// Try cache
	cacheKey := r.queryCacheKey(options)
	if r.cacheEnabled && r.cache != nil {
		if result, found := r.cache.GetList(ctx, cacheKey); found {
			r.recordMetrics("query", startTime, nil)
			return result.Data, result.Total, nil
		}
	}
	
	// Execute query
	entities, total, err := r.executeAdvancedQuery(ctx, query, args, options)
	if err != nil {
		r.recordMetrics("query", startTime, err)
		return nil, 0, err
	}
	
	// Cache result
	if r.cacheEnabled && r.cache != nil {
		r.cache.SetList(ctx, cacheKey, &CachedListResult[T]{
			Data:  entities,
			Total: total,
		})
	}
	
	r.recordMetrics("query", startTime, nil)
	return entities, total, nil
}

// Health check implementation
func (r *StandardizedRepositoryImpl[T, ID]) HealthCheck(ctx context.Context) error {
	// Check database connection
	if err := r.BaseRepositoryImpl.HealthCheck(ctx); err != nil {
		return err
	}
	
	// Check cache connection
	if r.cacheEnabled && r.cache != nil {
		if err := r.cache.HealthCheck(ctx); err != nil {
			return fmt.Errorf("cache health check failed: %w", err)
		}
	}
	
	// Check event publisher
	if r.eventEnabled && r.events != nil {
		if err := r.events.HealthCheck(ctx); err != nil {
			return fmt.Errorf("event publisher health check failed: %w", err)
		}
	}
	
	return nil
}

// Helper methods

func (r *StandardizedRepositoryImpl[T, ID]) recordAudit(ctx context.Context, action string, entity, oldEntity *T, changes map[string]interface{}) {
	if !r.auditEnabled || r.auditor == nil {
		return
	}
	
	// Extract entity ID for audit
	var entityID string
	if entity != nil {
		// Use reflection or type assertion to get ID
		entityID = fmt.Sprintf("%v", r.extractEntityID(entity))
	}
	
	r.auditor.Record(ctx, &repositories.AuditEntry{
		ID:         uuid.New(),
		EntityID:   entityID,
		EntityType: r.GetTableName(),
		Action:     action,
		UserID:     r.getUserIDFromContext(ctx),
		Changes:    changes,
		Metadata:   r.getMetadataFromContext(ctx),
		CreatedAt:  time.Now(),
	})
}

func (r *StandardizedRepositoryImpl[T, ID]) invalidateCache(ctx context.Context, entity *T) {
	if !r.cacheEnabled || r.cache == nil {
		return
	}
	
	// Invalidate entity cache
	entityID := r.extractEntityID(entity)
	r.cache.Delete(ctx, r.cacheKey(entityID))
	
	// Invalidate list caches (pattern-based)
	r.cache.DeletePattern(ctx, fmt.Sprintf("%s:list:*", r.GetTableName()))
}

func (r *StandardizedRepositoryImpl[T, ID]) invalidateCacheByID(ctx context.Context, id ID) {
	if !r.cacheEnabled || r.cache == nil {
		return
	}
	
	r.cache.Delete(ctx, r.cacheKey(id))
	r.cache.DeletePattern(ctx, fmt.Sprintf("%s:list:*", r.GetTableName()))
}

func (r *StandardizedRepositoryImpl[T, ID]) publishEvent(ctx context.Context, eventType string, entity, oldEntity *T) {
	if !r.eventEnabled || r.events == nil {
		return
	}
	
	event := &RepositoryEvent[T]{
		Type:      eventType,
		Entity:    entity,
		OldEntity: oldEntity,
		Timestamp: time.Now(),
		Metadata:  r.getMetadataFromContext(ctx),
	}
	
	r.events.Publish(ctx, event)
}

func (r *StandardizedRepositoryImpl[T, ID]) recordMetrics(operation string, startTime time.Time, err error) {
	if !r.metricsEnabled || r.metrics == nil {
		return
	}
	
	duration := time.Since(startTime)
	r.metrics.RecordOperation(operation, duration, err == nil)
}

func (r *StandardizedRepositoryImpl[T, ID]) cacheKey(id ID) string {
	return fmt.Sprintf("%s:entity:%v", r.GetTableName(), id)
}

func (r *StandardizedRepositoryImpl[T, ID]) listCacheKey(filters map[string]interface{}, limit, offset int) string {
	// Create a deterministic cache key from filters
	filterStr := r.serializeFilters(filters)
	return fmt.Sprintf("%s:list:%s:%d:%d", r.GetTableName(), filterStr, limit, offset)
}

func (r *StandardizedRepositoryImpl[T, ID]) queryCacheKey(options *repositories.QueryOptions) string {
	// Create a deterministic cache key from query options
	optionsStr := r.serializeQueryOptions(options)
	return fmt.Sprintf("%s:query:%s", r.GetTableName(), optionsStr)
}

func (r *StandardizedRepositoryImpl[T, ID]) serializeFilters(filters map[string]interface{}) string {
	// Simple serialization - in production, use a proper serialization method
	var parts []string
	for k, v := range filters {
		parts = append(parts, fmt.Sprintf("%s=%v", k, v))
	}
	return strings.Join(parts, "&")
}

func (r *StandardizedRepositoryImpl[T, ID]) serializeQueryOptions(options *repositories.QueryOptions) string {
	// Simple serialization - in production, use a proper serialization method
	return fmt.Sprintf("filters=%s&sort=%s:%s&limit=%d&offset=%d", 
		r.serializeFilters(options.Filters), 
		options.SortBy, 
		options.SortOrder, 
		options.Limit, 
		options.Offset)
}

func (r *StandardizedRepositoryImpl[T, ID]) extractEntityID(entity *T) ID {
	// This would use reflection or type assertion to extract the ID field
	// For now, return zero value - implement based on your entity structure
	var zero ID
	return zero
}

func (r *StandardizedRepositoryImpl[T, ID]) getUserIDFromContext(ctx context.Context) *uuid.UUID {
	if userID, ok := ctx.Value("user_id").(uuid.UUID); ok {
		return &userID
	}
	return nil
}

func (r *StandardizedRepositoryImpl[T, ID]) getMetadataFromContext(ctx context.Context) map[string]interface{} {
	metadata := make(map[string]interface{})
	
	if correlationID, ok := ctx.Value("correlation_id").(string); ok {
		metadata["correlation_id"] = correlationID
	}
	if userAgent, ok := ctx.Value("user_agent").(string); ok {
		metadata["user_agent"] = userAgent
	}
	if ipAddress, ok := ctx.Value("ip_address").(string); ok {
		metadata["ip_address"] = ipAddress
	}
	
	return metadata
}

// Transaction helpers
func (r *StandardizedRepositoryImpl[T, ID]) withTransaction(ctx context.Context, fn func(*sqlx.Tx) error) error {
	tx, err := r.GetDB().BeginTxx(ctx, nil)
	if err != nil {
		return err
	}
	
	defer func() {
		if p := recover(); p != nil {
			tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()
		} else {
			err = tx.Commit()
		}
	}()
	
	err = fn(tx)
	return err
}

func (r *StandardizedRepositoryImpl[T, ID]) createInTx(ctx context.Context, tx *sqlx.Tx, entity *T) error {
	// Implementation would use the transaction to create entity
	// This is a placeholder - implement based on your entity structure
	return nil
}

func (r *StandardizedRepositoryImpl[T, ID]) updateInTx(ctx context.Context, tx *sqlx.Tx, id ID, updates map[string]interface{}) error {
	// Implementation would use the transaction to update entity
	// This is a placeholder - implement based on your entity structure
	return nil
}

func (r *StandardizedRepositoryImpl[T, ID]) deleteInTx(ctx context.Context, tx *sqlx.Tx, id ID) error {
	// Implementation would use the transaction to delete entity
	// This is a placeholder - implement based on your entity structure
	return nil
}

func (r *StandardizedRepositoryImpl[T, ID]) buildAdvancedQuery(options *repositories.QueryOptions) (string, []interface{}) {
	// Build advanced query from options
	// This is a placeholder - implement based on your query builder
	return "", nil
}

func (r *StandardizedRepositoryImpl[T, ID]) executeAdvancedQuery(ctx context.Context, query string, args []interface{}, options *repositories.QueryOptions) ([]*T, int, error) {
	// Execute advanced query
	// This is a placeholder - implement based on your query execution
	return nil, 0, nil
}

// Supporting types

type CachedListResult[T any] struct {
	Data  []*T `json:"data"`
	Total int  `json:"total"`
}

type RepositoryEvent[T any] struct {
	Type      string                 `json:"type"`
	Entity    *T                     `json:"entity"`
	OldEntity *T                     `json:"old_entity,omitempty"`
	Timestamp time.Time              `json:"timestamp"`
	Metadata  map[string]interface{} `json:"metadata"`
}