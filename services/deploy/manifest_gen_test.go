package deploy

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewManifestGenerator(t *testing.T) {
	config := DefaultDeployerConfig()
	generator := NewManifestGenerator(config)

	assert.NotNil(t, generator)
}

func TestManifestGenerator_GenerateDeployment(t *testing.T) {
	config := DefaultDeployerConfig()
	generator := NewManifestGenerator(config)

	req := &DeployRequest{
		AppName:   "test-app",
		Namespace: "test-namespace",
		ImageName: "nginx",
		ImageTag:  "latest",
		Port:      8080,
		Replicas:  2,
		Environment: map[string]string{
			"ENV":        "production",
			"DEBUG":      "false",
			"LOG_LEVEL":  "info",
		},
		Resources: &ResourceLimits{
			CPU: ResourceSpec{
				Requests: "100m",
				Limits:   "500m",
			},
			Memory: ResourceSpec{
				Requests: "128Mi",
				Limits:   "512Mi",
			},
		},
		HealthCheck: &HealthCheckConfig{
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
		Labels: map[string]string{
			"version": "1.0.0",
			"owner":   "test-team",
		},
	}

	deployment, err := generator.GenerateDeployment(req)
	require.NoError(t, err)
	assert.NotNil(t, deployment)

	// Check basic deployment structure
	assert.Equal(t, "apps/v1", deployment["apiVersion"])
	assert.Equal(t, "Deployment", deployment["kind"])

	// Check metadata
	metadata := deployment["metadata"].(map[string]interface{})
	assert.Equal(t, "test-app", metadata["name"])
	assert.Equal(t, "test-namespace", metadata["namespace"])

	labels := metadata["labels"].(map[string]interface{})
	assert.Equal(t, "test-app", labels["app"])
	assert.Equal(t, "1.0.0", labels["version"])
	assert.Equal(t, "test-team", labels["owner"])

	// Check spec
	spec := deployment["spec"].(map[string]interface{})
	assert.Equal(t, int32(2), spec["replicas"])

	// Check template
	template := spec["template"].(map[string]interface{})
	templateSpec := template["spec"].(map[string]interface{})
	containers := templateSpec["containers"].([]map[string]interface{})
	assert.Len(t, containers, 1)

	container := containers[0]
	assert.Equal(t, "test-app", container["name"])
	assert.Equal(t, "nginx:latest", container["image"])

	// Check ports
	ports := container["ports"].([]map[string]interface{})
	assert.Len(t, ports, 1)
	assert.Equal(t, 8080, ports[0]["containerPort"])

	// Check environment variables
	env := container["env"].([]map[string]interface{})
	assert.Len(t, env, 3)

	// Check resources
	resources := container["resources"].(map[string]interface{})
	requests := resources["requests"].(map[string]interface{})
	limits := resources["limits"].(map[string]interface{})
	assert.Equal(t, "100m", requests["cpu"])
	assert.Equal(t, "128Mi", requests["memory"])
	assert.Equal(t, "500m", limits["cpu"])
	assert.Equal(t, "512Mi", limits["memory"])

	// Check health checks
	assert.Contains(t, container, "livenessProbe")
	assert.Contains(t, container, "readinessProbe")
}

func TestManifestGenerator_GenerateService(t *testing.T) {
	config := DefaultDeployerConfig()
	generator := NewManifestGenerator(config)

	req := &DeployRequest{
		AppName:   "test-app",
		Namespace: "test-namespace",
		ImageName: "nginx",
		ImageTag:  "latest",
		Port:      8080,
	}

	service, err := generator.GenerateService(req)
	require.NoError(t, err)
	assert.NotNil(t, service)

	// Check basic service structure
	assert.Equal(t, "v1", service["apiVersion"])
	assert.Equal(t, "Service", service["kind"])

	// Check metadata
	metadata := service["metadata"].(map[string]interface{})
	assert.Equal(t, "test-app", metadata["name"])
	assert.Equal(t, "test-namespace", metadata["namespace"])

	// Check spec
	spec := service["spec"].(map[string]interface{})
	assert.Equal(t, "ClusterIP", spec["type"])

	selector := spec["selector"].(map[string]interface{})
	assert.Equal(t, "test-app", selector["app"])

	ports := spec["ports"].([]map[string]interface{})
	assert.Len(t, ports, 1)
	assert.Equal(t, 8080, ports[0]["port"])
	assert.Equal(t, 8080, ports[0]["targetPort"])
	assert.Equal(t, "TCP", ports[0]["protocol"])
}

func TestManifestGenerator_GenerateIngress(t *testing.T) {
	config := DefaultDeployerConfig()
	generator := NewManifestGenerator(config)

	req := &DeployRequest{
		AppName:   "test-app",
		Namespace: "test-namespace",
		ImageName: "nginx",
		ImageTag:  "latest",
		Port:      8080,
		Ingress: &IngressConfig{
			Enabled:   true,
			Host:      "test-app.example.com",
			Path:      "/",
			TLS:       true,
			ClassName: "nginx",
			Annotations: map[string]string{
				"cert-manager.io/cluster-issuer": "letsencrypt-prod",
			},
		},
	}

	ingress, err := generator.GenerateIngress(req)
	require.NoError(t, err)
	assert.NotNil(t, ingress)

	// Check basic ingress structure
	assert.Equal(t, "networking.k8s.io/v1", ingress["apiVersion"])
	assert.Equal(t, "Ingress", ingress["kind"])

	// Check metadata
	metadata := ingress["metadata"].(map[string]interface{})
	assert.Equal(t, "test-app", metadata["name"])
	assert.Equal(t, "test-namespace", metadata["namespace"])

	annotations := metadata["annotations"].(map[string]interface{})
	assert.Equal(t, "letsencrypt-prod", annotations["cert-manager.io/cluster-issuer"])

	// Check spec
	spec := ingress["spec"].(map[string]interface{})
	assert.Equal(t, "nginx", spec["ingressClassName"])

	// Check rules
	rules := spec["rules"].([]map[string]interface{})
	assert.Len(t, rules, 1)
	rule := rules[0]
	assert.Equal(t, "test-app.example.com", rule["host"])

	// Check TLS
	tls := spec["tls"].([]map[string]interface{})
	assert.Len(t, tls, 1)
	tlsEntry := tls[0]
	hosts := tlsEntry["hosts"].([]string)
	assert.Contains(t, hosts, "test-app.example.com")
	assert.Equal(t, "test-app-tls", tlsEntry["secretName"])
}

func TestManifestGenerator_GenerateIngress_Disabled(t *testing.T) {
	config := DefaultDeployerConfig()
	generator := NewManifestGenerator(config)

	req := &DeployRequest{
		AppName:   "test-app",
		Namespace: "test-namespace",
		ImageName: "nginx",
		ImageTag:  "latest",
		Port:      8080,
		Ingress: &IngressConfig{
			Enabled: false,
		},
	}

	_, err := generator.GenerateIngress(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "ingress not enabled")
}

func TestManifestGenerator_GenerateConfigMap(t *testing.T) {
	config := DefaultDeployerConfig()
	generator := NewManifestGenerator(config)

	req := &DeployRequest{
		AppName:   "test-app",
		Namespace: "test-namespace",
		Environment: map[string]string{
			"DATABASE_URL": "postgres://localhost:5432/mydb",
			"REDIS_URL":    "redis://localhost:6379",
			"LOG_LEVEL":    "info",
		},
	}

	configMap, err := generator.GenerateConfigMap(req)
	require.NoError(t, err)
	assert.NotNil(t, configMap)

	// Check basic ConfigMap structure
	assert.Equal(t, "v1", configMap["apiVersion"])
	assert.Equal(t, "ConfigMap", configMap["kind"])

	// Check metadata
	metadata := configMap["metadata"].(map[string]interface{})
	assert.Equal(t, "test-app-config", metadata["name"])
	assert.Equal(t, "test-namespace", metadata["namespace"])

	// Check data
	data := configMap["data"].(map[string]string)
	assert.Equal(t, "postgres://localhost:5432/mydb", data["DATABASE_URL"])
	assert.Equal(t, "redis://localhost:6379", data["REDIS_URL"])
	assert.Equal(t, "info", data["LOG_LEVEL"])
}

func TestManifestGenerator_GenerateConfigMap_NoEnvironment(t *testing.T) {
	config := DefaultDeployerConfig()
	generator := NewManifestGenerator(config)

	req := &DeployRequest{
		AppName:   "test-app",
		Namespace: "test-namespace",
	}

	_, err := generator.GenerateConfigMap(req)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "no environment variables")
}

