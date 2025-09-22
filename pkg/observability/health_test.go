package observability

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestHealthService_RegisterChecker(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	hs := NewHealthService(config, nil)
	checker := &MockHealthChecker{name: "test-checker"}

	hs.RegisterChecker(checker)

	if len(hs.checkers) != 1 {
		t.Errorf("Expected 1 checker, got %d", len(hs.checkers))
	}

	if hs.checkers["test-checker"] != checker {
		t.Error("Checker not registered correctly")
	}
}

func TestHealthService_GetHealth(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	hs := NewHealthService(config, nil)

	healthyChecker := &MockHealthChecker{
		name:   "healthy-checker",
		status: HealthStatusHealthy,
	}
	unhealthyChecker := &MockHealthChecker{
		name:   "unhealthy-checker",
		status: HealthStatusUnhealthy,
	}

	hs.RegisterChecker(healthyChecker)
	hs.RegisterChecker(unhealthyChecker)

	hs.cache["healthy-checker"] = &HealthCheck{
		Name:        "healthy-checker",
		Status:      HealthStatusHealthy,
		Message:     "All good",
		LastChecked: time.Now(),
	}
	hs.cache["unhealthy-checker"] = &HealthCheck{
		Name:        "unhealthy-checker",
		Status:      HealthStatusUnhealthy,
		Message:     "Something wrong",
		LastChecked: time.Now(),
	}

	ctx := context.Background()
	response := hs.GetHealth(ctx)

	if response.Status != HealthStatusUnhealthy {
		t.Errorf("Expected overall status to be unhealthy, got %s", response.Status)
	}

	if response.ServiceName != "test-service" {
		t.Errorf("Expected service name to be 'test-service', got %s", response.ServiceName)
	}

	if response.Version != "1.0.0" {
		t.Errorf("Expected version to be '1.0.0', got %s", response.Version)
	}

	if len(response.Checks) != 2 {
		t.Errorf("Expected 2 checks, got %d", len(response.Checks))
	}

	if response.Summary["healthy"] != 1 {
		t.Errorf("Expected 1 healthy check, got %d", response.Summary["healthy"])
	}

	if response.Summary["unhealthy"] != 1 {
		t.Errorf("Expected 1 unhealthy check, got %d", response.Summary["unhealthy"])
	}
}

func TestHealthService_GetLiveness(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	hs := NewHealthService(config, nil)
	ctx := context.Background()
	response := hs.GetLiveness(ctx)

	if response.Status != HealthStatusHealthy {
		t.Errorf("Expected liveness to always be healthy, got %s", response.Status)
	}

	if len(response.Checks) != 0 {
		t.Errorf("Expected no checks for liveness, got %d", len(response.Checks))
	}

	if response.Summary["healthy"] != 1 {
		t.Errorf("Expected 1 healthy summary, got %d", response.Summary["healthy"])
	}
}

func TestHealthService_HTTPHandler(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	hs := NewHealthService(config, nil)
	handler := hs.HTTPHandler()

	tests := []struct {
		name           string
		path           string
		expectedStatus int
	}{
		{"health endpoint", "/health", http.StatusOK},
		{"readiness endpoint", "/health/readiness", http.StatusOK},
		{"liveness endpoint", "/health/liveness", http.StatusOK},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", tt.path, nil)
			w := httptest.NewRecorder()

			handler(w, req)

			if w.Code != tt.expectedStatus {
				t.Errorf("Expected status %d, got %d", tt.expectedStatus, w.Code)
			}

			contentType := w.Header().Get("Content-Type")
			if contentType != "application/json" {
				t.Errorf("Expected Content-Type to be 'application/json', got %s", contentType)
			}

			var response HealthResponse
			if err := json.NewDecoder(w.Body).Decode(&response); err != nil {
				t.Fatalf("Failed to decode response: %v", err)
			}

			if response.ServiceName != "test-service" {
				t.Errorf("Expected service name to be 'test-service', got %s", response.ServiceName)
			}
		})
	}
}

func TestHealthService_HTTPHandler_UnhealthyService(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	hs := NewHealthService(config, nil)

	hs.cache["unhealthy-check"] = &HealthCheck{
		Name:        "unhealthy-check",
		Status:      HealthStatusUnhealthy,
		Message:     "Service is down",
		LastChecked: time.Now(),
	}

	handler := hs.HTTPHandler()
	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()

	handler(w, req)

	if w.Code != http.StatusServiceUnavailable {
		t.Errorf("Expected status %d, got %d", http.StatusServiceUnavailable, w.Code)
	}
}

func TestDatabaseHealthChecker(t *testing.T) {
	var pingCalled bool
	pingFunc := func(ctx context.Context) error {
		pingCalled = true
		return nil
	}

	checker := NewDatabaseHealthChecker("test-db", pingFunc)

	if checker.Name() != "test-db" {
		t.Errorf("Expected name to be 'test-db', got %s", checker.Name())
	}

	ctx := context.Background()
	check := checker.Check(ctx)

	if !pingCalled {
		t.Error("Ping function should have been called")
	}

	if check.Status != HealthStatusHealthy {
		t.Errorf("Expected status to be healthy, got %s", check.Status)
	}

	if check.Name != "test-db" {
		t.Errorf("Expected check name to be 'test-db', got %s", check.Name)
	}
}

