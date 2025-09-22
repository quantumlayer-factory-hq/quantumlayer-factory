package observability

import (
	"context"
	"fmt"
	"time"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/exporters/jaeger"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/sdk/resource"
	"go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.21.0"
	oteltrace "go.opentelemetry.io/otel/trace"
)

// TracingService manages distributed tracing
type TracingService struct {
	config   *ObservabilityConfig
	tracer   oteltrace.Tracer
	provider *trace.TracerProvider
}

// NewTracingService creates a new tracing service
func NewTracingService(config *ObservabilityConfig) (*TracingService, error) {
	if !config.TracingEnabled {
		return &TracingService{
			config: config,
			tracer: otel.Tracer("noop"),
		}, nil
	}

	// Create Jaeger exporter
	exporter, err := jaeger.New(jaeger.WithCollectorEndpoint(
		jaeger.WithEndpoint(config.JaegerEndpoint),
	))
	if err != nil {
		return nil, fmt.Errorf("failed to create Jaeger exporter: %w", err)
	}

	// Create resource
	res, err := resource.New(context.Background(),
		resource.WithAttributes(
			semconv.ServiceName(config.ServiceName),
			semconv.ServiceVersion(config.ServiceVersion),
			semconv.DeploymentEnvironment(config.Environment),
		),
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create resource: %w", err)
	}

	// Create tracer provider
	provider := trace.NewTracerProvider(
		trace.WithBatcher(exporter),
		trace.WithResource(res),
		trace.WithSampler(trace.TraceIDRatioBased(config.TraceRatio)),
	)

	// Set global tracer provider
	otel.SetTracerProvider(provider)

	// Set global text map propagator
	otel.SetTextMapPropagator(propagation.NewCompositeTextMapPropagator(
		propagation.TraceContext{},
		propagation.Baggage{},
	))

	tracer := provider.Tracer(config.ServiceName)

	return &TracingService{
		config:   config,
		tracer:   tracer,
		provider: provider,
	}, nil
}

// StartSpan starts a new tracing span
func (ts *TracingService) StartSpan(ctx context.Context, name string, attrs ...attribute.KeyValue) (context.Context, oteltrace.Span) {
	return ts.tracer.Start(ctx, name, oteltrace.WithAttributes(attrs...))
}

// StartSpanWithOptions starts a span with additional options
func (ts *TracingService) StartSpanWithOptions(ctx context.Context, name string, opts ...oteltrace.SpanStartOption) (context.Context, oteltrace.Span) {
	return ts.tracer.Start(ctx, name, opts...)
}

// RecordError records an error in the current span
func (ts *TracingService) RecordError(span oteltrace.Span, err error, attrs ...attribute.KeyValue) {
	if err == nil {
		return
	}

	span.RecordError(err, oteltrace.WithAttributes(attrs...))
	span.SetStatus(codes.Error, err.Error())
}

// AddEvent adds an event to the current span
func (ts *TracingService) AddEvent(span oteltrace.Span, name string, attrs ...attribute.KeyValue) {
	span.AddEvent(name, oteltrace.WithAttributes(attrs...))
}

// Shutdown gracefully shuts down the tracing service
func (ts *TracingService) Shutdown(ctx context.Context) error {
	if ts.provider == nil {
		return nil
	}
	return ts.provider.Shutdown(ctx)
}

// Common span names
const (
	SpanGenerateApplication    = "generate.application"
	SpanCompileIR             = "ir.compile"
	SpanDetectOverlays        = "overlays.detect"
	SpanAgentGenerate         = "agent.generate"
	SpanLLMRequest            = "llm.request"
	SpanVerifyCode            = "verify.code"
	SpanExecuteGate           = "gate.execute"
	SpanRepairCode            = "repair.code"
	SpanBuildContainer        = "build.container"
	SpanScanSecurity          = "scan.security"
	SpanDeployApplication     = "deploy.application"
	SpanCreatePackage         = "package.create"
	SpanGenerateSBOM          = "sbom.generate"
	SpanSignPackage           = "package.sign"
	SpanDeliverPackage        = "package.deliver"
)

// Common attributes
var (
	AttrComponent      = attribute.Key("component")
	AttrOperation      = attribute.Key("operation")
	AttrLanguage       = attribute.Key("language")
	AttrFramework      = attribute.Key("framework")
	AttrProvider       = attribute.Key("provider")
	AttrModel          = attribute.Key("model")
	AttrOverlay        = attribute.Key("overlay")
	AttrAgentType      = attribute.Key("agent_type")
	AttrGateType       = attribute.Key("gate_type")
	AttrPackageSize    = attribute.Key("package_size")
	AttrSuccess        = attribute.Key("success")
	AttrErrorType      = attribute.Key("error_type")
	AttrTokensUsed     = attribute.Key("tokens_used")
	AttrCacheHit       = attribute.Key("cache_hit")
	AttrCost           = attribute.Key("cost")
	AttrLinesOfCode    = attribute.Key("lines_of_code")
	AttrFilesCount     = attribute.Key("files_count")
	AttrVulnCount      = attribute.Key("vulnerabilities_count")
	AttrQualityScore   = attribute.Key("quality_score")
)

// Helper functions for common tracing patterns

