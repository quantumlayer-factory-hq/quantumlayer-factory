package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/services/packager"
	"github.com/spf13/cobra"
)

type PackageFlags struct {
	// Basic package information
	name        string
	version     string
	description string
	author      string
	license     string

	// Source and build
	sourcePath     string
	buildArtifacts []string
	manifests      []string
	documentation  []string

	// Language and framework
	language  string
	framework string

	// Runtime configuration
	platforms    []string
	cpuLimit     string
	memoryLimit  string
	storageLimit string

	// Security options
	generateSBOM bool
	scanVulns    bool
	signPackage  bool
	signingKey   string

	// Compression
	compression      string
	compressionLevel int

	// Output options
	outputDir  string
	outputPath string

	// Publishing
	publish  []string
	tags     []string
	labels   []string
	public   bool

	// Documentation
	generateDocs bool
	docsFormat   string

	// Metadata
	repository string
	homepage   string
}

// NewPackageCmd creates the package command
func NewPackageCmd() *cobra.Command {
	var packageFlags PackageFlags

	packageCmd := &cobra.Command{
		Use:   "package",
		Short: "Package applications into .qlcapsule format",
		Long: `Package applications into secure .qlcapsule format with SBOM, attestation, and delivery channels.

The package command creates a compressed archive containing:
- Source code and build artifacts
- Software Bill of Materials (SBOM)
- Security attestation and digital signatures
- Deployment manifests and configuration
- Generated documentation

Examples:
  # Package a Go application
  qlf package my-app --source ./src --language go --framework gin

  # Package with custom configuration
  qlf package my-app --source ./src --version 1.2.0 --author "John Doe" --license MIT

  # Package and publish to multiple channels
  qlf package my-app --source ./src --publish registry,cdn

  # Package with security scanning
  qlf package my-app --source ./src --scan-vulns --sign --key ./private.pem`,
		RunE: func(cmd *cobra.Command, args []string) error {
			return runPackage(cmd, args, &packageFlags)
		},
	}

	// Package command initialization

	// Basic package information
	packageCmd.Flags().StringVarP(&packageFlags.name, "name", "n", "", "Package name")
	packageCmd.Flags().StringVarP(&packageFlags.version, "version", "v", "1.0.0", "Package version")
	packageCmd.Flags().StringVarP(&packageFlags.description, "description", "d", "", "Package description")
	packageCmd.Flags().StringVar(&packageFlags.author, "author", "", "Package author")
	packageCmd.Flags().StringVar(&packageFlags.license, "license", "", "Package license")

	// Source and build
	packageCmd.Flags().StringVarP(&packageFlags.sourcePath, "source", "s", ".", "Source code path")
	packageCmd.Flags().StringSliceVar(&packageFlags.buildArtifacts, "artifacts", []string{}, "Build artifacts to include")
	packageCmd.Flags().StringSliceVar(&packageFlags.manifests, "manifests", []string{}, "Deployment manifests to include")
	packageCmd.Flags().StringSliceVar(&packageFlags.documentation, "docs", []string{}, "Documentation files to include")

	// Language and framework
	packageCmd.Flags().StringVarP(&packageFlags.language, "language", "l", "", "Programming language")
	packageCmd.Flags().StringVarP(&packageFlags.framework, "framework", "f", "", "Framework or library")

	// Runtime configuration
	packageCmd.Flags().StringSliceVar(&packageFlags.platforms, "platforms", []string{"linux/amd64"}, "Target platforms")
	packageCmd.Flags().StringVar(&packageFlags.cpuLimit, "cpu", "", "CPU resource limit (e.g., 100m, 1.5)")
	packageCmd.Flags().StringVar(&packageFlags.memoryLimit, "memory", "", "Memory resource limit (e.g., 128Mi, 1Gi)")
	packageCmd.Flags().StringVar(&packageFlags.storageLimit, "storage", "", "Storage resource limit (e.g., 1Gi, 10Gi)")

	// Security options
	packageCmd.Flags().BoolVar(&packageFlags.generateSBOM, "sbom", true, "Generate Software Bill of Materials")
	packageCmd.Flags().BoolVar(&packageFlags.scanVulns, "scan-vulns", true, "Scan for vulnerabilities")
	packageCmd.Flags().BoolVar(&packageFlags.signPackage, "sign", false, "Sign the package")
	packageCmd.Flags().StringVar(&packageFlags.signingKey, "key", "", "Path to signing key")

	// Compression
	packageCmd.Flags().StringVar(&packageFlags.compression, "compression", "gzip", "Compression type (gzip, lz4, zstd)")
	packageCmd.Flags().IntVar(&packageFlags.compressionLevel, "compression-level", 6, "Compression level (1-9)")

	// Output options
	packageCmd.Flags().StringVar(&packageFlags.outputDir, "output-dir", "./packages", "Output directory")
	packageCmd.Flags().StringVarP(&packageFlags.outputPath, "output", "o", "", "Output file path")

	// Publishing
	packageCmd.Flags().StringSliceVar(&packageFlags.publish, "publish", []string{}, "Delivery channels to publish to")
	packageCmd.Flags().StringSliceVar(&packageFlags.tags, "tags", []string{}, "Package tags")
	packageCmd.Flags().StringSliceVar(&packageFlags.labels, "labels", []string{}, "Package labels (key=value)")
	packageCmd.Flags().BoolVar(&packageFlags.public, "public", false, "Make package public")

	// Documentation
	packageCmd.Flags().BoolVar(&packageFlags.generateDocs, "generate-docs", true, "Generate documentation")
	packageCmd.Flags().StringVar(&packageFlags.docsFormat, "docs-format", "markdown", "Documentation format (markdown, html)")

	// Metadata
	packageCmd.Flags().StringVar(&packageFlags.repository, "repository", "", "Source repository URL")
	packageCmd.Flags().StringVar(&packageFlags.homepage, "homepage", "", "Project homepage URL")

	// Required flags
	packageCmd.MarkFlagRequired("language")

	return packageCmd
}