func TestDatabaseHealthChecker_Error(t *testing.T) {
	pingFunc := func(ctx context.Context) error {
		return &TestError{message: "connection failed"}
	}

	checker := NewDatabaseHealthChecker("test-db", pingFunc)
	ctx := context.Background()
	check := checker.Check(ctx)

	if check.Status != HealthStatusUnhealthy {
		t.Errorf("Expected status to be unhealthy, got %s", check.Status)
	}

	if check.Message == "" {
		t.Error("Expected error message to be set")
	}

	if check.Details == nil {
		t.Error("Expected details to be set")
	}
}

func TestRedisHealthChecker(t *testing.T) {
	var pingCalled bool
	pingFunc := func(ctx context.Context) error {
		pingCalled = true
		return nil
	}

	checker := NewRedisHealthChecker("test-redis", pingFunc)

	if checker.Name() != "test-redis" {
		t.Errorf("Expected name to be 'test-redis', got %s", checker.Name())
	}

	ctx := context.Background()
	check := checker.Check(ctx)

	if !pingCalled {
		t.Error("Ping function should have been called")
	}

	if check.Status != HealthStatusHealthy {
		t.Errorf("Expected status to be healthy, got %s", check.Status)
	}
}

func TestHTTPHealthChecker(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer server.Close()

	checker := NewHTTPHealthChecker("test-http", server.URL, 5*time.Second, http.StatusOK)

	if checker.Name() != "test-http" {
		t.Errorf("Expected name to be 'test-http', got %s", checker.Name())
	}

	ctx := context.Background()
	check := checker.Check(ctx)

	if check.Status != HealthStatusHealthy {
		t.Errorf("Expected status to be healthy, got %s", check.Status)
	}

	if check.Details == nil {
		t.Error("Expected details to be set")
	}

	statusCode, ok := check.Details["status_code"].(int)
	if !ok || statusCode != http.StatusOK {
		t.Errorf("Expected status code to be %d, got %v", http.StatusOK, statusCode)
	}
}

func TestHTTPHealthChecker_WrongStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	checker := NewHTTPHealthChecker("test-http", server.URL, 5*time.Second, http.StatusOK)
	ctx := context.Background()
	check := checker.Check(ctx)

	if check.Status != HealthStatusUnhealthy {
		t.Errorf("Expected status to be unhealthy, got %s", check.Status)
	}
}

func TestDiskSpaceHealthChecker(t *testing.T) {
	checker := NewDiskSpaceHealthChecker("test-disk", "/", 90.0)

	if checker.Name() != "test-disk" {
		t.Errorf("Expected name to be 'test-disk', got %s", checker.Name())
	}

	ctx := context.Background()
	check := checker.Check(ctx)

	if check.Status != HealthStatusHealthy {
		t.Errorf("Expected status to be healthy, got %s", check.Status)
	}

	if check.Details == nil {
		t.Error("Expected details to be set")
	}

	usedPercent, ok := check.Details["used_percent"].(float64)
	if !ok {
		t.Error("Expected used_percent to be set in details")
	}

	if usedPercent != 25.0 {
		t.Errorf("Expected used_percent to be 25.0, got %f", usedPercent)
	}
}

func TestTemporalHealthChecker(t *testing.T) {
	var pingCalled bool
	pingFunc := func(ctx context.Context) error {
		pingCalled = true
		return nil
	}

	checker := NewTemporalHealthChecker("test-temporal", pingFunc)

	if checker.Name() != "test-temporal" {
		t.Errorf("Expected name to be 'test-temporal', got %s", checker.Name())
	}

	ctx := context.Background()
	check := checker.Check(ctx)

	if !pingCalled {
		t.Error("Ping function should have been called")
	}

	if check.Status != HealthStatusHealthy {
		t.Errorf("Expected status to be healthy, got %s", check.Status)
	}
}

func TestHealthService_StartStop(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	hs := NewHealthService(config, nil)
	hs.interval = 10 * time.Millisecond

	checker := &MockHealthChecker{
		name:   "test-checker",
		status: HealthStatusHealthy,
	}
	hs.RegisterChecker(checker)

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := hs.Start(ctx)
	if err != nil {
		t.Fatalf("Failed to start health service: %v", err)
	}

	time.Sleep(50 * time.Millisecond)

	hs.Stop()

	if len(hs.cache) == 0 {
		t.Error("Expected health checks to be cached")
	}

	if !checker.called {
		t.Error("Expected health checker to be called")
	}
}

type MockHealthChecker struct {
	name   string
	status HealthStatus
	called bool
}

func (m *MockHealthChecker) Name() string {
	return m.name
}

func (m *MockHealthChecker) Check(ctx context.Context) *HealthCheck {
	m.called = true
	return &HealthCheck{
		Name:   m.name,
		Status: m.status,
	}
}

func BenchmarkHealthService_GetHealth(b *testing.B) {
	config := &ObservabilityConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
	}

	hs := NewHealthService(config, nil)

	for i := 0; i < 10; i++ {
		checker := &MockHealthChecker{
			name:   "checker-" + string(rune(i)),
			status: HealthStatusHealthy,
		}
		hs.RegisterChecker(checker)
		hs.cache[checker.name] = &HealthCheck{
			Name:        checker.name,
			Status:      HealthStatusHealthy,
			LastChecked: time.Now(),
		}
	}

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		hs.GetHealth(ctx)
	}
}