package cache

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/agentscan/agentscan/internal/queue"
	"github.com/agentscan/agentscan/pkg/errors"
)

// Service provides caching functionality for frequently accessed data
type Service struct {
	redis  *queue.RedisClient
	config *Config
}

// Config holds cache configuration
type Config struct {
	DefaultTTL       time.Duration `json:"default_ttl"`
	ScanResultTTL    time.Duration `json:"scan_result_ttl"`
	FindingsTTL      time.Duration `json:"findings_ttl"`
	UserSessionTTL   time.Duration `json:"user_session_ttl"`
	RepositoryTTL    time.Duration `json:"repository_ttl"`
	EnableCompression bool         `json:"enable_compression"`
}

// DefaultConfig returns default cache configuration
func DefaultConfig() *Config {
	return &Config{
		DefaultTTL:       1 * time.Hour,
		ScanResultTTL:    24 * time.Hour,
		FindingsTTL:      6 * time.Hour,
		UserSessionTTL:   8 * time.Hour,
		RepositoryTTL:    30 * time.Minute,
		EnableCompression: true,
	}
}

// NewService creates a new cache service
func NewService(redis *queue.RedisClient, config *Config) *Service {
	if config == nil {
		config = DefaultConfig()
	}

	return &Service{
		redis:  redis,
		config: config,
	}
}

// CacheKey generates cache keys with consistent prefixes
type CacheKey struct {
	Prefix string
	ID     string
}

// String returns the formatted cache key
func (ck CacheKey) String() string {
	return fmt.Sprintf("%s:%s", ck.Prefix, ck.ID)
}

// Cache key prefixes
const (
	PrefixScanResult   = "scan_result"
	PrefixFindings     = "findings"
	PrefixUserSession  = "user_session"
	PrefixRepository   = "repository"
	PrefixScanStatus   = "scan_status"
	PrefixAgentHealth  = "agent_health"
	PrefixConsensus    = "consensus"
	PrefixStatistics   = "statistics"
)

// Set stores a value in cache with the specified TTL
func (s *Service) Set(ctx context.Context, key CacheKey, value interface{}, ttl time.Duration) error {
	data, err := s.serialize(value)
	if err != nil {
		return errors.NewInternalError("failed to serialize cache value").WithCause(err)
	}

	if ttl == 0 {
		ttl = s.config.DefaultTTL
	}

	if err := s.redis.Set(ctx, key.String(), data, ttl); err != nil {
		return errors.NewInternalError("failed to set cache value").WithCause(err)
	}

	return nil
}

// Get retrieves a value from cache
func (s *Service) Get(ctx context.Context, key CacheKey, dest interface{}) error {
	data, err := s.redis.Get(ctx, key.String())
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.NewNotFoundError("cache key")
		}
		return errors.NewInternalError("failed to get cache value").WithCause(err)
	}

	if err := s.deserialize(data, dest); err != nil {
		return errors.NewInternalError("failed to deserialize cache value").WithCause(err)
	}

	return nil
}

// Delete removes a value from cache
func (s *Service) Delete(ctx context.Context, key CacheKey) error {
	_, err := s.redis.Del(ctx, key.String())
	if err != nil {
		return errors.NewInternalError("failed to delete cache key").WithCause(err)
	}
	return nil
}

// Exists checks if a key exists in cache
func (s *Service) Exists(ctx context.Context, key CacheKey) (bool, error) {
	count, err := s.redis.Exists(ctx, key.String())
	if err != nil {
		return false, errors.NewInternalError("failed to check cache key existence").WithCause(err)
	}
	return count > 0, nil
}

// SetHash stores a hash in cache
func (s *Service) SetHash(ctx context.Context, key CacheKey, field string, value interface{}, ttl time.Duration) error {
	data, err := s.serialize(value)
	if err != nil {
		return errors.NewInternalError("failed to serialize hash value").WithCause(err)
	}

	if err := s.redis.HSet(ctx, key.String(), field, data); err != nil {
		return errors.NewInternalError("failed to set hash field").WithCause(err)
	}

	if ttl > 0 {
		if err := s.redis.Expire(ctx, key.String(), ttl); err != nil {
			return errors.NewInternalError("failed to set hash expiration").WithCause(err)
		}
	}

	return nil
}

// GetHash retrieves a hash field from cache
func (s *Service) GetHash(ctx context.Context, key CacheKey, field string, dest interface{}) error {
	data, err := s.redis.HGet(ctx, key.String(), field)
	if err != nil {
		if errors.IsNotFound(err) {
			return errors.NewNotFoundError("hash field")
		}
		return errors.NewInternalError("failed to get hash field").WithCause(err)
	}

	if err := s.deserialize(data, dest); err != nil {
		return errors.NewInternalError("failed to deserialize hash value").WithCause(err)
	}

	return nil
}

// GetAllHash retrieves all fields from a hash
func (s *Service) GetAllHash(ctx context.Context, key CacheKey) (map[string]string, error) {
	data, err := s.redis.HGetAll(ctx, key.String())
	if err != nil {
		return nil, errors.NewInternalError("failed to get hash").WithCause(err)
	}
	return data, nil
}

