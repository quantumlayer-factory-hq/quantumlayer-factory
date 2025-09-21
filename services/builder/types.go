package builder

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
)

// BuildRequest represents a container build request
type BuildRequest struct {
	ProjectPath    string            `json:"project_path"`
	Language       string            `json:"language"`
	Framework      string            `json:"framework"`
	ImageName      string            `json:"image_name"`
	ImageTag       string            `json:"image_tag"`
	Registry       string            `json:"registry,omitempty"`
	Port           int               `json:"port,omitempty"`
	BuildArgs      map[string]string `json:"build_args,omitempty"`
	SecurityScan   bool              `json:"security_scan"`
	Platform       []string          `json:"platform,omitempty"`
	PushToRegistry bool              `json:"push_to_registry"`
	Spec           *ir.IRSpec        `json:"spec,omitempty"`
}

// BuildResult represents the result of a container build
type BuildResult struct {
	Success       bool                   `json:"success"`
	ImageID       string                 `json:"image_id"`
	ImageName     string                 `json:"image_name"`
	ImageTag      string                 `json:"image_tag"`
	ImageSize     int64                  `json:"image_size"`
	BuildTime     time.Duration          `json:"build_time"`
	SecurityScan  *SecurityScanResult    `json:"security_scan,omitempty"`
	RegistryURL   string                 `json:"registry_url,omitempty"`
	Dockerfile    string                 `json:"dockerfile"`
	BuildLogs     []string               `json:"build_logs"`
	Warnings      []string               `json:"warnings"`
	Errors        []string               `json:"errors"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// SecurityScanResult represents vulnerability scan results
type SecurityScanResult struct {
	Scanner         string                 `json:"scanner"`
	ScanTime        time.Time              `json:"scan_time"`
	ImageID         string                 `json:"image_id"`
	ImageName       string                 `json:"image_name"`
	ImageTag        string                 `json:"image_tag"`
	TotalVulns      int                    `json:"total_vulnerabilities"`
	Critical        int                    `json:"critical"`
	High            int                    `json:"high"`
	Medium          int                    `json:"medium"`
	Low             int                    `json:"low"`
	Negligible      int                    `json:"negligible"`
	Unknown         int                    `json:"unknown"`
	Vulnerabilities []Vulnerability        `json:"vulnerabilities"`
	Passed          bool                   `json:"passed"`
	Threshold       VulnerabilityThreshold `json:"threshold"`
}

// Vulnerability represents a single vulnerability
type Vulnerability struct {
	ID           string   `json:"id"`
	Severity     string   `json:"severity"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Package      string   `json:"package"`
	Version      string   `json:"version"`
	FixedVersion string   `json:"fixed_version,omitempty"`
	CVSS         float64  `json:"cvss,omitempty"`
	References   []string `json:"references,omitempty"`
}

// VulnerabilityThreshold defines security scan thresholds
type VulnerabilityThreshold struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

// DockerfileTemplate represents a Dockerfile generation template
type DockerfileTemplate struct {
	Language     string            `json:"language"`
	Framework    string            `json:"framework"`
	BaseImage    string            `json:"base_image"`
	WorkDir      string            `json:"work_dir"`
	Dependencies []string          `json:"dependencies"`
	BuildSteps   []BuildStep       `json:"build_steps"`
	RunSteps     []BuildStep       `json:"run_steps"`
	ExposedPorts []int             `json:"exposed_ports"`
	HealthCheck  *HealthCheck      `json:"health_check,omitempty"`
	Environment  map[string]string `json:"environment"`
	Labels       map[string]string `json:"labels"`
	User         string            `json:"user"`
}

// BuildStep represents a single build step in Dockerfile
type BuildStep struct {
	Type        string            `json:"type"` // RUN, COPY, ADD, etc.
	Command     string            `json:"command"`
	Args        []string          `json:"args,omitempty"`
	Condition   string            `json:"condition,omitempty"`
	Environment map[string]string `json:"environment,omitempty"`
}

