package llm

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// CacheMetrics tracks cache performance metrics
type CacheMetrics struct {
	mu              sync.RWMutex
	Hits            int64     `json:"hits"`
	Misses          int64     `json:"misses"`
	TotalRequests   int64     `json:"total_requests"`
	CacheHitRatio   float64   `json:"cache_hit_ratio"`
	TotalSavedTime  float64   `json:"total_saved_time_seconds"`
	AverageLLMTime  float64   `json:"average_llm_time_seconds"`
	LastUpdate      time.Time `json:"last_update"`
	StartTime       time.Time `json:"start_time"`
}

// NewCacheMetrics creates a new cache metrics tracker
func NewCacheMetrics() *CacheMetrics {
	return &CacheMetrics{
		StartTime:  time.Now(),
		LastUpdate: time.Now(),
	}
}

// RecordHit records a cache hit
func (m *CacheMetrics) RecordHit() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Hits++
	m.TotalRequests++
	m.updateRatio()
	m.LastUpdate = time.Now()
}

// RecordMiss records a cache miss with the LLM request duration
func (m *CacheMetrics) RecordMiss(llmDuration time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Misses++
	m.TotalRequests++

	// Update average LLM time
	if m.Misses == 1 {
		m.AverageLLMTime = llmDuration.Seconds()
	} else {
		// Rolling average
		m.AverageLLMTime = (m.AverageLLMTime*float64(m.Misses-1) + llmDuration.Seconds()) / float64(m.Misses)
	}

	// Calculate time saved by cache hits (estimated)
	m.TotalSavedTime = float64(m.Hits) * m.AverageLLMTime

	m.updateRatio()
	m.LastUpdate = time.Now()
}

// updateRatio calculates the cache hit ratio (must be called with lock held)
func (m *CacheMetrics) updateRatio() {
	if m.TotalRequests > 0 {
		m.CacheHitRatio = float64(m.Hits) / float64(m.TotalRequests)
	}
}

// GetMetrics returns a copy of the current metrics
func (m *CacheMetrics) GetMetrics() CacheMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return CacheMetrics{
		Hits:           m.Hits,
		Misses:         m.Misses,
		TotalRequests:  m.TotalRequests,
		CacheHitRatio:  m.CacheHitRatio,
		TotalSavedTime: m.TotalSavedTime,
		AverageLLMTime: m.AverageLLMTime,
		LastUpdate:     m.LastUpdate,
		StartTime:      m.StartTime,
	}
}

// Reset resets all metrics
func (m *CacheMetrics) Reset() {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.Hits = 0
	m.Misses = 0
	m.TotalRequests = 0
	m.CacheHitRatio = 0
	m.TotalSavedTime = 0
	m.AverageLLMTime = 0
	m.StartTime = time.Now()
	m.LastUpdate = time.Now()
}

// String returns a formatted string representation of the metrics
func (m *CacheMetrics) String() string {
	metrics := m.GetMetrics()

	uptime := time.Since(metrics.StartTime)

	return fmt.Sprintf(`Cache Metrics:
  Uptime: %v
  Total Requests: %d
  Cache Hits: %d
  Cache Misses: %d
  Hit Ratio: %.2f%%
  Average LLM Time: %.2fs
  Total Time Saved: %.2fs
  Last Update: %v`,
		uptime,
		metrics.TotalRequests,
		metrics.Hits,
		metrics.Misses,
		metrics.CacheHitRatio*100,
		metrics.AverageLLMTime,
		metrics.TotalSavedTime,
		metrics.LastUpdate.Format("15:04:05"),
	)
}

// ToJSON returns the metrics as JSON
func (m *CacheMetrics) ToJSON() ([]byte, error) {
	metrics := m.GetMetrics()
	return json.MarshalIndent(metrics, "", "  ")
}

// MetricsEnabledCache wraps a cache with metrics tracking
type MetricsEnabledCache struct {
	cache   Cache
	metrics *CacheMetrics
}

// NewMetricsEnabledCache creates a cache wrapper with metrics
func NewMetricsEnabledCache(cache Cache) *MetricsEnabledCache {
	return &MetricsEnabledCache{
		cache:   cache,
		metrics: NewCacheMetrics(),
	}
}

// Get implements Cache interface with metrics
func (m *MetricsEnabledCache) Get(key string) (*GenerateResponse, bool) {
	response, found := m.cache.Get(key)
	if found {
		m.metrics.RecordHit()
	}
	return response, found
}

// Set implements Cache interface
func (m *MetricsEnabledCache) Set(key string, response *GenerateResponse) error {
	return m.cache.Set(key, response)
}

// Delete implements Cache interface
func (m *MetricsEnabledCache) Delete(key string) error {
	return m.cache.Delete(key)
}

// Close implements Cache interface
func (m *MetricsEnabledCache) Close() error {
	return m.cache.Close()
}

// GetMetrics returns the cache metrics
func (m *MetricsEnabledCache) GetMetrics() *CacheMetrics {
	return m.metrics
}

// RecordMiss records a cache miss (called externally when LLM request completes)
func (m *MetricsEnabledCache) RecordMiss(duration time.Duration) {
	m.metrics.RecordMiss(duration)
}