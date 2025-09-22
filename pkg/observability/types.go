package observability

import (
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
	"go.opentelemetry.io/otel/trace"
)

// ObservabilityConfig holds configuration for observability components
type ObservabilityConfig struct {
	ServiceName    string
	ServiceVersion string
	Environment    string

	// Tracing configuration
	TracingEnabled bool
	JaegerEndpoint string
	TraceRatio     float64

	// Metrics configuration
	MetricsEnabled     bool
	PrometheusEndpoint string
	MetricsPort        int

	// Logging configuration
	LogLevel  string
	LogFormat string // "json" or "text"
}

// DefaultConfig returns default observability configuration
func DefaultConfig() *ObservabilityConfig {
	return &ObservabilityConfig{
		ServiceName:        "quantumlayer-factory",
		ServiceVersion:     "1.0.0",
		Environment:        "development",
		TracingEnabled:     true,
		JaegerEndpoint:     "http://localhost:14268/api/traces",
		TraceRatio:         1.0,
		MetricsEnabled:     true,
		PrometheusEndpoint: "http://localhost:9090",
		MetricsPort:        8090,
		LogLevel:           "info",
		LogFormat:          "json",
	}
}

// Metrics holds all application metrics
type Metrics struct {
	// Generation metrics
	GenerationCounter    metric.Int64Counter
	GenerationDuration   metric.Float64Histogram
	IRCompilationTime    metric.Float64Histogram
	AgentExecutionTime   metric.Float64Histogram

	// LLM metrics
	LLMRequestCounter    metric.Int64Counter
	LLMTokenUsage        metric.Int64Counter
	LLMCostGauge         metric.Float64Gauge
	LLMLatency          metric.Float64Histogram
	LLMCacheHitCounter   metric.Int64Counter

	// Verification metrics
	VerificationCounter  metric.Int64Counter
	GateExecutionTime    metric.Float64Histogram
	RepairAttempts       metric.Int64Counter
	QualityScore         metric.Float64Gauge

	// Package metrics
	PackageCounter       metric.Int64Counter
	PackageSize          metric.Int64Histogram
	PackageBuildTime     metric.Float64Histogram
	SBOMGenerationTime   metric.Float64Histogram
	VulnScanTime         metric.Float64Histogram

	// Deploy metrics
	DeploymentCounter    metric.Int64Counter
	DeploymentTime       metric.Float64Histogram
	PreviewEnvironments  metric.Int64Gauge
	HealthCheckCounter   metric.Int64Counter

	// System metrics
	ActiveWorkflows      metric.Int64Gauge
	ErrorCounter         metric.Int64Counter
	ServiceUptime        metric.Float64Gauge
}

// TracingInfo holds tracing context and utilities
type TracingInfo struct {
	Tracer       trace.Tracer
	SpanRecorder *SpanRecorder
}

// SpanRecorder records span information for analysis
type SpanRecorder struct {
	spans []SpanInfo
}

// SpanInfo contains span metadata
type SpanInfo struct {
	Name        string
	StartTime   time.Time
	EndTime     time.Time
	Duration    time.Duration
	Attributes  []attribute.KeyValue
	Status      trace.Status
	Events      []trace.Event
}

// LLMMetrics holds LLM-specific monitoring data
type LLMMetrics struct {
	Provider    string
	Model       string
	TokensUsed  int64
	Cost        float64
	Latency     time.Duration
	CacheHit    bool
	Success     bool
	ErrorType   string
	RequestID   string
	Timestamp   time.Time
}

// AgentMetrics holds agent execution metrics
type AgentMetrics struct {
	AgentType   string
	Language    string
	Framework   string
	Duration    time.Duration
	LinesOfCode int
	FilesCount  int
	Success     bool
	ErrorType   string
	LLMUsed     bool
	Timestamp   time.Time
}

// VerificationMetrics holds verification gate metrics
type VerificationMetrics struct {
	GateName      string
	GateType      string
	Duration      time.Duration
	IssuesFound   int
	Success       bool
	RepairAttempt bool
	RepairSuccess bool
	QualityScore  float64
	Timestamp     time.Time
}