// SetList adds items to a list
func (s *Service) SetList(ctx context.Context, key CacheKey, values []interface{}, ttl time.Duration) error {
	// Clear existing list
	s.redis.Del(ctx, key.String())

	if len(values) == 0 {
		return nil
	}

	serializedValues := make([]interface{}, len(values))
	for i, value := range values {
		data, err := s.serialize(value)
		if err != nil {
			return errors.NewInternalError("failed to serialize list value").WithCause(err)
		}
		serializedValues[i] = data
	}

	if err := s.redis.LPush(ctx, key.String(), serializedValues...); err != nil {
		return errors.NewInternalError("failed to set list").WithCause(err)
	}

	if ttl > 0 {
		if err := s.redis.Expire(ctx, key.String(), ttl); err != nil {
			return errors.NewInternalError("failed to set list expiration").WithCause(err)
		}
	}

	return nil
}

// GetList retrieves all items from a list
func (s *Service) GetList(ctx context.Context, key CacheKey, dest interface{}) error {
	length, err := s.redis.LLen(ctx, key.String())
	if err != nil {
		return errors.NewInternalError("failed to get list length").WithCause(err)
	}

	if length == 0 {
		return errors.NewNotFoundError("list")
	}

	// Get all items from the list
	items := make([]string, 0, length)
	for i := int64(0); i < length; i++ {
		item, err := s.redis.Client().LIndex(ctx, key.String(), i).Result()
		if err != nil {
			if err == redis.Nil {
				break
			}
			return errors.NewInternalError("failed to get list item").WithCause(err)
		}
		items = append(items, item)
	}

	// Deserialize the entire list
	if err := s.deserialize(fmt.Sprintf("[%s]", joinStrings(items, ",")), dest); err != nil {
		return errors.NewInternalError("failed to deserialize list").WithCause(err)
	}

	return nil
}

// Increment atomically increments a counter
func (s *Service) Increment(ctx context.Context, key CacheKey, delta int64, ttl time.Duration) (int64, error) {
	result, err := s.redis.Client().IncrBy(ctx, key.String(), delta).Result()
	if err != nil {
		return 0, errors.NewInternalError("failed to increment counter").WithCause(err)
	}

	if ttl > 0 {
		if err := s.redis.Expire(ctx, key.String(), ttl); err != nil {
			return result, errors.NewInternalError("failed to set counter expiration").WithCause(err)
		}
	}

	return result, nil
}

// GetCounter retrieves a counter value
func (s *Service) GetCounter(ctx context.Context, key CacheKey) (int64, error) {
	result, err := s.redis.Client().Get(ctx, key.String()).Int64()
	if err != nil {
		if err == redis.Nil {
			return 0, nil
		}
		return 0, errors.NewInternalError("failed to get counter").WithCause(err)
	}
	return result, nil
}

// InvalidatePattern removes all keys matching a pattern
func (s *Service) InvalidatePattern(ctx context.Context, pattern string) error {
	keys, err := s.redis.Keys(ctx, pattern)
	if err != nil {
		return errors.NewInternalError("failed to get keys for pattern").WithCause(err)
	}

	if len(keys) == 0 {
		return nil
	}

	_, err = s.redis.Del(ctx, keys...)
	if err != nil {
		return errors.NewInternalError("failed to delete keys").WithCause(err)
	}

	return nil
}

// TTL returns the time to live for a key
func (s *Service) TTL(ctx context.Context, key CacheKey) (time.Duration, error) {
	ttl, err := s.redis.TTL(ctx, key.String())
	if err != nil {
		return 0, errors.NewInternalError("failed to get TTL").WithCause(err)
	}
	return ttl, nil
}

// Extend extends the TTL of a key
func (s *Service) Extend(ctx context.Context, key CacheKey, ttl time.Duration) error {
	if err := s.redis.Expire(ctx, key.String(), ttl); err != nil {
		return errors.NewInternalError("failed to extend TTL").WithCause(err)
	}
	return nil
}

// serialize converts a value to JSON bytes
func (s *Service) serialize(value interface{}) (string, error) {
	if str, ok := value.(string); ok {
		return str, nil
	}

	data, err := json.Marshal(value)
	if err != nil {
		return "", err
	}

	return string(data), nil
}

// deserialize converts JSON bytes to a value
func (s *Service) deserialize(data string, dest interface{}) error {
	if str, ok := dest.(*string); ok {
		*str = data
		return nil
	}

	return json.Unmarshal([]byte(data), dest)
}

// Helper function to join strings
func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	if len(strs) == 1 {
		return fmt.Sprintf(`"%s"`, strs[0])
	}

	result := fmt.Sprintf(`"%s"`, strs[0])
	for i := 1; i < len(strs); i++ {
		result += sep + fmt.Sprintf(`"%s"`, strs[i])
	}
	return result
}