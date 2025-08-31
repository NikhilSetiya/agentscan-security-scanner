package cache

import (
	"context"
	"errors"
	"strings"
	"time"
)

// ErrCacheMiss is returned when a cache key is not found
var ErrCacheMiss = errors.New("cache miss")

// Cache defines the interface for cache operations
type Cache interface {
	// Get retrieves a value from the cache
	Get(ctx context.Context, key string, dest interface{}) error
	
	// Set stores a value in the cache with TTL
	Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error
	
	// Delete removes a value from the cache
	Delete(ctx context.Context, key string) error
	
	// DeletePattern removes all keys matching a pattern
	DeletePattern(ctx context.Context, pattern string) error
	
	// Exists checks if a key exists in the cache
	Exists(ctx context.Context, key string) (bool, error)
	
	// TTL returns the time to live for a key
	TTL(ctx context.Context, key string) (time.Duration, error)
	
	// Increment increments a numeric value
	Increment(ctx context.Context, key string, delta int64) (int64, error)
	
	// SetNX sets a value only if the key doesn't exist
	SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error)
	
	// GetMultiple retrieves multiple values
	GetMultiple(ctx context.Context, keys []string) (map[string]interface{}, error)
	
	// SetMultiple stores multiple values
	SetMultiple(ctx context.Context, items map[string]interface{}, ttl time.Duration) error
	
	// Clear removes all keys with the cache prefix
	Clear(ctx context.Context) error
	
	// Stats returns cache statistics
	Stats(ctx context.Context) (*CacheStats, error)
}

// CacheStats represents cache statistics
type CacheStats struct {
	Hits   int64 `json:"hits"`
	Misses int64 `json:"misses"`
	Keys   int64 `json:"keys"`
}

// HitRate calculates the cache hit rate
func (s *CacheStats) HitRate() float64 {
	total := s.Hits + s.Misses
	if total == 0 {
		return 0
	}
	return float64(s.Hits) / float64(total)
}

// MemoryCache implements an in-memory cache
type MemoryCache struct {
	data      map[string]*cacheItem
	defaultTTL time.Duration
}

type cacheItem struct {
	value     interface{}
	expiresAt time.Time
}

// NewMemoryCache creates a new in-memory cache
func NewMemoryCache(defaultTTL time.Duration) *MemoryCache {
	return &MemoryCache{
		data:       make(map[string]*cacheItem),
		defaultTTL: defaultTTL,
	}
}

// Get retrieves a value from the memory cache
func (c *MemoryCache) Get(ctx context.Context, key string, dest interface{}) error {
	item, exists := c.data[key]
	if !exists {
		return ErrCacheMiss
	}
	
	if time.Now().After(item.expiresAt) {
		delete(c.data, key)
		return ErrCacheMiss
	}
	
	// Simple assignment for interface{} types
	switch v := dest.(type) {
	case *interface{}:
		*v = item.value
	case *string:
		if str, ok := item.value.(string); ok {
			*v = str
		} else {
			return errors.New("type mismatch: expected string")
		}
	case *int:
		if i, ok := item.value.(int); ok {
			*v = i
		} else {
			return errors.New("type mismatch: expected int")
		}
	case *int64:
		if i, ok := item.value.(int64); ok {
			*v = i
		} else {
			return errors.New("type mismatch: expected int64")
		}
	default:
		return errors.New("unsupported destination type")
	}
	
	return nil
}

// Set stores a value in the memory cache
func (c *MemoryCache) Set(ctx context.Context, key string, value interface{}, ttl time.Duration) error {
	if ttl == 0 {
		ttl = c.defaultTTL
	}
	
	c.data[key] = &cacheItem{
		value:     value,
		expiresAt: time.Now().Add(ttl),
	}
	
	return nil
}

// Delete removes a value from the memory cache
func (c *MemoryCache) Delete(ctx context.Context, key string) error {
	delete(c.data, key)
	return nil
}