// PackageMetrics holds packaging operation metrics
type PackageMetrics struct {
	PackageName       string
	Language          string
	Framework         string
	Size              int64
	CompressionRatio  float64
	BuildTime         time.Duration
	SBOMTime          time.Duration
	VulnScanTime      time.Duration
	VulnerabilitiesFound int
	DeliveryChannels  []string
	Success           bool
	ErrorType         string
	Timestamp         time.Time
}

// DeploymentMetrics holds deployment operation metrics
type DeploymentMetrics struct {
	Namespace     string
	AppName       string
	Language      string
	Replicas      int
	Duration      time.Duration
	HealthChecks  int
	Success       bool
	ErrorType     string
	TTL           time.Duration
	Timestamp     time.Time
}

// HealthStatus represents service health information
type HealthStatus struct {
	Service     string    `json:"service"`
	Status      string    `json:"status"` // "healthy", "unhealthy", "degraded"
	LastCheck   time.Time `json:"last_check"`
	Duration    time.Duration `json:"duration"`
	Message     string    `json:"message,omitempty"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

// SystemHealth represents overall system health
type SystemHealth struct {
	Status     string                  `json:"status"`
	Timestamp  time.Time              `json:"timestamp"`
	Services   map[string]HealthStatus `json:"services"`
	Summary    HealthSummary          `json:"summary"`
}

// HealthSummary provides aggregated health information
type HealthSummary struct {
	Healthy   int `json:"healthy"`
	Unhealthy int `json:"unhealthy"`
	Degraded  int `json:"degraded"`
	Total     int `json:"total"`
}

// Alert represents a monitoring alert
type Alert struct {
	Name        string            `json:"name"`
	Severity    string            `json:"severity"` // "critical", "warning", "info"
	Message     string            `json:"message"`
	Labels      map[string]string `json:"labels"`
	Timestamp   time.Time         `json:"timestamp"`
	Resolved    bool              `json:"resolved"`
	ResolvedAt  *time.Time        `json:"resolved_at,omitempty"`
}

// MetricLabels holds common metric labels
type MetricLabels struct {
	Service   string
	Component string
	Language  string
	Framework string
	Provider  string
	Model     string
	Overlay   string
	Result    string // "success", "error", "timeout"
}

// ToAttributes converts MetricLabels to OpenTelemetry attributes
func (ml *MetricLabels) ToAttributes() []attribute.KeyValue {
	attrs := []attribute.KeyValue{}

	if ml.Service != "" {
		attrs = append(attrs, attribute.String("service", ml.Service))
	}
	if ml.Component != "" {
		attrs = append(attrs, attribute.String("component", ml.Component))
	}
	if ml.Language != "" {
		attrs = append(attrs, attribute.String("language", ml.Language))
	}
	if ml.Framework != "" {
		attrs = append(attrs, attribute.String("framework", ml.Framework))
	}
	if ml.Provider != "" {
		attrs = append(attrs, attribute.String("provider", ml.Provider))
	}
	if ml.Model != "" {
		attrs = append(attrs, attribute.String("model", ml.Model))
	}
	if ml.Overlay != "" {
		attrs = append(attrs, attribute.String("overlay", ml.Overlay))
	}
	if ml.Result != "" {
		attrs = append(attrs, attribute.String("result", ml.Result))
	}

	return attrs
}

// Observer interface for observability operations
type Observer interface {
	// Tracing
	StartSpan(name string, attrs ...attribute.KeyValue) trace.Span
	RecordSpan(span trace.Span, duration time.Duration, success bool)

	// Metrics
	RecordGeneration(duration time.Duration, success bool, labels *MetricLabels)
	RecordLLMUsage(metrics *LLMMetrics)
	RecordVerification(metrics *VerificationMetrics)
	RecordPackaging(metrics *PackageMetrics)
	RecordDeployment(metrics *DeploymentMetrics)

	// Health
	CheckHealth(service string) HealthStatus
	UpdateHealth(service string, status HealthStatus)
	GetSystemHealth() SystemHealth
}