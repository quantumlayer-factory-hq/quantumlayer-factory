package packager

import (
	"crypto/sha256"
	"fmt"
	"time"
)

// CapsuleManifest represents the metadata for a .qlcapsule package
type CapsuleManifest struct {
	Version     string                 `json:"version"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Author      string                 `json:"author"`
	CreatedAt   time.Time              `json:"created_at"`

	// Package metadata
	Tags        []string               `json:"tags,omitempty"`
	License     string                 `json:"license,omitempty"`
	Repository  string                 `json:"repository,omitempty"`
	Homepage    string                 `json:"homepage,omitempty"`

	// Technical specifications
	Language    string                 `json:"language"`
	Framework   string                 `json:"framework"`
	Runtime     RuntimeSpec            `json:"runtime"`

	// Security and compliance
	SBOM        *SBOM                  `json:"sbom"`
	Attestation *Attestation           `json:"attestation"`
	Signatures  []DigitalSignature     `json:"signatures"`

	// Deployment configuration
	Deployment  DeploymentConfig       `json:"deployment"`

	// Files and artifacts
	Artifacts   []Artifact             `json:"artifacts"`

	// Custom metadata
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

// RuntimeSpec defines runtime requirements
type RuntimeSpec struct {
	MinVersion string            `json:"min_version,omitempty"`
	MaxVersion string            `json:"max_version,omitempty"`
	Platform   []string          `json:"platform"` // linux/amd64, darwin/arm64, etc.
	Arch       []string          `json:"arch"`     // amd64, arm64, etc.
	Environment map[string]string `json:"environment,omitempty"`
	Resources   ResourceLimits    `json:"resources"`
}

// ResourceLimits defines resource requirements
type ResourceLimits struct {
	CPU     string `json:"cpu,omitempty"`     // "100m", "1.5"
	Memory  string `json:"memory,omitempty"`  // "128Mi", "1Gi"
	Storage string `json:"storage,omitempty"` // "1Gi", "10Gi"
}

// SBOM (Software Bill of Materials)
type SBOM struct {
	Format      string      `json:"format"`      // "spdx", "cyclonedx"
	Version     string      `json:"version"`
	GeneratedAt time.Time   `json:"generated_at"`
	Tool        string      `json:"tool"`        // "syft", "cyclonedx"
	Components  []Component `json:"components"`
}

// Component represents a software component in SBOM
type Component struct {
	Type         string            `json:"type"`          // "library", "framework", "application"
	Name         string            `json:"name"`
	Version      string            `json:"version"`
	PackageURL   string            `json:"purl,omitempty"` // Package URL
	License      string            `json:"license,omitempty"`
	Supplier     string            `json:"supplier,omitempty"`
	Hash         string            `json:"hash,omitempty"`
	Dependencies []string          `json:"dependencies,omitempty"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities,omitempty"`
}

// Vulnerability represents a security vulnerability
type Vulnerability struct {
	ID          string    `json:"id"`          // CVE-2021-1234
	Description string    `json:"description"`
	Severity    string    `json:"severity"`    // Critical, High, Medium, Low
	Score       float64   `json:"score,omitempty"` // CVSS score
	URL         string    `json:"url,omitempty"`
	FixVersion  string    `json:"fix_version,omitempty"`
	Published   time.Time `json:"published,omitempty"`
}

// Attestation provides integrity and provenance information
type Attestation struct {
	BuildPlatform  string            `json:"build_platform"`
	BuildTimestamp time.Time         `json:"build_timestamp"`
	BuildHash      string            `json:"build_hash"`
	SourceRepo     string            `json:"source_repo,omitempty"`
	SourceCommit   string            `json:"source_commit,omitempty"`
	Builder        string            `json:"builder"` // "quantumlayer-factory"
	BuildArgs      map[string]string `json:"build_args,omitempty"`
	Reproducible   bool              `json:"reproducible"`
}

// DigitalSignature for package integrity
type DigitalSignature struct {
	Algorithm string    `json:"algorithm"` // "RSA", "ECDSA"
	KeyID     string    `json:"key_id"`
	Signature string    `json:"signature"` // Base64 encoded
	Timestamp time.Time `json:"timestamp"`
	Signer    string    `json:"signer"`
}

// DeploymentConfig contains deployment-specific configuration
type DeploymentConfig struct {
	Type        string                 `json:"type"` // "kubernetes", "docker", "serverless"
	Manifests   []DeploymentManifest   `json:"manifests"`
	Secrets     []SecretRef            `json:"secrets,omitempty"`
	ConfigMaps  []ConfigMapRef         `json:"config_maps,omitempty"`
	Networking  NetworkingConfig       `json:"networking,omitempty"`
	Storage     []StorageConfig        `json:"storage,omitempty"`
	Monitoring  MonitoringConfig       `json:"monitoring,omitempty"`
}

