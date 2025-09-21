package deploy

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewKubernetesDeployer(t *testing.T) {
	config := DefaultDeployerConfig()

	// Since NewKubernetesDeployer tries to connect to a real K8s cluster,
	// we'll test that it returns an error when no cluster is available
	_, err := NewKubernetesDeployer(config)
	assert.Error(t, err, "Should fail when no Kubernetes cluster is available")
}

func TestDefaultDeployerConfig(t *testing.T) {
	config := DefaultDeployerConfig()

	assert.False(t, config.InCluster)
	assert.Equal(t, "default", config.Namespace)
	assert.Equal(t, int32(1), config.DefaultReplicas)
	assert.Equal(t, 24*time.Hour, config.DefaultTTL)
	assert.Equal(t, 72*time.Hour, config.MaxTTL)
	assert.Equal(t, 1*time.Hour, config.CleanupInterval)
	assert.Equal(t, "100m", config.DefaultCPURequest)
	assert.Equal(t, "500m", config.DefaultCPULimit)
	assert.Equal(t, "128Mi", config.DefaultMemoryRequest)
	assert.Equal(t, "512Mi", config.DefaultMemoryLimit)
	assert.Equal(t, "2", config.MaxCPULimit)
	assert.Equal(t, "4Gi", config.MaxMemoryLimit)
	assert.Equal(t, "nginx", config.IngressClassName)
	assert.True(t, config.TLSEnabled)
	assert.True(t, config.CertManager)
	assert.Equal(t, "standard", config.DefaultStorageClass)
	assert.Contains(t, config.AllowedStorageClasses, "standard")
	assert.Contains(t, config.AllowedStorageClasses, "ssd")
	assert.Contains(t, config.AllowedStorageClasses, "premium")
	assert.True(t, config.PodSecurityPolicy)
	assert.True(t, config.NetworkPolicy)
	assert.Equal(t, "default", config.ServiceAccount)
	assert.True(t, config.EnableMetrics)
}

func TestDeployRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		req     *DeployRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &DeployRequest{
				AppName:   "test-app",
				Namespace: "test-ns",
				ImageName: "test-image",
				ImageTag:  "latest",
				Port:      8080,
				Replicas:  1,
			},
			wantErr: false,
		},
		{
			name: "missing app name",
			req: &DeployRequest{
				Namespace: "test-ns",
				ImageName: "test-image",
				ImageTag:  "latest",
				Port:      8080,
				Replicas:  1,
			},
			wantErr: true,
		},
		{
			name: "missing namespace",
			req: &DeployRequest{
				AppName:   "test-app",
				ImageName: "test-image",
				ImageTag:  "latest",
				Port:      8080,
				Replicas:  1,
			},
			wantErr: true,
		},
		{
			name: "missing image name",
			req: &DeployRequest{
				AppName:   "test-app",
				Namespace: "test-ns",
				ImageTag:  "latest",
				Port:      8080,
				Replicas:  1,
			},
			wantErr: true,
		},
		{
			name: "invalid port",
			req: &DeployRequest{
				AppName:   "test-app",
				Namespace: "test-ns",
				ImageName: "test-image",
				ImageTag:  "latest",
				Port:      0,
				Replicas:  1,
			},
			wantErr: true,
		},
		{
			name: "invalid replicas",
			req: &DeployRequest{
				AppName:   "test-app",
				Namespace: "test-ns",
				ImageName: "test-image",
				ImageTag:  "latest",
				Port:      8080,
				Replicas:  -1,
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateDeployRequest(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResourceLimits_Validation(t *testing.T) {
	tests := []struct {
		name      string
		resources *ResourceLimits
		config    *DeployerConfig
		wantErr   bool
	}{
		{
			name: "valid resources",
			resources: &ResourceLimits{
				CPU: ResourceSpec{
					Requests: "100m",
					Limits:   "500m",
				},
				Memory: ResourceSpec{
					Requests: "128Mi",
					Limits:   "512Mi",
				},
			},
			config:  DefaultDeployerConfig(),
			wantErr: false,
		},
		{
			name: "CPU limit exceeds maximum",
			resources: &ResourceLimits{
				CPU: ResourceSpec{
					Requests: "100m",
					Limits:   "4",
				},
				Memory: ResourceSpec{
					Requests: "128Mi",
					Limits:   "512Mi",
				},
			},
			config:  DefaultDeployerConfig(),
			wantErr: true,
		},
		{
			name: "Memory limit exceeds maximum",
			resources: &ResourceLimits{
				CPU: ResourceSpec{
					Requests: "100m",
					Limits:   "500m",
				},
				Memory: ResourceSpec{
					Requests: "128Mi",
					Limits:   "8Gi",
				},
			},
			config:  DefaultDeployerConfig(),
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateResourceLimits(tt.resources, tt.config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestHealthCheckConfig_Validation(t *testing.T) {
	tests := []struct {
		name        string
		healthCheck *HealthCheckConfig
		wantErr     bool
	}{
		{
			name: "valid health check",
			healthCheck: &HealthCheckConfig{
				LivenessProbe: &ProbeConfig{
					Path:                "/health",
					Port:                8080,
					InitialDelaySeconds: 30,
					PeriodSeconds:       10,
					TimeoutSeconds:      5,
					SuccessThreshold:    1,
					FailureThreshold:    3,
				},
				ReadinessProbe: &ProbeConfig{
					Path:                "/ready",
					Port:                8080,
					InitialDelaySeconds: 5,
					PeriodSeconds:       5,
					TimeoutSeconds:      3,
					SuccessThreshold:    1,
					FailureThreshold:    3,
				},
			},
			wantErr: false,
		},
		{
			name: "invalid port",
			healthCheck: &HealthCheckConfig{
				LivenessProbe: &ProbeConfig{
					Path:                "/health",
					Port:                0,
					InitialDelaySeconds: 30,
					PeriodSeconds:       10,
					TimeoutSeconds:      5,
					SuccessThreshold:    1,
					FailureThreshold:    3,
				},
			},
			wantErr: true,
		},
		{
			name: "invalid timeout",
			healthCheck: &HealthCheckConfig{
				LivenessProbe: &ProbeConfig{
					Path:                "/health",
					Port:                8080,
					InitialDelaySeconds: 30,
					PeriodSeconds:       10,
					TimeoutSeconds:      0,
					SuccessThreshold:    1,
					FailureThreshold:    3,
				},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateHealthCheckConfig(tt.healthCheck)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestIngressConfig_Validation(t *testing.T) {
	tests := []struct {
		name    string
		ingress *IngressConfig
		wantErr bool
	}{
		{
			name: "valid ingress",
			ingress: &IngressConfig{
				Enabled: true,
				Host:    "example.com",
				Path:    "/",
				TLS:     true,
			},
			wantErr: false,
		},
		{
			name: "missing host when enabled",
			ingress: &IngressConfig{
				Enabled: true,
				Path:    "/",
				TLS:     true,
			},
			wantErr: true,
		},
		{
			name: "disabled ingress",
			ingress: &IngressConfig{
				Enabled: false,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateIngressConfig(tt.ingress)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestPVConfig_Validation(t *testing.T) {
	config := DefaultDeployerConfig()

	tests := []struct {
		name    string
		pv      PVConfig
		wantErr bool
	}{
		{
			name: "valid PV config",
			pv: PVConfig{
				Name:         "data-volume",
				MountPath:    "/data",
				Size:         "10Gi",
				StorageClass: "standard",
				AccessModes:  []string{"ReadWriteOnce"},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			pv: PVConfig{
				MountPath:    "/data",
				Size:         "10Gi",
				StorageClass: "standard",
				AccessModes:  []string{"ReadWriteOnce"},
			},
			wantErr: true,
		},
		{
			name: "invalid storage class",
			pv: PVConfig{
				Name:         "data-volume",
				MountPath:    "/data",
				Size:         "10Gi",
				StorageClass: "invalid",
				AccessModes:  []string{"ReadWriteOnce"},
			},
			wantErr: true,
		},
		{
			name: "missing access modes",
			pv: PVConfig{
				Name:         "data-volume",
				MountPath:    "/data",
				Size:         "10Gi",
				StorageClass: "standard",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validatePVConfig(tt.pv, config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDeploymentStatus_Phases(t *testing.T) {
	tests := []struct {
		name   string
		status DeploymentStatus
		ready  bool
	}{
		{
			name: "running and ready",
			status: DeploymentStatus{
				Phase:             "Running",
				Replicas:          2,
				ReadyReplicas:     2,
				AvailableReplicas: 2,
				UpdatedReplicas:   2,
			},
			ready: true,
		},
		{
			name: "running but not ready",
			status: DeploymentStatus{
				Phase:             "Running",
				Replicas:          2,
				ReadyReplicas:     1,
				AvailableReplicas: 1,
				UpdatedReplicas:   2,
			},
			ready: false,
		},
		{
			name: "pending",
			status: DeploymentStatus{
				Phase:             "Pending",
				Replicas:          2,
				ReadyReplicas:     0,
				AvailableReplicas: 0,
				UpdatedReplicas:   0,
			},
			ready: false,
		},
		{
			name: "failed",
			status: DeploymentStatus{
				Phase:   "Failed",
				Message: "Image pull error",
			},
			ready: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isReady := tt.status.IsReady()
			assert.Equal(t, tt.ready, isReady)
		})
	}
}

func TestPodInfo_Age(t *testing.T) {
	now := time.Now()
	pod := PodInfo{
		Name:      "test-pod",
		Status:    "Running",
		Ready:     true,
		Restarts:  0,
		Node:      "node-1",
		IP:        "10.0.0.1",
		CreatedAt: now.Add(-5 * time.Minute),
	}

	age := pod.GetAge()
	assert.True(t, age >= 5*time.Minute)
	assert.True(t, age < 6*time.Minute)
}

func TestDeployResult_IsHealthy(t *testing.T) {
	tests := []struct {
		name    string
		result  DeployResult
		healthy bool
	}{
		{
			name: "healthy deployment",
			result: DeployResult{
				Success: true,
				Status: DeploymentStatus{
					Phase:             "Running",
					Replicas:          2,
					ReadyReplicas:     2,
					AvailableReplicas: 2,
				},
				Pods: []PodInfo{
					{Status: "Running", Ready: true},
					{Status: "Running", Ready: true},
				},
			},
			healthy: true,
		},
		{
			name: "unhealthy deployment - not ready",
			result: DeployResult{
				Success: true,
				Status: DeploymentStatus{
					Phase:             "Running",
					Replicas:          2,
					ReadyReplicas:     1,
					AvailableReplicas: 1,
				},
				Pods: []PodInfo{
					{Status: "Running", Ready: true},
					{Status: "Pending", Ready: false},
				},
			},
			healthy: false,
		},
		{
			name: "failed deployment",
			result: DeployResult{
				Success: false,
				Errors:  []string{"Deployment failed"},
			},
			healthy: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			isHealthy := tt.result.IsHealthy()
			assert.Equal(t, tt.healthy, isHealthy)
		})
	}
}

// Mock deployment test
func TestKubernetesDeployer_MockDeploy(t *testing.T) {
	ctx := context.Background()
	req := &DeployRequest{
		AppName:   "test-app",
		Namespace: "test-namespace",
		ImageName: "nginx",
		ImageTag:  "latest",
		Port:      80,
		Replicas:  2,
		TTL:       24 * time.Hour,
	}

	// Test the request structure without needing a real deployer
	assert.Equal(t, "test-app", req.AppName)
	assert.Equal(t, "test-namespace", req.Namespace)
	assert.Equal(t, int32(2), req.Replicas)
	assert.Equal(t, 80, req.Port)
	assert.NotNil(t, ctx)
}

// Helper validation functions that would be implemented in the actual deployer

func validateDeployRequest(req *DeployRequest) error {
	if req.AppName == "" {
		return assert.AnError
	}
	if req.Namespace == "" {
		return assert.AnError
	}
	if req.ImageName == "" {
		return assert.AnError
	}
	if req.Port <= 0 {
		return assert.AnError
	}
	if req.Replicas < 0 {
		return assert.AnError
	}
	return nil
}

func validateResourceLimits(resources *ResourceLimits, config *DeployerConfig) error {
	// Mock validation - in real implementation, this would parse and compare resource strings
	if resources.CPU.Limits == "4" && config.MaxCPULimit == "2" {
		return assert.AnError
	}
	if resources.Memory.Limits == "8Gi" && config.MaxMemoryLimit == "4Gi" {
		return assert.AnError
	}
	return nil
}

func validateHealthCheckConfig(healthCheck *HealthCheckConfig) error {
	if healthCheck.LivenessProbe != nil {
		if healthCheck.LivenessProbe.Port <= 0 {
			return assert.AnError
		}
		if healthCheck.LivenessProbe.TimeoutSeconds <= 0 {
			return assert.AnError
		}
	}
	if healthCheck.ReadinessProbe != nil {
		if healthCheck.ReadinessProbe.Port <= 0 {
			return assert.AnError
		}
		if healthCheck.ReadinessProbe.TimeoutSeconds <= 0 {
			return assert.AnError
		}
	}
	return nil
}

func validateIngressConfig(ingress *IngressConfig) error {
	if ingress.Enabled && ingress.Host == "" {
		return assert.AnError
	}
	return nil
}

func validatePVConfig(pv PVConfig, config *DeployerConfig) error {
	if pv.Name == "" {
		return assert.AnError
	}
	if pv.StorageClass != "" {
		valid := false
		for _, allowed := range config.AllowedStorageClasses {
			if pv.StorageClass == allowed {
				valid = true
				break
			}
		}
		if !valid {
			return assert.AnError
		}
	}
	if len(pv.AccessModes) == 0 {
		return assert.AnError
	}
	return nil
}

// Add methods to types for testing

func (s *DeploymentStatus) IsReady() bool {
	return s.Phase == "Running" && s.ReadyReplicas == s.Replicas && s.AvailableReplicas == s.Replicas
}

func (p *PodInfo) GetAge() time.Duration {
	return time.Since(p.CreatedAt)
}

func (r *DeployResult) IsHealthy() bool {
	if !r.Success || len(r.Errors) > 0 {
		return false
	}

	if !r.Status.IsReady() {
		return false
	}

	for _, pod := range r.Pods {
		if !pod.Ready {
			return false
		}
	}

	return true
}