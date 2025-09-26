package ir

import (
	"time"
)

// IRSpec represents a complete intermediate representation of an application specification
type IRSpec struct {
	Version        string                 `json:"version"`
	ID             string                 `json:"id,omitempty"`
	Brief          string                 `json:"brief"`
	App            AppSpec                `json:"app"`
	NonFunctionals NonFunctionalSpec      `json:"non_functionals"`
	API            APISpec                `json:"api"`
	Data           DataSpec               `json:"data"`
	UI             UISpec                 `json:"ui,omitempty"`
	Ops            OpsSpec                `json:"ops"`
	Acceptance     []AcceptanceCriteria   `json:"acceptance"`
	Questions      []BlockingQuestion     `json:"questions,omitempty"`
	Metadata       SpecMetadata           `json:"metadata"`
}

// AppSpec defines the core application characteristics
type AppSpec struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Type        string            `json:"type"` // "web", "api", "cli", "mobile", "desktop"
	Domain      string            `json:"domain"` // "ecommerce", "fintech", "healthcare", etc.
	Stack       TechStack         `json:"stack"`
	Features    []Feature         `json:"features"`
	Scale       ScaleRequirements `json:"scale"`
}

// TechStack specifies the technology choices
type TechStack struct {
	Backend  BackendStack  `json:"backend"`
	Frontend FrontendStack `json:"frontend,omitempty"`
	Database DatabaseStack `json:"database"`
	Cache    CacheStack    `json:"cache,omitempty"`
}

type BackendStack struct {
	Language   string   `json:"language"`   // "python", "go", "nodejs", "java"
	Framework  string   `json:"framework"`  // "fastapi", "gin", "express", "spring"
	Libraries  []string `json:"libraries"`
	Runtime    string   `json:"runtime"`    // "docker", "serverless"
}

type FrontendStack struct {
	Language  string   `json:"language"`  // "typescript", "javascript"
	Framework string   `json:"framework"` // "react", "vue", "angular", "svelte"
	Libraries []string `json:"libraries"`
	Build     string   `json:"build"`     // "vite", "webpack", "next"
}

type DatabaseStack struct {
	Type     string            `json:"type"`     // "postgresql", "mysql", "mongodb", "sqlite"
	Version  string            `json:"version"`
	Features []string          `json:"features"` // "migrations", "seeds", "indexes"
	Config   map[string]string `json:"config,omitempty"`
}

type CacheStack struct {
	Type    string            `json:"type"`    // "redis", "memcached", "in-memory"
	Version string            `json:"version"`
	Config  map[string]string `json:"config,omitempty"`
}

// Feature represents a functional capability
type Feature struct {
	Name         string   `json:"name"`
	Description  string   `json:"description"`
	Type         string   `json:"type"`         // "crud", "auth", "payment", "notification"
	Priority     string   `json:"priority"`     // "high", "medium", "low"
	Dependencies []string `json:"dependencies"` // other feature names
	Entities     []string `json:"entities"`     // data entities involved
	Operations   []string `json:"operations"`   // actions supported
}

// ScaleRequirements defines performance and capacity expectations
type ScaleRequirements struct {
	Users       ScaleMetric `json:"users"`
	Requests    ScaleMetric `json:"requests"`
	Storage     ScaleMetric `json:"storage"`
	Latency     string      `json:"latency"`     // "100ms", "1s"
	Uptime      string      `json:"uptime"`      // "99.9%", "99.99%"
	Concurrency int         `json:"concurrency"` // simultaneous users
}

type ScaleMetric struct {
	Initial int `json:"initial"`
	Peak    int `json:"peak"`
	Growth  int `json:"growth"` // percentage per year
}

// NonFunctionalSpec defines quality attributes
type NonFunctionalSpec struct {
	Security    SecuritySpec    `json:"security"`
	Performance PerformanceSpec `json:"performance"`
	Compliance  ComplianceSpec  `json:"compliance"`
	Monitoring  MonitoringSpec  `json:"monitoring"`
}

