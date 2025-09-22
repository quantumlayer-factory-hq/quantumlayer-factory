package observability

import (
	"context"
	"testing"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

func TestTracingService_InitializeTracing(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Environment:     "test",
		TracingEnabled:  true,
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		TraceRatio:      1.0,
	}

	ts := NewTracingService(config)

	err := ts.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	if ts.tracer == nil {
		t.Error("Tracer should not be nil after initialization")
	}

	if ts.shutdown == nil {
		t.Error("Shutdown function should not be nil after initialization")
	}

	err = ts.Shutdown(context.Background())
	if err != nil {
		t.Errorf("Failed to shutdown tracing: %v", err)
	}
}

func TestTracingService_StartSpan(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Environment:     "test",
		TracingEnabled:  true,
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		TraceRatio:      1.0,
	}

	ts := NewTracingService(config)
	err := ts.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer ts.Shutdown(context.Background())

	ctx := context.Background()
	spanName := "test-operation"
	attrs := []attribute.KeyValue{
		attribute.String("test.key", "test.value"),
		attribute.Int("test.number", 42),
	}

	spanCtx, span := ts.StartSpan(ctx, spanName, attrs...)

	if span == nil {
		t.Error("Span should not be nil")
	}

	if spanCtx == ctx {
		t.Error("Span context should be different from original context")
	}

	span.End()
}

func TestTracingService_TraceGeneration(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Environment:     "test",
		TracingEnabled:  true,
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		TraceRatio:      1.0,
	}

	ts := NewTracingService(config)
	err := ts.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer ts.Shutdown(context.Background())

	ctx := context.Background()
	requestID := "test-request-123"
	agentType := "go-generator"
	language := "go"
	framework := "gin"

	resultCtx := ts.TraceGeneration(ctx, requestID, agentType, language, framework, func(ctx context.Context) error {
		time.Sleep(10 * time.Millisecond)
		return nil
	})

	if resultCtx == ctx {
		t.Error("Result context should be different from original context")
	}

	span := trace.SpanFromContext(resultCtx)
	if span == nil {
		t.Error("Span should be present in result context")
	}
}

func TestTracingService_TraceLLMRequest(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Environment:     "test",
		TracingEnabled:  true,
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		TraceRatio:      1.0,
	}

	ts := NewTracingService(config)
	err := ts.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer ts.Shutdown(context.Background())

	ctx := context.Background()
	provider := "openai"
	model := "gpt-4"
	tokens := 150

	resultCtx := ts.TraceLLMRequest(ctx, provider, model, tokens, func(ctx context.Context) error {
		time.Sleep(5 * time.Millisecond)
		return nil
	})

	if resultCtx == ctx {
		t.Error("Result context should be different from original context")
	}

	span := trace.SpanFromContext(resultCtx)
	if span == nil {
		t.Error("Span should be present in result context")
	}
}

func TestTracingService_TracePackaging(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Environment:     "test",
		TracingEnabled:  true,
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		TraceRatio:      1.0,
	}

	ts := NewTracingService(config)
	err := ts.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer ts.Shutdown(context.Background())

	ctx := context.Background()
	packageID := "pkg-123"
	packageType := "qlcapsule"

	resultCtx := ts.TracePackaging(ctx, packageID, packageType, func(ctx context.Context) error {
		time.Sleep(20 * time.Millisecond)
		return nil
	})

	if resultCtx == ctx {
		t.Error("Result context should be different from original context")
	}

	span := trace.SpanFromContext(resultCtx)
	if span == nil {
		t.Error("Span should be present in result context")
	}
}

func TestTracingService_TraceDeployment(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Environment:     "test",
		TracingEnabled:  true,
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		TraceRatio:      1.0,
	}

	ts := NewTracingService(config)
	err := ts.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer ts.Shutdown(context.Background())

	ctx := context.Background()
	deploymentID := "deploy-456"
	namespace := "production"
	appName := "my-app"

	resultCtx := ts.TraceDeployment(ctx, deploymentID, namespace, appName, func(ctx context.Context) error {
		time.Sleep(15 * time.Millisecond)
		return nil
	})

	if resultCtx == ctx {
		t.Error("Result context should be different from original context")
	}

	span := trace.SpanFromContext(resultCtx)
	if span == nil {
		t.Error("Span should be present in result context")
	}
}

