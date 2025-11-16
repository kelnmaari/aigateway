// Package redis provides Redis client with advanced features
// Version: v3.0.6+ - Full Redis Integration
package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/sirupsen/logrus"
)

// Client wraps Redis client with application-specific methods
type Client struct {
	client    *redis.Client
	logger    *logrus.Logger
	keyPrefix string
	enabled   bool
}

// Config Redis client configuration
type Config struct {
	URL       string
	KeyPrefix string
	DB        int
	MaxRetries int
	PoolSize  int
}

// NewClient creates a new Redis client
func NewClient(cfg Config, logger *logrus.Logger) (*Client, error) {
	opts, err := redis.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	if cfg.DB != 0 {
		opts.DB = cfg.DB
	}
	if cfg.MaxRetries > 0 {
		opts.MaxRetries = cfg.MaxRetries
	}
	if cfg.PoolSize > 0 {
		opts.PoolSize = cfg.PoolSize
	}

	client := redis.NewClient(opts)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	logger.WithFields(logrus.Fields{
		"url":        cfg.URL,
		"db":         opts.DB,
		"pool_size":  opts.PoolSize,
		"key_prefix": cfg.KeyPrefix,
	}).Info("✅ Redis client connected")

	return &Client{
		client:    client,
		logger:    logger,
		keyPrefix: cfg.KeyPrefix,
		enabled:   true,
	}, nil
}

// Close closes Redis connection
func (c *Client) Close() error {
	if c.client != nil {
		return c.client.Close()
	}
	return nil
}

// IsEnabled returns true if Redis is enabled
func (c *Client) IsEnabled() bool {
	return c.enabled
}

// makeKey creates a key with prefix
func (c *Client) makeKey(key string) string {
	return c.keyPrefix + key
}

// ─────────────────────────────────────────────────────────────────────────────
// Basic Operations
// ─────────────────────────────────────────────────────────────────────────────

// Set sets a key-value pair with expiration
func (c *Client) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}

	return c.client.Set(ctx, c.makeKey(key), data, expiration).Err()
}

// Get gets a value by key
func (c *Client) Get(ctx context.Context, key string, dest interface{}) error {
	data, err := c.client.Get(ctx, c.makeKey(key)).Bytes()
	if err != nil {
		return err
	}

	return json.Unmarshal(data, dest)
}

// GetString gets a string value by key
func (c *Client) GetString(ctx context.Context, key string) (string, error) {
	return c.client.Get(ctx, c.makeKey(key)).Result()
}

// SetString sets a string value with expiration
func (c *Client) SetString(ctx context.Context, key string, value string, expiration time.Duration) error {
	return c.client.Set(ctx, c.makeKey(key), value, expiration).Err()
}

// Delete deletes a key
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	prefixedKeys := make([]string, len(keys))
	for i, key := range keys {
		prefixedKeys[i] = c.makeKey(key)
	}
	return c.client.Del(ctx, prefixedKeys...).Err()
}

// Exists checks if key exists
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	count, err := c.client.Exists(ctx, c.makeKey(key)).Result()
	return count > 0, err
}

// Expire sets expiration on a key
func (c *Client) Expire(ctx context.Context, key string, expiration time.Duration) error {
	return c.client.Expire(ctx, c.makeKey(key), expiration).Err()
}

// TTL gets time to live for a key
func (c *Client) TTL(ctx context.Context, key string) (time.Duration, error) {
	return c.client.TTL(ctx, c.makeKey(key)).Result()
}

// ─────────────────────────────────────────────────────────────────────────────
// Counter Operations (for rate limiting)
// ─────────────────────────────────────────────────────────────────────────────

// Incr increments a counter
func (c *Client) Incr(ctx context.Context, key string) (int64, error) {
	return c.client.Incr(ctx, c.makeKey(key)).Result()
}

// IncrBy increments a counter by value
func (c *Client) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	return c.client.IncrBy(ctx, c.makeKey(key), value).Result()
}

// Decr decrements a counter
func (c *Client) Decr(ctx context.Context, key string) (int64, error) {
	return c.client.Decr(ctx, c.makeKey(key)).Result()
}

// GetCounter gets counter value
func (c *Client) GetCounter(ctx context.Context, key string) (int64, error) {
	val, err := c.client.Get(ctx, c.makeKey(key)).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// SetNX sets a key only if it doesn't exist (returns true if set)
func (c *Client) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return false, fmt.Errorf("failed to marshal value: %w", err)
	}

	return c.client.SetNX(ctx, c.makeKey(key), data, expiration).Result()
}

// ─────────────────────────────────────────────────────────────────────────────
// List Operations
// ─────────────────────────────────────────────────────────────────────────────

// LPush pushes value to the left of a list
func (c *Client) LPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.LPush(ctx, c.makeKey(key), values...).Err()
}

// RPush pushes value to the right of a list
func (c *Client) RPush(ctx context.Context, key string, values ...interface{}) error {
	return c.client.RPush(ctx, c.makeKey(key), values...).Err()
}

// LRange gets a range of elements from a list
func (c *Client) LRange(ctx context.Context, key string, start, stop int64) ([]string, error) {
	return c.client.LRange(ctx, c.makeKey(key), start, stop).Result()
}

// LLen gets length of a list
func (c *Client) LLen(ctx context.Context, key string) (int64, error) {
	return c.client.LLen(ctx, c.makeKey(key)).Result()
}

// LTrim trims a list to specified range
func (c *Client) LTrim(ctx context.Context, key string, start, stop int64) error {
	return c.client.LTrim(ctx, c.makeKey(key), start, stop).Err()
}

