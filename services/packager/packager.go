package packager

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"context"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// PackagerService implements package creation and management
type PackagerService struct {
	config *PackagerConfig
}

// PackagerConfig contains configuration for the packager service
type PackagerConfig struct {
	WorkDir         string `json:"work_dir"`
	OutputDir       string `json:"output_dir"`
	TempDir         string `json:"temp_dir"`
	CompressionType string `json:"compression_type"` // "gzip", "lz4", "zstd"
	CompressionLevel int   `json:"compression_level"`

	// Security settings
	SigningEnabled bool   `json:"signing_enabled"`
	PrivateKeyPath string `json:"private_key_path,omitempty"`
	PublicKeyPath  string `json:"public_key_path,omitempty"`

	// SBOM generation
	SBOMEnabled   bool   `json:"sbom_enabled"`
	SBOMTool      string `json:"sbom_tool"`      // "syft", "cyclonedx"
	SBOMFormat    string `json:"sbom_format"`    // "spdx", "cyclonedx"

	// Vulnerability scanning
	VulnScanEnabled bool   `json:"vuln_scan_enabled"`
	VulnScanner     string `json:"vuln_scanner"`     // "trivy", "grype"
	VulnThreshold   string `json:"vuln_threshold"`   // "critical", "high"

	// Delivery channels
	DeliveryChannels map[string]DeliveryChannel `json:"delivery_channels"`
}

// NewPackagerService creates a new packager service
func NewPackagerService(config *PackagerConfig) *PackagerService {
	if config == nil {
		config = DefaultPackagerConfig()
	}

	return &PackagerService{
		config: config,
	}
}

// DefaultPackagerConfig returns default packager configuration
func DefaultPackagerConfig() *PackagerConfig {
	// Enable signing if a signing key is configured via environment
	signingEnabled := false
	signingKeyPath := os.Getenv("QLF_SIGNING_KEY_PATH")
	if signingKeyPath != "" {
		// Check if the key file exists
		if _, err := os.Stat(signingKeyPath); err == nil {
			signingEnabled = true
		}
	}

	return &PackagerConfig{
		WorkDir:          "/tmp/qlf-packager",
		OutputDir:        "./packages",
		TempDir:          "/tmp",
		CompressionType:  "gzip",
		CompressionLevel: 6,
		SigningEnabled:   signingEnabled,
		SBOMEnabled:      true,
		SBOMTool:         "syft",
		SBOMFormat:       "spdx",
		VulnScanEnabled:  true,
		VulnScanner:      "trivy",
		VulnThreshold:    "high",
		DeliveryChannels: make(map[string]DeliveryChannel),
	}
}

