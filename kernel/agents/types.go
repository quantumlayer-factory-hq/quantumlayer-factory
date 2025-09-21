package agents

import (
	"context"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
)

// Agent represents a specialized code generation agent
type Agent interface {
	// GetType returns the agent's specialization type
	GetType() AgentType

	// GetCapabilities returns what this agent can generate
	GetCapabilities() []string

	// CanHandle determines if this agent can handle the given specification
	CanHandle(spec *ir.IRSpec) bool

	// Generate creates code/artifacts from the specification
	Generate(ctx context.Context, req *GenerationRequest) (*GenerationResult, error)

	// Validate checks if the generated code meets requirements
	Validate(ctx context.Context, result *GenerationResult) (*ValidationResult, error)
}

// AgentType defines the type of agent
type AgentType string

const (
	AgentTypeBackend    AgentType = "backend"
	AgentTypeFrontend   AgentType = "frontend"
	AgentTypeDatabase   AgentType = "database"
	AgentTypeAPI        AgentType = "api"
	AgentTypeDevOps     AgentType = "devops"
	AgentTypeTest       AgentType = "test"
	AgentTypeDocumentation AgentType = "documentation"
)

// GenerationRequest contains the context and requirements for code generation
type GenerationRequest struct {
	Spec        *ir.IRSpec           `json:"spec"`
	Target      GenerationTarget     `json:"target"`
	Options     GenerationOptions    `json:"options"`
	Context     map[string]interface{} `json:"context,omitempty"`
	Overlays    []string             `json:"overlays,omitempty"`
}

// GenerationTarget specifies what to generate
type GenerationTarget struct {
	Type        string            `json:"type"`        // "controller", "model", "service", etc.
	Language    string            `json:"language"`    // "go", "python", "typescript"
	Framework   string            `json:"framework"`   // "gin", "fastapi", "react"
	OutputPath  string            `json:"output_path"` // where to generate files
	Templates   []string          `json:"templates"`   // template names to use
	Config      map[string]string `json:"config,omitempty"`
}

// GenerationOptions control how generation is performed
type GenerationOptions struct {
	OverwriteExisting bool              `json:"overwrite_existing"`
	CreateDirectories bool              `json:"create_directories"`
	FormatCode        bool              `json:"format_code"`
	ValidateOutput    bool              `json:"validate_output"`
	DryRun           bool              `json:"dry_run"`
	Verbose          bool              `json:"verbose"`
	Config           map[string]string `json:"config,omitempty"`
}

// GenerationResult contains the output of code generation
type GenerationResult struct {
	Success     bool                     `json:"success"`
	Files       []GeneratedFile          `json:"files"`
	Warnings    []string                 `json:"warnings,omitempty"`
	Errors      []string                 `json:"errors,omitempty"`
	Metadata    GenerationMetadata       `json:"metadata"`
	NextSteps   []string                 `json:"next_steps,omitempty"`
}

// GeneratedFile represents a file created by the agent
type GeneratedFile struct {
	Path        string            `json:"path"`
	Content     string            `json:"content"`
	Type        string            `json:"type"`        // "source", "config", "test", "doc"
	Language    string            `json:"language"`    // file language
	Size        int64             `json:"size"`        // file size in bytes
	Checksum    string            `json:"checksum"`    // content checksum
	Template    string            `json:"template,omitempty"` // template used
	Config      map[string]string `json:"config,omitempty"`
}

// GenerationMetadata contains information about the generation process
type GenerationMetadata struct {
	AgentType     AgentType         `json:"agent_type"`
	AgentVersion  string            `json:"agent_version"`
	GeneratedAt   time.Time         `json:"generated_at"`
	Duration      time.Duration     `json:"duration"`
	LinesOfCode   int               `json:"lines_of_code"`
	FilesCreated  int               `json:"files_created"`
	TemplatesUsed []string          `json:"templates_used"`
	Config        map[string]string `json:"config,omitempty"`
	LLMUsage      *LLMUsageMetadata `json:"llm_usage,omitempty"`
}

// LLMUsageMetadata contains information about LLM usage during generation
type LLMUsageMetadata struct {
	Provider         string  `json:"provider"`
	Model           string  `json:"model"`
	PromptTokens    int     `json:"prompt_tokens"`
	CompletionTokens int     `json:"completion_tokens"`
	TotalTokens     int     `json:"total_tokens"`
	Cost            float64 `json:"cost"`
}

// ValidationResult contains the result of validating generated code
type ValidationResult struct {
	Valid       bool              `json:"valid"`
	Errors      []ValidationError `json:"errors,omitempty"`
	Warnings    []string          `json:"warnings,omitempty"`
	Suggestions []string          `json:"suggestions,omitempty"`
	Metrics     ValidationMetrics `json:"metrics"`
}

// ValidationError represents a validation error
type ValidationError struct {
	File        string `json:"file"`
	Line        int    `json:"line,omitempty"`
	Column      int    `json:"column,omitempty"`
	Message     string `json:"message"`
	Type        string `json:"type"`     // "syntax", "semantic", "style"
	Severity    string `json:"severity"` // "error", "warning", "info"
	Rule        string `json:"rule,omitempty"`
}

// ValidationMetrics provides metrics about the validated code
type ValidationMetrics struct {
	LinesOfCode     int     `json:"lines_of_code"`
	CyclomaticComplexity int `json:"cyclomatic_complexity,omitempty"`
	TestCoverage    float64 `json:"test_coverage,omitempty"`
	CodeQuality     float64 `json:"code_quality"` // 0.0 to 1.0
	Performance     float64 `json:"performance,omitempty"` // relative score
}

// Factory creates and manages agents
type Factory interface {
	// CreateAgent creates an agent of the specified type
	CreateAgent(agentType AgentType) (Agent, error)

	// GetSupportedTypes returns all supported agent types
	GetSupportedTypes() []AgentType

	// GetAgent returns an existing agent instance
	GetAgent(agentType AgentType) (Agent, error)

	// GetBestAgent returns the best agent for handling the given spec
	GetBestAgent(spec *ir.IRSpec) (Agent, error)

	// RegisterAgent registers a new agent type
	RegisterAgent(agentType AgentType, creator AgentCreator) error
}

// AgentCreator is a function that creates agent instances
type AgentCreator func() Agent

// BaseAgent provides common functionality for all agents
type BaseAgent struct {
	agentType    AgentType
	version      string
	capabilities []string
	config       map[string]string
}

// NewBaseAgent creates a new base agent
func NewBaseAgent(agentType AgentType, version string, capabilities []string) *BaseAgent {
	return &BaseAgent{
		agentType:    agentType,
		version:      version,
		capabilities: capabilities,
		config:       make(map[string]string),
	}
}

// GetType implements Agent interface
func (a *BaseAgent) GetType() AgentType {
	return a.agentType
}

// GetCapabilities implements Agent interface
func (a *BaseAgent) GetCapabilities() []string {
	return a.capabilities
}

// SetConfig sets configuration for the agent
func (a *BaseAgent) SetConfig(key, value string) {
	a.config[key] = value
}

// GetConfig gets configuration value
func (a *BaseAgent) GetConfig(key string) (string, bool) {
	value, exists := a.config[key]
	return value, exists
}