// ─────────────────────────────────────────────────────────────────────────────
// Set Operations
// ─────────────────────────────────────────────────────────────────────────────

// SAdd adds members to a set
func (c *Client) SAdd(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SAdd(ctx, c.makeKey(key), members...).Err()
}

// SRem removes members from a set
func (c *Client) SRem(ctx context.Context, key string, members ...interface{}) error {
	return c.client.SRem(ctx, c.makeKey(key), members...).Err()
}

// SMembers gets all members of a set
func (c *Client) SMembers(ctx context.Context, key string) ([]string, error) {
	return c.client.SMembers(ctx, c.makeKey(key)).Result()
}

// SIsMember checks if member exists in a set
func (c *Client) SIsMember(ctx context.Context, key string, member interface{}) (bool, error) {
	return c.client.SIsMember(ctx, c.makeKey(key), member).Result()
}

// SCard gets cardinality (number of elements) of a set
func (c *Client) SCard(ctx context.Context, key string) (int64, error) {
	return c.client.SCard(ctx, c.makeKey(key)).Result()
}

// ─────────────────────────────────────────────────────────────────────────────
// Hash Operations
// ─────────────────────────────────────────────────────────────────────────────

// HSet sets field in a hash
func (c *Client) HSet(ctx context.Context, key string, field string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal value: %w", err)
	}
	return c.client.HSet(ctx, c.makeKey(key), field, data).Err()
}

// HGet gets field from a hash
func (c *Client) HGet(ctx context.Context, key string, field string, dest interface{}) error {
	data, err := c.client.HGet(ctx, c.makeKey(key), field).Bytes()
	if err != nil {
		return err
	}
	return json.Unmarshal(data, dest)
}

// HGetAll gets all fields from a hash
func (c *Client) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return c.client.HGetAll(ctx, c.makeKey(key)).Result()
}

// HDel deletes fields from a hash
func (c *Client) HDel(ctx context.Context, key string, fields ...string) error {
	return c.client.HDel(ctx, c.makeKey(key), fields...).Err()
}

// HExists checks if field exists in a hash
func (c *Client) HExists(ctx context.Context, key string, field string) (bool, error) {
	return c.client.HExists(ctx, c.makeKey(key), field).Result()
}

// ─────────────────────────────────────────────────────────────────────────────
// Sorted Set Operations (for time-based data)
// ─────────────────────────────────────────────────────────────────────────────

// ZAdd adds members to a sorted set
func (c *Client) ZAdd(ctx context.Context, key string, members ...redis.Z) error {
	return c.client.ZAdd(ctx, c.makeKey(key), members...).Err()
}

// ZRangeByScore gets members by score range
func (c *Client) ZRangeByScore(ctx context.Context, key string, min, max string) ([]string, error) {
	return c.client.ZRangeByScore(ctx, c.makeKey(key), &redis.ZRangeBy{
		Min: min,
		Max: max,
	}).Result()
}

// ZRemRangeByScore removes members by score range
func (c *Client) ZRemRangeByScore(ctx context.Context, key string, min, max string) error {
	return c.client.ZRemRangeByScore(ctx, c.makeKey(key), min, max).Err()
}

// ZCard gets cardinality of a sorted set
func (c *Client) ZCard(ctx context.Context, key string) (int64, error) {
	return c.client.ZCard(ctx, c.makeKey(key)).Result()
}

// ─────────────────────────────────────────────────────────────────────────────
// Pub/Sub Operations
// ─────────────────────────────────────────────────────────────────────────────

// Publish publishes a message to a channel
func (c *Client) Publish(ctx context.Context, channel string, message interface{}) error {
	data, err := json.Marshal(message)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	return c.client.Publish(ctx, c.makeKey(channel), data).Err()
}

// Subscribe subscribes to channels
func (c *Client) Subscribe(ctx context.Context, channels ...string) *redis.PubSub {
	prefixedChannels := make([]string, len(channels))
	for i, ch := range channels {
		prefixedChannels[i] = c.makeKey(ch)
	}
	return c.client.Subscribe(ctx, prefixedChannels...)
}

// ─────────────────────────────────────────────────────────────────────────────
// Pipeline Operations (for batch operations)
// ─────────────────────────────────────────────────────────────────────────────

// Pipeline creates a new pipeline
func (c *Client) Pipeline() redis.Pipeliner {
	return c.client.Pipeline()
}

// ─────────────────────────────────────────────────────────────────────────────
// Stats & Monitoring
// ─────────────────────────────────────────────────────────────────────────────

// Stats returns Redis client stats
func (c *Client) Stats() *redis.PoolStats {
	return c.client.PoolStats()
}

// Ping pings Redis server
func (c *Client) Ping(ctx context.Context) error {
	return c.client.Ping(ctx).Err()
}

// Info returns Redis server info
func (c *Client) Info(ctx context.Context) (string, error) {
	return c.client.Info(ctx).Result()
}

// DBSize returns number of keys in current database
func (c *Client) DBSize(ctx context.Context) (int64, error) {
	return c.client.DBSize(ctx).Result()
}

// FlushDB flushes current database (DANGEROUS!)
func (c *Client) FlushDB(ctx context.Context) error {
	c.logger.Warn("⚠️ Flushing Redis database!")
	return c.client.FlushDB(ctx).Err()
}

// Keys returns all keys matching pattern (use with caution in production)
func (c *Client) Keys(ctx context.Context, pattern string) ([]string, error) {
	return c.client.Keys(ctx, c.makeKey(pattern)).Result()
}

