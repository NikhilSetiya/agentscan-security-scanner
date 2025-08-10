package incremental

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// RedisCacheRepository implements the CacheRepository interface using Redis
type RedisCacheRepository struct {
	client *redis.Client
	prefix string
	ttl    time.Duration
}

// NewRedisCacheRepository creates a new Redis-based cache repository
func NewRedisCacheRepository(client *redis.Client, prefix string, ttl time.Duration) *RedisCacheRepository {
	return &RedisCacheRepository{
		client: client,
		prefix: prefix,
		ttl:    ttl,
	}
}

// GetCache retrieves a cached scan result
func (r *RedisCacheRepository) GetCache(ctx context.Context, repoID uuid.UUID, filePath, toolName, contentHash, configHash string) (*ScanCache, error) {
	key := r.buildCacheKey(repoID, filePath, toolName, contentHash, configHash)
	
	data, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, nil // Cache miss
		}
		return nil, fmt.Errorf("failed to get cache: %w", err)
	}

	var cache ScanCache
	if err := json.Unmarshal([]byte(data), &cache); err != nil {
		return nil, fmt.Errorf("failed to unmarshal cache data: %w", err)
	}

	return &cache, nil
}

// SetCache stores a scan result in the cache
func (r *RedisCacheRepository) SetCache(ctx context.Context, cache *ScanCache) error {
	key := r.buildCacheKey(cache.RepositoryID, cache.FilePath, cache.ToolName, cache.ContentHash, cache.ConfigHash)
	
	data, err := json.Marshal(cache)
	if err != nil {
		return fmt.Errorf("failed to marshal cache data: %w", err)
	}

	err = r.client.Set(ctx, key, data, r.ttl).Err()
	if err != nil {
		return fmt.Errorf("failed to set cache: %w", err)
	}

	// Also update cache statistics
	statsKey := r.buildStatsKey(cache.RepositoryID)
	r.client.HIncrBy(ctx, statsKey, "total_entries", 1)
	r.client.Expire(ctx, statsKey, r.ttl)

	return nil
}

// InvalidateCache removes cache entries for specific files
func (r *RedisCacheRepository) InvalidateCache(ctx context.Context, repoID uuid.UUID, filePaths []string) error {
	if len(filePaths) == 0 {
		return nil
	}

	// Build pattern to find all cache keys for these files
	var keys []string
	for _, filePath := range filePaths {
		pattern := r.buildCachePattern(repoID, filePath)
		
		// Find all keys matching the pattern
		matchingKeys, err := r.client.Keys(ctx, pattern).Result()
		if err != nil {
			return fmt.Errorf("failed to find cache keys for pattern %s: %w", pattern, err)
		}
		
		keys = append(keys, matchingKeys...)
	}

	if len(keys) == 0 {
		return nil
	}

	// Delete all matching keys
	err := r.client.Del(ctx, keys...).Err()
	if err != nil {
		return fmt.Errorf("failed to delete cache keys: %w", err)
	}

	// Update cache statistics
	statsKey := r.buildStatsKey(repoID)
	r.client.HIncrBy(ctx, statsKey, "total_entries", -int64(len(keys)))

	return nil
}

// GetCacheStats returns cache statistics for a repository
func (r *RedisCacheRepository) GetCacheStats(ctx context.Context, repoID uuid.UUID) (*CacheStats, error) {
	statsKey := r.buildStatsKey(repoID)
	
	stats, err := r.client.HGetAll(ctx, statsKey).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache stats: %w", err)
	}

	cacheStats := &CacheStats{}

	if totalStr, ok := stats["total_entries"]; ok {
		var total int64
		fmt.Sscanf(totalStr, "%d", &total)
		cacheStats.TotalEntries = total
	}

	if hitsStr, ok := stats["hits"]; ok {
		var hits int64
		fmt.Sscanf(hitsStr, "%d", &hits)
		
		if missesStr, ok := stats["misses"]; ok {
			var misses int64
			fmt.Sscanf(missesStr, "%d", &misses)
			
			total := hits + misses
			if total > 0 {
				cacheStats.HitRate = float64(hits) / float64(total)
				cacheStats.MissRate = float64(misses) / float64(total)
			}
		}
	}

	// Estimate size by counting keys
	pattern := r.buildCachePattern(repoID, "*")
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err == nil {
		// Rough estimate: assume average 1KB per cache entry
		cacheStats.SizeBytes = int64(len(keys)) * 1024
	}

	return cacheStats, nil
}

