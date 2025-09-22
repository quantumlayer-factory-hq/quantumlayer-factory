package observability

import (
	"context"
	"testing"
	"time"
)

func TestLLMMonitor_Initialize(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:        "test-service",
		ServiceVersion:     "1.0.0",
		Environment:        "test",
		MetricsEnabled:     true,
		PrometheusEndpoint: "http://localhost:9090",
	}

	ms := NewMetricsService(config)
	err := ms.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize metrics service: %v", err)
	}
	defer ms.Shutdown(context.Background())

	monitor := NewLLMMonitor(config, ms)

	if monitor.config != config {
		t.Error("Config should be set")
	}

	if monitor.metrics != ms {
		t.Error("Metrics service should be set")
	}

	if monitor.costTracker == nil {
		t.Error("Cost tracker should be initialized")
	}

	if monitor.circuitBreaker == nil {
		t.Error("Circuit breaker should be initialized")
	}
}

func TestLLMMonitor_RecordRequest(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:        "test-service",
		ServiceVersion:     "1.0.0",
		Environment:        "test",
		MetricsEnabled:     true,
		PrometheusEndpoint: "http://localhost:9090",
	}

	ms := NewMetricsService(config)
	err := ms.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize metrics service: %v", err)
	}
	defer ms.Shutdown(context.Background())

	monitor := NewLLMMonitor(config, ms)

	request := &LLMRequest{
		Provider:    "openai",
		Model:       "gpt-4",
		InputTokens: 100,
		MaxTokens:   200,
		Temperature: 0.7,
		RequestID:   "req-123",
		StartTime:   time.Now(),
	}

	monitor.RecordRequest(request)

	if len(monitor.activeRequests) != 1 {
		t.Errorf("Expected 1 active request, got %d", len(monitor.activeRequests))
	}

	if monitor.activeRequests["req-123"] != request {
		t.Error("Request should be tracked")
	}
}

func TestLLMMonitor_RecordResponse(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:        "test-service",
		ServiceVersion:     "1.0.0",
		Environment:        "test",
		MetricsEnabled:     true,
		PrometheusEndpoint: "http://localhost:9090",
	}

	ms := NewMetricsService(config)
	err := ms.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize metrics service: %v", err)
	}
	defer ms.Shutdown(context.Background())

	monitor := NewLLMMonitor(config, ms)

	request := &LLMRequest{
		Provider:    "openai",
		Model:       "gpt-4",
		InputTokens: 100,
		MaxTokens:   200,
		Temperature: 0.7,
		RequestID:   "req-123",
		StartTime:   time.Now().Add(-time.Second),
	}

	monitor.RecordRequest(request)

	response := &LLMResponse{
		RequestID:     "req-123",
		Success:       true,
		OutputTokens:  150,
		TotalTokens:   250,
		Cost:          0.05,
		ResponseTime:  time.Second,
		CacheHit:      false,
		FinishReason:  "stop",
		ResponseData:  "Generated response",
		EndTime:       time.Now(),
	}

	monitor.RecordResponse(response)

	if len(monitor.activeRequests) != 0 {
		t.Errorf("Expected 0 active requests after response, got %d", len(monitor.activeRequests))
	}
}

func TestLLMMonitor_RecordError(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:        "test-service",
		ServiceVersion:     "1.0.0",
		Environment:        "test",
		MetricsEnabled:     true,
		PrometheusEndpoint: "http://localhost:9090",
	}

	ms := NewMetricsService(config)
	err := ms.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize metrics service: %v", err)
	}
	defer ms.Shutdown(context.Background())

	monitor := NewLLMMonitor(config, ms)

	request := &LLMRequest{
		Provider:    "openai",
		Model:       "gpt-4",
		InputTokens: 100,
		RequestID:   "req-123",
		StartTime:   time.Now().Add(-time.Second),
	}

	monitor.RecordRequest(request)

	errorResp := &LLMError{
		RequestID:    "req-123",
		ErrorType:    "timeout",
		ErrorMessage: "Request timed out",
		Provider:     "openai",
		Model:        "gpt-4",
		RetryCount:   1,
		Timestamp:    time.Now(),
	}

	monitor.RecordError(errorResp)

	if len(monitor.activeRequests) != 0 {
		t.Errorf("Expected 0 active requests after error, got %d", len(monitor.activeRequests))
	}
}

