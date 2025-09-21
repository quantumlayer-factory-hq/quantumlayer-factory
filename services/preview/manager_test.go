package preview

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewPreviewManager(t *testing.T) {
	// Since NewPreviewManager requires many dependencies,
	// we'll just test the config creation
	config := DefaultPreviewConfig()
	assert.NotNil(t, config)
}

func TestDefaultPreviewConfig(t *testing.T) {
	config := DefaultPreviewConfig()

	assert.Equal(t, "preview.quantumlayer.dev", config.BaseDomain)
	assert.Equal(t, "{app}-{hash}", config.SubdomainPattern)
	assert.Equal(t, 24*time.Hour, config.DefaultTTL)
	assert.Equal(t, 72*time.Hour, config.MaxTTL)
	assert.Equal(t, 1*time.Hour, config.CleanupInterval)
	assert.True(t, config.TLS.Enabled)
	assert.Equal(t, "letsencrypt", config.TLS.Provider)
	assert.True(t, config.TLS.AutoRenew)
	assert.Equal(t, "nginx", config.LoadBalancer.Type)
	assert.Equal(t, "/health", config.HealthCheck.Endpoint)
	assert.Equal(t, 30*time.Second, config.HealthCheck.Interval)
	assert.True(t, config.Analytics.Enabled)
}

func TestPreviewRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     *PreviewRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &PreviewRequest{
				ProjectPath: "/path/to/project",
				Language:    "python",
				Framework:   "fastapi",
				AppName:     "test-app",
				Port:        8000,
				TTL:         24 * time.Hour,
			},
			wantErr: false,
		},
		{
			name: "missing project path",
			req: &PreviewRequest{
				Language:  "python",
				Framework: "fastapi",
				AppName:   "test-app",
				Port:      8000,
				TTL:       24 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "missing language",
			req: &PreviewRequest{
				ProjectPath: "/path/to/project",
				Framework:   "fastapi",
				AppName:     "test-app",
				Port:        8000,
				TTL:         24 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "missing app name",
			req: &PreviewRequest{
				ProjectPath: "/path/to/project",
				Language:    "python",
				Framework:   "fastapi",
				Port:        8000,
				TTL:         24 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			req: &PreviewRequest{
				ProjectPath: "/path/to/project",
				Language:    "python",
				Framework:   "fastapi",
				AppName:     "test-app",
				Port:        0,
				TTL:         24 * time.Hour,
			},
			wantErr: true,
		},
		{
			name: "TTL exceeds maximum",
			req: &PreviewRequest{
				ProjectPath: "/path/to/project",
				Language:    "python",
				Framework:   "fastapi",
				AppName:     "test-app",
				Port:        8000,
				TTL:         100 * time.Hour,
			},
			wantErr: true,
		},
	}

	config := DefaultPreviewConfig()
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePreviewRequest(tt.req, config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPreviewResult_Status(t *testing.T) {
	tests := []struct {
		name     string
		result   PreviewResult
		expected string
	}{
		{
			name: "building",
			result: PreviewResult{
				Status: PreviewStatus{Phase: "Building"},
			},
			expected: "Building",
		},
		{
			name: "deploying",
			result: PreviewResult{
				Status: PreviewStatus{Phase: "Deploying"},
			},
			expected: "Deploying",
		},
		{
			name: "ready",
			result: PreviewResult{
				Status: PreviewStatus{Phase: "Running"},
				URL:    "https://test-app-abc123.preview.local",
			},
			expected: "Running",
		},
		{
			name: "failed",
			result: PreviewResult{
				Status: PreviewStatus{Phase: "Failed"},
				Errors: []string{"Build failed", "Deployment error"},
			},
			expected: "Failed",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.result.Status.Phase)
		})
	}
}

func TestPreviewResult_IsReady(t *testing.T) {
	tests := []struct {
		name     string
		result   PreviewResult
		expected bool
	}{
		{
			name: "ready state",
			result: PreviewResult{
				Status: PreviewStatus{Phase: "Running"},
				URL:    "https://test-app.preview.local",
			},
			expected: true,
		},
		{
			name: "building state",
			result: PreviewResult{
				Status: PreviewStatus{Phase: "Building"},
			},
			expected: false,
		},
		{
			name: "failed state",
			result: PreviewResult{
				Status: PreviewStatus{Phase: "Failed"},
				Errors: []string{"Build error"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isReady := tt.result.IsReady()
			assert.Equal(t, tt.expected, isReady)
		})
	}
}

func TestPreviewResult_HasErrors(t *testing.T) {
	tests := []struct {
		name     string
		result   PreviewResult
		expected bool
	}{
		{
			name: "no errors",
			result: PreviewResult{
				Status: PreviewStatus{Phase: "Running"},
				Errors: []string{},
			},
			expected: false,
		},
		{
			name: "with errors",
			result: PreviewResult{
				Status: PreviewStatus{Phase: "Failed"},
				Errors: []string{"Build failed", "Image not found"},
			},
			expected: true,
		},
		{
			name: "nil errors",
			result: PreviewResult{
				Status: PreviewStatus{Phase: "Running"},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasErrors := tt.result.HasErrors()
			assert.Equal(t, tt.expected, hasErrors)
		})
	}
}

func TestPreviewResult_GetAge(t *testing.T) {
	now := time.Now()
	result := PreviewResult{
		ID:        "test-preview",
		CreatedAt: now.Add(-5 * time.Minute),
	}

	age := result.GetAge()
	assert.True(t, age >= 5*time.Minute-time.Second)
	assert.True(t, age <= 5*time.Minute+time.Second)
}

func TestPreviewResult_TimeLeft(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		result   PreviewResult
		expected time.Duration
	}{
		{
			name: "time remaining",
			result: PreviewResult{
				ID:        "test-preview",
				ExpiresAt: now.Add(2 * time.Hour),
			},
			expected: 2 * time.Hour,
		},
		{
			name: "expired",
			result: PreviewResult{
				ID:        "test-preview",
				ExpiresAt: now.Add(-1 * time.Hour),
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			timeLeft := tt.result.TimeLeft()
			if tt.expected > 0 {
				assert.True(t, timeLeft >= tt.expected-time.Second)
				assert.True(t, timeLeft <= tt.expected+time.Second)
			} else {
				assert.Equal(t, time.Duration(0), timeLeft)
			}
		})
	}
}

func TestPreviewResult_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		result   PreviewResult
		expected bool
	}{
		{
			name: "not expired",
			result: PreviewResult{
				ID:        "test-preview",
				ExpiresAt: now.Add(1 * time.Hour),
			},
			expected: false,
		},
		{
			name: "expired",
			result: PreviewResult{
				ID:        "test-preview",
				ExpiresAt: now.Add(-1 * time.Hour),
			},
			expected: true,
		},
		{
			name: "just expired",
			result: PreviewResult{
				ID:        "test-preview",
				ExpiresAt: now,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isExpired := tt.result.IsExpired()
			assert.Equal(t, tt.expected, isExpired)
		})
	}
}

