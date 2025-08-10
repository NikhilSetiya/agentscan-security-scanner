package incremental

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInMemoryCacheRepository_GetCache_Miss(t *testing.T) {
	cache := NewInMemoryCacheRepository()
	ctx := context.Background()

	repoID := uuid.New()
	result, err := cache.GetCache(ctx, repoID, "main.go", "semgrep", "hash123", "config456")

	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestInMemoryCacheRepository_SetAndGetCache(t *testing.T) {
	cache := NewInMemoryCacheRepository()
	ctx := context.Background()

	repoID := uuid.New()
	scanCache := &ScanCache{
		ID:           uuid.New(),
		RepositoryID: repoID,
		FilePath:     "main.go",
		ContentHash:  "hash123",
		ToolName:     "semgrep",
		ToolVersion:  "1.0.0",
		ConfigHash:   "config456",
		Results:      map[string]interface{}{"findings": []interface{}{}},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Set cache
	err := cache.SetCache(ctx, scanCache)
	require.NoError(t, err)

	// Get cache
	result, err := cache.GetCache(ctx, repoID, "main.go", "semgrep", "hash123", "config456")
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, scanCache.ID, result.ID)
	assert.Equal(t, scanCache.FilePath, result.FilePath)
	assert.Equal(t, scanCache.ToolName, result.ToolName)
	assert.Equal(t, scanCache.ContentHash, result.ContentHash)
}

func TestInMemoryCacheRepository_GetCache_Expired(t *testing.T) {
	cache := NewInMemoryCacheRepository()
	ctx := context.Background()

	repoID := uuid.New()
	scanCache := &ScanCache{
		ID:           uuid.New(),
		RepositoryID: repoID,
		FilePath:     "main.go",
		ContentHash:  "hash123",
		ToolName:     "semgrep",
		ToolVersion:  "1.0.0",
		ConfigHash:   "config456",
		Results:      map[string]interface{}{"findings": []interface{}{}},
		CreatedAt:    time.Now().Add(-25 * time.Hour), // Expired
		UpdatedAt:    time.Now().Add(-25 * time.Hour), // Expired
	}

	// Set expired cache directly
	key := cache.buildCacheKey(repoID, "main.go", "semgrep", "hash123", "config456")
	cache.cache[key] = scanCache

	// Try to get expired cache
	result, err := cache.GetCache(ctx, repoID, "main.go", "semgrep", "hash123", "config456")
	require.NoError(t, err)
	assert.Nil(t, result) // Should return nil for expired cache

	// Verify cache was cleaned up
	_, exists := cache.cache[key]
	assert.False(t, exists)
}

func TestInMemoryCacheRepository_InvalidateCache(t *testing.T) {
	cache := NewInMemoryCacheRepository()
	ctx := context.Background()

	repoID := uuid.New()
	
	// Set multiple cache entries
	caches := []*ScanCache{
		{
			ID:           uuid.New(),
			RepositoryID: repoID,
			FilePath:     "main.go",
			ContentHash:  "hash1",
			ToolName:     "semgrep",
			ToolVersion:  "1.0.0",
			ConfigHash:   "config1",
			Results:      map[string]interface{}{},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:           uuid.New(),
			RepositoryID: repoID,
			FilePath:     "utils.go",
			ContentHash:  "hash2",
			ToolName:     "semgrep",
			ToolVersion:  "1.0.0",
			ConfigHash:   "config1",
			Results:      map[string]interface{}{},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
		{
			ID:           uuid.New(),
			RepositoryID: repoID,
			FilePath:     "main.go",
			ContentHash:  "hash1",
			ToolName:     "eslint",
			ToolVersion:  "8.0.0",
			ConfigHash:   "config2",
			Results:      map[string]interface{}{},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}

	for _, c := range caches {
		err := cache.SetCache(ctx, c)
		require.NoError(t, err)
	}

	// Verify all caches are set
	assert.Len(t, cache.cache, 3)

	// Invalidate cache for main.go
	err := cache.InvalidateCache(ctx, repoID, []string{"main.go"})
	require.NoError(t, err)

	// Verify main.go caches are removed but utils.go remains
	result1, err := cache.GetCache(ctx, repoID, "main.go", "semgrep", "hash1", "config1")
	require.NoError(t, err)
	assert.Nil(t, result1)

	result2, err := cache.GetCache(ctx, repoID, "main.go", "eslint", "hash1", "config2")
	require.NoError(t, err)
	assert.Nil(t, result2)

	result3, err := cache.GetCache(ctx, repoID, "utils.go", "semgrep", "hash2", "config1")
	require.NoError(t, err)
	assert.NotNil(t, result3)
}

func TestInMemoryCacheRepository_GetCacheStats(t *testing.T) {
	cache := NewInMemoryCacheRepository()
	ctx := context.Background()

	repoID := uuid.New()

	// Initially no stats
	stats, err := cache.GetCacheStats(ctx, repoID)
	require.NoError(t, err)
	assert.Equal(t, int64(0), stats.TotalEntries)

	// Add some cache entries
	scanCache := &ScanCache{
		ID:           uuid.New(),
		RepositoryID: repoID,
		FilePath:     "main.go",
		ContentHash:  "hash123",
		ToolName:     "semgrep",
		ToolVersion:  "1.0.0",
		ConfigHash:   "config456",
		Results:      map[string]interface{}{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	err = cache.SetCache(ctx, scanCache)
	require.NoError(t, err)

	// Check stats updated
	stats, err = cache.GetCacheStats(ctx, repoID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), stats.TotalEntries)
}

func TestInMemoryCacheRepository_CleanupExpiredCache(t *testing.T) {
	cache := NewInMemoryCacheRepository()
	ctx := context.Background()

	repoID := uuid.New()

	// Add fresh cache entry
	freshCache := &ScanCache{
		ID:           uuid.New(),
		RepositoryID: repoID,
		FilePath:     "fresh.go",
		ContentHash:  "hash1",
		ToolName:     "semgrep",
		ToolVersion:  "1.0.0",
		ConfigHash:   "config1",
		Results:      map[string]interface{}{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}

	// Add expired cache entry
	expiredCache := &ScanCache{
		ID:           uuid.New(),
		RepositoryID: repoID,
		FilePath:     "expired.go",
		ContentHash:  "hash2",
		ToolName:     "semgrep",
		ToolVersion:  "1.0.0",
		ConfigHash:   "config1",
		Results:      map[string]interface{}{},
		CreatedAt:    time.Now().Add(-25 * time.Hour),
		UpdatedAt:    time.Now().Add(-25 * time.Hour),
	}

	err := cache.SetCache(ctx, freshCache)
	require.NoError(t, err)

	// Set expired cache directly
	expiredKey := cache.buildCacheKey(repoID, "expired.go", "semgrep", "hash2", "config1")
	cache.cache[expiredKey] = expiredCache

	// Verify both caches exist
	assert.Len(t, cache.cache, 2)

	// Cleanup expired cache
	err = cache.CleanupExpiredCache(ctx, 24*time.Hour)
	require.NoError(t, err)

	// Verify only fresh cache remains
	assert.Len(t, cache.cache, 1)

	// Verify fresh cache is still accessible
	result, err := cache.GetCache(ctx, repoID, "fresh.go", "semgrep", "hash1", "config1")
	require.NoError(t, err)
	assert.NotNil(t, result)

	// Verify expired cache is gone
	result, err = cache.GetCache(ctx, repoID, "expired.go", "semgrep", "hash2", "config1")
	require.NoError(t, err)
	assert.Nil(t, result)
}

func TestInMemoryCacheRepository_buildCacheKey(t *testing.T) {
	cache := NewInMemoryCacheRepository()
	repoID := uuid.New()

	key1 := cache.buildCacheKey(repoID, "main.go", "semgrep", "hash1", "config1")
	key2 := cache.buildCacheKey(repoID, "main.go", "semgrep", "hash2", "config1") // Different content hash
	key3 := cache.buildCacheKey(repoID, "main.go", "eslint", "hash1", "config1") // Different tool
	key4 := cache.buildCacheKey(repoID, "utils.go", "semgrep", "hash1", "config1") // Different file

	// All keys should be different
	assert.NotEqual(t, key1, key2)
	assert.NotEqual(t, key1, key3)
	assert.NotEqual(t, key1, key4)
	assert.NotEqual(t, key2, key3)
	assert.NotEqual(t, key2, key4)
	assert.NotEqual(t, key3, key4)

	// Same parameters should produce same key
	key1Duplicate := cache.buildCacheKey(repoID, "main.go", "semgrep", "hash1", "config1")
	assert.Equal(t, key1, key1Duplicate)
}

func TestInMemoryCacheRepository_keyMatchesFile(t *testing.T) {
	cache := NewInMemoryCacheRepository()
	repoID := uuid.New()

	tests := []struct {
		name     string
		key      string
		filePath string
		expected bool
	}{
		{
			name:     "exact match",
			key:      cache.buildCacheKey(repoID, "main.go", "semgrep", "hash1", "config1"),
			filePath: "main.go",
			expected: true,
		},
		{
			name:     "different file",
			key:      cache.buildCacheKey(repoID, "main.go", "semgrep", "hash1", "config1"),
			filePath: "utils.go",
			expected: false,
		},
		{
			name:     "different repo",
			key:      cache.buildCacheKey(uuid.New(), "main.go", "semgrep", "hash1", "config1"),
			filePath: "main.go",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := cache.keyMatchesFile(tt.key, repoID, tt.filePath)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestInMemoryCacheRepository_ConcurrentAccess(t *testing.T) {
	cache := NewInMemoryCacheRepository()
	ctx := context.Background()
	repoID := uuid.New()

	// Test concurrent reads and writes
	done := make(chan bool, 10)

	// Start multiple goroutines writing to cache
	for i := 0; i < 5; i++ {
		go func(index int) {
			scanCache := &ScanCache{
				ID:           uuid.New(),
				RepositoryID: repoID,
				FilePath:     fmt.Sprintf("file%d.go", index),
				ContentHash:  fmt.Sprintf("hash%d", index),
				ToolName:     "semgrep",
				ToolVersion:  "1.0.0",
				ConfigHash:   "config1",
				Results:      map[string]interface{}{},
				CreatedAt:    time.Now(),
				UpdatedAt:    time.Now(),
			}
			
			err := cache.SetCache(ctx, scanCache)
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Start multiple goroutines reading from cache
	for i := 0; i < 5; i++ {
		go func(index int) {
			_, err := cache.GetCache(ctx, repoID, fmt.Sprintf("file%d.go", index), "semgrep", fmt.Sprintf("hash%d", index), "config1")
			assert.NoError(t, err)
			done <- true
		}(i)
	}

	// Wait for all goroutines to complete
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify final state
	stats, err := cache.GetCacheStats(ctx, repoID)
	require.NoError(t, err)
	assert.Equal(t, int64(5), stats.TotalEntries)
}

// Benchmark tests
func BenchmarkInMemoryCacheRepository_SetCache(b *testing.B) {
	cache := NewInMemoryCacheRepository()
	ctx := context.Background()
	repoID := uuid.New()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		scanCache := &ScanCache{
			ID:           uuid.New(),
			RepositoryID: repoID,
			FilePath:     fmt.Sprintf("file%d.go", i),
			ContentHash:  fmt.Sprintf("hash%d", i),
			ToolName:     "semgrep",
			ToolVersion:  "1.0.0",
			ConfigHash:   "config1",
			Results:      map[string]interface{}{},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		
		err := cache.SetCache(ctx, scanCache)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkInMemoryCacheRepository_GetCache(b *testing.B) {
	cache := NewInMemoryCacheRepository()
	ctx := context.Background()
	repoID := uuid.New()

	// Pre-populate cache
	scanCache := &ScanCache{
		ID:           uuid.New(),
		RepositoryID: repoID,
		FilePath:     "main.go",
		ContentHash:  "hash123",
		ToolName:     "semgrep",
		ToolVersion:  "1.0.0",
		ConfigHash:   "config1",
		Results:      map[string]interface{}{},
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	cache.SetCache(ctx, scanCache)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := cache.GetCache(ctx, repoID, "main.go", "semgrep", "hash123", "config1")
		if err != nil {
			b.Fatal(err)
		}
	}
}