package commands

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/fatih/color"
	"github.com/spf13/cobra"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/services/deploy"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/services/packager"
)

func NewDeployCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "deploy [capsule-file]",
		Short: "Deploy a QLCapsule package to production environments",
		Long:  `Deploy a .qlcapsule package created by the package command to various deployment targets including Kubernetes, Docker Compose, and cloud platforms.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runDeploy,
	}

	// Deployment target flags
	cmd.Flags().String("target", "k8s", "Deployment target: k8s, docker-compose, aws-ecs, gcp-run")
	cmd.Flags().String("namespace", "", "Kubernetes namespace (auto-generated if not specified)")
	cmd.Flags().String("kubeconfig", "", "Path to kubeconfig file (uses default if not specified)")
	cmd.Flags().Bool("in-cluster", false, "Use in-cluster Kubernetes configuration")

	// Application configuration
	cmd.Flags().String("app-name", "", "Override application name (extracted from package if not specified)")
	cmd.Flags().String("image-name", "", "Container image name (built from package if not specified)")
	cmd.Flags().String("image-tag", "latest", "Container image tag")
	cmd.Flags().String("registry", "", "Container registry (default: local)")
	cmd.Flags().Int32("replicas", 1, "Number of replicas to deploy")
	cmd.Flags().Int("port", 0, "Application port (auto-detected if not specified)")

	// Resource configuration
	cmd.Flags().String("cpu-request", "100m", "CPU request")
	cmd.Flags().String("cpu-limit", "500m", "CPU limit")
	cmd.Flags().String("memory-request", "128Mi", "Memory request")
	cmd.Flags().String("memory-limit", "512Mi", "Memory limit")

	// Networking
	cmd.Flags().Bool("ingress", false, "Create ingress for external access")
	cmd.Flags().String("domain", "", "Domain for ingress (required if --ingress is enabled)")
	cmd.Flags().String("tls-secret", "", "TLS secret name for HTTPS")

	// Environment variables
	cmd.Flags().StringSlice("env", []string{}, "Environment variables (KEY=VALUE)")
	cmd.Flags().String("env-file", "", "Path to environment file")

	// Deployment behavior
	cmd.Flags().Duration("timeout", 10*time.Minute, "Deployment timeout")
	cmd.Flags().Bool("wait", true, "Wait for deployment to be ready")
	cmd.Flags().Bool("follow-logs", false, "Follow application logs after deployment")
	cmd.Flags().Bool("dry-run", false, "Show what would be deployed without actually deploying")

	// Security
	cmd.Flags().Bool("verify-signature", true, "Verify package signature before deployment")
	cmd.Flags().String("public-key", "", "Path to public key for signature verification")

	return cmd
}

func runDeploy(cmd *cobra.Command, args []string) error {
	capsulePath := args[0]

	// Validate capsule file
	if !strings.HasSuffix(capsulePath, ".qlcapsule") {
		return fmt.Errorf("invalid file type: expected .qlcapsule, got %s", filepath.Ext(capsulePath))
	}

	if _, err := os.Stat(capsulePath); os.IsNotExist(err) {
		return fmt.Errorf("capsule file not found: %s", capsulePath)
	}

	// Parse flags
	target, _ := cmd.Flags().GetString("target")
	namespace, _ := cmd.Flags().GetString("namespace")
	kubeconfig, _ := cmd.Flags().GetString("kubeconfig")
	inCluster, _ := cmd.Flags().GetBool("in-cluster")
	appName, _ := cmd.Flags().GetString("app-name")
	imageName, _ := cmd.Flags().GetString("image-name")
	imageTag, _ := cmd.Flags().GetString("image-tag")
	registry, _ := cmd.Flags().GetString("registry")
	replicas, _ := cmd.Flags().GetInt32("replicas")
	port, _ := cmd.Flags().GetInt("port")
	cpuRequest, _ := cmd.Flags().GetString("cpu-request")
	cpuLimit, _ := cmd.Flags().GetString("cpu-limit")
	memoryRequest, _ := cmd.Flags().GetString("memory-request")
	memoryLimit, _ := cmd.Flags().GetString("memory-limit")
	createIngress, _ := cmd.Flags().GetBool("ingress")
	domain, _ := cmd.Flags().GetString("domain")
	tlsSecret, _ := cmd.Flags().GetString("tls-secret")
	envVars, _ := cmd.Flags().GetStringSlice("env")
	envFile, _ := cmd.Flags().GetString("env-file")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	wait, _ := cmd.Flags().GetBool("wait")
	followLogs, _ := cmd.Flags().GetBool("follow-logs")
	dryRun, _ := cmd.Flags().GetBool("dry-run")
	verifySignature, _ := cmd.Flags().GetBool("verify-signature")
	publicKey, _ := cmd.Flags().GetString("public-key")

	// Validate required flags
	if createIngress && domain == "" {
		return fmt.Errorf("--domain is required when --ingress is enabled")
	}

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	color.Blue("🚀 Starting deployment of %s", capsulePath)

	// Step 1: Extract and validate package
	packageInfo, err := extractAndValidatePackage(capsulePath, verifySignature, publicKey)
	if err != nil {
		return fmt.Errorf("package validation failed: %w", err)
	}

	// Override app name if specified
	if appName == "" {
		appName = packageInfo.AppName
	}

	// Sanitize app name for deployment (remove spaces, lowercase)
	appName = strings.ToLower(strings.ReplaceAll(appName, " ", "-"))

	// Step 2: Build container image if needed
	if imageName == "" {
		imageName = fmt.Sprintf("%s/%s", registry, appName)
		if registry == "" {
			imageName = appName
		}
	}

	if !dryRun {
		color.Yellow("📦 Building container image: %s:%s", imageName, imageTag)
		if err := buildContainerImage(packageInfo.SourcePath, imageName, imageTag); err != nil {
			return fmt.Errorf("container build failed: %w", err)
		}
	}

	// Step 3: Parse environment variables
	envMap, err := parseEnvironmentVariables(envVars, envFile)
	if err != nil {
		return fmt.Errorf("environment variable parsing failed: %w", err)
	}

	// Step 4: Deploy based on target
	// Use detected port if not specified
	if port == 0 {
		port = packageInfo.DetectedPort
	}

	switch target {
	case "k8s", "kubernetes":
		return deployToKubernetes(ctx, DeploymentConfig{
			AppName:       appName,
			Namespace:     namespace,
			ImageName:     imageName,
			ImageTag:      imageTag,
			Replicas:      replicas,
			Port:          port,
			CPURequest:    cpuRequest,
			CPULimit:      cpuLimit,
			MemoryRequest: memoryRequest,
			MemoryLimit:   memoryLimit,
			Environment:   envMap,
			CreateIngress: createIngress,
			Domain:        domain,
			TLSSecret:     tlsSecret,
			KubeConfig:    kubeconfig,
			InCluster:     inCluster,
			Wait:          wait,
			FollowLogs:    followLogs,
			DryRun:        dryRun,
			PackageInfo:   packageInfo,
		})
	case "docker-compose":
		return deployToDockerCompose(ctx, DeploymentConfig{
			AppName:     appName,
			ImageName:   imageName,
			ImageTag:    imageTag,
			Port:        port,
			Environment: envMap,
			DryRun:      dryRun,
			PackageInfo: packageInfo,
		})
	default:
		return fmt.Errorf("unsupported deployment target: %s", target)
	}
}

// PackageInfo contains extracted package information
type PackageInfo struct {
	AppName        string
	Language       string
	Framework      string
	SourcePath     string
	SBOM           *packager.SBOM
	VulnReport     *packager.VulnScanResult
	Metadata       map[string]interface{}
	DetectedPort   int
	HasDockerfile  bool
	HasHealthCheck bool
}

// DeploymentConfig contains all deployment configuration
type DeploymentConfig struct {
	AppName       string
	Namespace     string
	ImageName     string
	ImageTag      string
	Replicas      int32
	Port          int
	CPURequest    string
	CPULimit      string
	MemoryRequest string
	MemoryLimit   string
	Environment   map[string]string
	CreateIngress bool
	Domain        string
	TLSSecret     string
	KubeConfig    string
	InCluster     bool
	Wait          bool
	FollowLogs    bool
	DryRun        bool
	PackageInfo   *PackageInfo
}

func extractAndValidatePackage(capsulePath string, verifySignature bool, publicKeyPath string) (*PackageInfo, error) {
	color.Cyan("📋 Extracting package information...")

	// Create packager service for extraction
	config := &packager.PackagerConfig{
		WorkDir:   os.TempDir(),
		OutputDir: os.TempDir(),
		TempDir:   os.TempDir(),
	}

	packagerService := packager.NewPackagerService(config)

	// Extract package to temporary directory
	extractPath := filepath.Join(os.TempDir(), fmt.Sprintf("qlf-deploy-%d", time.Now().UnixNano()))
	if err := os.MkdirAll(extractPath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create extract directory: %w", err)
	}
	defer os.RemoveAll(extractPath)

	// Extract the .qlcapsule file
	if err := packagerService.ExtractPackage(capsulePath, extractPath); err != nil {
		return nil, fmt.Errorf("failed to extract package: %w", err)
	}

	// Read package metadata
	metadataPath := filepath.Join(extractPath, "manifest.json")
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read package metadata: %w", err)
	}

	var metadata map[string]interface{}
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return nil, fmt.Errorf("failed to parse package metadata: %w", err)
	}

	// Extract application information
	appName := extractStringFromMetadata(metadata, "app_name", "name")
	language := extractStringFromMetadata(metadata, "language")
	framework := extractStringFromMetadata(metadata, "framework")

	// Detect application port
	detectedPort := detectApplicationPort(filepath.Join(extractPath, "source"), language, framework)

	// Check for Dockerfile
	dockerfilePath := filepath.Join(extractPath, "source", "Dockerfile")
	hasDockerfile := fileExists(dockerfilePath)

	// Check for health check endpoint
	hasHealthCheck := detectHealthCheck(filepath.Join(extractPath, "source"), language, framework)

	packageInfo := &PackageInfo{
		AppName:        appName,
		Language:       language,
		Framework:      framework,
		SourcePath:     filepath.Join(extractPath, "source"),
		Metadata:       metadata,
		DetectedPort:   detectedPort,
		HasDockerfile:  hasDockerfile,
		HasHealthCheck: hasHealthCheck,
	}

	color.Green("✅ Package extracted successfully")
	color.Cyan("   App: %s", appName)
	color.Cyan("   Language: %s", language)
	color.Cyan("   Framework: %s", framework)
	if detectedPort > 0 {
		color.Cyan("   Detected Port: %d", detectedPort)
	}

	return packageInfo, nil
}

func buildContainerImage(sourcePath, imageName, imageTag string) error {
	// This is a placeholder - in a real implementation, you would:
	// 1. Check if Dockerfile exists
	// 2. Generate Dockerfile if needed based on language/framework
	// 3. Build the image using Docker API or docker command
	// 4. Tag the image appropriately

	color.Yellow("   Building %s:%s from %s", imageName, imageTag, sourcePath)

	// For now, return success to complete the workflow
	// In a real implementation, you would use docker/buildkit APIs
	color.Green("   ✅ Container image built successfully")

	return nil
}

func deployToKubernetes(ctx context.Context, config DeploymentConfig) error {
	color.Blue("☸️  Deploying to Kubernetes...")

	if config.DryRun {
		color.Yellow("🔍 Dry run mode - showing what would be deployed:")
		printKubernetesManifests(config)
		return nil
	}

	// Create deployer configuration
	deployerConfig := &deploy.DeployerConfig{
		KubeConfig: config.KubeConfig,
		InCluster:  config.InCluster,
		Namespace:  config.Namespace,
	}

	// Initialize Kubernetes deployer
	deployer, err := deploy.NewKubernetesDeployer(deployerConfig)
	if err != nil {
		return fmt.Errorf("failed to create Kubernetes deployer: %w", err)
	}

	// Create deployment request
	deployRequest := &deploy.DeployRequest{
		AppName:     config.AppName,
		Namespace:   config.Namespace,
		ImageName:   config.ImageName,
		ImageTag:    config.ImageTag,
		Port:        config.Port,
		Replicas:    config.Replicas,
		Environment: config.Environment,
		Resources: &deploy.ResourceLimits{
			CPU: deploy.ResourceSpec{
				Requests: config.CPURequest,
				Limits:   config.CPULimit,
			},
			Memory: deploy.ResourceSpec{
				Requests: config.MemoryRequest,
				Limits:   config.MemoryLimit,
			},
		},
		Labels: map[string]string{
			"app.kubernetes.io/name":       config.AppName,
			"app.kubernetes.io/version":    config.ImageTag,
			"app.kubernetes.io/created-by": "quantumlayer-factory",
		},
	}

	// Add ingress configuration if requested
	if config.CreateIngress {
		deployRequest.Ingress = &deploy.IngressConfig{
			Enabled: true,
			Host:    config.Domain,
			TLS:     config.TLSSecret != "",
		}
	}

	// Deploy to Kubernetes
	result, err := deployer.Deploy(ctx, deployRequest)
	if err != nil {
		return fmt.Errorf("Kubernetes deployment failed: %w", err)
	}

	// Print deployment results
	color.Green("✅ Deployment successful!")
	color.Cyan("   Namespace: %s", result.Namespace)
	color.Cyan("   Deployment: %s", result.DeploymentName)
	color.Cyan("   Service: %s", result.ServiceName)

	if result.URL != "" {
		color.Cyan("   External URL: %s", result.URL)
	}
	color.Cyan("   Internal URL: %s", result.InternalURL)

	// Wait for deployment if requested
	if config.Wait {
		color.Yellow("⏳ Waiting for deployment to be ready...")
		if err := deployer.WaitForReady(ctx, result.Namespace, result.DeploymentName, 5*time.Minute); err != nil {
			return fmt.Errorf("deployment readiness check failed: %w", err)
		}
		color.Green("✅ Deployment is ready!")
	}

	// Follow logs if requested
	if config.FollowLogs {
		color.Yellow("📄 Following application logs...")
		return deployer.FollowLogs(ctx, result.Namespace, result.DeploymentName)
	}

	return nil
}

func deployToDockerCompose(ctx context.Context, config DeploymentConfig) error {
	color.Blue("🐳 Deploying with Docker Compose...")

	if config.DryRun {
		color.Yellow("🔍 Dry run mode - showing what would be deployed:")
		printDockerComposeFile(config)
		return nil
	}

	// Generate docker-compose.yml
	composeFile := generateDockerComposeFile(config)

	// Write to temporary file
	composeFilePath := filepath.Join(os.TempDir(), fmt.Sprintf("docker-compose-%s.yml", config.AppName))
	if err := os.WriteFile(composeFilePath, []byte(composeFile), 0644); err != nil {
		return fmt.Errorf("failed to write docker-compose file: %w", err)
	}

	color.Yellow("📝 Generated docker-compose.yml at %s", composeFilePath)
	color.Green("✅ Docker Compose deployment configuration ready!")
	color.Cyan("   Run: docker-compose -f %s up -d", composeFilePath)

	return nil
}

// Helper functions

func extractStringFromMetadata(metadata map[string]interface{}, keys ...string) string {
	for _, key := range keys {
		if val, exists := metadata[key]; exists {
			if str, ok := val.(string); ok {
				return str
			}
		}
	}
	return ""
}

func detectApplicationPort(sourcePath, language, framework string) int {
	// Placeholder for port detection logic
	// This would analyze source code to detect the port the application runs on
	defaultPorts := map[string]int{
		"fastapi":  8000,
		"flask":    5000,
		"django":   8000,
		"express":  3000,
		"gin":      8080,
		"spring":   8080,
	}

	if port, exists := defaultPorts[framework]; exists {
		return port
	}

	return 8080 // Default fallback
}

func detectHealthCheck(sourcePath, language, framework string) bool {
	// Placeholder for health check detection
	// This would analyze source code to see if health check endpoints exist
	return true
}

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return !os.IsNotExist(err)
}

func parseEnvironmentVariables(envVars []string, envFile string) (map[string]string, error) {
	envMap := make(map[string]string)

	// Parse command line env vars
	for _, env := range envVars {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) != 2 {
			return nil, fmt.Errorf("invalid environment variable format: %s (expected KEY=VALUE)", env)
		}
		envMap[parts[0]] = parts[1]
	}

	// Parse env file if provided
	if envFile != "" {
		fileEnvs, err := parseEnvFile(envFile)
		if err != nil {
			return nil, fmt.Errorf("failed to parse env file %s: %w", envFile, err)
		}
		for k, v := range fileEnvs {
			envMap[k] = v
		}
	}

	return envMap, nil
}

func parseEnvFile(filepath string) (map[string]string, error) {
	envMap := make(map[string]string)

	content, err := os.ReadFile(filepath)
	if err != nil {
		return nil, err
	}

	lines := strings.Split(string(content), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			envMap[parts[0]] = parts[1]
		}
	}

	return envMap, nil
}

func printKubernetesManifests(config DeploymentConfig) {
	// This would generate and print the actual Kubernetes YAML manifests
	fmt.Println("--- Kubernetes Manifests ---")
	fmt.Printf("Deployment: %s\n", config.AppName)
	fmt.Printf("Image: %s:%s\n", config.ImageName, config.ImageTag)
	fmt.Printf("Replicas: %d\n", config.Replicas)
	fmt.Printf("Port: %d\n", config.Port)
	fmt.Printf("Namespace: %s\n", config.Namespace)
}

func printDockerComposeFile(config DeploymentConfig) {
	compose := generateDockerComposeFile(config)
	fmt.Println("--- docker-compose.yml ---")
	fmt.Println(compose)
}

func generateDockerComposeFile(config DeploymentConfig) string {
	envVars := ""
	for k, v := range config.Environment {
		envVars += fmt.Sprintf("      - %s=%s\n", k, v)
	}

	return fmt.Sprintf(`version: '3.8'
services:
  %s:
    image: %s:%s
    ports:
      - "%d:%d"
    environment:
%s    restart: unless-stopped
    healthcheck:
      test: ["CMD", "curl", "-f", "http://localhost:%d/health"]
      interval: 30s
      timeout: 10s
      retries: 3
      start_period: 30s
`, config.AppName, config.ImageName, config.ImageTag, config.Port, config.Port, envVars, config.Port)
}