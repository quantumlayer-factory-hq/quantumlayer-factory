package preview

import (
	"context"
	"time"
)

// PreviewRequest represents a preview environment request
type PreviewRequest struct {
	ID            string            `json:"id"`
	AppName       string            `json:"app_name"`
	ProjectPath   string            `json:"project_path"`
	Language      string            `json:"language"`
	Framework     string            `json:"framework"`
	Port          int               `json:"port"`
	Environment   map[string]string `json:"environment,omitempty"`
	TTL           time.Duration     `json:"ttl"`
	Subdomain     string            `json:"subdomain,omitempty"`
	CustomDomain  string            `json:"custom_domain,omitempty"`
	TLS           bool              `json:"tls"`
	BasicAuth     *BasicAuthConfig  `json:"basic_auth,omitempty"`
	Metadata      map[string]interface{} `json:"metadata,omitempty"`
}

// PreviewResult represents the result of a preview deployment
type PreviewResult struct {
	ID            string                 `json:"id"`
	AppName       string                 `json:"app_name"`
	URL           string                 `json:"url"`
	InternalURL   string                 `json:"internal_url"`
	Status        PreviewStatus          `json:"status"`
	BuildResult   *BuildInfo             `json:"build_result,omitempty"`
	DeployResult  *DeployInfo            `json:"deploy_result,omitempty"`
	HealthCheck   *HealthStatus          `json:"health_check,omitempty"`
	Analytics     *AnalyticsInfo         `json:"analytics,omitempty"`
	CreatedAt     time.Time              `json:"created_at"`
	UpdatedAt     time.Time              `json:"updated_at"`
	ExpiresAt     time.Time              `json:"expires_at"`
	TTL           time.Duration          `json:"ttl"`
	Logs          []string               `json:"logs,omitempty"`
	Warnings      []string               `json:"warnings"`
	Errors        []string               `json:"errors"`
	Metadata      map[string]interface{} `json:"metadata"`
}

// PreviewStatus represents the status of a preview
type PreviewStatus struct {
	Phase       string    `json:"phase"`       // Creating, Building, Deploying, Running, Failed, Expired
	Message     string    `json:"message"`
	Progress    int       `json:"progress"`    // 0-100
	LastUpdated time.Time `json:"last_updated"`
}

// BuildInfo represents build information
type BuildInfo struct {
	ImageID       string        `json:"image_id"`
	ImageName     string        `json:"image_name"`
	ImageTag      string        `json:"image_tag"`
	BuildTime     time.Duration `json:"build_time"`
	ImageSize     int64         `json:"image_size"`
	SecurityScan  *SecurityInfo `json:"security_scan,omitempty"`
}

// SecurityInfo represents security scan information
type SecurityInfo struct {
	Scanner        string `json:"scanner"`
	TotalVulns     int    `json:"total_vulnerabilities"`
	Critical       int    `json:"critical"`
	High           int    `json:"high"`
	Medium         int    `json:"medium"`
	Low            int    `json:"low"`
	Passed         bool   `json:"passed"`
}

// DeployInfo represents deployment information
type DeployInfo struct {
	Namespace      string    `json:"namespace"`
	DeploymentName string    `json:"deployment_name"`
	ServiceName    string    `json:"service_name"`
	IngressName    string    `json:"ingress_name"`
	Replicas       int32     `json:"replicas"`
	ReadyReplicas  int32     `json:"ready_replicas"`
	DeployTime     time.Duration `json:"deploy_time"`
}

// HealthStatus represents health check status
type HealthStatus struct {
	Healthy        bool      `json:"healthy"`
	LastCheck      time.Time `json:"last_check"`
	ResponseTime   time.Duration `json:"response_time"`
	StatusCode     int       `json:"status_code"`
	ErrorMessage   string    `json:"error_message,omitempty"`
	ChecksTotal    int       `json:"checks_total"`
	ChecksFailed   int       `json:"checks_failed"`
}

// AnalyticsInfo represents analytics information
type AnalyticsInfo struct {
	TotalRequests  int64     `json:"total_requests"`
	UniqueVisitors int64     `json:"unique_visitors"`
	LastAccess     time.Time `json:"last_access"`
	AvgResponseTime time.Duration `json:"avg_response_time"`
}

// BasicAuthConfig represents basic authentication configuration
type BasicAuthConfig struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

