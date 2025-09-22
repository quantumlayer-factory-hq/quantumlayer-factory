package llm

import (
	"context"
	"fmt"
	"log"
	"time"
)

// CachedClient wraps an LLM client with caching capabilities
type CachedClient struct {
	client      Client
	cache       Cache
	config      CacheConfig
	metricsCache *MetricsEnabledCache
}

// NewCachedClient creates a new cached client wrapper
func NewCachedClient(client Client, cache Cache, config CacheConfig) *CachedClient {
	metricsCache := NewMetricsEnabledCache(cache)
	return &CachedClient{
		client:       client,
		cache:        cache,
		config:       config,
		metricsCache: metricsCache,
	}
}

// Generate implements the Client interface with caching
func (c *CachedClient) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	// Skip caching if disabled
	if !c.config.Enabled {
		return c.client.Generate(ctx, req)
	}

	// Generate cache key
	cacheKey := req.CacheKey
	if cacheKey == "" {
		cacheKey = GenerateCacheKey(req)
	}

	// Try to get from cache first
	if cachedResponse, found := c.metricsCache.Get(cacheKey); found {
		log.Printf("[LLM Cache] Cache HIT for key: %s", cacheKey)

		// Mark as cached and return
		cachedResponse.Cached = true
		cachedResponse.Duration = 0 // Cache retrieval is essentially instant
		return cachedResponse, nil
	}

	log.Printf("[LLM Cache] Cache MISS for key: %s", cacheKey)

	// Not in cache, make actual request
	startTime := time.Now()
	response, err := c.client.Generate(ctx, req)
	if err != nil {
		// Still record the miss even if the request failed
		c.metricsCache.RecordMiss(time.Since(startTime))
		return nil, err
	}

	// Record actual duration and miss
	duration := time.Since(startTime)
	response.Duration = duration
	response.Cached = false
	c.metricsCache.RecordMiss(duration)

	// Store in cache (async to not slow down response)
	go func() {
		if err := c.metricsCache.Set(cacheKey, response); err != nil {
			log.Printf("[LLM Cache] Failed to cache response for key %s: %v", cacheKey, err)
		} else {
			log.Printf("[LLM Cache] Cached response for key: %s (TTL: %v)", cacheKey, c.config.TTL)
		}
	}()

	return response, nil
}

// GenerateStream implements the Client interface
// Note: Streaming responses are not cached as they're inherently real-time
func (c *CachedClient) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *StreamChunk, error) {
	// Streaming responses are not cached
	return c.client.GenerateStream(ctx, req)
}

// GetProvider returns the provider type from the wrapped client
func (c *CachedClient) GetProvider() Provider {
	return c.client.GetProvider()
}

// GetModels returns available models from the wrapped client
func (c *CachedClient) GetModels() []Model {
	return c.client.GetModels()
}

// Close closes both the cache and the wrapped client
func (c *CachedClient) Close() error {
	var clientErr, cacheErr error

	if c.client != nil {
		clientErr = c.client.Close()
	}

	if c.cache != nil {
		cacheErr = c.cache.Close()
	}

	// Return the first error encountered
	if clientErr != nil {
		return fmt.Errorf("failed to close client: %w", clientErr)
	}
	if cacheErr != nil {
		return fmt.Errorf("failed to close cache: %w", cacheErr)
	}

	return nil
}

// Health checks the health of the wrapped client
func (c *CachedClient) Health(ctx context.Context) error {
	return c.client.Health(ctx)
}

// GetCacheStats returns cache statistics
func (c *CachedClient) GetCacheStats() (*CacheStats, error) {
	if redisCache, ok := c.cache.(*RedisCache); ok {
		return redisCache.GetStats()
	}
	return nil, fmt.Errorf("cache statistics not available for this cache type")
}

// ClearCache clears all cached responses
func (c *CachedClient) ClearCache() error {
	if redisCache, ok := c.cache.(*RedisCache); ok {
		return redisCache.ClearAll()
	}
	return fmt.Errorf("cache clearing not available for this cache type")
}

// InvalidateCache invalidates a specific cached response
func (c *CachedClient) InvalidateCache(key string) error {
	return c.cache.Delete(key)
}

// WarmCache pre-populates the cache with commonly used requests
func (c *CachedClient) WarmCache(ctx context.Context, requests []*GenerateRequest) error {
	log.Printf("[LLM Cache] Starting cache warming with %d requests", len(requests))

	for i, req := range requests {
		// Generate response and cache it
		_, err := c.Generate(ctx, req)
		if err != nil {
			log.Printf("[LLM Cache] Failed to warm cache for request %d: %v", i, err)
			continue
		}

		// Brief pause to avoid overwhelming the LLM provider
		time.Sleep(100 * time.Millisecond)
	}

	log.Printf("[LLM Cache] Cache warming completed")
	return nil
}

// GetCacheMetrics returns the current cache metrics
func (c *CachedClient) GetCacheMetrics() *CacheMetrics {
	if c.metricsCache != nil {
		return c.metricsCache.GetMetrics()
	}
	return nil
}

// PrintCacheMetrics logs the current cache metrics
func (c *CachedClient) PrintCacheMetrics() {
	if c.metricsCache != nil {
		log.Println(c.metricsCache.GetMetrics().String())
	}
}