func TestTracingService_TracingDisabled(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Environment:     "test",
		TracingEnabled:  false,
	}

	ts := NewTracingService(config)
	err := ts.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}

	ctx := context.Background()
	spanCtx, span := ts.StartSpan(ctx, "test-span")

	if spanCtx != ctx {
		t.Error("When tracing is disabled, context should remain unchanged")
	}

	if !span.SpanContext().IsValid() {
		t.Error("Should have a valid span context even when tracing is disabled")
	}

	span.End()
}

func TestTracingService_AddSpanAttributes(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Environment:     "test",
		TracingEnabled:  true,
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		TraceRatio:      1.0,
	}

	ts := NewTracingService(config)
	err := ts.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer ts.Shutdown(context.Background())

	ctx := context.Background()
	spanCtx, span := ts.StartSpan(ctx, "test-span")

	attrs := []attribute.KeyValue{
		attribute.String("custom.field", "custom.value"),
		attribute.Bool("custom.flag", true),
	}

	ts.AddSpanAttributes(spanCtx, attrs...)

	span.End()
}

func TestTracingService_RecordError(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Environment:     "test",
		TracingEnabled:  true,
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		TraceRatio:      1.0,
	}

	ts := NewTracingService(config)
	err := ts.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer ts.Shutdown(context.Background())

	ctx := context.Background()
	spanCtx, span := ts.StartSpan(ctx, "test-span")

	testErr := &TestError{message: "test error message"}
	ts.RecordError(spanCtx, testErr)

	span.End()
}

type TestError struct {
	message string
}

func (e *TestError) Error() string {
	return e.message
}

func TestTracingService_ExtractTraceID(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Environment:     "test",
		TracingEnabled:  true,
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		TraceRatio:      1.0,
	}

	ts := NewTracingService(config)
	err := ts.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer ts.Shutdown(context.Background())

	ctx := context.Background()
	spanCtx, span := ts.StartSpan(ctx, "test-span")

	traceID := ts.ExtractTraceID(spanCtx)
	if traceID == "" {
		t.Error("Trace ID should not be empty")
	}

	if len(traceID) != 32 {
		t.Errorf("Trace ID should be 32 characters long, got %d", len(traceID))
	}

	span.End()
}

func TestTracingService_ExtractSpanID(t *testing.T) {
	config := &ObservabilityConfig{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Environment:     "test",
		TracingEnabled:  true,
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		TraceRatio:      1.0,
	}

	ts := NewTracingService(config)
	err := ts.Initialize()
	if err != nil {
		t.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer ts.Shutdown(context.Background())

	ctx := context.Background()
	spanCtx, span := ts.StartSpan(ctx, "test-span")

	spanID := ts.ExtractSpanID(spanCtx)
	if spanID == "" {
		t.Error("Span ID should not be empty")
	}

	if len(spanID) != 16 {
		t.Errorf("Span ID should be 16 characters long, got %d", len(spanID))
	}

	span.End()
}

func BenchmarkTracingService_StartSpan(b *testing.B) {
	config := &ObservabilityConfig{
		ServiceName:     "test-service",
		ServiceVersion:  "1.0.0",
		Environment:     "test",
		TracingEnabled:  true,
		JaegerEndpoint:  "http://localhost:14268/api/traces",
		TraceRatio:      1.0,
	}

	ts := NewTracingService(config)
	err := ts.Initialize()
	if err != nil {
		b.Fatalf("Failed to initialize tracing: %v", err)
	}
	defer ts.Shutdown(context.Background())

	ctx := context.Background()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, span := ts.StartSpan(ctx, "benchmark-span")
		span.End()
	}
}