// PreviewURLConfig represents preview URL configuration
type PreviewURLConfig struct {
	Domain        string            `json:"domain"`
	Subdomains    []string          `json:"subdomains"`
	TLS           TLSConfig         `json:"tls"`
	LoadBalancer  LoadBalancerConfig `json:"load_balancer"`
	Analytics     AnalyticsConfig   `json:"analytics"`
	Security      SecurityConfig    `json:"security"`
}

// TLSConfig represents TLS configuration
type TLSConfig struct {
	Enabled       bool   `json:"enabled"`
	Provider      string `json:"provider"`      // letsencrypt, manual
	Email         string `json:"email"`
	CertDir       string `json:"cert_dir"`
	AutoRenew     bool   `json:"auto_renew"`
}

// LoadBalancerConfig represents load balancer configuration
type LoadBalancerConfig struct {
	Type         string            `json:"type"`         // nginx, traefik, envoy
	Annotations  map[string]string `json:"annotations"`
	ClassName    string            `json:"class_name"`
}

// AnalyticsConfig represents analytics configuration
type AnalyticsConfig struct {
	Enabled   bool   `json:"enabled"`
	Provider  string `json:"provider"`  // internal, google-analytics
	TrackingID string `json:"tracking_id,omitempty"`
}

// SecurityConfig represents security configuration
type SecurityConfig struct {
	BasicAuth    bool              `json:"basic_auth"`
	IPWhitelist  []string          `json:"ip_whitelist"`
	RateLimit    RateLimitConfig   `json:"rate_limit"`
	Headers      map[string]string `json:"headers"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	Enabled bool `json:"enabled"`
	RPM     int  `json:"rpm"`     // Requests per minute
	Burst   int  `json:"burst"`   // Burst size
}

// PreviewManager interface defines preview management capabilities
type PreviewManager interface {
	// Create creates a new preview environment
	Create(ctx context.Context, req *PreviewRequest) (*PreviewResult, error)

	// Get retrieves a preview by ID
	Get(ctx context.Context, id string) (*PreviewResult, error)

	// List lists all previews
	List(ctx context.Context) ([]*PreviewResult, error)

	// Update updates a preview
	Update(ctx context.Context, id string, updates map[string]interface{}) (*PreviewResult, error)

	// Delete removes a preview
	Delete(ctx context.Context, id string) error

	// Extend extends preview TTL
	Extend(ctx context.Context, id string, ttl time.Duration) error

	// GetLogs retrieves preview logs
	GetLogs(ctx context.Context, id string, lines int) ([]string, error)

	// GetStatus gets detailed status
	GetStatus(ctx context.Context, id string) (*PreviewStatus, error)

	// CleanupExpired removes expired previews
	CleanupExpired(ctx context.Context) error
}

// URLManager interface defines URL management capabilities
type URLManager interface {
	// GenerateSubdomain generates a unique subdomain
	GenerateSubdomain(appName string) (string, error)

	// ReserveSubdomain reserves a subdomain
	ReserveSubdomain(ctx context.Context, subdomain string, previewID string, ttl time.Duration) error

	// ReleaseSubdomain releases a subdomain
	ReleaseSubdomain(ctx context.Context, subdomain string) error

	// GetURL constructs the full URL for a subdomain
	GetURL(subdomain string, tls bool) string

	// ValidateSubdomain validates subdomain format
	ValidateSubdomain(subdomain string) error

	// IsSubdomainAvailable checks if subdomain is available
	IsSubdomainAvailable(ctx context.Context, subdomain string) (bool, error)
}

// TLSProvider interface defines TLS certificate management
type TLSProvider interface {
	// ProvisionCertificate provisions a TLS certificate
	ProvisionCertificate(ctx context.Context, domain string) (*TLSCertificate, error)

	// RevokeCertificate revokes a TLS certificate
	RevokeCertificate(ctx context.Context, domain string) error

	// RenewCertificate renews a TLS certificate
	RenewCertificate(ctx context.Context, domain string) (*TLSCertificate, error)

	// GetCertificate retrieves a certificate
	GetCertificate(ctx context.Context, domain string) (*TLSCertificate, error)

	// ListCertificates lists all certificates
	ListCertificates(ctx context.Context) ([]*TLSCertificate, error)
}

// TLSCertificate represents a TLS certificate
type TLSCertificate struct {
	Domain      string    `json:"domain"`
	CertPEM     []byte    `json:"cert_pem"`
	KeyPEM      []byte    `json:"key_pem"`
	IssuedAt    time.Time `json:"issued_at"`
	ExpiresAt   time.Time `json:"expires_at"`
	Issuer      string    `json:"issuer"`
	SerialNumber string   `json:"serial_number"`
}

// HealthMonitor interface defines health monitoring capabilities
type HealthMonitor interface {
	// StartMonitoring starts health monitoring for a preview
	StartMonitoring(ctx context.Context, previewID, url string) error

	// StopMonitoring stops health monitoring
	StopMonitoring(ctx context.Context, previewID string) error

	// GetHealthStatus gets current health status
	GetHealthStatus(ctx context.Context, previewID string) (*HealthStatus, error)

	// SetHealthCheck configures health check
	SetHealthCheck(ctx context.Context, previewID string, config HealthCheckConfig) error
}

// HealthCheckConfig represents health check configuration
type HealthCheckConfig struct {
	Endpoint     string        `json:"endpoint"`
	Interval     time.Duration `json:"interval"`
	Timeout      time.Duration `json:"timeout"`
	HealthyThreshold   int     `json:"healthy_threshold"`
	UnhealthyThreshold int     `json:"unhealthy_threshold"`
	ExpectedStatus     int     `json:"expected_status"`
	ExpectedBody       string  `json:"expected_body,omitempty"`
}

// AnalyticsTracker interface defines analytics tracking capabilities
type AnalyticsTracker interface {
	// TrackRequest tracks a request
	TrackRequest(ctx context.Context, previewID, ip, userAgent string) error

	// GetAnalytics gets analytics data
	GetAnalytics(ctx context.Context, previewID string) (*AnalyticsInfo, error)

	// GetAnalyticsReport gets detailed analytics report
	GetAnalyticsReport(ctx context.Context, previewID string, from, to time.Time) (map[string]interface{}, error)
}

// PreviewConfig represents preview service configuration
type PreviewConfig struct {
	// Domain configuration
	BaseDomain       string        `json:"base_domain"`
	SubdomainPattern string        `json:"subdomain_pattern"`

	// TTL configuration
	DefaultTTL       time.Duration `json:"default_ttl"`
	MaxTTL           time.Duration `json:"max_ttl"`
	CleanupInterval  time.Duration `json:"cleanup_interval"`

	// TLS configuration
	TLS              TLSConfig     `json:"tls"`

	// Load balancer configuration
	LoadBalancer     LoadBalancerConfig `json:"load_balancer"`

	// Health monitoring
	HealthCheck      HealthCheckConfig  `json:"health_check"`

	// Analytics
	Analytics        AnalyticsConfig    `json:"analytics"`

	// Security
	Security         SecurityConfig     `json:"security"`

	// Resource limits
	MaxPreviews      int           `json:"max_previews"`
	MaxPreviewsPerUser int         `json:"max_previews_per_user"`

	// Storage
	StorageBackend   string        `json:"storage_backend"`  // redis, etcd, postgres
	StorageConfig    map[string]interface{} `json:"storage_config"`
}

// DefaultPreviewConfig returns default preview configuration
func DefaultPreviewConfig() *PreviewConfig {
	return &PreviewConfig{
		BaseDomain:       "preview.quantumlayer.dev",
		SubdomainPattern: "{app}-{hash}",
		DefaultTTL:       24 * time.Hour,
		MaxTTL:           72 * time.Hour,
		CleanupInterval:  1 * time.Hour,
		TLS: TLSConfig{
			Enabled:   true,
			Provider:  "letsencrypt",
			AutoRenew: true,
		},
		LoadBalancer: LoadBalancerConfig{
			Type:      "nginx",
			ClassName: "nginx",
			Annotations: map[string]string{
				"nginx.ingress.kubernetes.io/rewrite-target": "/",
			},
		},
		HealthCheck: HealthCheckConfig{
			Endpoint:           "/health",
			Interval:           30 * time.Second,
			Timeout:            10 * time.Second,
			HealthyThreshold:   2,
			UnhealthyThreshold: 3,
			ExpectedStatus:     200,
		},
		Analytics: AnalyticsConfig{
			Enabled:  true,
			Provider: "internal",
		},
		Security: SecurityConfig{
			BasicAuth: false,
			RateLimit: RateLimitConfig{
				Enabled: true,
				RPM:     100,
				Burst:   20,
			},
			Headers: map[string]string{
				"X-Frame-Options":           "SAMEORIGIN",
				"X-Content-Type-Options":    "nosniff",
				"X-XSS-Protection":          "1; mode=block",
				"Referrer-Policy":           "strict-origin-when-cross-origin",
			},
		},
		MaxPreviews:        100,
		MaxPreviewsPerUser: 10,
		StorageBackend:     "redis",
	}
}