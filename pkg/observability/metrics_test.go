package observability

import (
	"context"
	"testing"
	"time"
)

func TestMetricsService_Initialize(t *testing.T) {
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
		t.Fatalf("Failed to initialize metrics: %v", err)
	}

	if ms.metrics == nil {
		t.Error("Metrics should not be nil after initialization")
	}

	if ms.shutdown == nil {
		t.Error("Shutdown function should not be nil after initialization")
	}

	err = ms.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Failed to shutdown metrics: %v", err)
	}
}

func TestMetricsService_RecordGeneration(t *testing.T) {
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
		t.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer ms.Shutdown(context.Background())

	labels := &MetricLabels{
		AgentType: "go-generator",
		Language:  "go",
		Framework: "gin",
		Result:    "success",
	}

	duration := 2 * time.Second
	success := true

	ms.RecordGeneration(duration, success, labels)
}

func TestMetricsService_RecordLLMRequest(t *testing.T) {
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
		t.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer ms.Shutdown(context.Background())

	labels := &MetricLabels{
		Provider: "openai",
		Model:    "gpt-4",
		Result:   "success",
	}

	duration := 1500 * time.Millisecond
	tokens := 150
	cost := 0.03

	ms.RecordLLMRequest(duration, tokens, cost, labels)
}

func TestMetricsService_RecordPackaging(t *testing.T) {
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
		t.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer ms.Shutdown(context.Background())

	labels := &MetricLabels{
		PackageType: "qlcapsule",
		Result:      "success",
	}

	duration := 30 * time.Second
	sizeBytes := int64(1024 * 1024 * 5)

	ms.RecordPackaging(duration, sizeBytes, labels)
}

func TestMetricsService_RecordDeployment(t *testing.T) {
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
		t.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer ms.Shutdown(context.Background())

	labels := &MetricLabels{
		Namespace: "production",
		AppName:   "my-app",
		Result:    "success",
	}

	duration := 120 * time.Second

	ms.RecordDeployment(duration, labels)
}

func TestMetricsService_RecordVerification(t *testing.T) {
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
		t.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer ms.Shutdown(context.Background())

	labels := &MetricLabels{
		GateType: "security",
		Result:   "passed",
	}

	duration := 5 * time.Second
	score := 8.5

	ms.RecordVerification(duration, score, labels)
}

func TestMetricsService_RecordHealthCheck(t *testing.T) {
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
		t.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer ms.Shutdown(context.Background())

	labels := &MetricLabels{
		Component: "database",
		CheckName: "postgres-ping",
	}

	duration := 50 * time.Millisecond
	status := 1.0

	ms.RecordHealthCheck(duration, status, labels)
}

func TestMetricsService_IncrementError(t *testing.T) {
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
		t.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer ms.Shutdown(context.Background())

	labels := &MetricLabels{
		Component: "llm",
		ErrorType: "timeout",
	}

	ms.IncrementError(labels)
}

func TestMetricsService_SetActiveWorkflows(t *testing.T) {
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
		t.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer ms.Shutdown(context.Background())

	count := 15
	ms.SetActiveWorkflows(count)
}

func TestMetricsService_SetActivePreviewEnvironments(t *testing.T) {
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
		t.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer ms.Shutdown(context.Background())

	count := 8
	ms.SetActivePreviewEnvironments(count)
}

func TestMetricsService_RecordQualityScore(t *testing.T) {
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
		t.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer ms.Shutdown(context.Background())

	labels := &MetricLabels{
		Language:  "go",
		Framework: "gin",
	}

	score := 9.2
	ms.RecordQualityScore(score, labels)
}

func TestMetricsService_MetricsDisabled(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:    "test-service",
		ServiceVersion: "1.0.0",
		Environment:    "test",
		MetricsEnabled: false,
	}

	ms := NewMetricsService(config)
	err := ms.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize metrics: %v", err)
	}

	if ms.metrics != nil {
		t.Error("Metrics should be nil when disabled")
	}

	labels := &MetricLabels{
		AgentType: "test-agent",
		Result:    "success",
	}

	ms.RecordGeneration(time.Second, true, labels)
}

func TestMetricLabels_ToAttributes(t *testing.T) {
	labels := &MetricLabels{
		AgentType:     "go-generator",
		Language:      "go",
		Framework:     "gin",
		Provider:      "openai",
		Model:         "gpt-4",
		Result:        "success",
		Component:     "generation",
		ErrorType:     "timeout",
		PackageType:   "qlcapsule",
		Namespace:     "production",
		AppName:       "my-app",
		GateType:      "security",
		CheckName:     "database-ping",
	}

	attrs := labels.ToAttributes()

	if len(attrs) == 0 {
		t.Error("Attributes should not be empty")
	}

	found := make(map[string]bool)
	for _, attr := range attrs {
		found[string(attr.Key)] = true
	}

	expectedKeys := []string{
		"agent_type", "language", "framework", "provider", "model",
		"result", "component", "error_type", "package_type",
		"namespace", "app_name", "gate_type", "check_name",
	}

	for _, key := range expectedKeys {
		if !found[key] {
			t.Errorf("Expected attribute key %s not found", key)
		}
	}
}

func TestMetricLabels_ToPromLabels(t *testing.T) {
	labels := &MetricLabels{
		AgentType: "go-generator",
		Language:  "go",
		Framework: "gin",
		Result:    "success",
	}

	promLabels := labels.ToPromLabels()

	if len(promLabels) == 0 {
		t.Error("Prometheus labels should not be empty")
	}

	if promLabels["agent_type"] != "go-generator" {
		t.Errorf("Expected agent_type to be 'go-generator', got %s", promLabels["agent_type"])
	}

	if promLabels["language"] != "go" {
		t.Errorf("Expected language to be 'go', got %s", promLabels["language"])
	}

	if promLabels["framework"] != "gin" {
		t.Errorf("Expected framework to be 'gin', got %s", promLabels["framework"])
	}

	if promLabels["result"] != "success" {
		t.Errorf("Expected result to be 'success', got %s", promLabels["result"])
	}
}

func BenchmarkMetricsService_RecordGeneration(b *testing.B) {
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
		b.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer ms.Shutdown(context.Background())

	labels := &MetricLabels{
		AgentType: "go-generator",
		Language:  "go",
		Framework: "gin",
		Result:    "success",
	}

	duration := time.Second
	success := true

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ms.RecordGeneration(duration, success, labels)
	}
}

func BenchmarkMetricsService_IncrementError(b *testing.B) {
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
		b.Fatalf("Failed to initialize metrics: %v", err)
	}
	defer ms.Shutdown(context.Background())

	labels := &MetricLabels{
		Component: "llm",
		ErrorType: "timeout",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ms.IncrementError(labels)
	}
}