type SecuritySpec struct {
	Authentication []string          `json:"authentication"` // "jwt", "oauth2", "basic"
	Authorization  []string          `json:"authorization"`  // "rbac", "acl", "policy"
	Encryption     []string          `json:"encryption"`     // "tls", "aes256", "bcrypt"
	Audit          bool              `json:"audit"`
	Compliance     []string          `json:"compliance"`     // "gdpr", "hipaa", "pci"
	Config         map[string]string `json:"config,omitempty"`
}

type PerformanceSpec struct {
	ResponseTime string            `json:"response_time"` // "100ms", "1s"
	Throughput   string            `json:"throughput"`    // "1000rps", "10000rpm"
	Memory       string            `json:"memory"`        // "512MB", "2GB"
	CPU          string            `json:"cpu"`           // "2cores", "4cores"
	Config       map[string]string `json:"config,omitempty"`
}

type ComplianceSpec struct {
	Standards   []string          `json:"standards"`   // "pci-dss", "hipaa", "gdpr"
	DataRetention string          `json:"data_retention"` // "7years", "30days"
	AuditLog    bool              `json:"audit_log"`
	Config      map[string]string `json:"config,omitempty"`
}

type MonitoringSpec struct {
	Metrics []string          `json:"metrics"` // "cpu", "memory", "requests", "errors"
	Logging string            `json:"logging"` // "structured", "json", "plain"
	Tracing bool              `json:"tracing"`
	Alerts  []AlertRule       `json:"alerts"`
	Config  map[string]string `json:"config,omitempty"`
}

type AlertRule struct {
	Name      string `json:"name"`
	Condition string `json:"condition"` // "error_rate > 5%"
	Action    string `json:"action"`    // "email", "slack", "pagerduty"
}

// APISpec defines the API structure
type APISpec struct {
	Type      string      `json:"type"`      // "rest", "graphql", "grpc"
	Version   string      `json:"version"`   // "v1", "v2"
	BaseURL   string      `json:"base_url"`  // "/api/v1"
	Auth      AuthSpec    `json:"auth"`
	Endpoints []Endpoint  `json:"endpoints"`
	Schemas   []Schema    `json:"schemas"`
	Config    APIConfig   `json:"config"`
}

type AuthSpec struct {
	Type     string   `json:"type"`     // "bearer", "api-key", "oauth2"
	Scopes   []string `json:"scopes"`
	Required bool     `json:"required"`
	Issuer   string   `json:"issuer,omitempty"`   // "external" for external auth providers
	JWKSURL  string   `json:"jwks_url,omitempty"` // URL for JWKS endpoint
}

type Endpoint struct {
	Path        string            `json:"path"`        // "/users/{id}"
	Method      string            `json:"method"`      // "GET", "POST", "PUT", "DELETE"
	Summary     string            `json:"summary"`
	Description string            `json:"description"`
	Parameters  []Parameter       `json:"parameters"`
	RequestBody *RequestBody      `json:"request_body,omitempty"`
	Responses   map[string]Response `json:"responses"`
	Auth        bool              `json:"auth"`
	RateLimit   *RateLimit        `json:"rate_limit,omitempty"`
}

type Parameter struct {
	Name        string `json:"name"`
	In          string `json:"in"`          // "path", "query", "header"
	Type        string `json:"type"`        // "string", "integer", "boolean"
	Required    bool   `json:"required"`
	Description string `json:"description"`
	Example     string `json:"example,omitempty"`
}

type RequestBody struct {
	Required    bool   `json:"required"`
	ContentType string `json:"content_type"` // "application/json"
	Schema      string `json:"schema"`       // reference to schema
	Example     string `json:"example,omitempty"`
}

type Response struct {
	Description string `json:"description"`
	Schema      string `json:"schema,omitempty"`
	Example     string `json:"example,omitempty"`
}

type RateLimit struct {
	Requests int    `json:"requests"` // requests per window
	Window   string `json:"window"`   // "1m", "1h", "1d"
}