func runPackage(cmd *cobra.Command, args []string, packageFlags *PackageFlags) error {
	ctx := context.Background()

	// Determine package name
	name := packageFlags.name
	if name == "" {
		if len(args) > 0 {
			name = args[0]
		} else {
			name = filepath.Base(packageFlags.sourcePath)
		}
	}

	// Validate inputs
	if err := validatePackageInputs(name, packageFlags); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Create packager service
	config := createPackagerConfig(packageFlags)
	packagerService := packager.NewPackagerService(config)

	// Create package request
	request, err := createPackageRequest(name, packageFlags)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	fmt.Printf("Creating package: %s v%s\n", name, packageFlags.version)
	fmt.Printf("Source: %s\n", packageFlags.sourcePath)
	fmt.Printf("Language: %s\n", packageFlags.language)
	if packageFlags.framework != "" {
		fmt.Printf("Framework: %s\n", packageFlags.framework)
	}

	// Create package
	result, err := packagerService.CreatePackage(ctx, request)
	if err != nil {
		return fmt.Errorf("failed to create package: %w", err)
	}

	// Display results
	fmt.Printf("\n✅ Package created successfully!\n")
	fmt.Printf("📦 Package: %s\n", result.CapsulePath)
	fmt.Printf("📏 Size: %s\n", formatSize(result.Size))
	fmt.Printf("🔍 Hash: %s\n", result.Hash)
	fmt.Printf("⏱️  Build time: %s\n", result.BuildTime.Round(time.Millisecond))

	if result.Compressed {
		fmt.Printf("🗜️  Compression ratio: %.2f%%\n", result.CompressionRatio*100)
	}

	if result.VulnScanResult != nil {
		fmt.Printf("\n🛡️  Security scan results:\n")
		fmt.Printf("   Critical: %d\n", result.VulnScanResult.Critical)
		fmt.Printf("   High: %d\n", result.VulnScanResult.High)
		fmt.Printf("   Medium: %d\n", result.VulnScanResult.Medium)
		fmt.Printf("   Low: %d\n", result.VulnScanResult.Low)
		fmt.Printf("   Fixable: %d\n", result.VulnScanResult.Fixable)
	}

	// Generate documentation if requested
	if packageFlags.generateDocs {
		if err := generateDocumentation(result, packageFlags); err != nil {
			fmt.Printf("⚠️  Warning: Failed to generate documentation: %v\n", err)
		}
	}

	// Publish to channels if requested
	if len(packageFlags.publish) > 0 {
		if err := publishPackage(ctx, result.CapsulePath, packageFlags); err != nil {
			return fmt.Errorf("failed to publish package: %w", err)
		}
	}

	fmt.Printf("\n🎉 Package %s ready!\n", name)
	return nil
}

func validatePackageInputs(name string, packageFlags *PackageFlags) error {
	if name == "" {
		return fmt.Errorf("package name is required")
	}

	if packageFlags.sourcePath == "" {
		return fmt.Errorf("source path is required")
	}

	// Check if source path exists
	if _, err := os.Stat(packageFlags.sourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", packageFlags.sourcePath)
	}

	if packageFlags.language == "" {
		return fmt.Errorf("language is required")
	}

	// Validate signing configuration
	if packageFlags.signPackage && packageFlags.signingKey == "" {
		return fmt.Errorf("signing key is required when signing is enabled")
	}

	if packageFlags.signingKey != "" {
		if _, err := os.Stat(packageFlags.signingKey); os.IsNotExist(err) {
			return fmt.Errorf("signing key file does not exist: %s", packageFlags.signingKey)
		}
	}

	return nil
}