func TestManifestGenerator_GeneratePVC(t *testing.T) {
	config := DefaultDeployerConfig()
	generator := NewManifestGenerator(config)

	req := &DeployRequest{
		AppName:   "test-app",
		Namespace: "test-namespace",
	}

	pvConfig := PVConfig{
		Name:         "data-volume",
		MountPath:    "/data",
		Size:         "10Gi",
		StorageClass: "ssd",
		AccessModes:  []string{"ReadWriteOnce"},
	}

	pvc, err := generator.GeneratePVC(req, pvConfig)
	require.NoError(t, err)
	assert.NotNil(t, pvc)

	// Check basic PVC structure
	assert.Equal(t, "v1", pvc["apiVersion"])
	assert.Equal(t, "PersistentVolumeClaim", pvc["kind"])

	// Check metadata
	metadata := pvc["metadata"].(map[string]interface{})
	assert.Equal(t, "data-volume", metadata["name"])
	assert.Equal(t, "test-namespace", metadata["namespace"])

	// Check spec
	spec := pvc["spec"].(map[string]interface{})
	assert.Equal(t, "ssd", spec["storageClassName"])

	accessModes := spec["accessModes"].([]string)
	assert.Contains(t, accessModes, "ReadWriteOnce")

	resources := spec["resources"].(map[string]interface{})
	requests := resources["requests"].(map[string]interface{})
	assert.Equal(t, "10Gi", requests["storage"])
}

