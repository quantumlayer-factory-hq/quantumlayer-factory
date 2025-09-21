package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Cache interface for LLM response caching
type Cache interface {
	Get(key string) (*GenerateResponse, bool)
	Set(key string, response *GenerateResponse) error
	Delete(key string) error
	Close() error
}

// RedisCache implements Cache using Redis
type RedisCache struct {
	client *redis.Client
	ttl    time.Duration
	prefix string
}

// NewCache creates a new cache instance
func NewCache(config CacheConfig) (*RedisCache, error) {
	// Parse Redis URL or use default
	redisURL := config.RedisURL
	if redisURL == "" {
		redisURL = "redis://localhost:6379"
	}

	// Parse Redis options
	opt, err := redis.ParseURL(redisURL)
	if err != nil {
		return nil, fmt.Errorf("failed to parse Redis URL: %w", err)
	}

	client := redis.NewClient(opt)

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	ttl := config.TTL
	if ttl == 0 {
		ttl = 1 * time.Hour // Default TTL
	}

	return &RedisCache{
		client: client,
		ttl:    ttl,
		prefix: "qlf:llm:cache:",
	}, nil
}

// Get retrieves a cached response
func (c *RedisCache) Get(key string) (*GenerateResponse, bool) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	fullKey := c.prefix + key
	data, err := c.client.Get(ctx, fullKey).Result()
	if err != nil {
		if err == redis.Nil {
			return nil, false // Key not found
		}
		return nil, false // Other error
	}

	var response GenerateResponse
	if err := json.Unmarshal([]byte(data), &response); err != nil {
		return nil, false
	}

	return &response, true
}

// Set stores a response in cache
func (c *RedisCache) Set(key string, response *GenerateResponse) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	data, err := json.Marshal(response)
	if err != nil {
		return fmt.Errorf("failed to marshal response: %w", err)
	}

	fullKey := c.prefix + key
	return c.client.Set(ctx, fullKey, data, c.ttl).Err()
}

// Delete removes a cached response
func (c *RedisCache) Delete(key string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Second)
	defer cancel()

	fullKey := c.prefix + key
	return c.client.Del(ctx, fullKey).Err()
}

// Close closes the Redis connection
func (c *RedisCache) Close() error {
	return c.client.Close()
}

// GetStats returns cache statistics
func (c *RedisCache) GetStats() (*CacheStats, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Get all cache keys
	pattern := c.prefix + "*"
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to get cache keys: %w", err)
	}

	totalKeys := len(keys)
	totalSize := int64(0)

	// Sample a few keys to estimate total size
	sampleSize := 10
	if len(keys) < sampleSize {
		sampleSize = len(keys)
	}

	for i := 0; i < sampleSize; i++ {
		size, err := c.client.MemoryUsage(ctx, keys[i]).Result()
		if err == nil {
			totalSize += size
		}
	}

	// Estimate total size
	if sampleSize > 0 {
		avgSize := totalSize / int64(sampleSize)
		totalSize = avgSize * int64(totalKeys)
	}

	return &CacheStats{
		TotalKeys:     totalKeys,
		EstimatedSize: totalSize,
	}, nil
}

// ClearAll clears all cached responses
func (c *RedisCache) ClearAll() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pattern := c.prefix + "*"
	keys, err := c.client.Keys(ctx, pattern).Result()
	if err != nil {
		return fmt.Errorf("failed to get cache keys: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	return c.client.Del(ctx, keys...).Err()
}

// CacheStats represents cache statistics
type CacheStats struct {
	TotalKeys     int   `json:"total_keys"`
	EstimatedSize int64 `json:"estimated_size_bytes"`
}

// GenerateCacheKey generates a cache key for a request
func GenerateCacheKey(req *GenerateRequest) string {
	// Create a hash of the important request parameters
	key := fmt.Sprintf("%s:%s:%d:%.2f",
		req.Prompt,
		req.SystemPrompt,
		req.MaxTokens,
		req.Temperature,
	)

	// Include model if specified
	if req.Model != "" {
		key = fmt.Sprintf("%s:%s", key, req.Model)
	}

	// Simple hash (in production, use a proper hash function)
	hash := hashString(key)
	return fmt.Sprintf("req:%x", hash)
}

// Simple string hash function
func hashString(s string) uint32 {
	h := uint32(0)
	for _, c := range s {
		h = h*31 + uint32(c)
	}
	return h
}