func createPackagerConfig(packageFlags *PackageFlags) *packager.PackagerConfig {
	config := packager.DefaultPackagerConfig()

	config.OutputDir = packageFlags.outputDir
	config.CompressionType = packageFlags.compression
	config.CompressionLevel = packageFlags.compressionLevel
	config.SBOMEnabled = packageFlags.generateSBOM
	config.VulnScanEnabled = packageFlags.scanVulns
	config.SigningEnabled = packageFlags.signPackage

	if packageFlags.signingKey != "" {
		config.PrivateKeyPath = packageFlags.signingKey
	}

	return config
}

func createPackageRequest(name string, packageFlags *PackageFlags) (*packager.PackageRequest, error) {
	// Parse labels
	labels := make(map[string]string)
	for _, label := range packageFlags.labels {
		parts := strings.SplitN(label, "=", 2)
		if len(parts) == 2 {
			labels[parts[0]] = parts[1]
		}
	}

	// Create runtime spec
	runtime := packager.RuntimeSpec{
		Platform: packageFlags.platforms,
		Resources: packager.ResourceLimits{
			CPU:     packageFlags.cpuLimit,
			Memory:  packageFlags.memoryLimit,
			Storage: packageFlags.storageLimit,
		},
	}

	request := &packager.PackageRequest{
		Name:           name,
		Version:        packageFlags.version,
		Description:    packageFlags.description,
		Author:         packageFlags.author,
		License:        packageFlags.license,
		SourcePath:     packageFlags.sourcePath,
		BuildArtifacts: packageFlags.buildArtifacts,
		Manifests:      packageFlags.manifests,
		Documentation:  packageFlags.documentation,
		Language:       packageFlags.language,
		Framework:      packageFlags.framework,
		Runtime:        runtime,
		SigningKey:     packageFlags.signingKey,
		GenerateSBOM:   packageFlags.generateSBOM,
		ScanVulns:      packageFlags.scanVulns,
		Compression:    packageFlags.compression,
		CompressionLevel: packageFlags.compressionLevel,
		Tags:           packageFlags.tags,
		Labels:         labels,
	}

	// Add repository and homepage to labels if provided
	if packageFlags.repository != "" {
		if request.Labels == nil {
			request.Labels = make(map[string]string)
		}
		request.Labels["repository"] = packageFlags.repository
	}

	if packageFlags.homepage != "" {
		if request.Labels == nil {
			request.Labels = make(map[string]string)
		}
		request.Labels["homepage"] = packageFlags.homepage
	}

	return request, nil
}

func generateDocumentation(result *packager.PackageResult, packageFlags *PackageFlags) error {
	fmt.Printf("📚 Generating documentation...\n")

	config := packager.DefaultDocsConfig()
	config.Format = packageFlags.docsFormat
	config.OutputDir = filepath.Join(packageFlags.outputDir, "docs")

	generator := packager.NewDocsGenerator(config)

	docRequest := &packager.DocumentationRequest{
		Manifest: &result.Manifest,
		Format:   packageFlags.docsFormat,
		Sections: []string{
			"overview",
			"installation",
			"configuration",
			"deployment",
			"monitoring",
		},
	}

	if config.IncludeAPI {
		docRequest.Sections = append(docRequest.Sections, "api")
	}

	if config.IncludeSBOM {
		docRequest.Sections = append(docRequest.Sections, "sbom")
	}

	if config.IncludeVulns {
		docRequest.Sections = append(docRequest.Sections, "security")
	}

	docResult, err := generator.GenerateDocumentation(context.Background(), docRequest)
	if err != nil {
		return err
	}

	fmt.Printf("📄 Documentation generated: %s\n", docResult.OutputPath)
	fmt.Printf("   Format: %s\n", docResult.Format)
	fmt.Printf("   Size: %s\n", formatSize(docResult.Size))
	fmt.Printf("   Sections: %s\n", strings.Join(docResult.Sections, ", "))

	return nil
}

