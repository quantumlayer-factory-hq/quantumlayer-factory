package overlays

import (
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
)

// OverlayType defines the category of overlay
type OverlayType string

const (
	OverlayTypeDomain     OverlayType = "domain"     // Industry/domain-specific patterns
	OverlayTypeCompliance OverlayType = "compliance" // Regulatory compliance requirements
	OverlayTypeCapability OverlayType = "capability" // Technical capabilities/features
)

// Priority defines overlay precedence for conflict resolution
type Priority int

const (
	PriorityLow    Priority = 1
	PriorityMedium Priority = 5
	PriorityHigh   Priority = 10
)

// Overlay represents a composable modification to the code generation process
type Overlay interface {
	// GetMetadata returns overlay identification and configuration
	GetMetadata() OverlayMetadata

	// ApplyToIR modifies an IR specification based on overlay requirements
	ApplyToIR(spec *ir.IRSpec) (*ir.IRSpec, error)

	// GetPromptEnhancements returns prompt modifications for agents
	GetPromptEnhancements() []PromptEnhancement

	// GetValidationRules returns additional validation rules
	GetValidationRules() []ValidationRule

	// GetDependencies returns other overlays this one depends on
	GetDependencies() []string

	// Validate checks if the overlay configuration is valid
	Validate() error
}

// OverlayMetadata contains overlay identification and configuration
type OverlayMetadata struct {
	Name        string            `yaml:"name" json:"name"`
	Version     string            `yaml:"version" json:"version"`
	Type        OverlayType       `yaml:"type" json:"type"`
	Priority    Priority          `yaml:"priority" json:"priority"`
	Description string            `yaml:"description" json:"description"`
	Author      string            `yaml:"author,omitempty" json:"author,omitempty"`
	Tags        []string          `yaml:"tags,omitempty" json:"tags,omitempty"`
	CreatedAt   time.Time         `yaml:"created_at" json:"created_at"`
	UpdatedAt   time.Time         `yaml:"updated_at" json:"updated_at"`
	Config      map[string]string `yaml:"config,omitempty" json:"config,omitempty"`
}

// PromptEnhancement defines modifications to agent prompts
type PromptEnhancement struct {
	AgentType   string            `yaml:"agent_type" json:"agent_type"`     // backend, frontend, database, etc.
	Section     string            `yaml:"section" json:"section"`           // system, context, examples, etc.
	Content     string            `yaml:"content" json:"content"`           // content to inject
	Position    string            `yaml:"position" json:"position"`         // before, after, replace
	Conditions  map[string]string `yaml:"conditions,omitempty" json:"conditions,omitempty"` // when to apply
	Priority    Priority          `yaml:"priority" json:"priority"`
}

// ValidationRule defines additional validation requirements
type ValidationRule struct {
	Name        string            `yaml:"name" json:"name"`
	Type        string            `yaml:"type" json:"type"`         // security, compliance, performance, etc.
	Severity    string            `yaml:"severity" json:"severity"` // error, warning, info
	Pattern     string            `yaml:"pattern,omitempty" json:"pattern,omitempty"`
	Message     string            `yaml:"message" json:"message"`
	Conditions  map[string]string `yaml:"conditions,omitempty" json:"conditions,omitempty"`
	Remediation string            `yaml:"remediation,omitempty" json:"remediation,omitempty"`
}

// OverlaySpec defines the structure of overlay YAML files
type OverlaySpec struct {
	Metadata           OverlayMetadata     `yaml:"metadata" json:"metadata"`
	Dependencies       []string            `yaml:"dependencies,omitempty" json:"dependencies,omitempty"`
	IRModifications    []IRModification    `yaml:"ir_modifications,omitempty" json:"ir_modifications,omitempty"`
	PromptEnhancements []PromptEnhancement `yaml:"prompt_enhancements,omitempty" json:"prompt_enhancements,omitempty"`
	ValidationRules    []ValidationRule    `yaml:"validation_rules,omitempty" json:"validation_rules,omitempty"`
	Templates          map[string]string   `yaml:"templates,omitempty" json:"templates,omitempty"`
}

// IRModification defines changes to apply to IR specifications
type IRModification struct {
	Path      string      `yaml:"path" json:"path"`           // JSON path to modify (e.g., "app.features")
	Operation string      `yaml:"operation" json:"operation"` // add, remove, replace, merge
	Value     interface{} `yaml:"value" json:"value"`         // value to apply
	Condition string      `yaml:"condition,omitempty" json:"condition,omitempty"` // when to apply
}

// ResolverConfig configures overlay resolution behavior
type ResolverConfig struct {
	OverlayPaths       []string `yaml:"overlay_paths" json:"overlay_paths"`
	ConflictResolution string   `yaml:"conflict_resolution" json:"conflict_resolution"` // priority, merge, error
	EnableCaching      bool     `yaml:"enable_caching" json:"enable_caching"`
	CacheTTL           int      `yaml:"cache_ttl" json:"cache_ttl"` // seconds
}

// ResolverResult contains the result of overlay resolution
type ResolverResult struct {
	ResolvedOverlays []string                 `json:"resolved_overlays"`
	AppliedOrder     []string                 `json:"applied_order"`
	Conflicts        []ConflictInfo           `json:"conflicts,omitempty"`
	IRSpec           *ir.IRSpec               `json:"ir_spec"`
	PromptChanges    map[string][]PromptEnhancement `json:"prompt_changes"`
	ValidationRules  []ValidationRule         `json:"validation_rules"`
	Metadata         map[string]OverlayMetadata `json:"metadata"`
}

// ConflictInfo describes conflicts between overlays
type ConflictInfo struct {
	Path        string   `json:"path"`
	Overlays    []string `json:"overlays"`
	Resolution  string   `json:"resolution"`
	Description string   `json:"description"`
}

// Resolver interface for overlay resolution engines
type Resolver interface {
	// LoadOverlay loads an overlay from file or registry
	LoadOverlay(name string) (Overlay, error)

	// ResolveOverlays resolves a list of overlay names into a coherent set
	ResolveOverlays(overlayNames []string, baseIR *ir.IRSpec) (*ResolverResult, error)

	// ValidateOverlays checks overlay compatibility and dependencies
	ValidateOverlays(overlayNames []string) error

	// ListAvailable returns available overlays
	ListAvailable() ([]OverlayMetadata, error)

	// GetConflicts identifies potential conflicts between overlays
	GetConflicts(overlayNames []string) ([]ConflictInfo, error)
}

// Registry interface for overlay storage and discovery
type Registry interface {
	// Register adds an overlay to the registry
	Register(overlay Overlay) error

	// Get retrieves an overlay by name
	Get(name string) (Overlay, error)

	// List returns all available overlays
	List() ([]OverlayMetadata, error)

	// Search finds overlays matching criteria
	Search(criteria map[string]string) ([]OverlayMetadata, error)

	// Update updates an existing overlay
	Update(overlay Overlay) error

	// Remove removes an overlay from the registry
	Remove(name string) error
}