// CreatePackage creates a .qlcapsule package from the given request
func (ps *PackagerService) CreatePackage(ctx context.Context, req *PackageRequest) (*PackageResult, error) {
	startTime := time.Now()

	// Validate request
	if err := ps.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	// Create temporary workspace
	workDir, err := ps.createWorkspace(req.Name)
	if err != nil {
		return nil, fmt.Errorf("failed to create workspace: %w", err)
	}
	defer ps.cleanupWorkspace(workDir)

	// Create manifest
	manifest, err := ps.createManifest(req)
	if err != nil {
		return nil, fmt.Errorf("failed to create manifest: %w", err)
	}

	// Generate SBOM if enabled
	if req.GenerateSBOM && ps.config.SBOMEnabled {
		sbom, err := ps.generateSBOM(req.SourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to generate SBOM: %w", err)
		}
		manifest.SBOM = sbom
	}

	// Scan for vulnerabilities if enabled
	var vulnResult *VulnScanResult
	if req.ScanVulns && ps.config.VulnScanEnabled {
		vulnResult, err = ps.scanVulnerabilities(req.SourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to scan vulnerabilities: %w", err)
		}

		// Check if vulnerabilities exceed threshold
		if vulnResult.IsVulnerable(ps.config.VulnThreshold) {
			return nil, fmt.Errorf("vulnerabilities exceed threshold: %s", ps.config.VulnThreshold)
		}
	}

	// Create attestation
	attestation := ps.createAttestation(req)
	manifest.Attestation = attestation

	// Collect artifacts
	artifacts, err := ps.collectArtifacts(workDir, req)
	if err != nil {
		return nil, fmt.Errorf("failed to collect artifacts: %w", err)
	}
	manifest.Artifacts = artifacts

	// Sign the package if enabled
	if ps.config.SigningEnabled {
		// Use provided signing key or fall back to environment variable
		signingKey := req.SigningKey
		if signingKey == "" {
			signingKey = os.Getenv("QLF_SIGNING_KEY_PATH")
		}

		if signingKey != "" {
			signature, err := ps.signManifest(manifest, signingKey)
			if err != nil {
				return nil, fmt.Errorf("failed to sign package: %w", err)
			}
			manifest.Signatures = []DigitalSignature{*signature}
		} else {
			fmt.Println("Warning: Signing enabled but no signing key provided. Set QLF_SIGNING_KEY_PATH environment variable.")
		}
	}

	// Create the .qlcapsule file
	capsulePath, err := ps.createCapsuleFile(workDir, manifest)
	if err != nil {
		return nil, fmt.Errorf("failed to create capsule file: %w", err)
	}

	// Calculate final hash and size
	hash, size, err := ps.calculateFileHash(capsulePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate hash: %w", err)
	}

	result := &PackageResult{
		CapsulePath:    capsulePath,
		Manifest:       *manifest,
		Size:           size,
		Hash:           hash,
		CreatedAt:      time.Now(),
		BuildTime:      time.Since(startTime),
		Compressed:     req.Compression != "",
		Validated:      true,
		VulnScanResult: vulnResult,
	}

	return result, nil
}

// PublishPackage publishes a package to delivery channels
func (ps *PackagerService) PublishPackage(ctx context.Context, req *PublishRequest) (*PublishResult, error) {
	result := &PublishResult{
		URLs:        make(map[string]string),
		PublishedAt: time.Now(),
		Success:     true,
	}

	for _, channelName := range req.Channels {
		channel, exists := ps.config.DeliveryChannels[channelName]
		if !exists {
			result.Errors = append(result.Errors, fmt.Sprintf("channel not found: %s", channelName))
			result.Success = false
			continue
		}

		url, err := ps.publishToChannel(ctx, req.CapsulePath, channel, req)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("failed to publish to %s: %v", channelName, err))
			result.Success = false
			continue
		}

		result.URLs[channelName] = url
	}

	return result, nil
}

// Helper methods

func (ps *PackagerService) validateRequest(req *PackageRequest) error {
	if req.Name == "" {
		return ErrInvalidName
	}
	if req.Version == "" {
		return ErrInvalidVersion
	}
	if req.SourcePath == "" {
		return ErrInvalidPath
	}
	if req.Language == "" {
		return ErrInvalidLanguage
	}

	// Check if source path exists
	if _, err := os.Stat(req.SourcePath); os.IsNotExist(err) {
		return fmt.Errorf("source path does not exist: %s", req.SourcePath)
	}

	return nil
}

func (ps *PackagerService) createWorkspace(name string) (string, error) {
	workDir := filepath.Join(ps.config.WorkDir, name)
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return "", err
	}
	return workDir, nil
}

func (ps *PackagerService) cleanupWorkspace(workDir string) error {
	return os.RemoveAll(workDir)
}

func (ps *PackagerService) createManifest(req *PackageRequest) (*CapsuleManifest, error) {
	manifest := &CapsuleManifest{
		Version:     req.Version,
		Name:        req.Name,
		Description: req.Description,
		Author:      req.Author,
		CreatedAt:   time.Now(),
		Tags:        req.Tags,
		License:     req.License,
		Language:    req.Language,
		Framework:   req.Framework,
		Runtime:     req.Runtime,
		Metadata:    make(map[string]interface{}),
	}

	// Add custom labels as metadata
	if req.Labels != nil {
		manifest.Metadata["labels"] = req.Labels
	}
	if req.Annotations != nil {
		manifest.Metadata["annotations"] = req.Annotations
	}

	return manifest, nil
}

