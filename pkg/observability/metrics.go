package observability

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	promexporter "go.opentelemetry.io/otel/exporters/prometheus"
	"go.opentelemetry.io/otel/metric"
	sdkmetric "go.opentelemetry.io/otel/sdk/metric"
)

// MetricsService manages Prometheus metrics collection
type MetricsService struct {
	config     *ObservabilityConfig
	provider   *sdkmetric.MeterProvider
	meter      metric.Meter
	metrics    *Metrics
	server     *http.Server
}

// NewMetricsService creates a new metrics service
func NewMetricsService(config *ObservabilityConfig) (*MetricsService, error) {
	if !config.MetricsEnabled {
		return &MetricsService{
			config: config,
		}, nil
	}

	// Create Prometheus exporter
	exporter, err := promexporter.New()
	if err != nil {
		return nil, fmt.Errorf("failed to create Prometheus exporter: %w", err)
	}

	// Create meter provider
	provider := sdkmetric.NewMeterProvider(sdkmetric.WithReader(exporter))
	otel.SetMeterProvider(provider)

	meter := provider.Meter(config.ServiceName)

	// Initialize metrics
	metrics, err := initializeMetrics(meter)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize metrics: %w", err)
	}

	// Create HTTP server for metrics endpoint
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())

	server := &http.Server{
		Addr:    fmt.Sprintf(":%d", config.MetricsPort),
		Handler: mux,
	}

	ms := &MetricsService{
		config:   config,
		provider: provider,
		meter:    meter,
		metrics:  metrics,
		server:   server,
	}

	return ms, nil
}

// Start starts the metrics HTTP server
func (ms *MetricsService) Start() error {
	if !ms.config.MetricsEnabled {
		return nil
	}

	go func() {
		if err := ms.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			fmt.Printf("Metrics server error: %v\n", err)
		}
	}()

	return nil
}

// Shutdown gracefully shuts down the metrics service
func (ms *MetricsService) Shutdown(ctx context.Context) error {
	if ms.server != nil {
		if err := ms.server.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown metrics server: %w", err)
		}
	}

	if ms.provider != nil {
		if err := ms.provider.Shutdown(ctx); err != nil {
			return fmt.Errorf("failed to shutdown meter provider: %w", err)
		}
	}

	return nil
}

// GetMetrics returns the metrics instance
func (ms *MetricsService) GetMetrics() *Metrics {
	return ms.metrics
}

