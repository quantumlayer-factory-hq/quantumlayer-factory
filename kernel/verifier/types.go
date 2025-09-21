package verifier

import (
	"context"
	"time"
)

// Gate represents a verification gate in the pipeline
type Gate interface {
	// GetType returns the gate type identifier
	GetType() GateType

	// GetName returns a human-readable name for this gate
	GetName() string

	// CanVerify determines if this gate can verify the given artifacts
	CanVerify(artifacts []Artifact) bool

	// Verify performs the verification and returns results
	Verify(ctx context.Context, req *VerificationRequest) (*VerificationResult, error)

	// GetConfiguration returns the current gate configuration
	GetConfiguration() GateConfig
}

// GateType defines the type of verification gate
type GateType string

const (
	GateTypeStatic     GateType = "static"
	GateTypeUnit       GateType = "unit"
	GateTypeIntegration GateType = "integration"
	GateTypeContract    GateType = "contract"
	GateTypeSecurity    GateType = "security"
	GateTypePerformance GateType = "performance"
	GateTypeMutation    GateType = "mutation"
	GateTypeRuntime     GateType = "runtime"
)

// VerificationRequest contains the context for verification
type VerificationRequest struct {
	RequestID   string                 `json:"request_id"`
	Artifacts   []Artifact             `json:"artifacts"`
	Config      GateConfig             `json:"config"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Timeout     time.Duration          `json:"timeout"`
	Environment string                 `json:"environment"` // "development", "staging", "production"
}

// VerificationResult contains the outcome of verification
type VerificationResult struct {
	Success     bool                   `json:"success"`
	GateType    GateType               `json:"gate_type"`
	GateName    string                 `json:"gate_name"`
	Artifacts   []Artifact             `json:"artifacts"`
	Issues      []Issue                `json:"issues,omitempty"`
	Warnings    []string               `json:"warnings,omitempty"`
	Metrics     VerificationMetrics    `json:"metrics"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
	Duration    time.Duration          `json:"duration"`
	Timestamp   time.Time              `json:"timestamp"`
	RepairHints []RepairHint           `json:"repair_hints,omitempty"`
}

// Artifact represents a file or resource to be verified
type Artifact struct {
	Path        string            `json:"path"`
	Type        ArtifactType      `json:"type"`
	Language    string            `json:"language,omitempty"`
	Framework   string            `json:"framework,omitempty"`
	Content     string            `json:"content,omitempty"`
	Size        int64             `json:"size"`
	Hash        string            `json:"hash"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// ArtifactType defines the type of artifact
type ArtifactType string

const (
	ArtifactTypeSource       ArtifactType = "source"
	ArtifactTypeTest         ArtifactType = "test"
	ArtifactTypeConfig       ArtifactType = "config"
	ArtifactTypeDocumentation ArtifactType = "documentation"
	ArtifactTypeSchema       ArtifactType = "schema"
	ArtifactTypeDeployment   ArtifactType = "deployment"
	ArtifactTypeBinary       ArtifactType = "binary"
)

// Issue represents a verification issue found by a gate
type Issue struct {
	ID          string      `json:"id"`
	Type        IssueType   `json:"type"`
	Severity    Severity    `json:"severity"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	File        string      `json:"file,omitempty"`
	Line        int         `json:"line,omitempty"`
	Column      int         `json:"column,omitempty"`
	Rule        string      `json:"rule,omitempty"`
	Category    string      `json:"category,omitempty"`
	Fix         *Fix        `json:"fix,omitempty"`
	References  []Reference `json:"references,omitempty"`
}

// IssueType categorizes the type of issue
type IssueType string

const (
	IssueTypeSyntax      IssueType = "syntax"
	IssueTypeSemantic    IssueType = "semantic"
	IssueTypeStyle       IssueType = "style"
	IssueTypeSecurity    IssueType = "security"
	IssueTypePerformance IssueType = "performance"
	IssueTypeMaintenance IssueType = "maintenance"
	IssueTypeCompliance  IssueType = "compliance"
)

// Severity indicates the severity level of an issue
type Severity string

const (
	SeverityInfo     Severity = "info"
	SeverityWarning  Severity = "warning"
	SeverityError    Severity = "error"
	SeverityCritical Severity = "critical"
	SeverityBlocking Severity = "blocking"
)

// Fix represents a potential fix for an issue
type Fix struct {
	Description string `json:"description"`
	Diff        string `json:"diff,omitempty"`
	Automated   bool   `json:"automated"`
	Confidence  float64 `json:"confidence"` // 0.0 to 1.0
}

// Reference provides additional information about an issue
type Reference struct {
	Type        string `json:"type"`        // "url", "cwe", "cve", "documentation"
	Value       string `json:"value"`
	Description string `json:"description,omitempty"`
}

// VerificationMetrics provides metrics about the verification process
type VerificationMetrics struct {
	FilesScanned    int           `json:"files_scanned"`
	LinesScanned    int           `json:"lines_scanned"`
	IssuesFound     int           `json:"issues_found"`
	IssuesFixed     int           `json:"issues_fixed"`
	Duration        time.Duration `json:"duration"`
	ResourceUsage   ResourceUsage `json:"resource_usage"`
}

// ResourceUsage tracks resource consumption during verification
type ResourceUsage struct {
	CPUTime    time.Duration `json:"cpu_time"`
	Memory     int64         `json:"memory_bytes"`
	DiskIO     int64         `json:"disk_io_bytes"`
	NetworkIO  int64         `json:"network_io_bytes"`
}