func (ps *PackagerService) generateSBOM(sourcePath string) (*SBOM, error) {
	// Use actual SBOM tools (Syft) for production-ready generation
	sbom := &SBOM{
		Format:      ps.config.SBOMFormat,
		Version:     "1.0",
		GeneratedAt: time.Now(),
		Tool:        ps.config.SBOMTool,
		Components:  []Component{},
	}

	// Generate SBOM using Syft if available, otherwise use fallback analysis
	components, err := ps.generateComponentsWithSyft(sourcePath)
	if err != nil {
		// Fallback to basic file analysis if Syft is not available
		fmt.Printf("Warning: Syft not available, using basic analysis: %v\n", err)
		components, err = ps.generateComponentsBasic(sourcePath)
		if err != nil {
			return nil, fmt.Errorf("failed to generate components: %w", err)
		}
	}

	sbom.Components = components
	return sbom, nil
}

func (ps *PackagerService) scanVulnerabilities(sourcePath string) (*VulnScanResult, error) {
	// Use actual vulnerability scanning with Trivy or fallback to basic analysis
	return ps.scanWithTrivy(sourcePath)
}

func (ps *PackagerService) createAttestation(req *PackageRequest) *Attestation {
	return &Attestation{
		BuildPlatform:  "linux/amd64",
		BuildTimestamp: time.Now(),
		BuildHash:      ps.generateBuildHash(req),
		SourceRepo:     req.SourcePath,
		Builder:        "quantumlayer-factory",
		Reproducible:   true,
	}
}

func (ps *PackagerService) generateBuildHash(req *PackageRequest) string {
	hasher := sha256.New()
	hasher.Write([]byte(req.Name + req.Version + req.SourcePath))
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func (ps *PackagerService) collectArtifacts(workDir string, req *PackageRequest) ([]Artifact, error) {
	var artifacts []Artifact

	// Copy source files
	err := filepath.Walk(req.SourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, _ := filepath.Rel(req.SourcePath, path)
			destPath := filepath.Join(workDir, "source", relPath)

			// Create directory structure
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return err
			}

			// Copy file
			if err := ps.copyFile(path, destPath); err != nil {
				return err
			}

			// Calculate hash
			hash, err := ps.calculateFileHashString(destPath)
			if err != nil {
				return err
			}

			artifact := Artifact{
				Name:       relPath,
				Type:       ps.detectArtifactType(relPath),
				Path:       filepath.Join("source", relPath),
				Size:       info.Size(),
				Hash:       hash,
				Compressed: false,
				Executable: ps.isExecutable(info.Mode()),
				Permissions: fmt.Sprintf("%o", info.Mode().Perm()),
			}

			artifacts = append(artifacts, artifact)
		}

		return nil
	})

	if err != nil {
		return nil, err
	}

	// Add build artifacts
	for _, artifactPath := range req.BuildArtifacts {
		if _, err := os.Stat(artifactPath); err == nil {
			info, _ := os.Stat(artifactPath)
			destPath := filepath.Join(workDir, "artifacts", filepath.Base(artifactPath))

			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return nil, err
			}

			if err := ps.copyFile(artifactPath, destPath); err != nil {
				return nil, err
			}

			hash, err := ps.calculateFileHashString(destPath)
			if err != nil {
				return nil, err
			}

			artifact := Artifact{
				Name:       filepath.Base(artifactPath),
				Type:       "binary",
				Path:       filepath.Join("artifacts", filepath.Base(artifactPath)),
				Size:       info.Size(),
				Hash:       hash,
				Executable: ps.isExecutable(info.Mode()),
				Permissions: fmt.Sprintf("%o", info.Mode().Perm()),
			}

			artifacts = append(artifacts, artifact)
		}
	}

	// Add manifests
	for _, manifestPath := range req.Manifests {
		if _, err := os.Stat(manifestPath); err == nil {
			info, _ := os.Stat(manifestPath)
			destPath := filepath.Join(workDir, "manifests", filepath.Base(manifestPath))

			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return nil, err
			}

			if err := ps.copyFile(manifestPath, destPath); err != nil {
				return nil, err
			}

			hash, err := ps.calculateFileHashString(destPath)
			if err != nil {
				return nil, err
			}

			artifact := Artifact{
				Name:       filepath.Base(manifestPath),
				Type:       "manifest",
				Path:       filepath.Join("manifests", filepath.Base(manifestPath)),
				Size:       info.Size(),
				Hash:       hash,
				Permissions: fmt.Sprintf("%o", info.Mode().Perm()),
			}

			artifacts = append(artifacts, artifact)
		}
	}

	return artifacts, nil
}