// DeploymentManifest represents a deployment manifest file
type DeploymentManifest struct {
	Name     string `json:"name"`
	Type     string `json:"type"` // "Deployment", "Service", "Ingress"
	Path     string `json:"path"` // Path within the capsule
	Required bool   `json:"required"`
}

// SecretRef references a required secret
type SecretRef struct {
	Name        string   `json:"name"`
	Keys        []string `json:"keys"`
	Description string   `json:"description,omitempty"`
	Required    bool     `json:"required"`
}

// ConfigMapRef references a required config map
type ConfigMapRef struct {
	Name        string   `json:"name"`
	Keys        []string `json:"keys,omitempty"`
	Description string   `json:"description,omitempty"`
	Required    bool     `json:"required"`
}

// NetworkingConfig defines networking requirements
type NetworkingConfig struct {
	Ports     []PortConfig `json:"ports,omitempty"`
	Ingress   bool         `json:"ingress,omitempty"`
	LoadBalancer bool      `json:"load_balancer,omitempty"`
	DNSPolicy string       `json:"dns_policy,omitempty"`
}

// PortConfig defines a port configuration
type PortConfig struct {
	Name       string `json:"name"`
	Port       int    `json:"port"`
	TargetPort int    `json:"target_port,omitempty"`
	Protocol   string `json:"protocol,omitempty"` // TCP, UDP
	External   bool   `json:"external"`
}

// StorageConfig defines storage requirements
type StorageConfig struct {
	Name        string `json:"name"`
	Type        string `json:"type"` // "persistent", "ephemeral"
	Size        string `json:"size"`
	AccessMode  string `json:"access_mode,omitempty"`
	StorageClass string `json:"storage_class,omitempty"`
	MountPath   string `json:"mount_path"`
}

// MonitoringConfig defines monitoring configuration
type MonitoringConfig struct {
	HealthCheck  HealthCheckConfig `json:"health_check,omitempty"`
	Metrics      MetricsConfig     `json:"metrics,omitempty"`
	Logging      LoggingConfig     `json:"logging,omitempty"`
	Alerting     AlertingConfig    `json:"alerting,omitempty"`
}

// HealthCheckConfig defines health check configuration
type HealthCheckConfig struct {
	Enabled     bool   `json:"enabled"`
	Path        string `json:"path,omitempty"`
	Port        int    `json:"port,omitempty"`
	Interval    string `json:"interval,omitempty"`
	Timeout     string `json:"timeout,omitempty"`
	Retries     int    `json:"retries,omitempty"`
	StartPeriod string `json:"start_period,omitempty"`
}

// MetricsConfig defines metrics collection
type MetricsConfig struct {
	Enabled bool   `json:"enabled"`
	Path    string `json:"path,omitempty"`
	Port    int    `json:"port,omitempty"`
	Format  string `json:"format,omitempty"` // "prometheus", "statsd"
}

// LoggingConfig defines logging configuration
type LoggingConfig struct {
	Level  string            `json:"level,omitempty"`  // "debug", "info", "warn", "error"
	Format string            `json:"format,omitempty"` // "json", "text"
	Fields map[string]string `json:"fields,omitempty"`
}

// AlertingConfig defines alerting rules
type AlertingConfig struct {
	Enabled bool         `json:"enabled"`
	Rules   []AlertRule  `json:"rules,omitempty"`
}

