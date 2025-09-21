package deploy

import (
	"context"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"k8s.io/client-go/kubernetes"
)

// DeployRequest represents a Kubernetes deployment request
type DeployRequest struct {
	AppName       string            `json:"app_name"`
	Namespace     string            `json:"namespace"`
	ImageName     string            `json:"image_name"`
	ImageTag      string            `json:"image_tag"`
	Registry      string            `json:"registry,omitempty"`
	Port          int               `json:"port"`
	Replicas      int32             `json:"replicas"`
	Environment   map[string]string `json:"environment,omitempty"`
	Resources     *ResourceLimits   `json:"resources,omitempty"`
	HealthCheck   *HealthCheckConfig `json:"health_check,omitempty"`
	Ingress       *IngressConfig    `json:"ingress,omitempty"`
	PersistentVolumes []PVConfig    `json:"persistent_volumes,omitempty"`
	TTL           time.Duration     `json:"ttl"`
	Labels        map[string]string `json:"labels,omitempty"`
	Annotations   map[string]string `json:"annotations,omitempty"`
	Spec          *ir.IRSpec        `json:"spec,omitempty"`
}

// DeployResult represents the result of a Kubernetes deployment
type DeployResult struct {
	Success        bool                   `json:"success"`
	Namespace      string                 `json:"namespace"`
	AppName        string                 `json:"app_name"`
	DeploymentName string                 `json:"deployment_name"`
	ServiceName    string                 `json:"service_name"`
	IngressName    string                 `json:"ingress_name,omitempty"`
	URL            string                 `json:"url,omitempty"`
	InternalURL    string                 `json:"internal_url"`
	Status         DeploymentStatus       `json:"status"`
	Pods           []PodInfo              `json:"pods"`
	Events         []string               `json:"events"`
	Warnings       []string               `json:"warnings"`
	Errors         []string               `json:"errors"`
	Metadata       map[string]interface{} `json:"metadata"`
	CreatedAt      time.Time              `json:"created_at"`
	ExpiresAt      time.Time              `json:"expires_at"`
}

// ResourceLimits defines CPU and memory limits
type ResourceLimits struct {
	CPU    ResourceSpec `json:"cpu"`
	Memory ResourceSpec `json:"memory"`
}

// ResourceSpec defines resource requests and limits
type ResourceSpec struct {
	Requests string `json:"requests"`
	Limits   string `json:"limits"`
}

// HealthCheckConfig defines health check configuration
type HealthCheckConfig struct {
	LivenessProbe  *ProbeConfig `json:"liveness_probe,omitempty"`
	ReadinessProbe *ProbeConfig `json:"readiness_probe,omitempty"`
	StartupProbe   *ProbeConfig `json:"startup_probe,omitempty"`
}

// ProbeConfig defines a probe configuration
type ProbeConfig struct {
	Path                string        `json:"path"`
	Port                int           `json:"port"`
	InitialDelaySeconds int32         `json:"initial_delay_seconds"`
	PeriodSeconds       int32         `json:"period_seconds"`
	TimeoutSeconds      int32         `json:"timeout_seconds"`
	SuccessThreshold    int32         `json:"success_threshold"`
	FailureThreshold    int32         `json:"failure_threshold"`
}

// IngressConfig defines ingress configuration
type IngressConfig struct {
	Enabled     bool              `json:"enabled"`
	Host        string            `json:"host"`
	Path        string            `json:"path"`
	TLS         bool              `json:"tls"`
	Annotations map[string]string `json:"annotations,omitempty"`
	ClassName   string            `json:"class_name,omitempty"`
}

// PVConfig defines persistent volume configuration
type PVConfig struct {
	Name         string `json:"name"`
	MountPath    string `json:"mount_path"`
	Size         string `json:"size"`
	StorageClass string `json:"storage_class,omitempty"`
	AccessModes  []string `json:"access_modes"`
}