// HealthCheck represents container health check configuration
type HealthCheck struct {
	Command     []string      `json:"command"`
	Interval    time.Duration `json:"interval"`
	Timeout     time.Duration `json:"timeout"`
	StartPeriod time.Duration `json:"start_period"`
	Retries     int           `json:"retries"`
}

// Builder interface defines container building capabilities
type Builder interface {
	// Build builds a container image from the project
	Build(ctx context.Context, req *BuildRequest) (*BuildResult, error)

	// GenerateDockerfile generates a Dockerfile for the project
	GenerateDockerfile(req *BuildRequest) (string, error)

	// ScanImage scans the built image for vulnerabilities
	ScanImage(ctx context.Context, imageID string) (*SecurityScanResult, error)

	// PushImage pushes the image to a registry
	PushImage(ctx context.Context, imageID, registry, name, tag string) error

	// GetSupportedLanguages returns supported languages and frameworks
	GetSupportedLanguages() map[string][]string

	// ValidateProject validates that the project can be built
	ValidateProject(projectPath string) error
}

// Registry interface defines container registry operations
type Registry interface {
	// Push pushes an image to the registry
	Push(ctx context.Context, imageID, name, tag string) (string, error)

	// Pull pulls an image from the registry
	Pull(ctx context.Context, name, tag string) (string, error)

	// Delete removes an image from the registry
	Delete(ctx context.Context, name, tag string) error

	// List lists images in the registry
	List(ctx context.Context, namespace string) ([]string, error)

	// GetManifest gets image manifest
	GetManifest(ctx context.Context, name, tag string) (map[string]interface{}, error)
}

// Scanner interface defines security scanning capabilities
type Scanner interface {
	// Scan scans an image for vulnerabilities
	Scan(ctx context.Context, imageID string, threshold VulnerabilityThreshold) (*SecurityScanResult, error)

	// ScanFile scans a specific file or package for vulnerabilities
	ScanFile(ctx context.Context, filePath string) (*SecurityScanResult, error)

	// GetDatabase returns the vulnerability database version
	GetDatabase(ctx context.Context) (string, error)

	// UpdateDatabase updates the vulnerability database
	UpdateDatabase(ctx context.Context) error
}

// BuilderConfig represents builder service configuration
type BuilderConfig struct {
	// Docker daemon configuration
	DockerHost     string `json:"docker_host"`
	DockerVersion  string `json:"docker_version"`

	// Registry configuration
	DefaultRegistry string            `json:"default_registry"`
	Registries      map[string]string `json:"registries"`

	// Security scanning
	EnableScanning       bool                   `json:"enable_scanning"`
	Scanner              string                 `json:"scanner"` // trivy, snyk, etc.
	DefaultThreshold     VulnerabilityThreshold `json:"default_threshold"`

	// Build configuration
	BuildTimeout         time.Duration `json:"build_timeout"`
	DefaultPlatforms     []string      `json:"default_platforms"`
	EnableBuildCache     bool          `json:"enable_build_cache"`
	BuildCacheSize       int64         `json:"build_cache_size"`

	// Template configuration
	TemplateDir          string        `json:"template_dir"`
	CustomTemplates      bool          `json:"custom_templates"`

	// Resource limits
	MaxConcurrentBuilds  int           `json:"max_concurrent_builds"`
	MaxImageSize         int64         `json:"max_image_size"`
	CleanupInterval      time.Duration `json:"cleanup_interval"`
}