func TestManifestGenerator_GenerateAll(t *testing.T) {
	config := DefaultDeployerConfig()
	generator := NewManifestGenerator(config)

	req := &DeployRequest{
		AppName:   "test-app",
		Namespace: "test-namespace",
		ImageName: "nginx",
		ImageTag:  "latest",
		Port:      8080,
		Replicas:  2,
		Environment: map[string]string{
			"ENV": "production",
		},
		Ingress: &IngressConfig{
			Enabled: true,
			Host:    "test-app.example.com",
			Path:    "/",
			TLS:     true,
		},
		PersistentVolumes: []PVConfig{
			{
				Name:        "data-volume",
				MountPath:   "/data",
				Size:        "5Gi",
				AccessModes: []string{"ReadWriteOnce"},
			},
		},
	}

	manifests, err := generator.GenerateAll(req)
	require.NoError(t, err)
	assert.NotNil(t, manifests)

	// Check that all expected manifests are generated
	assert.Contains(t, manifests, "deployment")
	assert.Contains(t, manifests, "service")
	assert.Contains(t, manifests, "ingress")
	assert.Contains(t, manifests, "configmap")
	assert.Contains(t, manifests, "pvc-0")

	// Verify each manifest type
	deployment := manifests["deployment"]
	assert.Equal(t, "Deployment", deployment["kind"])

	service := manifests["service"]
	assert.Equal(t, "Service", service["kind"])

	ingress := manifests["ingress"]
	assert.Equal(t, "Ingress", ingress["kind"])

	configMap := manifests["configmap"]
	assert.Equal(t, "ConfigMap", configMap["kind"])

	pvc := manifests["pvc-0"]
	assert.Equal(t, "PersistentVolumeClaim", pvc["kind"])
}

func TestManifestGenerator_GenerateDeployment_WithPersistentVolumes(t *testing.T) {
	config := DefaultDeployerConfig()
	generator := NewManifestGenerator(config)

	req := &DeployRequest{
		AppName:   "test-app",
		Namespace: "test-namespace",
		ImageName: "postgres",
		ImageTag:  "13",
		Port:      5432,
		Replicas:  1,
		PersistentVolumes: []PVConfig{
			{
				Name:        "postgres-data",
				MountPath:   "/var/lib/postgresql/data",
				Size:        "20Gi",
				AccessModes: []string{"ReadWriteOnce"},
			},
			{
				Name:        "postgres-config",
				MountPath:   "/etc/postgresql",
				Size:        "1Gi",
				AccessModes: []string{"ReadWriteOnce"},
			},
		},
	}

	deployment, err := generator.GenerateDeployment(req)
	require.NoError(t, err)

	// Check that volumes are added to pod spec
	spec := deployment["spec"].(map[string]interface{})
	template := spec["template"].(map[string]interface{})
	templateSpec := template["spec"].(map[string]interface{})

	volumes := templateSpec["volumes"].([]map[string]interface{})
	assert.Len(t, volumes, 2)

	// Check volume mounts in container
	containers := templateSpec["containers"].([]map[string]interface{})
	container := containers[0]
	volumeMounts := container["volumeMounts"].([]map[string]interface{})
	assert.Len(t, volumeMounts, 2)

	// Verify mount paths
	mountPaths := make([]string, len(volumeMounts))
	for i, mount := range volumeMounts {
		mountPaths[i] = mount["mountPath"].(string)
	}
	assert.Contains(t, mountPaths, "/var/lib/postgresql/data")
	assert.Contains(t, mountPaths, "/etc/postgresql")
}