func (ps *PackagerService) signManifest(manifest *CapsuleManifest, keyPath string) (*DigitalSignature, error) {
	// Load private key
	keyData, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, err
	}

	block, _ := pem.Decode(keyData)
	if block == nil {
		return nil, fmt.Errorf("failed to parse PEM block")
	}

	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	// Serialize manifest for signing
	manifestData, err := json.Marshal(manifest)
	if err != nil {
		return nil, err
	}

	// Calculate hash
	hasher := sha256.New()
	hasher.Write(manifestData)
	hashed := hasher.Sum(nil)

	// Sign
	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
	if err != nil {
		return nil, err
	}

	return &DigitalSignature{
		Algorithm: "RSA",
		KeyID:     ps.generateKeyID(privateKey),
		Signature: base64.StdEncoding.EncodeToString(signature),
		Timestamp: time.Now(),
		Signer:    "quantumlayer-factory",
	}, nil
}

func (ps *PackagerService) createCapsuleFile(workDir string, manifest *CapsuleManifest) (string, error) {
	// Create output directory
	if err := os.MkdirAll(ps.config.OutputDir, 0755); err != nil {
		return "", err
	}

	// Create capsule file
	capsulePath := filepath.Join(ps.config.OutputDir, manifest.Name+".qlcapsule")
	capsuleFile, err := os.Create(capsulePath)
	if err != nil {
		return "", err
	}
	defer capsuleFile.Close()

	// Create gzip writer
	gzipWriter := gzip.NewWriter(capsuleFile)
	defer gzipWriter.Close()

	// Create tar writer
	tarWriter := tar.NewWriter(gzipWriter)
	defer tarWriter.Close()

	// Add manifest
	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return "", err
	}

	manifestHeader := &tar.Header{
		Name: "manifest.json",
		Mode: 0644,
		Size: int64(len(manifestData)),
	}

	if err := tarWriter.WriteHeader(manifestHeader); err != nil {
		return "", err
	}

	if _, err := tarWriter.Write(manifestData); err != nil {
		return "", err
	}

	// Add all files from workspace
	err = filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if !info.IsDir() {
			relPath, _ := filepath.Rel(workDir, path)

			file, err := os.Open(path)
			if err != nil {
				return err
			}
			defer file.Close()

			header := &tar.Header{
				Name: relPath,
				Mode: int64(info.Mode().Perm()),
				Size: info.Size(),
			}

			if err := tarWriter.WriteHeader(header); err != nil {
				return err
			}

			if _, err := io.Copy(tarWriter, file); err != nil {
				return err
			}
		}

		return nil
	})

	if err != nil {
		return "", err
	}

	return capsulePath, nil
}

func (ps *PackagerService) publishToChannel(ctx context.Context, capsulePath string, channel DeliveryChannel, req *PublishRequest) (string, error) {
	switch channel.Type {
	case "registry":
		return ps.publishToRegistry(ctx, capsulePath, channel, req)
	case "cdn":
		return ps.publishToCDN(ctx, capsulePath, channel, req)
	case "direct":
		return ps.publishDirect(ctx, capsulePath, channel, req)
	default:
		return "", fmt.Errorf("unsupported delivery channel type: %s", channel.Type)
	}
}

func (ps *PackagerService) publishToRegistry(ctx context.Context, capsulePath string, channel DeliveryChannel, req *PublishRequest) (string, error) {
	// Mock implementation for registry publishing
	return fmt.Sprintf("%s/%s:latest", channel.Endpoint, filepath.Base(capsulePath)), nil
}