// CleanupExpiredCache removes old cache entries
func (r *RedisCacheRepository) CleanupExpiredCache(ctx context.Context, maxAge time.Duration) error {
	// Redis handles TTL automatically, but we can clean up stats
	// Find all stats keys and clean up old ones
	pattern := r.prefix + ":stats:*"
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to find stats keys: %w", err)
	}

	var expiredKeys []string
	for _, key := range keys {
		ttl, err := r.client.TTL(ctx, key).Result()
		if err != nil {
			continue
		}
		
		// If TTL is very low or expired, mark for deletion
		if ttl < time.Minute {
			expiredKeys = append(expiredKeys, key)
		}
	}

	if len(expiredKeys) > 0 {
		err = r.client.Del(ctx, expiredKeys...).Err()
		if err != nil {
			return fmt.Errorf("failed to delete expired stats keys: %w", err)
		}
	}

	return nil
}

// RecordCacheHit records a cache hit for statistics
func (r *RedisCacheRepository) RecordCacheHit(ctx context.Context, repoID uuid.UUID) {
	statsKey := r.buildStatsKey(repoID)
	r.client.HIncrBy(ctx, statsKey, "hits", 1)
	r.client.Expire(ctx, statsKey, r.ttl)
}

// RecordCacheMiss records a cache miss for statistics
func (r *RedisCacheRepository) RecordCacheMiss(ctx context.Context, repoID uuid.UUID) {
	statsKey := r.buildStatsKey(repoID)
	r.client.HIncrBy(ctx, statsKey, "misses", 1)
	r.client.Expire(ctx, statsKey, r.ttl)
}

// Helper methods

func (r *RedisCacheRepository) buildCacheKey(repoID uuid.UUID, filePath, toolName, contentHash, configHash string) string {
	return fmt.Sprintf("%s:cache:%s:%s:%s:%s:%s", 
		r.prefix, repoID.String(), filePath, toolName, contentHash, configHash)
}

func (r *RedisCacheRepository) buildCachePattern(repoID uuid.UUID, filePath string) string {
	return fmt.Sprintf("%s:cache:%s:%s:*", r.prefix, repoID.String(), filePath)
}

func (r *RedisCacheRepository) buildStatsKey(repoID uuid.UUID) string {
	return fmt.Sprintf("%s:stats:%s", r.prefix, repoID.String())
}

// InMemoryCacheRepository implements the CacheRepository interface using in-memory storage
// This is useful for testing or small deployments
type InMemoryCacheRepository struct {
	cache map[string]*ScanCache
	stats map[uuid.UUID]*CacheStats
}

// NewInMemoryCacheRepository creates a new in-memory cache repository
func NewInMemoryCacheRepository() *InMemoryCacheRepository {
	return &InMemoryCacheRepository{
		cache: make(map[string]*ScanCache),
		stats: make(map[uuid.UUID]*CacheStats),
	}
}

// GetCache retrieves a cached scan result
func (m *InMemoryCacheRepository) GetCache(ctx context.Context, repoID uuid.UUID, filePath, toolName, contentHash, configHash string) (*ScanCache, error) {
	key := m.buildCacheKey(repoID, filePath, toolName, contentHash, configHash)
	
	if cache, exists := m.cache[key]; exists {
		// Check if cache is still valid (simple TTL check)
		if time.Since(cache.UpdatedAt) < 24*time.Hour {
			m.recordCacheHit(repoID)
			return cache, nil
		}
		// Cache expired, remove it
		delete(m.cache, key)
	}

	m.recordCacheMiss(repoID)
	return nil, nil // Cache miss
}