// DeploymentStatus represents the status of a deployment
type DeploymentStatus struct {
	Phase             string    `json:"phase"`             // Pending, Running, Succeeded, Failed, Unknown
	Replicas          int32     `json:"replicas"`
	ReadyReplicas     int32     `json:"ready_replicas"`
	AvailableReplicas int32     `json:"available_replicas"`
	UpdatedReplicas   int32     `json:"updated_replicas"`
	LastUpdateTime    time.Time `json:"last_update_time"`
	Message           string    `json:"message,omitempty"`
}

// PodInfo represents information about a pod
type PodInfo struct {
	Name      string            `json:"name"`
	Status    string            `json:"status"`
	Ready     bool              `json:"ready"`
	Restarts  int32             `json:"restarts"`
	Age       time.Duration     `json:"age"`
	Node      string            `json:"node"`
	IP        string            `json:"ip"`
	Labels    map[string]string `json:"labels"`
	CreatedAt time.Time         `json:"created_at"`
}

// NamespaceInfo represents information about a namespace
type NamespaceInfo struct {
	Name        string            `json:"name"`
	Status      string            `json:"status"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	CreatedAt   time.Time         `json:"created_at"`
	ExpiresAt   time.Time         `json:"expires_at"`
	TTL         time.Duration     `json:"ttl"`
}

// Deployer interface defines Kubernetes deployment capabilities
type Deployer interface {
	// Deploy deploys an application to Kubernetes
	Deploy(ctx context.Context, req *DeployRequest) (*DeployResult, error)

	// GetStatus gets the status of a deployment
	GetStatus(ctx context.Context, namespace, appName string) (*DeployResult, error)

	// Delete removes a deployment
	Delete(ctx context.Context, namespace, appName string) error

	// List lists deployments in a namespace
	List(ctx context.Context, namespace string) ([]*DeployResult, error)

	// Scale scales a deployment
	Scale(ctx context.Context, namespace, appName string, replicas int32) error

	// GetLogs gets logs from pods
	GetLogs(ctx context.Context, namespace, appName string, lines int64) ([]string, error)

	// ValidateManifests validates Kubernetes manifests
	ValidateManifests(manifests map[string]interface{}) error
}

// NamespaceManager interface defines namespace management capabilities
type NamespaceManager interface {
	// CreateNamespace creates a new namespace
	CreateNamespace(ctx context.Context, name string, labels map[string]string, ttl time.Duration) (*NamespaceInfo, error)

	// DeleteNamespace deletes a namespace
	DeleteNamespace(ctx context.Context, name string) error

	// GetNamespace gets namespace information
	GetNamespace(ctx context.Context, name string) (*NamespaceInfo, error)

	// ListNamespaces lists namespaces
	ListNamespaces(ctx context.Context) ([]*NamespaceInfo, error)

	// CleanupExpiredNamespaces removes expired namespaces
	CleanupExpiredNamespaces(ctx context.Context) error

	// ExtendTTL extends namespace TTL
	ExtendTTL(ctx context.Context, name string, ttl time.Duration) error
}

// ManifestGenerator interface defines manifest generation capabilities
type ManifestGenerator interface {
	// GenerateDeployment generates a Deployment manifest
	GenerateDeployment(req *DeployRequest) (map[string]interface{}, error)

	// GenerateService generates a Service manifest
	GenerateService(req *DeployRequest) (map[string]interface{}, error)

	// GenerateIngress generates an Ingress manifest
	GenerateIngress(req *DeployRequest) (map[string]interface{}, error)

	// GenerateConfigMap generates a ConfigMap manifest
	GenerateConfigMap(req *DeployRequest) (map[string]interface{}, error)

	// GenerateSecret generates a Secret manifest
	GenerateSecret(req *DeployRequest) (map[string]interface{}, error)

	// GeneratePVC generates a PersistentVolumeClaim manifest
	GeneratePVC(req *DeployRequest, pv PVConfig) (map[string]interface{}, error)

	// GenerateAll generates all required manifests
	GenerateAll(req *DeployRequest) (map[string]map[string]interface{}, error)
}

// DeployerConfig represents deployer service configuration
type DeployerConfig struct {
	// Kubernetes configuration
	KubeConfig      string `json:"kube_config,omitempty"`
	InCluster       bool   `json:"in_cluster"`
	Namespace       string `json:"default_namespace"`

	// Deployment defaults
	DefaultReplicas   int32         `json:"default_replicas"`
	DefaultTTL        time.Duration `json:"default_ttl"`
	MaxTTL            time.Duration `json:"max_ttl"`
	CleanupInterval   time.Duration `json:"cleanup_interval"`

	// Resource limits
	DefaultCPURequest    string `json:"default_cpu_request"`
	DefaultCPULimit      string `json:"default_cpu_limit"`
	DefaultMemoryRequest string `json:"default_memory_request"`
	DefaultMemoryLimit   string `json:"default_memory_limit"`
	MaxCPULimit          string `json:"max_cpu_limit"`
	MaxMemoryLimit       string `json:"max_memory_limit"`

	// Ingress configuration
	IngressClassName    string            `json:"ingress_class_name"`
	IngressAnnotations  map[string]string `json:"ingress_annotations"`
	TLSEnabled          bool              `json:"tls_enabled"`
	CertManager         bool              `json:"cert_manager"`

	// Storage configuration
	DefaultStorageClass string   `json:"default_storage_class"`
	AllowedStorageClasses []string `json:"allowed_storage_classes"`

	// Security
	PodSecurityPolicy   bool              `json:"pod_security_policy"`
	NetworkPolicy       bool              `json:"network_policy"`
	ServiceAccount      string            `json:"service_account"`
	ImagePullSecrets    []string          `json:"image_pull_secrets"`

	// Monitoring
	EnableMetrics       bool              `json:"enable_metrics"`
	PrometheusLabels    map[string]string `json:"prometheus_labels"`
}

// DefaultDeployerConfig returns default deployer configuration
func DefaultDeployerConfig() *DeployerConfig {
	return &DeployerConfig{
		InCluster:            false,
		Namespace:            "default",
		DefaultReplicas:      1,
		DefaultTTL:           24 * time.Hour,
		MaxTTL:               72 * time.Hour,
		CleanupInterval:      1 * time.Hour,
		DefaultCPURequest:    "100m",
		DefaultCPULimit:      "500m",
		DefaultMemoryRequest: "128Mi",
		DefaultMemoryLimit:   "512Mi",
		MaxCPULimit:          "2",
		MaxMemoryLimit:       "4Gi",
		IngressClassName:     "nginx",
		IngressAnnotations: map[string]string{
			"nginx.ingress.kubernetes.io/rewrite-target": "/",
		},
		TLSEnabled:            true,
		CertManager:           true,
		DefaultStorageClass:   "standard",
		AllowedStorageClasses: []string{"standard", "ssd", "premium"},
		PodSecurityPolicy:     true,
		NetworkPolicy:         true,
		ServiceAccount:        "default",
		EnableMetrics:         true,
		PrometheusLabels: map[string]string{
			"app.kubernetes.io/managed-by": "quantumlayer-factory",
		},
	}
}

// KubernetesClient wraps the Kubernetes client with additional functionality
type KubernetesClient struct {
	clientset kubernetes.Interface
	config    *DeployerConfig
}

// NewKubernetesClient creates a new Kubernetes client wrapper
func NewKubernetesClient(clientset kubernetes.Interface, config *DeployerConfig) *KubernetesClient {
	return &KubernetesClient{
		clientset: clientset,
		config:    config,
	}
}

// GetClientset returns the underlying Kubernetes clientset
func (kc *KubernetesClient) GetClientset() kubernetes.Interface {
	return kc.clientset
}

// GetConfig returns the deployer configuration
func (kc *KubernetesClient) GetConfig() *DeployerConfig {
	return kc.config
}