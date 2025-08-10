package cache

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/agentscan/agentscan/internal/queue"
	"github.com/agentscan/agentscan/pkg/config"
)

func setupTestCache(t *testing.T) *Service {
	// Setup test Redis client
	redisConfig := &config.RedisConfig{
		Host:     "localhost",
		Port:     6379,
		Password: "",
		DB:       1, // Use different DB for tests
		PoolSize: 5,
	}

	redisClient, err := queue.NewRedisClient(redisConfig)
	require.NoError(t, err)

	// Clear test database
	err = redisClient.FlushDB(context.Background())
	require.NoError(t, err)

	return NewService(redisClient, DefaultConfig())
}

func TestCacheService_SetAndGet(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	key := CacheKey{Prefix: "test", ID: "123"}
	value := map[string]interface{}{
		"name": "test",
		"age":  30,
	}

	// Test Set
	err := cache.Set(ctx, key, value, 1*time.Minute)
	assert.NoError(t, err)

	// Test Get
	var result map[string]interface{}
	err = cache.Get(ctx, key, &result)
	assert.NoError(t, err)
	assert.Equal(t, "test", result["name"])
	assert.Equal(t, float64(30), result["age"]) // JSON unmarshaling converts to float64
}

func TestCacheService_Exists(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	key := CacheKey{Prefix: "test", ID: "exists"}

	// Test non-existent key
	exists, err := cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.False(t, exists)

	// Set value
	err = cache.Set(ctx, key, "test value", 1*time.Minute)
	assert.NoError(t, err)

	// Test existing key
	exists, err = cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.True(t, exists)
}

func TestCacheService_Delete(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	key := CacheKey{Prefix: "test", ID: "delete"}

	// Set value
	err := cache.Set(ctx, key, "test value", 1*time.Minute)
	assert.NoError(t, err)

	// Verify exists
	exists, err := cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.True(t, exists)

	// Delete
	err = cache.Delete(ctx, key)
	assert.NoError(t, err)

	// Verify deleted
	exists, err = cache.Exists(ctx, key)
	assert.NoError(t, err)
	assert.False(t, exists)
}

func TestCacheService_Hash(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	key := CacheKey{Prefix: "test", ID: "hash"}
	field := "field1"
	value := map[string]string{"key": "value"}

	// Test SetHash
	err := cache.SetHash(ctx, key, field, value, 1*time.Minute)
	assert.NoError(t, err)

	// Test GetHash
	var result map[string]string
	err = cache.GetHash(ctx, key, field, &result)
	assert.NoError(t, err)
	assert.Equal(t, "value", result["key"])

	// Test GetAllHash
	allFields, err := cache.GetAllHash(ctx, key)
	assert.NoError(t, err)
	assert.Contains(t, allFields, field)
}

func TestCacheService_List(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	key := CacheKey{Prefix: "test", ID: "list"}
	values := []interface{}{"item1", "item2", "item3"}

	// Test SetList
	err := cache.SetList(ctx, key, values, 1*time.Minute)
	assert.NoError(t, err)

	// Test GetList
	var result []string
	err = cache.GetList(ctx, key, &result)
	assert.NoError(t, err)
	assert.Len(t, result, 3)
	assert.Contains(t, result, "item1")
	assert.Contains(t, result, "item2")
	assert.Contains(t, result, "item3")
}

func TestCacheService_Counter(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	key := CacheKey{Prefix: "test", ID: "counter"}

	// Test Increment
	count, err := cache.Increment(ctx, key, 5, 1*time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, int64(5), count)

	// Test Increment again
	count, err = cache.Increment(ctx, key, 3, 1*time.Minute)
	assert.NoError(t, err)
	assert.Equal(t, int64(8), count)

	// Test GetCounter
	count, err = cache.GetCounter(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, int64(8), count)
}

func TestCacheService_TTL(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	key := CacheKey{Prefix: "test", ID: "ttl"}
	ttl := 30 * time.Second

	// Set with TTL
	err := cache.Set(ctx, key, "test value", ttl)
	assert.NoError(t, err)

	// Check TTL
	remainingTTL, err := cache.TTL(ctx, key)
	assert.NoError(t, err)
	assert.True(t, remainingTTL > 0)
	assert.True(t, remainingTTL <= ttl)

	// Extend TTL
	newTTL := 1 * time.Minute
	err = cache.Extend(ctx, key, newTTL)
	assert.NoError(t, err)

	// Check extended TTL
	remainingTTL, err = cache.TTL(ctx, key)
	assert.NoError(t, err)
	assert.True(t, remainingTTL > ttl) // Should be longer than original
}

func TestCacheService_InvalidatePattern(t *testing.T) {
	cache := setupTestCache(t)
	ctx := context.Background()

	// Set multiple keys with same prefix
	keys := []CacheKey{
		{Prefix: "test", ID: "pattern1"},
		{Prefix: "test", ID: "pattern2"},
		{Prefix: "other", ID: "pattern3"},
	}

	for _, key := range keys {
		err := cache.Set(ctx, key, "test value", 1*time.Minute)
		assert.NoError(t, err)
	}

	// Invalidate pattern
	err := cache.InvalidatePattern(ctx, "test:*")
	assert.NoError(t, err)

	// Check that test keys are deleted
	exists, err := cache.Exists(ctx, keys[0])
	assert.NoError(t, err)
	assert.False(t, exists)

	exists, err = cache.Exists(ctx, keys[1])
	assert.NoError(t, err)
	assert.False(t, exists)

	// Check that other key still exists
	exists, err = cache.Exists(ctx, keys[2])
	assert.NoError(t, err)
	assert.True(t, exists)
}

func BenchmarkCacheService_Set(b *testing.B) {
	cache := setupTestCache(&testing.T{})
	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		key := CacheKey{Prefix: "bench", ID: string(rune(i))}
		cache.Set(ctx, key, "benchmark value", 1*time.Minute)
	}
}

func BenchmarkCacheService_Get(b *testing.B) {
	cache := setupTestCache(&testing.T{})
	ctx := context.Background()

	// Pre-populate cache
	key := CacheKey{Prefix: "bench", ID: "get"}
	cache.Set(ctx, key, "benchmark value", 1*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var result string
		cache.Get(ctx, key, &result)
	}
}

func BenchmarkCacheService_Increment(b *testing.B) {
	cache := setupTestCache(&testing.T{})
	ctx := context.Background()

	key := CacheKey{Prefix: "bench", ID: "counter"}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Increment(ctx, key, 1, 1*time.Minute)
	}
}