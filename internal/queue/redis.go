package queue

import (
	"context"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/agentscan/agentscan/pkg/config"
	"github.com/agentscan/agentscan/pkg/errors"
)

// RedisClient wraps the Redis client with additional functionality
type RedisClient struct {
	client *redis.Client
	config *config.RedisConfig
}

// NewRedisClient creates a new Redis client
func NewRedisClient(cfg *config.RedisConfig) (*RedisClient, error) {
	if cfg == nil {
		return nil, errors.NewValidationError("Redis configuration is required")
	}

	// Create Redis client options
	opts := &redis.Options{
		Addr:     fmt.Sprintf("%s:%d", cfg.Host, cfg.Port),
		Password: cfg.Password,
		DB:       cfg.DB,
		PoolSize: cfg.PoolSize,

		// Connection timeouts
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,

		// Pool timeouts
		PoolTimeout:     4 * time.Second,
		ConnMaxIdleTime: 5 * time.Minute,

		// Retry configuration
		MaxRetries:      3,
		MinRetryBackoff: 8 * time.Millisecond,
		MaxRetryBackoff: 512 * time.Millisecond,
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		client.Close()
		return nil, errors.NewInternalError("failed to connect to Redis").WithCause(err)
	}

	return &RedisClient{
		client: client,
		config: cfg,
	}, nil
}

// Close closes the Redis connection
func (r *RedisClient) Close() error {
	if r.client != nil {
		return r.client.Close()
	}
	return nil
}

// Health checks the Redis connection health
func (r *RedisClient) Health(ctx context.Context) error {
	if r.client == nil {
		return errors.NewInternalError("Redis client is nil")
	}

	if err := r.client.Ping(ctx).Err(); err != nil {
		return errors.NewInternalError("Redis health check failed").WithCause(err)
	}

	return nil
}

// Client returns the underlying Redis client
func (r *RedisClient) Client() *redis.Client {
	return r.client
}

// Config returns the Redis configuration
func (r *RedisClient) Config() *config.RedisConfig {
	return r.config
}

// Stats returns Redis connection statistics
func (r *RedisClient) Stats() *redis.PoolStats {
	return r.client.PoolStats()
}

// FlushDB flushes the current database (use with caution)
func (r *RedisClient) FlushDB(ctx context.Context) error {
	if err := r.client.FlushDB(ctx).Err(); err != nil {
		return errors.NewInternalError("failed to flush Redis database").WithCause(err)
	}
	return nil
}

// Keys returns all keys matching the pattern
func (r *RedisClient) Keys(ctx context.Context, pattern string) ([]string, error) {
	keys, err := r.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, errors.NewInternalError("failed to get Redis keys").WithCause(err)
	}
	return keys, nil
}

// Exists checks if keys exist
func (r *RedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	count, err := r.client.Exists(ctx, keys...).Result()
	if err != nil {
		return 0, errors.NewInternalError("failed to check key existence").WithCause(err)
	}
	return count, nil
}

// Del deletes keys
func (r *RedisClient) Del(ctx context.Context, keys ...string) (int64, error) {
	count, err := r.client.Del(ctx, keys...).Result()
	if err != nil {
		return 0, errors.NewInternalError("failed to delete keys").WithCause(err)
	}
	return count, nil
}

// Set sets a key-value pair with optional expiration
func (r *RedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	if err := r.client.Set(ctx, key, value, expiration).Err(); err != nil {
		return errors.NewInternalError("failed to set Redis key").WithCause(err)
	}
	return nil
}

// Get gets a value by key
func (r *RedisClient) Get(ctx context.Context, key string) (string, error) {
	val, err := r.client.Get(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", errors.NewNotFoundError("key")
		}
		return "", errors.NewInternalError("failed to get Redis key").WithCause(err)
	}
	return val, nil
}

// HSet sets hash field
func (r *RedisClient) HSet(ctx context.Context, key string, values ...interface{}) error {
	if err := r.client.HSet(ctx, key, values...).Err(); err != nil {
		return errors.NewInternalError("failed to set Redis hash").WithCause(err)
	}
	return nil
}

// HGet gets hash field
func (r *RedisClient) HGet(ctx context.Context, key, field string) (string, error) {
	val, err := r.client.HGet(ctx, key, field).Result()
	if err != nil {
		if err == redis.Nil {
			return "", errors.NewNotFoundError("hash field")
		}
		return "", errors.NewInternalError("failed to get Redis hash field").WithCause(err)
	}
	return val, nil
}

// HGetAll gets all hash fields
func (r *RedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	val, err := r.client.HGetAll(ctx, key).Result()
	if err != nil {
		return nil, errors.NewInternalError("failed to get Redis hash").WithCause(err)
	}
	return val, nil
}

// LPush pushes elements to the left of a list
func (r *RedisClient) LPush(ctx context.Context, key string, values ...interface{}) error {
	if err := r.client.LPush(ctx, key, values...).Err(); err != nil {
		return errors.NewInternalError("failed to push to Redis list").WithCause(err)
	}
	return nil
}

// RPop pops an element from the right of a list
func (r *RedisClient) RPop(ctx context.Context, key string) (string, error) {
	val, err := r.client.RPop(ctx, key).Result()
	if err != nil {
		if err == redis.Nil {
			return "", errors.NewNotFoundError("list element")
		}
		return "", errors.NewInternalError("failed to pop from Redis list").WithCause(err)
	}
	return val, nil
}

// BRPop blocks and pops an element from the right of a list
func (r *RedisClient) BRPop(ctx context.Context, timeout time.Duration, keys ...string) ([]string, error) {
	val, err := r.client.BRPop(ctx, timeout, keys...).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, errors.NewNotFoundError("list element")
		}
		return nil, errors.NewInternalError("failed to block pop from Redis list").WithCause(err)
	}
	return val, nil
}

// LLen returns the length of a list
func (r *RedisClient) LLen(ctx context.Context, key string) (int64, error) {
	length, err := r.client.LLen(ctx, key).Result()
	if err != nil {
		return 0, errors.NewInternalError("failed to get Redis list length").WithCause(err)
	}
	return length, nil
}

// ZAdd adds elements to a sorted set
func (r *RedisClient) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	if err := r.client.ZAdd(ctx, key, members...).Err(); err != nil {
		return errors.NewInternalError("failed to add to Redis sorted set").WithCause(err)
	}
	return nil
}

// ZPopMin pops the minimum element from a sorted set
func (r *RedisClient) ZPopMin(ctx context.Context, key string, count ...int64) ([]redis.Z, error) {
	val, err := r.client.ZPopMin(ctx, key, count...).Result()
	if err != nil {
		return nil, errors.NewInternalError("failed to pop min from Redis sorted set").WithCause(err)
	}
	return val, nil
}

// ZCard returns the cardinality of a sorted set
func (r *RedisClient) ZCard(ctx context.Context, key string) (int64, error) {
	count, err := r.client.ZCard(ctx, key).Result()
	if err != nil {
		return 0, errors.NewInternalError("failed to get Redis sorted set cardinality").WithCause(err)
	}
	return count, nil
}

// Expire sets a timeout on a key
func (r *RedisClient) Expire(ctx context.Context, key string, expiration time.Duration) error {
	if err := r.client.Expire(ctx, key, expiration).Err(); err != nil {
		return errors.NewInternalError("failed to set Redis key expiration").WithCause(err)
	}
	return nil
}

// TTL returns the time to live of a key
func (r *RedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	ttl, err := r.client.TTL(ctx, key).Result()
	if err != nil {
		return 0, errors.NewInternalError("failed to get Redis key TTL").WithCause(err)
	}
	return ttl, nil
}