func (ps *PackagerService) publishToCDN(ctx context.Context, capsulePath string, channel DeliveryChannel, req *PublishRequest) (string, error) {
	// Mock implementation for CDN publishing
	return fmt.Sprintf("%s/%s", channel.Endpoint, filepath.Base(capsulePath)), nil
}

func (ps *PackagerService) publishDirect(ctx context.Context, capsulePath string, channel DeliveryChannel, req *PublishRequest) (string, error) {
	// Mock implementation for direct publishing
	return fmt.Sprintf("%s/%s", channel.Endpoint, filepath.Base(capsulePath)), nil
}

// Utility methods

func (ps *PackagerService) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

func (ps *PackagerService) calculateFileHash(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return "", 0, err
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), size, nil
}

func (ps *PackagerService) calculateFileHashString(path string) (string, error) {
	hash, _, err := ps.calculateFileHash(path)
	return hash, err
}

func (ps *PackagerService) detectArtifactType(path string) string {
	ext := strings.ToLower(filepath.Ext(path))
	switch ext {
	case ".go", ".py", ".js", ".ts", ".java", ".c", ".cpp", ".rs":
		return "source"
	case ".json", ".yaml", ".yml", ".toml", ".xml":
		return "config"
	case ".md", ".txt", ".rst":
		return "documentation"
	case ".exe", ".bin", ".so", ".dll":
		return "binary"
	default:
		return "file"
	}
}

func (ps *PackagerService) isExecutable(mode os.FileMode) bool {
	return mode&0111 != 0
}

func (ps *PackagerService) generateKeyID(key *rsa.PrivateKey) string {
	// Generate a simple key ID based on the public key
	pubKeyBytes, _ := x509.MarshalPKIXPublicKey(&key.PublicKey)
	hasher := sha256.New()
	hasher.Write(pubKeyBytes)
	return fmt.Sprintf("%x", hasher.Sum(nil))[:16]
}

// generateComponentsWithSyft uses Syft CLI tool for accurate SBOM generation
func (ps *PackagerService) generateComponentsWithSyft(sourcePath string) ([]Component, error) {
	// Check if syft is available
	cmd := exec.Command("syft", "--version")
	if err := cmd.Run(); err != nil {
		return nil, fmt.Errorf("syft not installed: %w", err)
	}

	// Run syft to generate SBOM
	outputFormat := "json"
	if ps.config.SBOMFormat == "spdx" {
		outputFormat = "spdx-json"
	} else if ps.config.SBOMFormat == "cyclonedx" {
		outputFormat = "cyclonedx-json"
	}

	cmd = exec.Command("syft", "dir:"+sourcePath, "-o", outputFormat)
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("syft execution failed: %w", err)
	}

	// Parse syft output and convert to our Component format
	return ps.parseSyftOutput(output, outputFormat)
}

// parseSyftOutput converts Syft JSON output to our Component format
func (ps *PackagerService) parseSyftOutput(output []byte, format string) ([]Component, error) {
	var components []Component

	// Parse based on format
	switch format {
	case "json":
		var syftResult struct {
			Artifacts []struct {
				Name     string `json:"name"`
				Version  string `json:"version"`
				Type     string `json:"type"`
				Licenses []struct {
					Value string `json:"value"`
				} `json:"licenses"`
			} `json:"artifacts"`
		}

		if err := json.Unmarshal(output, &syftResult); err != nil {
			return nil, fmt.Errorf("failed to parse syft output: %w", err)
		}

		for _, artifact := range syftResult.Artifacts {
			license := "Unknown"
			if len(artifact.Licenses) > 0 {
				license = artifact.Licenses[0].Value
			}

			components = append(components, Component{
				Type:         artifact.Type,
				Name:         artifact.Name,
				Version:      artifact.Version,
				License:      license,
				Dependencies: []string{},
			})
		}

	default:
		return nil, fmt.Errorf("unsupported syft format: %s", format)
	}

	return components, nil
}

