package database

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/agentscan/agentscan/pkg/errors"
)

// RepositoryCache provides caching functionality for repositories
type RepositoryCache struct {
	client     *redis.Client
	defaultTTL time.Duration
	keyPrefix  string
}

// NewRepositoryCache creates a new repository cache
func NewRepositoryCache(redisURL string, defaultTTL time.Duration) *RepositoryCache {
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		// Fallback to default configuration
		opts = &redis.Options{
			Addr: "localhost:6379",
		}
	}

	client := redis.NewClient(opts)

	return &RepositoryCache{
		client:     client,
		defaultTTL: defaultTTL,
		keyPrefix:  "agentscan:repo:",
	}
}

// Get retrieves a cached entity
func (rc *RepositoryCache) Get(ctx context.Context, key string) (interface{}, bool) {
	fullKey := rc.keyPrefix + key

	data, err := rc.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false // Cache miss
		}
		// Log error but don't fail the operation
		return nil, false
	}

	var entity interface{}
	if err := json.Unmarshal([]byte(data), &entity); err != nil {
		// Log error but don't fail the operation
		return nil, false
	}

	return entity, true
}

// Set stores an entity in cache
func (rc *RepositoryCache) Set(ctx context.Context, key string, entity interface{}) error {
	fullKey := rc.keyPrefix + key

	data, err := json.Marshal(entity)
	if err != nil {
		return errors.NewInternalError("failed to serialize entity for cache").WithCause(err)
	}

	err = rc.client.Set(ctx, fullKey, data, rc.defaultTTL).Err()
	if err != nil {
		// Log error but don't fail the operation
		return nil
	}

	return nil
}

// GetList retrieves a cached list result
func (rc *RepositoryCache) GetList(ctx context.Context, key string) (*CachedListResult[interface{}], bool) {
	fullKey := rc.keyPrefix + key

	data, err := rc.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false // Cache miss
		}
		return nil, false
	}

	var result CachedListResult[interface{}]
	if err := json.Unmarshal([]byte(data), &result); err != nil {
		return nil, false
	}

	return &result, true
}

// SetList stores a list result in cache
func (rc *RepositoryCache) SetList(ctx context.Context, key string, result *CachedListResult[interface{}]) error {
	fullKey := rc.keyPrefix + key

	data, err := json.Marshal(result)
	if err != nil {
		return errors.NewInternalError("failed to serialize list result for cache").WithCause(err)
	}

	err = rc.client.Set(ctx, fullKey, data, rc.defaultTTL).Err()
	if err != nil {
		// Log error but don't fail the operation
		return nil
	}

	return nil
}

// Delete removes a cached entity
func (rc *RepositoryCache) Delete(ctx context.Context, key string) error {
	fullKey := rc.keyPrefix + key
	return rc.client.Del(ctx, fullKey).Err()
}

// DeletePattern removes cached entities matching a pattern
func (rc *RepositoryCache) DeletePattern(ctx context.Context, pattern string) error {
	fullPattern := rc.keyPrefix + pattern

	// Get all keys matching the pattern
	keys, err := rc.client.Keys(ctx, fullPattern).Result()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	// Delete all matching keys
	return rc.client.Del(ctx, keys...).Err()
}

// HealthCheck checks if the cache is available
func (rc *RepositoryCache) HealthCheck(ctx context.Context) error {
	return rc.client.Ping(ctx).Err()
}

// GetStats returns cache statistics
func (rc *RepositoryCache) GetStats(ctx context.Context) (*repositories.CacheStats, error) {
	info, err := rc.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, err
	}

	stats := &repositories.CacheStats{}

	// Parse Redis INFO output for statistics
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.Contains(line, "keyspace_hits:") {
			fmt.Sscanf(line, "keyspace_hits:%d", &stats.TotalHits)
		} else if strings.Contains(line, "keyspace_misses:") {
			fmt.Sscanf(line, "keyspace_misses:%d", &stats.TotalMisses)
		}
	}

	// Calculate hit rate
	total := stats.TotalHits + stats.TotalMisses
	if total > 0 {
		stats.HitRate = float64(stats.TotalHits) / float64(total)
		stats.MissRate = float64(stats.TotalMisses) / float64(total)
	}

	return stats, nil
}

// Close closes the cache connection
func (rc *RepositoryCache) Close() error {
	return rc.client.Close()
}