// DeletePattern removes all keys matching a pattern
func (c *MemoryCache) DeletePattern(ctx context.Context, pattern string) error {
	// Simple pattern matching with wildcards
	for key := range c.data {
		if matchPattern(key, pattern) {
			delete(c.data, key)
		}
	}
	return nil
}

// Exists checks if a key exists in the memory cache
func (c *MemoryCache) Exists(ctx context.Context, key string) (bool, error) {
	item, exists := c.data[key]
	if !exists {
		return false, nil
	}
	
	if time.Now().After(item.expiresAt) {
		delete(c.data, key)
		return false, nil
	}
	
	return true, nil
}

// TTL returns the time to live for a key
func (c *MemoryCache) TTL(ctx context.Context, key string) (time.Duration, error) {
	item, exists := c.data[key]
	if !exists {
		return -2 * time.Second, nil // Key doesn't exist
	}
	
	ttl := time.Until(item.expiresAt)
	if ttl < 0 {
		delete(c.data, key)
		return -2 * time.Second, nil // Key expired
	}
	
	return ttl, nil
}

// Increment increments a numeric value
func (c *MemoryCache) Increment(ctx context.Context, key string, delta int64) (int64, error) {
	item, exists := c.data[key]
	if !exists {
		c.data[key] = &cacheItem{
			value:     delta,
			expiresAt: time.Now().Add(c.defaultTTL),
		}
		return delta, nil
	}
	
	if time.Now().After(item.expiresAt) {
		delete(c.data, key)
		c.data[key] = &cacheItem{
			value:     delta,
			expiresAt: time.Now().Add(c.defaultTTL),
		}
		return delta, nil
	}
	
	if val, ok := item.value.(int64); ok {
		newVal := val + delta
		item.value = newVal
		return newVal, nil
	}
	
	return 0, errors.New("value is not numeric")
}

// SetNX sets a value only if the key doesn't exist
func (c *MemoryCache) SetNX(ctx context.Context, key string, value interface{}, ttl time.Duration) (bool, error) {
	exists, err := c.Exists(ctx, key)
	if err != nil {
		return false, err
	}
	
	if exists {
		return false, nil
	}
	
	return true, c.Set(ctx, key, value, ttl)
}

// GetMultiple retrieves multiple values
func (c *MemoryCache) GetMultiple(ctx context.Context, keys []string) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	for _, key := range keys {
		var value interface{}
		if err := c.Get(ctx, key, &value); err == nil {
			result[key] = value
		}
	}
	
	return result, nil
}

// SetMultiple stores multiple values
func (c *MemoryCache) SetMultiple(ctx context.Context, items map[string]interface{}, ttl time.Duration) error {
	for key, value := range items {
		if err := c.Set(ctx, key, value, ttl); err != nil {
			return err
		}
	}
	return nil
}

// Clear removes all keys
func (c *MemoryCache) Clear(ctx context.Context) error {
	c.data = make(map[string]*cacheItem)
	return nil
}

// Stats returns cache statistics
func (c *MemoryCache) Stats(ctx context.Context) (*CacheStats, error) {
	// Clean up expired items first
	now := time.Now()
	validKeys := 0
	
	for key, item := range c.data {
		if now.After(item.expiresAt) {
			delete(c.data, key)
		} else {
			validKeys++
		}
	}
	
	return &CacheStats{
		Keys: int64(validKeys),
		// Memory cache doesn't track hits/misses
	}, nil
}

// matchPattern performs simple pattern matching with wildcards
func matchPattern(text, pattern string) bool {
	if pattern == "*" {
		return true
	}
	
	if strings.Contains(pattern, "*") {
		parts := strings.Split(pattern, "*")
		if len(parts) == 2 {
			prefix := parts[0]
			suffix := parts[1]
			return strings.HasPrefix(text, prefix) && strings.HasSuffix(text, suffix)
		}
	}
	
	return text == pattern
}