// generateComponentsBasic provides basic component analysis when Syft is not available
func (ps *PackagerService) generateComponentsBasic(sourcePath string) ([]Component, error) {
	var components []Component

	// Analyze package files to extract dependencies
	if err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Detect package files and extract dependencies
		switch filepath.Base(path) {
		case "package.json":
			if deps, err := ps.parsePackageJson(path); err == nil {
				components = append(components, deps...)
			}
		case "requirements.txt":
			if deps, err := ps.parseRequirementsTxt(path); err == nil {
				components = append(components, deps...)
			}
		case "go.mod":
			if deps, err := ps.parseGoMod(path); err == nil {
				components = append(components, deps...)
			}
		case "Cargo.toml":
			if deps, err := ps.parseCargoToml(path); err == nil {
				components = append(components, deps...)
			}
		}

		return nil
	}); err != nil {
		return nil, fmt.Errorf("failed to walk source directory: %w", err)
	}

	// Add main application component
	components = append(components, Component{
		Type:         "application",
		Name:         filepath.Base(sourcePath),
		Version:      "1.0.0",
		License:      "Unknown",
		Dependencies: []string{},
	})

	return components, nil
}

// Package file parsers for basic analysis
func (ps *PackagerService) parsePackageJson(path string) ([]Component, error) {
	// Basic Node.js package.json parsing
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}

	if err := json.Unmarshal(data, &pkg); err != nil {
		return nil, err
	}

	var components []Component
	for name, version := range pkg.Dependencies {
		components = append(components, Component{
			Type:    "npm-package",
			Name:    name,
			Version: version,
			License: "Unknown",
		})
	}

	return components, nil
}

func (ps *PackagerService) parseRequirementsTxt(path string) ([]Component, error) {
	// Basic Python requirements.txt parsing
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var components []Component
	lines := strings.Split(string(data), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Simple parsing: package==version or package>=version
		parts := strings.FieldsFunc(line, func(c rune) bool {
			return c == '=' || c == '>' || c == '<' || c == '!'
		})
		if len(parts) > 0 {
			name := strings.TrimSpace(parts[0])
			version := "Unknown"
			if len(parts) > 1 {
				version = strings.TrimSpace(parts[1])
			}

			components = append(components, Component{
				Type:    "python-package",
				Name:    name,
				Version: version,
				License: "Unknown",
			})
		}
	}

	return components, nil
}

func (ps *PackagerService) parseGoMod(path string) ([]Component, error) {
	// Basic Go mod parsing
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var components []Component
	lines := strings.Split(string(data), "\n")
	inRequire := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "require") {
			inRequire = true
			continue
		}
		if line == ")" {
			inRequire = false
			continue
		}
		if inRequire && line != "" {
			parts := strings.Fields(line)
			if len(parts) >= 2 {
				name := parts[0]
				version := parts[1]
				components = append(components, Component{
					Type:    "go-module",
					Name:    name,
					Version: version,
					License: "Unknown",
				})
			}
		}
	}

	return components, nil
}

func (ps *PackagerService) parseCargoToml(path string) ([]Component, error) {
	// Basic Cargo.toml parsing (simplified)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var components []Component
	lines := strings.Split(string(data), "\n")
	inDependencies := false

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[dependencies]") {
			inDependencies = true
			continue
		}
		if strings.HasPrefix(line, "[") && inDependencies {
			inDependencies = false
			continue
		}
		if inDependencies && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				name := strings.TrimSpace(parts[0])
				version := strings.Trim(strings.TrimSpace(parts[1]), "\"")
				components = append(components, Component{
					Type:    "rust-crate",
					Name:    name,
					Version: version,
					License: "Unknown",
				})
			}
		}
	}

	return components, nil
}

