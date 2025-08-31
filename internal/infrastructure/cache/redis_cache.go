package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
)

// RedisCache implements a Redis-based cache
type RedisCache struct {
	client     *redis.Client
	defaultTTL time.Duration
	keyPrefix  string
}

// NewRedisCache creates a new Redis cache instance
func NewRedisCache(client *redis.Client, defaultTTL time.Duration, keyPrefix string) *RedisCache {
	return &RedisCache{
		client:     client,
		defaultTTL: defaultTTL,
		keyPrefix:  keyPrefix,
	}
}

// Get retrieves a value from the cache
func (c *RedisCache) Get(ctx context.Context, key string, dest interface{}) error {
	fullKey := c.keyPrefix + key
	
	val, err := c.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return ErrCacheMiss
		}
		return fmt.Errorf("failed to get cache value: %w", err)
	}
	
	if err := json.Unmarshal([]byte(val), dest); err != nil {
		return fmt.Errorf("failed to unmarshal cache value: %w", err)
	}
	
	return nil
}

// Set stores a value in the cache
func (c *RedisCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	fullKey := c.keyPrefix + key
	
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal cache value: %w", err)
	}
	
	if ttl == 0 {
		ttl = c.defaultTTL
	}
	
	if err := c.client.Set(ctx, fullKey, data, ttl).Err(); err != nil {
		return fmt.Errorf("failed to set cache value: %w", err)
	}
	
	return nil
}

// Delete removes a value from the cache
func (c *RedisCache) Delete(ctx context.Context, key string) error {
	fullKey := c.keyPrefix + key
	
	if err := c.client.Del(ctx, fullKey).Err(); err != nil {
		return fmt.Errorf("failed to delete cache value: %w", err)
	}
	
	return nil
}

// DeletePattern removes all keys matching a pattern
func (c *RedisCache) DeletePattern(ctx context.Context, pattern string) error {
	fullPattern := c.keyPrefix + pattern
	
	keys, err := c.client.Keys(ctx, fullPattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys for pattern: %w", err)
	}
	
	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to delete keys: %w", err)
		}
	}
	
	return nil
}

// Exists checks if a key exists in the cache
func (c *RedisCache) Exists(ctx context.Context, key string) (bool, error) {
	fullKey := c.keyPrefix + key
	
	count, err := c.client.Exists(ctx, fullKey).Result()
	if err != nil {
		return false, fmt.Errorf("failed to check key existence: %w", err)
	}
	
	return count > 0, nil
}

// TTL returns the time to live for a key
func (c *RedisCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	fullKey := c.keyPrefix + key
	
	ttl, err := c.client.TTL(ctx, fullKey).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to get TTL: %w", err)
	}
	
	return ttl, nil
}

// Increment increments a numeric value in the cache
func (c *RedisCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	fullKey := c.keyPrefix + key
	
	val, err := c.client.IncrBy(ctx, fullKey, delta).Result()
	if err != nil {
		return 0, fmt.Errorf("failed to increment value: %w", err)
	}
	
	return val, nil
}

// SetNX sets a value only if the key doesn't exist (atomic operation)
func (c *RedisCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	fullKey := c.keyPrefix + key
	
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal cache value: %w", err)
	}
	
	if ttl == 0 {
		ttl = c.defaultTTL
	}
	
	success, err := c.client.SetNX(ctx, fullKey, data, ttl).Result()
	if err != nil {
		return false, fmt.Errorf("failed to set cache value: %w", err)
	}
	
	return success, nil
}

// GetMultiple retrieves multiple values from the cache
func (c *RedisCache) GetMultiple(ctx context.Context, keys []string) (map[string]interface{}, error) {
	if len(keys) == 0 {
		return make(map[string]interface{}), nil
	}
	
	fullKeys := make([]string, len(keys))
	for i, key := range keys {
		fullKeys[i] = c.keyPrefix + key
	}
	
	values, err := c.client.MGet(ctx, fullKeys...).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get multiple cache values: %w", err)
	}
	
	result := make(map[string]interface{})
	for i, val := range values {
		if val != nil {
			var data interface{}
			if err := json.Unmarshal([]byte(val.(string)), &data); err == nil {
				result[keys[i]] = data
			}
		}
	}
	
	return result, nil
}

// SetMultiple stores multiple values in the cache
func (c *RedisCache) SetMultiple(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	if len(items) == 0 {
		return nil
	}
	
	pipe := c.client.Pipeline()
	
	for key, value := range items {
		fullKey := c.keyPrefix + key
		
		data, err := json.Marshal(value)
		if err != nil {
			return fmt.Errorf("failed to marshal cache value for key %s: %w", key, err)
		}
		
		if ttl == 0 {
			ttl = c.defaultTTL
		}
		
		pipe.Set(ctx, fullKey, data, ttl)
	}
	
	_, err := pipe.Exec(ctx)
	if err != nil {
		return fmt.Errorf("failed to set multiple cache values: %w", err)
	}
	
	return nil
}

// Clear removes all keys with the cache prefix
func (c *RedisCache) Clear(ctx context.Context) error {
	pattern := c.keyPrefix + "*"
	
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get keys for clearing: %w", err)
	}
	
	if len(keys) > 0 {
		if err := c.client.Del(ctx, keys...).Err(); err != nil {
			return fmt.Errorf("failed to clear cache: %w", err)
		}
	}
	
	return nil
}

// Stats returns cache statistics
func (c *RedisCache) Stats(ctx context.Context) (*CacheStats, error) {
	info, err := c.client.Info(ctx, "stats").Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache stats: %w", err)
	}
	
	stats := &CacheStats{}
	
	// Parse Redis INFO stats
	lines := strings.Split(info, "\r\n")
	for _, line := range lines {
		if strings.Contains(line, "keyspace_hits:") {
			fmt.Sscanf(line, "keyspace_hits:%d", &stats.Hits)
		} else if strings.Contains(line, "keyspace_misses:") {
			fmt.Sscanf(line, "keyspace_misses:%d", &stats.Misses)
		}
	}
	
	// Get key count for our prefix
	keys, err := c.client.Keys(ctx, c.keyPrefix+"*").Result()
	if err == nil {
		stats.Keys = int64(len(keys))
	}
	
	return stats, nil
}