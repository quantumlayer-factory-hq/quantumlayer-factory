package packager

import (
	"archive/tar"
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
	return &PackagerConfig{
		WorkDir:          "/tmp/qlf-packager",
		OutputDir:        "./packages",
		TempDir:          "/tmp",
		CompressionType:  "gzip",
		CompressionLevel: 6,
		SigningEnabled:   false,
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
	if ps.config.SigningEnabled && req.SigningKey != "" {
		signature, err := ps.signManifest(manifest, req.SigningKey)
		if err != nil {
			return nil, fmt.Errorf("failed to sign package: %w", err)
		}
		manifest.Signatures = []DigitalSignature{*signature}
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
	// This is a mock implementation - in production would use actual SBOM tools
	sbom := &SBOM{
		Format:      ps.config.SBOMFormat,
		Version:     "1.0",
		GeneratedAt: time.Now(),
		Tool:        ps.config.SBOMTool,
		Components:  []Component{},
	}

	// Mock component for demonstration
	component := Component{
		Type:    "application",
		Name:    filepath.Base(sourcePath),
		Version: "1.0.0",
		License: "MIT",
	}

	sbom.Components = append(sbom.Components, component)

	return sbom, nil
}

func (ps *PackagerService) scanVulnerabilities(sourcePath string) (*VulnScanResult, error) {
	// This is a mock implementation - in production would use actual vulnerability scanners
	result := &VulnScanResult{
		TotalVulns:   0,
		Critical:     0,
		High:         0,
		Medium:       0,
		Low:          0,
		Fixable:      0,
		ScanTime:     time.Now(),
		Scanner:      ps.config.VulnScanner,
		ScanDuration: time.Second,
	}

	return result, nil
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