type APIConfig struct {
	CORS        bool              `json:"cors"`
	Compression bool              `json:"compression"`
	Versioning  string            `json:"versioning"` // "header", "path", "query"
	Pagination  string            `json:"pagination"` // "offset", "cursor", "page"
	Config      map[string]string `json:"config,omitempty"`
}

// DataSpec defines the data model
type DataSpec struct {
	Entities      []Entity           `json:"entities"`
	Relationships []Relationship     `json:"relationships"`
	Migrations    []Migration        `json:"migrations"`
	Seeds         []Seed             `json:"seeds"`
	Indexes       []Index            `json:"indexes"`
	Config        DataConfig         `json:"config"`
}

type Entity struct {
	Name        string            `json:"name"`
	Description string            `json:"description"`
	Fields      []Field           `json:"fields"`
	Constraints []Constraint      `json:"constraints"`
	Config      map[string]string `json:"config,omitempty"`
}

type Field struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`        // "string", "integer", "boolean", "date"
	Required    bool              `json:"required"`
	Unique      bool              `json:"unique"`
	Default     string            `json:"default,omitempty"`
	Validation  []ValidationRule  `json:"validation"`
	Description string            `json:"description"`
	Config      map[string]string `json:"config,omitempty"`
}

type ValidationRule struct {
	Type    string `json:"type"`    // "min", "max", "pattern", "enum"
	Value   string `json:"value"`
	Message string `json:"message"`
}

type Constraint struct {
	Type   string   `json:"type"`   // "primary_key", "foreign_key", "unique", "check"
	Fields []string `json:"fields"`
	Target string   `json:"target,omitempty"` // for foreign keys
}

type Relationship struct {
	From       string `json:"from"`        // entity name
	To         string `json:"to"`          // entity name
	Type       string `json:"type"`        // "one_to_one", "one_to_many", "many_to_many"
	ForeignKey string `json:"foreign_key,omitempty"`
	Through    string `json:"through,omitempty"` // for many_to_many
}

type Migration struct {
	Version     string   `json:"version"`
	Description string   `json:"description"`
	Operations  []string `json:"operations"` // "create_table", "add_column", etc.
}

type Seed struct {
	Entity string                 `json:"entity"`
	Data   []map[string]interface{} `json:"data"`
}

type Index struct {
	Name   string   `json:"name"`
	Entity string   `json:"entity"`
	Fields []string `json:"fields"`
	Type   string   `json:"type"` // "btree", "hash", "gin"
	Unique bool     `json:"unique"`
}

type DataConfig struct {
	Migrations  bool              `json:"migrations"`
	Seeds       bool              `json:"seeds"`
	Soft_Delete bool              `json:"soft_delete"`
	Timestamps  bool              `json:"timestamps"` // created_at, updated_at
	Config      map[string]string `json:"config,omitempty"`
}

// UISpec defines the user interface (optional)
type UISpec struct {
	Type       string     `json:"type"`       // "spa", "mpa", "mobile", "desktop"
	Pages      []Page     `json:"pages"`
	Components []Component `json:"components"`
	Theme      Theme      `json:"theme"`
	Config     UIConfig   `json:"config"`
}

type Page struct {
	Name        string   `json:"name"`
	Path        string   `json:"path"`
	Title       string   `json:"title"`
	Description string   `json:"description"`
	Components  []string `json:"components"` // component names
	Auth        bool     `json:"auth"`
}

type Component struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"`        // "form", "table", "chart", "modal"
	Props       map[string]string `json:"props"`
	Description string            `json:"description"`
}

type Theme struct {
	Primary   string `json:"primary"`   // color
	Secondary string `json:"secondary"` // color
	Style     string `json:"style"`     // "material", "bootstrap", "tailwind"
}

type UIConfig struct {
	Responsive bool              `json:"responsive"`
	PWA        bool              `json:"pwa"`
	I18n       bool              `json:"i18n"`
	Config     map[string]string `json:"config,omitempty"`
}