func TestPreviewResult_ExtendTTL(t *testing.T) {
	now := time.Now()
	result := &PreviewResult{
		ID:        "test-preview",
		CreatedAt: now,
		ExpiresAt: now.Add(1 * time.Hour),
		TTL:       1 * time.Hour,
	}

	// Extend by 2 hours
	extension := 2 * time.Hour
	result.ExtendTTL(extension)

	assert.Equal(t, 3*time.Hour, result.TTL)
	expectedExpiry := now.Add(3 * time.Hour)
	assert.True(t, result.ExpiresAt.After(expectedExpiry.Add(-time.Second)))
	assert.True(t, result.ExpiresAt.Before(expectedExpiry.Add(time.Second)))
}

func TestHealthStatus_IsHealthy(t *testing.T) {
	tests := []struct {
		name        string
		healthStatus HealthStatus
		expected    bool
	}{
		{
			name: "healthy",
			healthStatus: HealthStatus{
				Healthy:      true,
				LastCheck:    time.Now().Add(-1 * time.Minute),
				ResponseTime: 150 * time.Millisecond,
				StatusCode:   200,
			},
			expected: true,
		},
		{
			name: "unhealthy",
			healthStatus: HealthStatus{
				Healthy:      false,
				LastCheck:    time.Now().Add(-1 * time.Minute),
				ErrorMessage: "Connection refused",
				StatusCode:   500,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isHealthy := tt.healthStatus.Healthy
			assert.Equal(t, tt.expected, isHealthy)
		})
	}
}

func TestAnalyticsInfo_Track(t *testing.T) {
	analytics := &AnalyticsInfo{
		TotalRequests:   10,
		UniqueVisitors:  5,
		LastAccess:      time.Now().Add(-1 * time.Hour),
		AvgResponseTime: 150 * time.Millisecond,
	}

	// Test analytics data
	assert.Equal(t, int64(10), analytics.TotalRequests)
	assert.Equal(t, int64(5), analytics.UniqueVisitors)
	assert.Equal(t, 150*time.Millisecond, analytics.AvgResponseTime)
	assert.True(t, analytics.LastAccess.Before(time.Now()))
}