// DefaultBuilderConfig returns default builder configuration
func DefaultBuilderConfig() *BuilderConfig {
	return &BuilderConfig{
		DockerHost:           "unix:///var/run/docker.sock",
		DockerVersion:        "1.41",
		DefaultRegistry:      "docker.io",
		EnableScanning:       true,
		Scanner:              "trivy",
		DefaultThreshold: VulnerabilityThreshold{
			Critical: 0,
			High:     5,
			Medium:   20,
			Low:      100,
		},
		BuildTimeout:         30 * time.Minute,
		DefaultPlatforms:     []string{"linux/amd64"},
		EnableBuildCache:     true,
		BuildCacheSize:       10 * 1024 * 1024 * 1024, // 10GB
		CustomTemplates:      false,
		MaxConcurrentBuilds:  5,
		MaxImageSize:         2 * 1024 * 1024 * 1024, // 2GB
		CleanupInterval:      1 * time.Hour,
	}
}

// SecurityScanResult methods

// PassesThreshold checks if the scan result passes the vulnerability threshold
func (r *SecurityScanResult) PassesThreshold(threshold VulnerabilityThreshold) bool {
	return r.Critical <= threshold.Critical &&
		r.High <= threshold.High &&
		r.Medium <= threshold.Medium &&
		r.Low <= threshold.Low
}

// TotalVulnerabilities returns the total number of vulnerabilities
func (r *SecurityScanResult) TotalVulnerabilities() int {
	return r.Critical + r.High + r.Medium + r.Low
}

// HasCriticalVulnerabilities returns true if there are critical vulnerabilities
func (r *SecurityScanResult) HasCriticalVulnerabilities() bool {
	return r.Critical > 0
}

// Summary returns a summary string of the scan results
func (r *SecurityScanResult) Summary() string {
	return fmt.Sprintf("Scanner: %s, Critical: %d, High: %d, Medium: %d, Low: %d",
		r.Scanner, r.Critical, r.High, r.Medium, r.Low)
}

// ToJSON converts the scan result to JSON
func (r *SecurityScanResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// GetVulnerabilitiesBySeverity returns vulnerabilities filtered by severity
func (r *SecurityScanResult) GetVulnerabilitiesBySeverity(severity string) []Vulnerability {
	var filtered []Vulnerability
	for _, vuln := range r.Vulnerabilities {
		if vuln.Severity == severity {
			filtered = append(filtered, vuln)
		}
	}
	return filtered
}

// GetFixableVulnerabilities returns vulnerabilities that have fixes available
func (r *SecurityScanResult) GetFixableVulnerabilities() []Vulnerability {
	var fixable []Vulnerability
	for _, vuln := range r.Vulnerabilities {
		if vuln.IsFixable() {
			fixable = append(fixable, vuln)
		}
	}
	return fixable
}

// FilterBySeverity returns a new SecurityScanResult with only specified severities
func (r *SecurityScanResult) FilterBySeverity(severities []string) *SecurityScanResult {
	severityMap := make(map[string]bool)
	for _, s := range severities {
		severityMap[s] = true
	}

	filtered := &SecurityScanResult{
		Scanner:   r.Scanner,
		ScanTime:  r.ScanTime,
		ImageID:   r.ImageID,
		ImageName: r.ImageName,
		ImageTag:  r.ImageTag,
		Threshold: r.Threshold,
	}

	for _, vuln := range r.Vulnerabilities {
		if severityMap[vuln.Severity] {
			filtered.Vulnerabilities = append(filtered.Vulnerabilities, vuln)
			switch vuln.Severity {
			case "CRITICAL":
				filtered.Critical++
			case "HIGH":
				filtered.High++
			case "MEDIUM":
				filtered.Medium++
			case "LOW":
				filtered.Low++
			}
		}
	}

	return filtered
}

// Vulnerability methods

// IsFixable returns true if the vulnerability has a fix available
func (v *Vulnerability) IsFixable() bool {
	return v.FixedVersion != ""
}

// VulnerabilityThreshold methods

// IsValid checks if the threshold values are valid
func (t *VulnerabilityThreshold) IsValid() bool {
	return t.Critical >= 0 && t.High >= 0 && t.Medium >= 0 && t.Low >= 0
}