// OpsSpec defines deployment and operations
type OpsSpec struct {
	Environment []Environment    `json:"environment"`
	CI_CD       CICDSpec         `json:"ci_cd"`
	Monitoring  OpsMonitoring    `json:"monitoring"`
	Backup      BackupSpec       `json:"backup"`
	Scaling     ScalingSpec      `json:"scaling"`
	Config      OpsConfig        `json:"config"`
}

type Environment struct {
	Name     string            `json:"name"`     // "dev", "staging", "prod"
	URL      string            `json:"url"`
	Provider string            `json:"provider"` // "aws", "gcp", "azure", "k8s"
	Region   string            `json:"region"`
	Config   map[string]string `json:"config"`
}

type CICDSpec struct {
	Provider string   `json:"provider"` // "github", "gitlab", "jenkins"
	Triggers []string `json:"triggers"` // "push", "pr", "tag"
	Stages   []string `json:"stages"`   // "test", "build", "deploy"
	Config   map[string]string `json:"config,omitempty"`
}

type OpsMonitoring struct {
	Provider string   `json:"provider"` // "datadog", "newrelic", "prometheus"
	Dashboards []string `json:"dashboards"`
	Alerts   []string `json:"alerts"`
	Config   map[string]string `json:"config,omitempty"`
}

type BackupSpec struct {
	Frequency string `json:"frequency"` // "daily", "weekly"
	Retention string `json:"retention"` // "30d", "1y"
	Provider  string `json:"provider"`  // "s3", "gcs"
	Config    map[string]string `json:"config,omitempty"`
}

type ScalingSpec struct {
	Type     string `json:"type"`     // "horizontal", "vertical", "auto"
	Min      int    `json:"min"`      // minimum instances
	Max      int    `json:"max"`      // maximum instances
	Triggers []string `json:"triggers"` // "cpu", "memory", "requests"
	Config   map[string]string `json:"config,omitempty"`
}

type OpsConfig struct {
	SSL         bool              `json:"ssl"`
	CDN         bool              `json:"cdn"`
	LoadBalancer bool             `json:"load_balancer"`
	Config      map[string]string `json:"config,omitempty"`
}

// AcceptanceCriteria defines success criteria
type AcceptanceCriteria struct {
	ID          string `json:"id"`
	Description string `json:"description"`
	Type        string `json:"type"`        // "functional", "performance", "security"
	Priority    string `json:"priority"`    // "must", "should", "could"
	Testable    bool   `json:"testable"`
	Automated   bool   `json:"automated"`
}

// BlockingQuestion represents ambiguities that need clarification
type BlockingQuestion struct {
	ID       string   `json:"id"`
	Question string   `json:"question"`
	Context  string   `json:"context"`
	Type     string   `json:"type"`     // "technical", "business", "security"
	Options  []string `json:"options"`  // possible answers
	Required bool     `json:"required"`
}

// SpecMetadata contains metadata about the specification
type SpecMetadata struct {
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
	Version     string            `json:"version"`
	CreatedBy   string            `json:"created_by,omitempty"`
	Confidence  float64           `json:"confidence"`  // 0.0 to 1.0
	Completeness float64          `json:"completeness"` // 0.0 to 1.0
	Tags        []string          `json:"tags"`
	Source      string            `json:"source"`      // "manual", "ai", "template"
	Config      map[string]string `json:"config,omitempty"`
}

// Schema represents a data schema definition
type Schema struct {
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`        // "object", "array"
	Properties  map[string]Property    `json:"properties"`
	Required    []string               `json:"required"`
	Description string                 `json:"description"`
	Example     map[string]interface{} `json:"example,omitempty"`
}

type Property struct {
	Type        string      `json:"type"`        // "string", "integer", "boolean", "array", "object"
	Format      string      `json:"format,omitempty"` // "email", "uri", "date-time"
	Description string      `json:"description"`
	Example     interface{} `json:"example,omitempty"`
	Items       *Property   `json:"items,omitempty"`       // for arrays
	Properties  map[string]Property `json:"properties,omitempty"` // for objects
	Required    []string    `json:"required,omitempty"`    // for objects
}