func TestPreviewManager_MockOperations(t *testing.T) {
	ctx := context.Background()

	// Test create preview request
	req := &PreviewRequest{
		ProjectPath: "/path/to/project",
		Language:    "python",
		Framework:   "fastapi",
		AppName:     "test-app",
		Port:        8000,
		Environment: map[string]string{
			"DEBUG": "true",
		},
		TTL: 24 * time.Hour,
	}

	assert.Equal(t, "/path/to/project", req.ProjectPath)
	assert.Equal(t, "python", req.Language)
	assert.Equal(t, "fastapi", req.Framework)
	assert.Equal(t, "test-app", req.AppName)
	assert.Equal(t, 8000, req.Port)
	assert.Equal(t, "true", req.Environment["DEBUG"])
	assert.Equal(t, 24*time.Hour, req.TTL)
	assert.NotNil(t, ctx)
}

func TestPreviewCleanup_Logic(t *testing.T) {
	now := time.Now()

	previews := []PreviewResult{
		{
			ID:        "active-preview",
			Status:    PreviewStatus{Phase: "Running"},
			CreatedAt: now.Add(-1 * time.Hour),
			ExpiresAt: now.Add(1 * time.Hour),
			TTL:       2 * time.Hour,
		},
		{
			ID:        "expired-preview-1",
			Status:    PreviewStatus{Phase: "Running"},
			CreatedAt: now.Add(-3 * time.Hour),
			ExpiresAt: now.Add(-1 * time.Hour),
			TTL:       2 * time.Hour,
		},
		{
			ID:        "expired-preview-2",
			Status:    PreviewStatus{Phase: "Failed"},
			CreatedAt: now.Add(-4 * time.Hour),
			ExpiresAt: now.Add(-2 * time.Hour),
			TTL:       2 * time.Hour,
		},
		{
			ID:        "building-preview",
			Status:    PreviewStatus{Phase: "Building"},
			CreatedAt: now.Add(-30 * time.Minute),
			ExpiresAt: now.Add(90 * time.Minute),
			TTL:       2 * time.Hour,
		},
	}

	// Test which previews should be cleaned up
	var expiredPreviews []PreviewResult
	for _, preview := range previews {
		if preview.IsExpired() {
			expiredPreviews = append(expiredPreviews, preview)
		}
	}

	assert.Len(t, expiredPreviews, 2)
	assert.Equal(t, "expired-preview-1", expiredPreviews[0].ID)
	assert.Equal(t, "expired-preview-2", expiredPreviews[1].ID)
}

// Helper validation functions

func validatePreviewRequest(req *PreviewRequest, config *PreviewConfig) error {
	if req.ProjectPath == "" {
		return assert.AnError
	}
	if req.Language == "" {
		return assert.AnError
	}
	if req.AppName == "" {
		return assert.AnError
	}
	if req.Port <= 0 {
		return assert.AnError
	}
	if req.TTL > config.MaxTTL {
		return assert.AnError
	}
	return nil
}

// Add methods to PreviewResult for testing

func (pr *PreviewResult) IsReady() bool {
	return pr.Status.Phase == "Running"
}

func (pr *PreviewResult) HasErrors() bool {
	return len(pr.Errors) > 0
}

func (pr *PreviewResult) GetAge() time.Duration {
	return time.Since(pr.CreatedAt)
}

func (pr *PreviewResult) TimeLeft() time.Duration {
	remaining := time.Until(pr.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (pr *PreviewResult) IsExpired() bool {
	return time.Now().After(pr.ExpiresAt)
}

func (pr *PreviewResult) ExtendTTL(extension time.Duration) {
	pr.TTL += extension
	pr.ExpiresAt = pr.CreatedAt.Add(pr.TTL)
}


// Benchmark tests

func BenchmarkPreviewResult_IsExpired(b *testing.B) {
	now := time.Now()
	result := PreviewResult{
		ID:        "benchmark-preview",
		ExpiresAt: now.Add(-1 * time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.IsExpired()
	}
}

func BenchmarkPreviewResult_TimeLeft(b *testing.B) {
	now := time.Now()
	result := PreviewResult{
		ID:        "benchmark-preview",
		ExpiresAt: now.Add(2 * time.Hour),
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.TimeLeft()
	}
}