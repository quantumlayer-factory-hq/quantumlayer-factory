package packager

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewPackagerService(t *testing.T) {
	config := DefaultPackagerConfig()
	service := NewPackagerService(config)

	assert.NotNil(t, service)
	assert.Equal(t, config, service.config)
}

func TestDefaultPackagerConfig(t *testing.T) {
	config := DefaultPackagerConfig()

	assert.Equal(t, "/tmp/qlf-packager", config.WorkDir)
	assert.Equal(t, "./packages", config.OutputDir)
	assert.Equal(t, "gzip", config.CompressionType)
	assert.Equal(t, 6, config.CompressionLevel)
	assert.True(t, config.SBOMEnabled)
	assert.True(t, config.VulnScanEnabled)
	assert.False(t, config.SigningEnabled)
	assert.Equal(t, "syft", config.SBOMTool)
	assert.Equal(t, "trivy", config.VulnScanner)
}

func TestPackageRequest_Validation(t *testing.T) {
	service := NewPackagerService(nil)

	tests := []struct {
		name    string
		req     *PackageRequest
		wantErr bool
	}{
		{
			name: "valid request",
			req: &PackageRequest{
				Name:       "test-package",
				Version:    "1.0.0",
				SourcePath: "/tmp",
				Language:   "go",
				Framework:  "gin",
				Runtime: RuntimeSpec{
					Platform: []string{"linux/amd64"},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			req: &PackageRequest{
				Version:    "1.0.0",
				SourcePath: "/tmp",
				Language:   "go",
			},
			wantErr: true,
		},
		{
			name: "missing version",
			req: &PackageRequest{
				Name:       "test-package",
				SourcePath: "/tmp",
				Language:   "go",
			},
			wantErr: true,
		},
		{
			name: "missing source path",
			req: &PackageRequest{
				Name:     "test-package",
				Version:  "1.0.0",
				Language: "go",
			},
			wantErr: true,
		},
		{
			name: "missing language",
			req: &PackageRequest{
				Name:       "test-package",
				Version:    "1.0.0",
				SourcePath: "/tmp",
			},
			wantErr: true,
		},
		{
			name: "non-existent source path",
			req: &PackageRequest{
				Name:       "test-package",
				Version:    "1.0.0",
				SourcePath: "/non/existent/path",
				Language:   "go",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := service.validateRequest(tt.req)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestCapsuleManifest_Validation(t *testing.T) {
	tests := []struct {
		name     string
		manifest *CapsuleManifest
		wantErr  bool
	}{
		{
			name: "valid manifest",
			manifest: &CapsuleManifest{
				Name:     "test-package",
				Version:  "1.0.0",
				Language: "go",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			manifest: &CapsuleManifest{
				Version:  "1.0.0",
				Language: "go",
			},
			wantErr: true,
		},
		{
			name: "missing version",
			manifest: &CapsuleManifest{
				Name:     "test-package",
				Language: "go",
			},
			wantErr: true,
		},
		{
			name: "missing language",
			manifest: &CapsuleManifest{
				Name:    "test-package",
				Version: "1.0.0",
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.manifest.Validate()
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestVulnScanResult_IsVulnerable(t *testing.T) {
	tests := []struct {
		name      string
		result    *VulnScanResult
		threshold string
		expected  bool
	}{
		{
			name: "critical threshold with critical vuln",
			result: &VulnScanResult{
				Critical: 1,
				High:     0,
				Medium:   0,
				Low:      0,
			},
			threshold: "critical",
			expected:  true,
		},
		{
			name: "critical threshold without critical vuln",
			result: &VulnScanResult{
				Critical: 0,
				High:     1,
				Medium:   0,
				Low:      0,
			},
			threshold: "critical",
			expected:  false,
		},
		{
			name: "high threshold with high vuln",
			result: &VulnScanResult{
				Critical: 0,
				High:     1,
				Medium:   0,
				Low:      0,
			},
			threshold: "high",
			expected:  true,
		},
		{
			name: "medium threshold with medium vuln",
			result: &VulnScanResult{
				Critical: 0,
				High:     0,
				Medium:   1,
				Low:      0,
			},
			threshold: "medium",
			expected:  true,
		},
		{
			name: "low threshold with low vuln",
			result: &VulnScanResult{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      1,
			},
			threshold: "low",
			expected:  true,
		},
		{
			name: "no vulnerabilities",
			result: &VulnScanResult{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      0,
			},
			threshold: "critical",
			expected:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.result.IsVulnerable(tt.threshold)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVulnScanResult_GetSeverityCount(t *testing.T) {
	result := &VulnScanResult{
		Critical: 2,
		High:     3,
		Medium:   5,
		Low:      7,
	}

	assert.Equal(t, 2, result.GetSeverityCount("critical"))
	assert.Equal(t, 3, result.GetSeverityCount("high"))
	assert.Equal(t, 5, result.GetSeverityCount("medium"))
	assert.Equal(t, 7, result.GetSeverityCount("low"))
	assert.Equal(t, 0, result.GetSeverityCount("unknown"))
}

func TestPackagerService_CreateManifest(t *testing.T) {
	service := NewPackagerService(nil)

	req := &PackageRequest{
		Name:        "test-package",
		Version:     "1.0.0",
		Description: "Test package",
		Author:      "Test Author",
		License:     "MIT",
		Language:    "go",
		Framework:   "gin",
		Tags:        []string{"web", "api"},
		Labels: map[string]string{
			"env": "test",
		},
		Annotations: map[string]string{
			"note": "test package",
		},
	}

	manifest, err := service.createManifest(req)
	require.NoError(t, err)
	assert.NotNil(t, manifest)

	assert.Equal(t, "test-package", manifest.Name)
	assert.Equal(t, "1.0.0", manifest.Version)
	assert.Equal(t, "Test package", manifest.Description)
	assert.Equal(t, "Test Author", manifest.Author)
	assert.Equal(t, "MIT", manifest.License)
	assert.Equal(t, "go", manifest.Language)
	assert.Equal(t, "gin", manifest.Framework)
	assert.Equal(t, []string{"web", "api"}, manifest.Tags)
	assert.NotEmpty(t, manifest.CreatedAt)
	assert.NotNil(t, manifest.Metadata)
}

func TestPackagerService_GenerateSBOM(t *testing.T) {
	service := NewPackagerService(nil)

	sbom, err := service.generateSBOM("/tmp")
	require.NoError(t, err)
	assert.NotNil(t, sbom)

	assert.Equal(t, "spdx", sbom.Format)
	assert.Equal(t, "syft", sbom.Tool)
	assert.NotEmpty(t, sbom.GeneratedAt)
	assert.NotEmpty(t, sbom.Components)
}

func TestPackagerService_ScanVulnerabilities(t *testing.T) {
	service := NewPackagerService(nil)

	result, err := service.scanVulnerabilities("/tmp")
	require.NoError(t, err)
	assert.NotNil(t, result)

	assert.Equal(t, "trivy", result.Scanner)
	assert.NotEmpty(t, result.ScanTime)
	assert.Equal(t, 0, result.TotalVulns)
}

func TestPackagerService_CreateAttestation(t *testing.T) {
	service := NewPackagerService(nil)

	req := &PackageRequest{
		Name:       "test-package",
		Version:    "1.0.0",
		SourcePath: "/tmp/test",
	}

	attestation := service.createAttestation(req)
	assert.NotNil(t, attestation)

	assert.Equal(t, "quantumlayer-factory", attestation.Builder)
	assert.Equal(t, "linux/amd64", attestation.BuildPlatform)
	assert.True(t, attestation.Reproducible)
	assert.NotEmpty(t, attestation.BuildTimestamp)
	assert.NotEmpty(t, attestation.BuildHash)
}

func TestPackagerService_DetectArtifactType(t *testing.T) {
	service := NewPackagerService(nil)

	tests := []struct {
		path     string
		expected string
	}{
		{"main.go", "source"},
		{"app.py", "source"},
		{"index.js", "source"},
		{"config.json", "config"},
		{"app.yaml", "config"},
		{"README.md", "documentation"},
		{"app.exe", "binary"},
		{"lib.so", "binary"},
		{"data.txt", "documentation"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := service.detectArtifactType(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestPackagerService_WorkspaceManagement(t *testing.T) {
	service := NewPackagerService(nil)

	// Create workspace
	workDir, err := service.createWorkspace("test-package")
	require.NoError(t, err)
	assert.NotEmpty(t, workDir)
	assert.Contains(t, workDir, "test-package")

	// Check if directory exists
	_, err = os.Stat(workDir)
	assert.NoError(t, err)

	// Cleanup workspace
	err = service.cleanupWorkspace(workDir)
	assert.NoError(t, err)

	// Check if directory is removed
	_, err = os.Stat(workDir)
	assert.True(t, os.IsNotExist(err))
}

func TestDeliveryService_DefaultConfig(t *testing.T) {
	config := DefaultDeliveryConfig()

	assert.Equal(t, "direct", config.DefaultChannel)
	assert.Equal(t, 3, config.RetryAttempts)
	assert.Equal(t, time.Second*5, config.RetryDelay)
	assert.Equal(t, time.Minute*10, config.Timeout)
	assert.True(t, config.VerifyChecksums)
	assert.True(t, config.EnableCompression)
}

func TestDeliveryService_ChannelManagement(t *testing.T) {
	service := NewDeliveryService(nil)

	// Register channel
	channel := DeliveryChannel{
		Name:     "test-registry",
		Type:     "registry",
		Endpoint: "https://registry.example.com",
		Enabled:  true,
	}

	service.RegisterChannel("test-registry", channel)

	// Get channel
	retrievedChannel, exists := service.GetChannel("test-registry")
	assert.True(t, exists)
	assert.Equal(t, channel, retrievedChannel)

	// Get non-existent channel
	_, exists = service.GetChannel("non-existent")
	assert.False(t, exists)
}

func TestDeliveryService_ValidatePackageName(t *testing.T) {
	service := NewDeliveryService(nil)

	tests := []struct {
		name     string
		expected bool
	}{
		{"test-package.qlcapsule", true},
		{"my-app-v1.0.0.qlcapsule", true},
		{"", false},
		{"test-package", false},
		{"../test-package.qlcapsule", false},
		{"test/package.qlcapsule", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := service.isValidPackageName(tt.name)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDocsGenerator_DefaultConfig(t *testing.T) {
	config := DefaultDocsConfig()

	assert.Equal(t, "./docs", config.OutputDir)
	assert.Equal(t, "./templates", config.TemplateDir)
	assert.Equal(t, "markdown", config.Format)
	assert.True(t, config.IncludeSBOM)
	assert.True(t, config.IncludeVulns)
	assert.True(t, config.IncludeAPI)
}

func TestDocsGenerator_GetDefaultSections(t *testing.T) {
	generator := NewDocsGenerator(nil)

	sections := generator.getDefaultSections()
	assert.Contains(t, sections, "overview")
	assert.Contains(t, sections, "installation")
	assert.Contains(t, sections, "configuration")
	assert.Contains(t, sections, "deployment")
	assert.Contains(t, sections, "monitoring")
	assert.Contains(t, sections, "api")
	assert.Contains(t, sections, "sbom")
	assert.Contains(t, sections, "security")
}

func TestCreateDeliveryChannels(t *testing.T) {
	// Test registry channel creation
	registryConfig := RegistryConfig{
		URL:       "https://registry.example.com",
		Username:  "user",
		Password:  "pass",
		Namespace: "myorg",
		TLS:       true,
	}

	registryChannel := CreateRegistryChannel("my-registry", registryConfig)
	assert.Equal(t, "my-registry", registryChannel.Name)
	assert.Equal(t, "registry", registryChannel.Type)
	assert.Equal(t, "https://registry.example.com", registryChannel.Endpoint)
	assert.True(t, registryChannel.Enabled)

	// Test CDN channel creation
	cdnConfig := CDNConfig{
		URL:        "https://cdn.example.com",
		APIKey:     "api-key",
		BucketName: "packages",
		Region:     "us-east-1",
		PublicRead: true,
		CacheTTL:   3600,
	}

	cdnChannel := CreateCDNChannel("my-cdn", cdnConfig)
	assert.Equal(t, "my-cdn", cdnChannel.Name)
	assert.Equal(t, "cdn", cdnChannel.Type)
	assert.Equal(t, "https://cdn.example.com", cdnChannel.Endpoint)

	// Test direct channel creation
	directConfig := DirectConfig{
		BaseURL:     "https://packages.example.com",
		StoragePath: "/var/packages",
		ServeHTTP:   true,
		HTTPPort:    8080,
	}

	directChannel := CreateDirectChannel("my-direct", directConfig)
	assert.Equal(t, "my-direct", directChannel.Name)
	assert.Equal(t, "direct", directChannel.Type)
	assert.Equal(t, "https://packages.example.com", directChannel.Endpoint)

	// Test package manager channel creation
	pmConfig := PackageManagerConfig{
		Type:       "npm",
		Repository: "https://registry.npmjs.org",
		APIKey:     "npm-key",
		Namespace:  "myorg",
	}

	pmChannel := CreatePackageManagerChannel("my-npm", pmConfig)
	assert.Equal(t, "my-npm", pmChannel.Name)
	assert.Equal(t, "package_manager", pmChannel.Type)
	assert.Equal(t, "https://registry.npmjs.org", pmChannel.Endpoint)
}

// Benchmark tests

func BenchmarkManifestValidation(b *testing.B) {
	manifest := &CapsuleManifest{
		Name:     "test-package",
		Version:  "1.0.0",
		Language: "go",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = manifest.Validate()
	}
}

func BenchmarkVulnScanResultCheck(b *testing.B) {
	result := &VulnScanResult{
		Critical: 2,
		High:     3,
		Medium:   5,
		Low:      7,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.IsVulnerable("high")
	}
}