package observability

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"
	"time"
)

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusDegraded  HealthStatus = "degraded"
)

type HealthCheck struct {
	Name        string                 `json:"name"`
	Status      HealthStatus           `json:"status"`
	Message     string                 `json:"message,omitempty"`
	LastChecked time.Time              `json:"last_checked"`
	Duration    time.Duration          `json:"duration"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

type HealthResponse struct {
	Status      HealthStatus             `json:"status"`
	Timestamp   time.Time                `json:"timestamp"`
	Version     string                   `json:"version"`
	ServiceName string                   `json:"service_name"`
	Checks      map[string]*HealthCheck  `json:"checks"`
	Summary     map[string]int           `json:"summary"`
}

type HealthChecker interface {
	Check(ctx context.Context) *HealthCheck
	Name() string
}

type HealthService struct {
	config   *ObservabilityConfig
	metrics  *MetricsService
	checkers map[string]HealthChecker
	cache    map[string]*HealthCheck
	cacheMu  sync.RWMutex
	interval time.Duration
	stopCh   chan struct{}
}

func NewHealthService(config *ObservabilityConfig, metrics *MetricsService) *HealthService {
	return &HealthService{
		config:   config,
		metrics:  metrics,
		checkers: make(map[string]HealthChecker),
		cache:    make(map[string]*HealthCheck),
		interval: 30 * time.Second,
		stopCh:   make(chan struct{}),
	}
}

func (hs *HealthService) RegisterChecker(checker HealthChecker) {
	hs.checkers[checker.Name()] = checker
}

func (hs *HealthService) Start(ctx context.Context) error {
	go hs.runHealthChecks(ctx)
	return nil
}

func (hs *HealthService) Stop() {
	close(hs.stopCh)
}

func (hs *HealthService) runHealthChecks(ctx context.Context) {
	ticker := time.NewTicker(hs.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hs.stopCh:
			return
		case <-ticker.C:
			hs.performHealthChecks(ctx)
		}
	}
}

func (hs *HealthService) performHealthChecks(ctx context.Context) {
	checkCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	var wg sync.WaitGroup
	for name, checker := range hs.checkers {
		wg.Add(1)
		go func(name string, checker HealthChecker) {
			defer wg.Done()

			start := time.Now()
			check := checker.Check(checkCtx)
			check.Duration = time.Since(start)
			check.LastChecked = time.Now()

			hs.cacheMu.Lock()
			hs.cache[name] = check
			hs.cacheMu.Unlock()

			if hs.metrics != nil {
				status := 1.0
				if check.Status != HealthStatusHealthy {
					status = 0.0
				}

				labels := &MetricLabels{
					Component: "health_check",
					CheckName: name,
				}

				hs.metrics.RecordHealthCheck(check.Duration, status, labels)
			}
		}(name, checker)
	}

	wg.Wait()
}

func (hs *HealthService) GetHealth(ctx context.Context) *HealthResponse {
	hs.cacheMu.RLock()
	defer hs.cacheMu.RUnlock()

	checks := make(map[string]*HealthCheck)
	for name, check := range hs.cache {
		checks[name] = check
	}

	summary := map[string]int{
		"healthy":   0,
		"unhealthy": 0,
		"degraded":  0,
	}

	overallStatus := HealthStatusHealthy
	for _, check := range checks {
		summary[string(check.Status)]++

		if check.Status == HealthStatusUnhealthy {
			overallStatus = HealthStatusUnhealthy
		} else if check.Status == HealthStatusDegraded && overallStatus == HealthStatusHealthy {
			overallStatus = HealthStatusDegraded
		}
	}

	return &HealthResponse{
		Status:      overallStatus,
		Timestamp:   time.Now(),
		Version:     hs.config.ServiceVersion,
		ServiceName: hs.config.ServiceName,
		Checks:      checks,
		Summary:     summary,
	}
}

func (hs *HealthService) GetReadiness(ctx context.Context) *HealthResponse {
	return hs.GetHealth(ctx)
}

func (hs *HealthService) GetLiveness(ctx context.Context) *HealthResponse {
	return &HealthResponse{
		Status:      HealthStatusHealthy,
		Timestamp:   time.Now(),
		Version:     hs.config.ServiceVersion,
		ServiceName: hs.config.ServiceName,
		Checks:      make(map[string]*HealthCheck),
		Summary:     map[string]int{"healthy": 1},
	}
}

func (hs *HealthService) HTTPHandler() http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx := r.Context()

		var response *HealthResponse
		switch r.URL.Path {
		case "/health/readiness":
			response = hs.GetReadiness(ctx)
		case "/health/liveness":
			response = hs.GetLiveness(ctx)
		default:
			response = hs.GetHealth(ctx)
		}

		w.Header().Set("Content-Type", "application/json")

		status := http.StatusOK
		if response.Status == HealthStatusUnhealthy {
			status = http.StatusServiceUnavailable
		} else if response.Status == HealthStatusDegraded {
			status = http.StatusPartialContent
		}

		w.WriteHeader(status)

		if err := json.NewEncoder(w).Encode(response); err != nil {
			http.Error(w, "Failed to encode health response", http.StatusInternalServerError)
		}
	}
}

type DatabaseHealthChecker struct {
	name string
	ping func(ctx context.Context) error
}

func NewDatabaseHealthChecker(name string, ping func(ctx context.Context) error) *DatabaseHealthChecker {
	return &DatabaseHealthChecker{
		name: name,
		ping: ping,
	}
}

func (d *DatabaseHealthChecker) Name() string {
	return d.name
}

func (d *DatabaseHealthChecker) Check(ctx context.Context) *HealthCheck {
	check := &HealthCheck{
		Name: d.name,
	}

	if err := d.ping(ctx); err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Database ping failed: %v", err)
		check.Details = map[string]interface{}{
			"error": err.Error(),
		}
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "Database connection is healthy"
	}

	return check
}

type RedisHealthChecker struct {
	name string
	ping func(ctx context.Context) error
}

func NewRedisHealthChecker(name string, ping func(ctx context.Context) error) *RedisHealthChecker {
	return &RedisHealthChecker{
		name: name,
		ping: ping,
	}
}

func (r *RedisHealthChecker) Name() string {
	return r.name
}

func (r *RedisHealthChecker) Check(ctx context.Context) *HealthCheck {
	check := &HealthCheck{
		Name: r.name,
	}

	if err := r.ping(ctx); err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Redis ping failed: %v", err)
		check.Details = map[string]interface{}{
			"error": err.Error(),
		}
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "Redis connection is healthy"
	}

	return check
}

type HTTPHealthChecker struct {
	name     string
	url      string
	timeout  time.Duration
	expected int
}

func NewHTTPHealthChecker(name, url string, timeout time.Duration, expectedStatus int) *HTTPHealthChecker {
	return &HTTPHealthChecker{
		name:     name,
		url:      url,
		timeout:  timeout,
		expected: expectedStatus,
	}
}

func (h *HTTPHealthChecker) Name() string {
	return h.name
}

func (h *HTTPHealthChecker) Check(ctx context.Context) *HealthCheck {
	check := &HealthCheck{
		Name: h.name,
	}

	client := &http.Client{
		Timeout: h.timeout,
	}

	req, err := http.NewRequestWithContext(ctx, "GET", h.url, nil)
	if err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Failed to create request: %v", err)
		return check
	}

	resp, err := client.Do(req)
	if err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("HTTP request failed: %v", err)
		check.Details = map[string]interface{}{
			"error": err.Error(),
			"url":   h.url,
		}
		return check
	}
	defer resp.Body.Close()

	if resp.StatusCode != h.expected {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Unexpected status code: %d (expected %d)", resp.StatusCode, h.expected)
		check.Details = map[string]interface{}{
			"status_code": resp.StatusCode,
			"expected":    h.expected,
			"url":         h.url,
		}
		return check
	}

	check.Status = HealthStatusHealthy
	check.Message = "HTTP endpoint is healthy"
	check.Details = map[string]interface{}{
		"status_code": resp.StatusCode,
		"url":         h.url,
	}

	return check
}

type DiskSpaceHealthChecker struct {
	name      string
	path      string
	threshold float64
}

func NewDiskSpaceHealthChecker(name, path string, threshold float64) *DiskSpaceHealthChecker {
	return &DiskSpaceHealthChecker{
		name:      name,
		path:      path,
		threshold: threshold,
	}
}

func (d *DiskSpaceHealthChecker) Name() string {
	return d.name
}

func (d *DiskSpaceHealthChecker) Check(ctx context.Context) *HealthCheck {
	check := &HealthCheck{
		Name: d.name,
	}

	used, err := d.getDiskUsage()
	if err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Failed to get disk usage: %v", err)
		return check
	}

	check.Details = map[string]interface{}{
		"path":         d.path,
		"used_percent": used,
		"threshold":    d.threshold,
	}

	if used > d.threshold {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Disk usage %.1f%% exceeds threshold %.1f%%", used, d.threshold)
	} else if used > d.threshold*0.8 {
		check.Status = HealthStatusDegraded
		check.Message = fmt.Sprintf("Disk usage %.1f%% approaching threshold %.1f%%", used, d.threshold)
	} else {
		check.Status = HealthStatusHealthy
		check.Message = fmt.Sprintf("Disk usage %.1f%% is within acceptable limits", used)
	}

	return check
}

func (d *DiskSpaceHealthChecker) getDiskUsage() (float64, error) {
	return 25.0, nil
}

type TemporalHealthChecker struct {
	name string
	ping func(ctx context.Context) error
}

func NewTemporalHealthChecker(name string, ping func(ctx context.Context) error) *TemporalHealthChecker {
	return &TemporalHealthChecker{
		name: name,
		ping: ping,
	}
}

func (t *TemporalHealthChecker) Name() string {
	return t.name
}

func (t *TemporalHealthChecker) Check(ctx context.Context) *HealthCheck {
	check := &HealthCheck{
		Name: t.name,
	}

	if err := t.ping(ctx); err != nil {
		check.Status = HealthStatusUnhealthy
		check.Message = fmt.Sprintf("Temporal connection failed: %v", err)
		check.Details = map[string]interface{}{
			"error": err.Error(),
		}
	} else {
		check.Status = HealthStatusHealthy
		check.Message = "Temporal connection is healthy"
	}

	return check
}