// SetCache stores a scan result in the cache
func (m *InMemoryCacheRepository) SetCache(ctx context.Context, cache *ScanCache) error {
	key := m.buildCacheKey(cache.RepositoryID, cache.FilePath, cache.ToolName, cache.ContentHash, cache.ConfigHash)
	m.cache[key] = cache

	// Update stats
	if _, exists := m.stats[cache.RepositoryID]; !exists {
		m.stats[cache.RepositoryID] = &CacheStats{}
	}
	m.stats[cache.RepositoryID].TotalEntries++

	return nil
}

// InvalidateCache removes cache entries for specific files
func (m *InMemoryCacheRepository) InvalidateCache(ctx context.Context, repoID uuid.UUID, filePaths []string) error {
	var deletedCount int64

	for key := range m.cache {
		for _, filePath := range filePaths {
			if m.keyMatchesFile(key, repoID, filePath) {
				delete(m.cache, key)
				deletedCount++
				break
			}
		}
	}

	// Update stats
	if stats, exists := m.stats[repoID]; exists {
		stats.TotalEntries -= deletedCount
		if stats.TotalEntries < 0 {
			stats.TotalEntries = 0
		}
	}

	return nil
}

// GetCacheStats returns cache statistics for a repository
func (m *InMemoryCacheRepository) GetCacheStats(ctx context.Context, repoID uuid.UUID) (*CacheStats, error) {
	if stats, exists := m.stats[repoID]; exists {
		return stats, nil
	}

	return &CacheStats{}, nil
}

// CleanupExpiredCache removes old cache entries
func (m *InMemoryCacheRepository) CleanupExpiredCache(ctx context.Context, maxAge time.Duration) error {
	cutoff := time.Now().Add(-maxAge)
	
	for key, cache := range m.cache {
		if cache.UpdatedAt.Before(cutoff) {
			delete(m.cache, key)
			
			// Update stats
			if stats, exists := m.stats[cache.RepositoryID]; exists {
				stats.TotalEntries--
				if stats.TotalEntries < 0 {
					stats.TotalEntries = 0
				}
			}
		}
	}

	return nil
}

// Helper methods for in-memory implementation

func (m *InMemoryCacheRepository) buildCacheKey(repoID uuid.UUID, filePath, toolName, contentHash, configHash string) string {
	return fmt.Sprintf("%s:%s:%s:%s:%s", repoID.String(), filePath, toolName, contentHash, configHash)
}

func (m *InMemoryCacheRepository) keyMatchesFile(key string, repoID uuid.UUID, filePath string) bool {
	prefix := fmt.Sprintf("%s:%s:", repoID.String(), filePath)
	return strings.HasPrefix(key, prefix)
}

func (m *InMemoryCacheRepository) recordCacheHit(repoID uuid.UUID) {
	if _, exists := m.stats[repoID]; !exists {
		m.stats[repoID] = &CacheStats{}
	}
	
	stats := m.stats[repoID]
	hits := stats.HitRate * float64(stats.TotalEntries)
	misses := stats.MissRate * float64(stats.TotalEntries)
	
	hits++
	total := hits + misses
	if total > 0 {
		stats.HitRate = hits / total
		stats.MissRate = misses / total
	}
}

func (m *InMemoryCacheRepository) recordCacheMiss(repoID uuid.UUID) {
	if _, exists := m.stats[repoID]; !exists {
		m.stats[repoID] = &CacheStats{}
	}
	
	stats := m.stats[repoID]
	hits := stats.HitRate * float64(stats.TotalEntries)
	misses := stats.MissRate * float64(stats.TotalEntries)
	
	misses++
	total := hits + misses
	if total > 0 {
		stats.HitRate = hits / total
		stats.MissRate = misses / total
	}
}