func TestCostTracker_TrackCost(t *testing.T) {
	tracker := NewCostTracker(100.0)

	err := tracker.TrackCost("openai", "gpt-4", 0.05)
	if err != nil {
		t.Errorf("Failed to track cost: %v", err)
	}

	totalCost := tracker.GetTotalCost()
	if totalCost != 0.05 {
		t.Errorf("Expected total cost to be 0.05, got %f", totalCost)
	}

	providerCost := tracker.GetProviderCost("openai")
	if providerCost != 0.05 {
		t.Errorf("Expected OpenAI cost to be 0.05, got %f", providerCost)
	}

	modelCost := tracker.GetModelCost("openai", "gpt-4")
	if modelCost != 0.05 {
		t.Errorf("Expected GPT-4 cost to be 0.05, got %f", modelCost)
	}
}

func TestCostTracker_BudgetExceeded(t *testing.T) {
	tracker := NewCostTracker(0.10)

	err := tracker.TrackCost("openai", "gpt-4", 0.15)
	if err == nil {
		t.Error("Expected error when budget is exceeded")
	}

	if err.Error() != "monthly budget of $0.10 exceeded (would be $0.15)" {
		t.Errorf("Unexpected error message: %s", err.Error())
	}
}

func TestCostTracker_Reset(t *testing.T) {
	tracker := NewCostTracker(100.0)

	tracker.TrackCost("openai", "gpt-4", 0.05)
	tracker.TrackCost("anthropic", "claude", 0.03)

	if tracker.GetTotalCost() != 0.08 {
		t.Errorf("Expected total cost to be 0.08, got %f", tracker.GetTotalCost())
	}

	tracker.Reset()

	if tracker.GetTotalCost() != 0.0 {
		t.Errorf("Expected total cost to be 0.0 after reset, got %f", tracker.GetTotalCost())
	}

	if tracker.GetProviderCost("openai") != 0.0 {
		t.Errorf("Expected OpenAI cost to be 0.0 after reset, got %f", tracker.GetProviderCost("openai"))
	}
}

func TestCircuitBreaker_ShouldAllowRequest(t *testing.T) {
	cb := NewCircuitBreaker("openai", 5, 10*time.Second)

	for i := 0; i < 3; i++ {
		if !cb.ShouldAllowRequest() {
			t.Errorf("Should allow request %d", i+1)
		}
		cb.RecordFailure()
	}

	if !cb.ShouldAllowRequest() {
		t.Error("Should still allow request before threshold")
	}
	cb.RecordFailure()

	if !cb.ShouldAllowRequest() {
		t.Error("Should still allow request at threshold")
	}
	cb.RecordFailure()

	if cb.ShouldAllowRequest() {
		t.Error("Should not allow request after threshold exceeded")
	}
}

func TestCircuitBreaker_RecordSuccess(t *testing.T) {
	cb := NewCircuitBreaker("openai", 3, 10*time.Second)

	for i := 0; i < 4; i++ {
		cb.RecordFailure()
	}

	if cb.ShouldAllowRequest() {
		t.Error("Should not allow request after failures")
	}

	cb.RecordSuccess()

	if !cb.ShouldAllowRequest() {
		t.Error("Should allow request after success")
	}

	if cb.GetFailureCount() != 0 {
		t.Errorf("Expected failure count to be 0 after success, got %d", cb.GetFailureCount())
	}
}

func TestCircuitBreaker_Reset(t *testing.T) {
	cb := NewCircuitBreaker("openai", 3, 10*time.Second)

	for i := 0; i < 4; i++ {
		cb.RecordFailure()
	}

	if cb.ShouldAllowRequest() {
		t.Error("Should not allow request after failures")
	}

	cb.Reset()

	if !cb.ShouldAllowRequest() {
		t.Error("Should allow request after reset")
	}

	if cb.GetFailureCount() != 0 {
		t.Errorf("Expected failure count to be 0 after reset, got %d", cb.GetFailureCount())
	}
}

func TestCircuitBreaker_Recovery(t *testing.T) {
	cb := NewCircuitBreaker("openai", 3, 50*time.Millisecond)

	for i := 0; i < 4; i++ {
		cb.RecordFailure()
	}

	if cb.ShouldAllowRequest() {
		t.Error("Should not allow request immediately after failures")
	}

	time.Sleep(60 * time.Millisecond)

	if !cb.ShouldAllowRequest() {
		t.Error("Should allow request after recovery period")
	}
}