func publishPackage(ctx context.Context, capsulePath string, packageFlags *PackageFlags) error {
	fmt.Printf("🚀 Publishing package...\n")

	config := packager.DefaultDeliveryConfig()
	deliveryService := packager.NewDeliveryService(config)

	// Load delivery channel configurations from environment/config
	for _, channelName := range packageFlags.publish {
		switch channelName {
		case "registry":
			registryConfig := loadRegistryConfig()
			if registryConfig == nil {
				fmt.Printf("⚠️  Registry configuration not found. Please set REGISTRY_* environment variables\n")
				continue
			}
			channel := packager.CreateRegistryChannel("registry", *registryConfig)
			deliveryService.RegisterChannel("registry", channel)

		case "cdn":
			cdnConfig := loadCDNConfig()
			if cdnConfig == nil {
				fmt.Printf("⚠️  CDN configuration not found. Please set CDN_* environment variables\n")
				continue
			}
			cdnConfig.PublicRead = packageFlags.public
			channel := packager.CreateCDNChannel("cdn", *cdnConfig)
			deliveryService.RegisterChannel("cdn", channel)

		case "direct":
			directConfig := loadDirectConfig()
			if directConfig == nil {
				fmt.Printf("⚠️  Direct delivery configuration not found. Please set DIRECT_* environment variables\n")
				continue
			}
			channel := packager.CreateDirectChannel("direct", *directConfig)
			deliveryService.RegisterChannel("direct", channel)

		default:
			fmt.Printf("⚠️  Unknown delivery channel: %s\n", channelName)
			continue
		}
	}

	// Publish to channels
	results, errors := deliveryService.DeliverToMultipleChannels(ctx, capsulePath, packageFlags.publish, make(map[string]interface{}))

	if len(results) > 0 {
		fmt.Printf("✅ Package published successfully!\n")
		for channel, result := range results {
			fmt.Printf("   %s: %s\n", channel, result.URL)
		}
	}

	if len(errors) > 0 {
		fmt.Printf("❌ Publishing failed:\n")
		for channel, err := range errors {
			fmt.Printf("   %s: %s\n", channel, err.Error())
		}
	}

	return nil
}

func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

// loadRegistryConfig loads registry configuration from environment variables
func loadRegistryConfig() *packager.RegistryConfig {
	url := os.Getenv("REGISTRY_URL")
	username := os.Getenv("REGISTRY_USERNAME")
	password := os.Getenv("REGISTRY_PASSWORD")
	namespace := os.Getenv("REGISTRY_NAMESPACE")

	if url == "" || username == "" || password == "" {
		return nil
	}

	// Set defaults
	if namespace == "" {
		namespace = "quantumlayer"
	}

	tlsStr := os.Getenv("REGISTRY_TLS")
	tls := tlsStr != "false" // Default to true

	return &packager.RegistryConfig{
		URL:       url,
		Username:  username,
		Password:  password,
		Namespace: namespace,
		TLS:       tls,
	}
}

// loadCDNConfig loads CDN configuration from environment variables
func loadCDNConfig() *packager.CDNConfig {
	url := os.Getenv("CDN_URL")
	apiKey := os.Getenv("CDN_API_KEY")
	bucketName := os.Getenv("CDN_BUCKET_NAME")

	if url == "" || apiKey == "" || bucketName == "" {
		return nil
	}

	// Set defaults
	region := os.Getenv("CDN_REGION")
	if region == "" {
		region = "us-east-1"
	}

	cacheTTLStr := os.Getenv("CDN_CACHE_TTL")
	cacheTTL := 3600 // Default to 1 hour
	if cacheTTLStr != "" {
		if ttl, err := strconv.Atoi(cacheTTLStr); err == nil {
			cacheTTL = ttl
		}
	}

	return &packager.CDNConfig{
		URL:        url,
		APIKey:     apiKey,
		BucketName: bucketName,
		Region:     region,
		PublicRead: false, // Will be set from flags
		CacheTTL:   cacheTTL,
	}
}

// loadDirectConfig loads direct delivery configuration from environment variables
func loadDirectConfig() *packager.DirectConfig {
	baseURL := os.Getenv("DIRECT_BASE_URL")
	storagePath := os.Getenv("DIRECT_STORAGE_PATH")

	if baseURL == "" || storagePath == "" {
		return nil
	}

	// Set defaults
	serveHTTPStr := os.Getenv("DIRECT_SERVE_HTTP")
	serveHTTP := serveHTTPStr != "false" // Default to true

	httpPortStr := os.Getenv("DIRECT_HTTP_PORT")
	httpPort := 8080 // Default port
	if httpPortStr != "" {
		if port, err := strconv.Atoi(httpPortStr); err == nil {
			httpPort = port
		}
	}

	return &packager.DirectConfig{
		BaseURL:     baseURL,
		StoragePath: storagePath,
		ServeHTTP:   serveHTTP,
		HTTPPort:    httpPort,
	}
}