// TraceIRCompilation traces IR compilation with overlay detection
func (ts *TracingService) TraceIRCompilation(ctx context.Context, brief string, overlays []string) (context.Context, oteltrace.Span) {
	attrs := []attribute.KeyValue{
		AttrComponent.String("ir_compiler"),
		AttrOperation.String("compile_brief"),
		attribute.Int("brief_length", len(brief)),
		attribute.Int("overlays_count", len(overlays)),
	}

	if len(overlays) > 0 {
		attrs = append(attrs, AttrOverlay.StringSlice(overlays))
	}

	return ts.StartSpan(ctx, SpanCompileIR, attrs...)
}

// TraceAgentGeneration traces agent code generation
func (ts *TracingService) TraceAgentGeneration(ctx context.Context, agentType, language, framework string, llmUsed bool) (context.Context, oteltrace.Span) {
	attrs := []attribute.KeyValue{
		AttrComponent.String("agent_factory"),
		AttrOperation.String("generate_code"),
		AttrAgentType.String(agentType),
		AttrLanguage.String(language),
		attribute.Bool("llm_used", llmUsed),
	}

	if framework != "" {
		attrs = append(attrs, AttrFramework.String(framework))
	}

	return ts.StartSpan(ctx, SpanAgentGenerate, attrs...)
}

// TraceLLMRequest traces LLM API requests
func (ts *TracingService) TraceLLMRequest(ctx context.Context, provider, model string, tokensRequested int) (context.Context, oteltrace.Span) {
	attrs := []attribute.KeyValue{
		AttrComponent.String("llm_client"),
		AttrOperation.String("generate_completion"),
		AttrProvider.String(provider),
		AttrModel.String(model),
		attribute.Int("tokens_requested", tokensRequested),
	}

	return ts.StartSpan(ctx, SpanLLMRequest, attrs...)
}

// TraceVerificationGate traces verification gate execution
func (ts *TracingService) TraceVerificationGate(ctx context.Context, gateName, gateType string) (context.Context, oteltrace.Span) {
	attrs := []attribute.KeyValue{
		AttrComponent.String("verifier"),
		AttrOperation.String("execute_gate"),
		attribute.String("gate_name", gateName),
		AttrGateType.String(gateType),
	}

	return ts.StartSpan(ctx, SpanExecuteGate, attrs...)
}

// TracePackageCreation traces package creation process
func (ts *TracingService) TracePackageCreation(ctx context.Context, packageName, language, framework string) (context.Context, oteltrace.Span) {
	attrs := []attribute.KeyValue{
		AttrComponent.String("packager"),
		AttrOperation.String("create_package"),
		attribute.String("package_name", packageName),
		AttrLanguage.String(language),
	}

	if framework != "" {
		attrs = append(attrs, AttrFramework.String(framework))
	}

	return ts.StartSpan(ctx, SpanCreatePackage, attrs...)
}

// TraceDeployment traces application deployment
func (ts *TracingService) TraceDeployment(ctx context.Context, appName, namespace string, replicas int) (context.Context, oteltrace.Span) {
	attrs := []attribute.KeyValue{
		AttrComponent.String("deploy_service"),
		AttrOperation.String("deploy_application"),
		attribute.String("app_name", appName),
		attribute.String("namespace", namespace),
		attribute.Int("replicas", replicas),
	}

	return ts.StartSpan(ctx, SpanDeployApplication, attrs...)
}

// RecordSpanResult records common span completion attributes
func (ts *TracingService) RecordSpanResult(span oteltrace.Span, duration time.Duration, success bool, errorType string) {
	attrs := []attribute.KeyValue{
		attribute.Int64("duration_ms", duration.Milliseconds()),
		AttrSuccess.Bool(success),
	}

	if !success && errorType != "" {
		attrs = append(attrs, AttrErrorType.String(errorType))
		span.SetStatus(codes.Error, errorType)
	} else {
		span.SetStatus(codes.Ok, "")
	}

	span.SetAttributes(attrs...)
}

// WithTracing wraps a function with automatic tracing
func (ts *TracingService) WithTracing(ctx context.Context, spanName string, fn func(context.Context) error, attrs ...attribute.KeyValue) error {
	ctx, span := ts.StartSpan(ctx, spanName, attrs...)
	defer span.End()

	start := time.Now()
	err := fn(ctx)
	duration := time.Since(start)

	if err != nil {
		ts.RecordError(span, err)
		ts.RecordSpanResult(span, duration, false, err.Error())
	} else {
		ts.RecordSpanResult(span, duration, true, "")
	}

	return err
}

// WithTracingResult wraps a function with automatic tracing and result capture
func WithTracingResult[T any](ts *TracingService, ctx context.Context, spanName string, fn func(context.Context) (T, error), attrs ...attribute.KeyValue) (T, error) {
	ctx, span := ts.StartSpan(ctx, spanName, attrs...)
	defer span.End()

	start := time.Now()
	result, err := fn(ctx)
	duration := time.Since(start)

	if err != nil {
		ts.RecordError(span, err)
		ts.RecordSpanResult(span, duration, false, err.Error())
	} else {
		ts.RecordSpanResult(span, duration, true, "")
	}

	return result, err
}