func TestLLMMonitor_GetActiveRequests(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:        "test-service",
		ServiceVersion:     "1.0.0",
		Environment:        "test",
		MetricsEnabled:     true,
		PrometheusEndpoint: "http://localhost:9090",
	}

	ms := NewMetricsService(config)
	err := ms.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize metrics service: %v", err)
	}
	defer ms.Shutdown(context.Background())

	monitor := NewLLMMonitor(config, ms)

	request1 := &LLMRequest{
		Provider:  "openai",
		Model:     "gpt-4",
		RequestID: "req-1",
		StartTime: time.Now(),
	}

	request2 := &LLMRequest{
		Provider:  "anthropic",
		Model:     "claude",
		RequestID: "req-2",
		StartTime: time.Now(),
	}

	monitor.RecordRequest(request1)
	monitor.RecordRequest(request2)

	activeRequests := monitor.GetActiveRequests()

	if len(activeRequests) != 2 {
		t.Errorf("Expected 2 active requests, got %d", len(activeRequests))
	}

	found1, found2 := false, false
	for _, req := range activeRequests {
		if req.RequestID == "req-1" {
			found1 = true
		}
		if req.RequestID == "req-2" {
			found2 = true
		}
	}

	if !found1 {
		t.Error("Request 1 should be in active requests")
	}

	if !found2 {
		t.Error("Request 2 should be in active requests")
	}
}

func TestLLMMonitor_GetStats(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:        "test-service",
		ServiceVersion:     "1.0.0",
		Environment:        "test",
		MetricsEnabled:     true,
		PrometheusEndpoint: "http://localhost:9090",
	}

	ms := NewMetricsService(config)
	err := ms.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize metrics service: %v", err)
	}
	defer ms.Shutdown(context.Background())

	monitor := NewLLMMonitor(config, ms)

	request := &LLMRequest{
		Provider:    "openai",
		Model:       "gpt-4",
		InputTokens: 100,
		RequestID:   "req-123",
		StartTime:   time.Now().Add(-time.Second),
	}

	monitor.RecordRequest(request)

	response := &LLMResponse{
		RequestID:     "req-123",
		Success:       true,
		OutputTokens:  150,
		TotalTokens:   250,
		Cost:          0.05,
		ResponseTime:  time.Second,
		CacheHit:      false,
		FinishReason:  "stop",
		ResponseData:  "Generated response",
		EndTime:       time.Now(),
	}

	monitor.RecordResponse(response)

	stats := monitor.GetStats()

	if stats.TotalRequests != 1 {
		t.Errorf("Expected 1 total request, got %d", stats.TotalRequests)
	}

	if stats.SuccessfulRequests != 1 {
		t.Errorf("Expected 1 successful request, got %d", stats.SuccessfulRequests)
	}

	if stats.TotalCost != 0.05 {
		t.Errorf("Expected total cost to be 0.05, got %f", stats.TotalCost)
	}

	if stats.TotalTokens != 250 {
		t.Errorf("Expected total tokens to be 250, got %d", stats.TotalTokens)
	}

	if len(stats.ProviderStats) != 1 {
		t.Errorf("Expected 1 provider stat, got %d", len(stats.ProviderStats))
	}

	openaiStats, exists := stats.ProviderStats["openai"]
	if !exists {
		t.Error("Expected OpenAI stats to exist")
	}

	if openaiStats.RequestCount != 1 {
		t.Errorf("Expected OpenAI request count to be 1, got %d", openaiStats.RequestCount)
	}

	if openaiStats.TotalCost != 0.05 {
		t.Errorf("Expected OpenAI total cost to be 0.05, got %f", openaiStats.TotalCost)
	}
}

func BenchmarkLLMMonitor_RecordRequest(b *testing.B) {
	config := &ObservabilityConfig{
		ServiceName:        "test-service",
		ServiceVersion:     "1.0.0",
		Environment:        "test",
		MetricsEnabled:     true,
		PrometheusEndpoint: "http://localhost:9090",
	}

	ms := NewMetricsService(config)
	err := ms.Initialize()
	if err != nil {
		b.Fatalf("Failed to initialize metrics service: %v", err)
	}
	defer ms.Shutdown(context.Background())

	monitor := NewLLMMonitor(config, ms)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		request := &LLMRequest{
			Provider:    "openai",
			Model:       "gpt-4",
			InputTokens: 100,
			RequestID:   "req-" + string(rune(i)),
			StartTime:   time.Now(),
		}
		monitor.RecordRequest(request)
	}
}

func BenchmarkCostTracker_TrackCost(b *testing.B) {
	tracker := NewCostTracker(1000.0)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		tracker.TrackCost("openai", "gpt-4", 0.01)
	}
}