// AlertRule defines an alerting rule
type AlertRule struct {
	Name        string            `json:"name"`
	Condition   string            `json:"condition"`
	Threshold   float64           `json:"threshold,omitempty"`
	Duration    string            `json:"duration,omitempty"`
	Severity    string            `json:"severity"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// Artifact represents a file or resource in the package
type Artifact struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"` // "binary", "config", "documentation", "manifest"
	Path        string            `json:"path"`
	Size        int64             `json:"size"`
	Hash        string            `json:"hash"` // SHA256
	Compressed  bool              `json:"compressed"`
	Executable  bool              `json:"executable,omitempty"`
	Permissions string            `json:"permissions,omitempty"` // "755", "644"
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// PackageRequest represents a request to create a package
type PackageRequest struct {
	Name          string                 `json:"name"`
	Version       string                 `json:"version"`
	Description   string                 `json:"description,omitempty"`
	Author        string                 `json:"author,omitempty"`
	License       string                 `json:"license,omitempty"`

	// Source information
	SourcePath    string                 `json:"source_path"`
	BuildArtifacts []string              `json:"build_artifacts"`
	Manifests     []string               `json:"manifests"`
	Documentation []string               `json:"documentation,omitempty"`

	// Configuration
	Language      string                 `json:"language"`
	Framework     string                 `json:"framework"`
	Runtime       RuntimeSpec            `json:"runtime"`

	// Security options
	SigningKey    string                 `json:"signing_key,omitempty"`
	GenerateSBOM  bool                   `json:"generate_sbom"`
	ScanVulns     bool                   `json:"scan_vulnerabilities"`

	// Compression options
	Compression   string                 `json:"compression,omitempty"` // "gzip", "lz4", "zstd"
	CompressionLevel int                 `json:"compression_level,omitempty"`

	// Custom metadata
	Tags          []string               `json:"tags,omitempty"`
	Labels        map[string]string      `json:"labels,omitempty"`
	Annotations   map[string]string      `json:"annotations,omitempty"`
}

// PackageResult represents the result of package creation
type PackageResult struct {
	CapsulePath   string        `json:"capsule_path"`
	Manifest      CapsuleManifest `json:"manifest"`
	Size          int64         `json:"size"`
	Hash          string        `json:"hash"`
	CreatedAt     time.Time     `json:"created_at"`
	BuildTime     time.Duration `json:"build_time"`
	Compressed    bool          `json:"compressed"`
	CompressionRatio float64    `json:"compression_ratio,omitempty"`

	// Validation results
	Validated     bool          `json:"validated"`
	Signature     string        `json:"signature,omitempty"`

	// Security scan results
	VulnScanResult *VulnScanResult `json:"vulnerability_scan,omitempty"`
}

// VulnScanResult represents vulnerability scan results
type VulnScanResult struct {
	TotalVulns    int              `json:"total_vulnerabilities"`
	Critical      int              `json:"critical"`
	High          int              `json:"high"`
	Medium        int              `json:"medium"`
	Low           int              `json:"low"`
	Fixable       int              `json:"fixable"`
	ScanTime      time.Time        `json:"scan_time"`
	Scanner       string           `json:"scanner"`
	ScanDuration  time.Duration    `json:"scan_duration"`
	Vulnerabilities []Vulnerability `json:"vulnerabilities,omitempty"`
}

// DeliveryChannel represents a delivery channel configuration
type DeliveryChannel struct {
	Name        string            `json:"name"`
	Type        string            `json:"type"` // "registry", "cdn", "direct", "package_manager"
	Endpoint    string            `json:"endpoint"`
	Credentials map[string]string `json:"credentials,omitempty"`
	Config      map[string]interface{} `json:"config,omitempty"`
	Enabled     bool              `json:"enabled"`
}

// PublishRequest represents a request to publish a package
type PublishRequest struct {
	CapsulePath string            `json:"capsule_path"`
	Channels    []string          `json:"channels"`
	Tags        []string          `json:"tags,omitempty"`
	Public      bool              `json:"public"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// PublishResult represents the result of package publishing
type PublishResult struct {
	URLs        map[string]string `json:"urls"` // channel -> URL
	PublishedAt time.Time         `json:"published_at"`
	Success     bool              `json:"success"`
	Errors      []string          `json:"errors,omitempty"`
}

// Helper methods

// CalculateHash calculates SHA256 hash of the manifest
func (m *CapsuleManifest) CalculateHash() string {
	// This would be implemented to calculate a deterministic hash
	// of the manifest for integrity verification
	hasher := sha256.New()
	// Implementation would serialize manifest deterministically
	_ = hasher // TODO: implement deterministic serialization
	return ""
}

// Validate validates the manifest structure
func (m *CapsuleManifest) Validate() error {
	if m.Name == "" {
		return ErrInvalidName
	}
	if m.Version == "" {
		return ErrInvalidVersion
	}
	if m.Language == "" {
		return ErrInvalidLanguage
	}
	return nil
}

// IsVulnerable checks if the package has vulnerabilities above threshold
func (r *VulnScanResult) IsVulnerable(threshold string) bool {
	switch threshold {
	case "critical":
		return r.Critical > 0
	case "high":
		return r.Critical > 0 || r.High > 0
	case "medium":
		return r.Critical > 0 || r.High > 0 || r.Medium > 0
	case "low":
		return r.Critical > 0 || r.High > 0 || r.Medium > 0 || r.Low > 0
	default:
		return r.Critical > 0
	}
}

// GetSeverityCount returns count of vulnerabilities by severity
func (r *VulnScanResult) GetSeverityCount(severity string) int {
	switch severity {
	case "critical":
		return r.Critical
	case "high":
		return r.High
	case "medium":
		return r.Medium
	case "low":
		return r.Low
	default:
		return 0
	}
}

// Common errors
var (
	ErrInvalidName     = fmt.Errorf("invalid package name")
	ErrInvalidVersion  = fmt.Errorf("invalid package version")
	ErrInvalidLanguage = fmt.Errorf("invalid language")
	ErrInvalidPath     = fmt.Errorf("invalid path")
	ErrSigningFailed   = fmt.Errorf("signing failed")
	ErrCompressionFailed = fmt.Errorf("compression failed")
	ErrValidationFailed = fmt.Errorf("validation failed")
)