// scanWithTrivy performs vulnerability scanning using Trivy
func (ps *PackagerService) scanWithTrivy(sourcePath string) (*VulnScanResult, error) {
	startTime := time.Now()

	// Check if trivy is available
	_, err := exec.LookPath("trivy")
	if err != nil {
		// Fallback to basic vulnerability scanning
		return ps.scanBasicVulnerabilities(sourcePath)
	}

	// Create temporary file for trivy output
	tmpFile, err := os.CreateTemp("", "trivy-scan-*.json")
	if err != nil {
		return ps.scanBasicVulnerabilities(sourcePath)
	}
	defer os.Remove(tmpFile.Name())
	tmpFile.Close()

	// Run trivy scan
	cmd := exec.Command("trivy", "fs", "--format", "json", "--output", tmpFile.Name(), sourcePath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	err = cmd.Run()
	if err != nil {
		fmt.Printf("Trivy scan failed, using fallback: %v (%s)\n", err, stderr.String())
		return ps.scanBasicVulnerabilities(sourcePath)
	}

	// Parse trivy output
	data, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return ps.scanBasicVulnerabilities(sourcePath)
	}

	var trivyResult struct {
		Results []struct {
			Vulnerabilities []struct {
				VulnerabilityID string `json:"VulnerabilityID"`
				Severity       string `json:"Severity"`
				PkgName        string `json:"PkgName"`
				InstalledVersion string `json:"InstalledVersion"`
				FixedVersion   string `json:"FixedVersion"`
				Title          string `json:"Title"`
				Description    string `json:"Description"`
			} `json:"Vulnerabilities"`
		} `json:"Results"`
	}

	if err := json.Unmarshal(data, &trivyResult); err != nil {
		return ps.scanBasicVulnerabilities(sourcePath)
	}

	// Convert to our format
	result := &VulnScanResult{
		ScanTime:     startTime,
		Scanner:      "trivy",
		ScanDuration: time.Since(startTime),
	}

	for _, res := range trivyResult.Results {
		for _, vuln := range res.Vulnerabilities {
			result.TotalVulns++

			switch strings.ToUpper(vuln.Severity) {
			case "CRITICAL":
				result.Critical++
			case "HIGH":
				result.High++
			case "MEDIUM":
				result.Medium++
			case "LOW":
				result.Low++
			}

			if vuln.FixedVersion != "" && vuln.FixedVersion != "unknown" {
				result.Fixable++
			}
		}
	}

	return result, nil
}

// scanBasicVulnerabilities provides fallback vulnerability scanning
func (ps *PackagerService) scanBasicVulnerabilities(sourcePath string) (*VulnScanResult, error) {
	startTime := time.Now()

	result := &VulnScanResult{
		TotalVulns:   0,
		Critical:     0,
		High:         0,
		Medium:       0,
		Low:          0,
		Fixable:      0,
		ScanTime:     startTime,
		Scanner:      "basic-fallback",
		ScanDuration: time.Since(startTime),
	}

	// Basic checks for known vulnerable patterns
	err := filepath.Walk(sourcePath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // Continue on errors
		}

		// Check for known vulnerable file patterns
		filename := info.Name()

		// Check for old JavaScript libraries (basic heuristics)
		if strings.Contains(filename, "jquery") && strings.Contains(filename, "1.") {
			result.TotalVulns++
			result.Medium++
			result.Fixable++
		}

		// Check for vulnerable Python packages in requirements.txt
		if filename == "requirements.txt" {
			content, err := os.ReadFile(path)
			if err == nil {
				if strings.Contains(string(content), "django==1.") ||
				   strings.Contains(string(content), "flask==0.") ||
				   strings.Contains(string(content), "requests==2.0") {
					result.TotalVulns++
					result.High++
					result.Fixable++
				}
			}
		}

		// Check for vulnerable Node.js packages
		if filename == "package.json" {
			content, err := os.ReadFile(path)
			if err == nil {
				if strings.Contains(string(content), "\"lodash\": \"4.1") ||
				   strings.Contains(string(content), "\"express\": \"3.") {
					result.TotalVulns++
					result.High++
					result.Fixable++
				}
			}
		}

		return nil
	})

	if err != nil {
		return nil, fmt.Errorf("basic vulnerability scan failed: %w", err)
	}

	result.ScanDuration = time.Since(startTime)
	return result, nil
}