// RecordGeneration records generation metrics
func (ms *MetricsService) RecordGeneration(duration time.Duration, success bool, labels *MetricLabels) {
	if !ms.config.MetricsEnabled {
		return
	}

	attrs := labels.ToAttributes()
	attrs = append(attrs, AttrSuccess.Bool(success))

	ms.metrics.GenerationCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	ms.metrics.GenerationDuration.Record(context.Background(), duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordIRCompilation records IR compilation metrics
func (ms *MetricsService) RecordIRCompilation(duration time.Duration, success bool, overlaysDetected int) {
	if !ms.config.MetricsEnabled {
		return
	}

	attrs := []attribute.KeyValue{
		AttrComponent.String("ir_compiler"),
		AttrSuccess.Bool(success),
		attribute.Int("overlays_detected", overlaysDetected),
	}

	ms.metrics.IRCompilationTime.Record(context.Background(), duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordAgentExecution records agent execution metrics
func (ms *MetricsService) RecordAgentExecution(metrics *AgentMetrics) {
	if !ms.config.MetricsEnabled {
		return
	}

	attrs := []attribute.KeyValue{
		AttrComponent.String("agent_factory"),
		AttrAgentType.String(metrics.AgentType),
		AttrLanguage.String(metrics.Language),
		AttrFramework.String(metrics.Framework),
		AttrSuccess.Bool(metrics.Success),
		attribute.Bool("llm_used", metrics.LLMUsed),
		AttrLinesOfCode.Int(metrics.LinesOfCode),
		AttrFilesCount.Int(metrics.FilesCount),
	}

	if !metrics.Success && metrics.ErrorType != "" {
		attrs = append(attrs, AttrErrorType.String(metrics.ErrorType))
	}

	ms.metrics.AgentExecutionTime.Record(context.Background(), metrics.Duration.Seconds(), metric.WithAttributes(attrs...))
}

// RecordLLMUsage records LLM usage metrics
func (ms *MetricsService) RecordLLMUsage(metrics *LLMMetrics) {
	if !ms.config.MetricsEnabled {
		return
	}

	attrs := []attribute.KeyValue{
		AttrComponent.String("llm_client"),
		AttrProvider.String(metrics.Provider),
		AttrModel.String(metrics.Model),
		AttrSuccess.Bool(metrics.Success),
		AttrCacheHit.Bool(metrics.CacheHit),
		AttrTokensUsed.Int64(metrics.TokensUsed),
		AttrCost.Float64(metrics.Cost),
	}

	if !metrics.Success && metrics.ErrorType != "" {
		attrs = append(attrs, AttrErrorType.String(metrics.ErrorType))
	}

	ms.metrics.LLMRequestCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	ms.metrics.LLMTokenUsage.Add(context.Background(), metrics.TokensUsed, metric.WithAttributes(attrs...))
	ms.metrics.LLMLatency.Record(context.Background(), metrics.Latency.Seconds(), metric.WithAttributes(attrs...))

	if metrics.CacheHit {
		ms.metrics.LLMCacheHitCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	}

	// Update cost gauge (this would typically be done periodically)
	ms.metrics.LLMCostGauge.Record(context.Background(), metrics.Cost, metric.WithAttributes(attrs...))
}

// RecordVerification records verification gate metrics
func (ms *MetricsService) RecordVerification(metrics *VerificationMetrics) {
	if !ms.config.MetricsEnabled {
		return
	}

	attrs := []attribute.KeyValue{
		AttrComponent.String("verifier"),
		attribute.String("gate_name", metrics.GateName),
		AttrGateType.String(metrics.GateType),
		AttrSuccess.Bool(metrics.Success),
		attribute.Int("issues_found", metrics.IssuesFound),
		attribute.Bool("repair_attempted", metrics.RepairAttempt),
		attribute.Bool("repair_success", metrics.RepairSuccess),
		AttrQualityScore.Float64(metrics.QualityScore),
	}

	ms.metrics.VerificationCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	ms.metrics.GateExecutionTime.Record(context.Background(), metrics.Duration.Seconds(), metric.WithAttributes(attrs...))
	ms.metrics.QualityScore.Record(context.Background(), metrics.QualityScore, metric.WithAttributes(attrs...))

	if metrics.RepairAttempt {
		ms.metrics.RepairAttempts.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	}
}

// RecordPackaging records packaging metrics
func (ms *MetricsService) RecordPackaging(metrics *PackageMetrics) {
	if !ms.config.MetricsEnabled {
		return
	}

	attrs := []attribute.KeyValue{
		AttrComponent.String("packager"),
		attribute.String("package_name", metrics.PackageName),
		AttrLanguage.String(metrics.Language),
		AttrFramework.String(metrics.Framework),
		AttrSuccess.Bool(metrics.Success),
		AttrPackageSize.Int64(metrics.Size),
		attribute.Float64("compression_ratio", metrics.CompressionRatio),
		AttrVulnCount.Int(metrics.VulnerabilitiesFound),
		attribute.Int("delivery_channels", len(metrics.DeliveryChannels)),
	}

	if !metrics.Success && metrics.ErrorType != "" {
		attrs = append(attrs, AttrErrorType.String(metrics.ErrorType))
	}

	ms.metrics.PackageCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	ms.metrics.PackageSize.Record(context.Background(), metrics.Size, metric.WithAttributes(attrs...))
	ms.metrics.PackageBuildTime.Record(context.Background(), metrics.BuildTime.Seconds(), metric.WithAttributes(attrs...))
	ms.metrics.SBOMGenerationTime.Record(context.Background(), metrics.SBOMTime.Seconds(), metric.WithAttributes(attrs...))
	ms.metrics.VulnScanTime.Record(context.Background(), metrics.VulnScanTime.Seconds(), metric.WithAttributes(attrs...))
}

// RecordDeployment records deployment metrics
func (ms *MetricsService) RecordDeployment(metrics *DeploymentMetrics) {
	if !ms.config.MetricsEnabled {
		return
	}

	attrs := []attribute.KeyValue{
		AttrComponent.String("deploy_service"),
		attribute.String("namespace", metrics.Namespace),
		attribute.String("app_name", metrics.AppName),
		AttrLanguage.String(metrics.Language),
		AttrSuccess.Bool(metrics.Success),
		attribute.Int("replicas", metrics.Replicas),
		attribute.Int("health_checks", metrics.HealthChecks),
		attribute.Int64("ttl_seconds", int64(metrics.TTL.Seconds())),
	}

	if !metrics.Success && metrics.ErrorType != "" {
		attrs = append(attrs, AttrErrorType.String(metrics.ErrorType))
	}

	ms.metrics.DeploymentCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	ms.metrics.DeploymentTime.Record(context.Background(), metrics.Duration.Seconds(), metric.WithAttributes(attrs...))
}

// UpdateActiveWorkflows updates the active workflows gauge
func (ms *MetricsService) UpdateActiveWorkflows(count int64) {
	if !ms.config.MetricsEnabled {
		return
	}

	ms.metrics.ActiveWorkflows.Record(context.Background(), count)
}

// UpdatePreviewEnvironments updates the preview environments gauge
func (ms *MetricsService) UpdatePreviewEnvironments(count int64) {
	if !ms.config.MetricsEnabled {
		return
	}

	ms.metrics.PreviewEnvironments.Record(context.Background(), count)
}

// RecordError records an error occurrence
func (ms *MetricsService) RecordError(component, errorType string) {
	if !ms.config.MetricsEnabled {
		return
	}

	attrs := []attribute.KeyValue{
		AttrComponent.String(component),
		AttrErrorType.String(errorType),
	}

	ms.metrics.ErrorCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
}

// RecordHealthCheck records health check metrics
func (ms *MetricsService) RecordHealthCheck(duration time.Duration, status float64, labels *MetricLabels) {
	if !ms.config.MetricsEnabled {
		return
	}

	attrs := labels.ToAttributes()
	ms.metrics.HealthCheckCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
}

// UpdateServiceUptime updates service uptime gauge
func (ms *MetricsService) UpdateServiceUptime(service string, uptimeSeconds float64) {
	if !ms.config.MetricsEnabled {
		return
	}

	attrs := []attribute.KeyValue{
		attribute.String("service", service),
	}

	ms.metrics.ServiceUptime.Record(context.Background(), uptimeSeconds, metric.WithAttributes(attrs...))
}

// initializeMetrics creates all metric instruments
func initializeMetrics(meter metric.Meter) (*Metrics, error) {
	// Generation metrics
	generationCounter, err := meter.Int64Counter(
		"qlf_generations_total",
		metric.WithDescription("Total number of application generations"),
	)
	if err != nil {
		return nil, err
	}

	generationDuration, err := meter.Float64Histogram(
		"qlf_generation_duration_seconds",
		metric.WithDescription("Duration of application generation"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	irCompilationTime, err := meter.Float64Histogram(
		"qlf_ir_compilation_duration_seconds",
		metric.WithDescription("Duration of IR compilation"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	agentExecutionTime, err := meter.Float64Histogram(
		"qlf_agent_execution_duration_seconds",
		metric.WithDescription("Duration of agent code generation"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	// LLM metrics
	llmRequestCounter, err := meter.Int64Counter(
		"qlf_llm_requests_total",
		metric.WithDescription("Total number of LLM requests"),
	)
	if err != nil {
		return nil, err
	}

	llmTokenUsage, err := meter.Int64Counter(
		"qlf_llm_tokens_total",
		metric.WithDescription("Total number of tokens used"),
	)
	if err != nil {
		return nil, err
	}

	llmCostGauge, err := meter.Float64Gauge(
		"qlf_llm_cost_usd",
		metric.WithDescription("Current LLM costs in USD"),
		metric.WithUnit("USD"),
	)
	if err != nil {
		return nil, err
	}

	llmLatency, err := meter.Float64Histogram(
		"qlf_llm_latency_seconds",
		metric.WithDescription("LLM request latency"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	llmCacheHitCounter, err := meter.Int64Counter(
		"qlf_llm_cache_hits_total",
		metric.WithDescription("Total number of LLM cache hits"),
	)
	if err != nil {
		return nil, err
	}

	// Verification metrics
	verificationCounter, err := meter.Int64Counter(
		"qlf_verifications_total",
		metric.WithDescription("Total number of verification gate executions"),
	)
	if err != nil {
		return nil, err
	}

	gateExecutionTime, err := meter.Float64Histogram(
		"qlf_gate_execution_duration_seconds",
		metric.WithDescription("Duration of verification gate execution"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	repairAttempts, err := meter.Int64Counter(
		"qlf_repair_attempts_total",
		metric.WithDescription("Total number of repair attempts"),
	)
	if err != nil {
		return nil, err
	}

	qualityScore, err := meter.Float64Gauge(
		"qlf_quality_score",
		metric.WithDescription("Code quality score"),
	)
	if err != nil {
		return nil, err
	}

	// Package metrics
	packageCounter, err := meter.Int64Counter(
		"qlf_packages_created_total",
		metric.WithDescription("Total number of packages created"),
	)
	if err != nil {
		return nil, err
	}

	packageSize, err := meter.Int64Histogram(
		"qlf_package_size_bytes",
		metric.WithDescription("Package size in bytes"),
		metric.WithUnit("bytes"),
	)
	if err != nil {
		return nil, err
	}

	packageBuildTime, err := meter.Float64Histogram(
		"qlf_package_build_duration_seconds",
		metric.WithDescription("Duration of package creation"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	sbomGenerationTime, err := meter.Float64Histogram(
		"qlf_sbom_generation_duration_seconds",
		metric.WithDescription("Duration of SBOM generation"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	vulnScanTime, err := meter.Float64Histogram(
		"qlf_vuln_scan_duration_seconds",
		metric.WithDescription("Duration of vulnerability scanning"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	// Deploy metrics
	deploymentCounter, err := meter.Int64Counter(
		"qlf_deployments_total",
		metric.WithDescription("Total number of deployments"),
	)
	if err != nil {
		return nil, err
	}

	deploymentTime, err := meter.Float64Histogram(
		"qlf_deployment_duration_seconds",
		metric.WithDescription("Duration of application deployment"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	previewEnvironments, err := meter.Int64Gauge(
		"qlf_preview_environments_active",
		metric.WithDescription("Number of active preview environments"),
	)
	if err != nil {
		return nil, err
	}

	healthCheckCounter, err := meter.Int64Counter(
		"qlf_health_checks_total",
		metric.WithDescription("Total number of health checks performed"),
	)
	if err != nil {
		return nil, err
	}

	// System metrics
	activeWorkflows, err := meter.Int64Gauge(
		"qlf_active_workflows",
		metric.WithDescription("Number of active workflows"),
	)
	if err != nil {
		return nil, err
	}

	errorCounter, err := meter.Int64Counter(
		"qlf_errors_total",
		metric.WithDescription("Total number of errors"),
	)
	if err != nil {
		return nil, err
	}

	serviceUptime, err := meter.Float64Gauge(
		"qlf_service_uptime_seconds",
		metric.WithDescription("Service uptime in seconds"),
		metric.WithUnit("s"),
	)
	if err != nil {
		return nil, err
	}

	return &Metrics{
		GenerationCounter:    generationCounter,
		GenerationDuration:   generationDuration,
		IRCompilationTime:    irCompilationTime,
		AgentExecutionTime:   agentExecutionTime,
		LLMRequestCounter:    llmRequestCounter,
		LLMTokenUsage:        llmTokenUsage,
		LLMCostGauge:         llmCostGauge,
		LLMLatency:          llmLatency,
		LLMCacheHitCounter:   llmCacheHitCounter,
		VerificationCounter:  verificationCounter,
		GateExecutionTime:    gateExecutionTime,
		RepairAttempts:       repairAttempts,
		QualityScore:         qualityScore,
		PackageCounter:       packageCounter,
		PackageSize:          packageSize,
		PackageBuildTime:     packageBuildTime,
		SBOMGenerationTime:   sbomGenerationTime,
		VulnScanTime:         vulnScanTime,
		DeploymentCounter:    deploymentCounter,
		DeploymentTime:       deploymentTime,
		PreviewEnvironments:  previewEnvironments,
		HealthCheckCounter:   healthCheckCounter,
		ActiveWorkflows:      activeWorkflows,
		ErrorCounter:         errorCounter,
		ServiceUptime:        serviceUptime,
	}, nil
}

// MiddlewareFunc returns HTTP middleware for automatic metrics collection
func (ms *MetricsService) MiddlewareFunc() func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			start := time.Now()

			// Create a response writer that captures status code
			rw := &responseWriter{ResponseWriter: w, statusCode: http.StatusOK}

			// Process request
			next.ServeHTTP(rw, r)

			// Record metrics
			duration := time.Since(start)
			attrs := []attribute.KeyValue{
				attribute.String("method", r.Method),
				attribute.String("path", r.URL.Path),
				attribute.Int("status_code", rw.statusCode),
				AttrSuccess.Bool(rw.statusCode < 400),
			}

			// Record request metrics
			ms.metrics.GenerationDuration.Record(r.Context(), duration.Seconds(), metric.WithAttributes(attrs...))

			if rw.statusCode >= 400 {
				ms.RecordError("http_server", fmt.Sprintf("status_%d", rw.statusCode))
			}
		})
	}
}

// responseWriter wraps http.ResponseWriter to capture status code
type responseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (rw *responseWriter) WriteHeader(code int) {
	rw.statusCode = code
	rw.ResponseWriter.WriteHeader(code)
}

// Common metric recording helpers

// WithMetrics wraps a function with automatic metrics recording
func (ms *MetricsService) WithMetrics(component string, fn func() error) error {
	start := time.Now()
	err := fn()
	duration := time.Since(start)

	labels := &MetricLabels{
		Component: component,
		Result:    "success",
	}

	if err != nil {
		labels.Result = "error"
		ms.RecordError(component, err.Error())
	}

	ms.RecordGeneration(duration, err == nil, labels)
	return err
}

// IncrementCounter increments a named counter with labels
func (ms *MetricsService) IncrementCounter(name string, labels *MetricLabels) {
	if !ms.config.MetricsEnabled {
		return
	}

	attrs := labels.ToAttributes()

	switch name {
	case "generations":
		ms.metrics.GenerationCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	case "llm_requests":
		ms.metrics.LLMRequestCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	case "verifications":
		ms.metrics.VerificationCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	case "packages":
		ms.metrics.PackageCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	case "deployments":
		ms.metrics.DeploymentCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	}
}

// RecordHistogram records a histogram value with labels
func (ms *MetricsService) RecordHistogram(name string, value float64, labels *MetricLabels) {
	if !ms.config.MetricsEnabled {
		return
	}

	attrs := labels.ToAttributes()

	switch name {
	case "generation_duration":
		ms.metrics.GenerationDuration.Record(context.Background(), value, metric.WithAttributes(attrs...))
	case "agent_execution_time":
		ms.metrics.AgentExecutionTime.Record(context.Background(), value, metric.WithAttributes(attrs...))
	case "llm_latency":
		ms.metrics.LLMLatency.Record(context.Background(), value, metric.WithAttributes(attrs...))
	case "gate_execution_time":
		ms.metrics.GateExecutionTime.Record(context.Background(), value, metric.WithAttributes(attrs...))
	case "package_build_time":
		ms.metrics.PackageBuildTime.Record(context.Background(), value, metric.WithAttributes(attrs...))
	case "deployment_time":
		ms.metrics.DeploymentTime.Record(context.Background(), value, metric.WithAttributes(attrs...))
	}
}

// UpdateGauge updates a gauge value with labels
func (ms *MetricsService) UpdateGauge(name string, value float64, labels *MetricLabels) {
	if !ms.config.MetricsEnabled {
		return
	}

	attrs := labels.ToAttributes()

	switch name {
	case "llm_cost":
		ms.metrics.LLMCostGauge.Record(context.Background(), value, metric.WithAttributes(attrs...))
	case "quality_score":
		ms.metrics.QualityScore.Record(context.Background(), value, metric.WithAttributes(attrs...))
	case "active_workflows":
		ms.metrics.ActiveWorkflows.Record(context.Background(), int64(value), metric.WithAttributes(attrs...))
	case "preview_environments":
		ms.metrics.PreviewEnvironments.Record(context.Background(), int64(value), metric.WithAttributes(attrs...))
	case "service_uptime":
		ms.metrics.ServiceUptime.Record(context.Background(), value, metric.WithAttributes(attrs...))
	}
}