func TestManifestGenerator_ProbeConfig(t *testing.T) {
	config := DefaultDeployerConfig()
	generator := NewManifestGenerator(config).(*ManifestGeneratorImpl)

	probe := &ProbeConfig{
		Path:                "/health",
		Port:                8080,
		InitialDelaySeconds: 30,
		PeriodSeconds:       10,
		TimeoutSeconds:      5,
		SuccessThreshold:    1,
		FailureThreshold:    3,
	}

	probeConfig := generator.buildProbe(probe)
	assert.NotNil(t, probeConfig)

	httpGet := probeConfig["httpGet"].(map[string]interface{})
	assert.Equal(t, "/health", httpGet["path"])
	assert.Equal(t, 8080, httpGet["port"])

	assert.Equal(t, int32(30), probeConfig["initialDelaySeconds"])
	assert.Equal(t, int32(10), probeConfig["periodSeconds"])
	assert.Equal(t, int32(5), probeConfig["timeoutSeconds"])
	assert.Equal(t, int32(1), probeConfig["successThreshold"])
	assert.Equal(t, int32(3), probeConfig["failureThreshold"])
}

func TestManifestGenerator_Labels(t *testing.T) {
	config := DefaultDeployerConfig()
	generator := NewManifestGenerator(config)

	req := &DeployRequest{
		AppName:   "test-app",
		Namespace: "test-namespace",
		ImageName: "nginx",
		ImageTag:  "latest",
		Port:      8080,
		Labels: map[string]string{
			"version":     "1.2.3",
			"environment": "production",
			"team":        "backend",
		},
	}

	deployment, err := generator.GenerateDeployment(req)
	require.NoError(t, err)

	metadata := deployment["metadata"].(map[string]interface{})
	labels := metadata["labels"].(map[string]interface{})

	// Check standard labels
	assert.Equal(t, "test-app", labels["app"])
	assert.Equal(t, "test-app", labels["app.kubernetes.io/name"])
	assert.Equal(t, "test-app", labels["app.kubernetes.io/instance"])
	assert.Equal(t, "app", labels["app.kubernetes.io/component"])
	assert.Equal(t, "quantumlayer-factory", labels["app.kubernetes.io/managed-by"])

	// Check custom labels
	assert.Equal(t, "1.2.3", labels["version"])
	assert.Equal(t, "production", labels["environment"])
	assert.Equal(t, "backend", labels["team"])
}

// Benchmark tests

func BenchmarkManifestGenerator_GenerateDeployment(b *testing.B) {
	config := DefaultDeployerConfig()
	generator := NewManifestGenerator(config)

	req := &DeployRequest{
		AppName:   "benchmark-app",
		Namespace: "benchmark-ns",
		ImageName: "nginx",
		ImageTag:  "latest",
		Port:      8080,
		Replicas:  3,
		Environment: map[string]string{
			"ENV1": "value1",
			"ENV2": "value2",
			"ENV3": "value3",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.GenerateDeployment(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkManifestGenerator_GenerateAll(b *testing.B) {
	config := DefaultDeployerConfig()
	generator := NewManifestGenerator(config)

	req := &DeployRequest{
		AppName:   "benchmark-app",
		Namespace: "benchmark-ns",
		ImageName: "nginx",
		ImageTag:  "latest",
		Port:      8080,
		Replicas:  2,
		Environment: map[string]string{
			"DATABASE_URL": "postgres://localhost/db",
			"REDIS_URL":    "redis://localhost:6379",
		},
		Ingress: &IngressConfig{
			Enabled: true,
			Host:    "benchmark.example.com",
			Path:    "/",
		},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := generator.GenerateAll(req)
		if err != nil {
			b.Fatal(err)
		}
	}
}