// RepairHint provides guidance for fixing issues
type RepairHint struct {
	IssueID     string  `json:"issue_id"`
	Suggestion  string  `json:"suggestion"`
	Confidence  float64 `json:"confidence"`
	Automated   bool    `json:"automated"`
	References  []string `json:"references,omitempty"`
}

// GateConfig contains configuration for a verification gate
type GateConfig struct {
	Enabled      bool                   `json:"enabled"`
	Timeout      time.Duration          `json:"timeout"`
	Parallel     bool                   `json:"parallel"`
	Rules        map[string]interface{} `json:"rules,omitempty"`
	Thresholds   map[string]float64     `json:"thresholds,omitempty"`
	Environment  map[string]string      `json:"environment,omitempty"`
	CustomConfig map[string]interface{} `json:"custom_config,omitempty"`
}

// Pipeline represents a sequence of verification gates
type Pipeline interface {
	// AddGate adds a gate to the pipeline
	AddGate(gate Gate) error

	// RemoveGate removes a gate from the pipeline
	RemoveGate(gateType GateType) error

	// Execute runs all gates in the pipeline
	Execute(ctx context.Context, artifacts []Artifact) (*PipelineResult, error)

	// GetGates returns all gates in the pipeline
	GetGates() []Gate
}

// PipelineResult contains the results of running a verification pipeline
type PipelineResult struct {
	Success      bool                  `json:"success"`
	RequestID    string                `json:"request_id"`
	GateResults  []VerificationResult  `json:"gate_results"`
	TotalIssues  int                   `json:"total_issues"`
	TotalWarnings int                  `json:"total_warnings"`
	Duration     time.Duration         `json:"duration"`
	Metrics      PipelineMetrics       `json:"metrics"`
	Artifacts    []Artifact            `json:"artifacts"`
	Summary      PipelineSummary       `json:"summary"`
}

// PipelineMetrics provides overall pipeline metrics
type PipelineMetrics struct {
	GatesExecuted   int           `json:"gates_executed"`
	GatesPassed     int           `json:"gates_passed"`
	GatesFailed     int           `json:"gates_failed"`
	TotalDuration   time.Duration `json:"total_duration"`
	ParallelGates   int           `json:"parallel_gates"`
	SequentialGates int           `json:"sequential_gates"`
}

// PipelineSummary provides a high-level summary of the pipeline execution
type PipelineSummary struct {
	Quality      QualityScore `json:"quality"`
	Security     QualityScore `json:"security"`
	Maintainability QualityScore `json:"maintainability"`
	Performance  QualityScore `json:"performance"`
	Compliance   QualityScore `json:"compliance"`
	Overall      QualityScore `json:"overall"`
}

// QualityScore represents a quality metric score
type QualityScore struct {
	Score       float64 `json:"score"`        // 0.0 to 1.0
	Grade       string  `json:"grade"`        // A, B, C, D, F
	Issues      int     `json:"issues"`
	Trend       string  `json:"trend"`        // "improving", "stable", "declining"
	Benchmark   float64 `json:"benchmark,omitempty"` // Industry benchmark
}

// Runner represents a specific verification tool runner
type Runner interface {
	// GetName returns the runner name
	GetName() string

	// GetVersion returns the runner version
	GetVersion() string

	// CanRun determines if this runner can process the given artifacts
	CanRun(artifacts []Artifact) bool

	// Run executes the verification tool
	Run(ctx context.Context, artifacts []Artifact, config map[string]interface{}) (*RunnerResult, error)

	// GetDefaultConfig returns the default configuration for this runner
	GetDefaultConfig() map[string]interface{}
}

// RunnerResult contains the output from a verification tool runner
type RunnerResult struct {
	Success    bool                   `json:"success"`
	ExitCode   int                    `json:"exit_code"`
	Stdout     string                 `json:"stdout,omitempty"`
	Stderr     string                 `json:"stderr,omitempty"`
	Issues     []Issue                `json:"issues"`
	Metrics    map[string]interface{} `json:"metrics,omitempty"`
	Duration   time.Duration          `json:"duration"`
	Artifacts  []Artifact             `json:"artifacts,omitempty"`
}

// Factory creates and manages verification gates
type Factory interface {
	// CreateGate creates a gate of the specified type
	CreateGate(gateType GateType, config GateConfig) (Gate, error)

	// CreatePipeline creates a new verification pipeline
	CreatePipeline() Pipeline

	// GetSupportedGates returns all supported gate types
	GetSupportedGates() []GateType

	// RegisterRunner registers a new runner for a gate type
	RegisterRunner(gateType GateType, runner Runner) error
}

// Repository stores verification results and artifacts
type Repository interface {
	// StoreResult stores a verification result
	StoreResult(ctx context.Context, result *VerificationResult) error

	// GetResult retrieves a verification result by ID
	GetResult(ctx context.Context, requestID string) (*VerificationResult, error)

	// StoreArtifact stores an artifact
	StoreArtifact(ctx context.Context, artifact *Artifact) error

	// GetArtifact retrieves an artifact by path and hash
	GetArtifact(ctx context.Context, path, hash string) (*Artifact, error)

	// SearchSimilarIssues finds similar issues for repair hints
	SearchSimilarIssues(ctx context.Context, issue Issue) ([]Issue, error)
}