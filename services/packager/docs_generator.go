package packager

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"
)

// DocsGenerator generates documentation for packages
type DocsGenerator struct {
	config *DocsConfig
}

// DocsConfig contains configuration for documentation generation
type DocsConfig struct {
	OutputDir      string            `json:"output_dir"`
	TemplateDir    string            `json:"template_dir"`
	Format         string            `json:"format"`         // "markdown", "html", "pdf"
	IncludeSBOM    bool              `json:"include_sbom"`
	IncludeVulns   bool              `json:"include_vulns"`
	IncludeAPI     bool              `json:"include_api"`
	CustomSections map[string]string `json:"custom_sections"`
}

// DocumentationRequest represents a request to generate documentation
type DocumentationRequest struct {
	Manifest      *CapsuleManifest  `json:"manifest"`
	OutputPath    string            `json:"output_path"`
	Format        string            `json:"format"`
	Sections      []string          `json:"sections"`
	CustomData    map[string]interface{} `json:"custom_data,omitempty"`
}

// DocumentationResult represents the result of documentation generation
type DocumentationResult struct {
	OutputPath   string        `json:"output_path"`
	Format       string        `json:"format"`
	Size         int64         `json:"size"`
	GeneratedAt  time.Time     `json:"generated_at"`
	GenerationTime time.Duration `json:"generation_time"`
	Sections     []string      `json:"sections"`
}

// NewDocsGenerator creates a new documentation generator
func NewDocsGenerator(config *DocsConfig) *DocsGenerator {
	if config == nil {
		config = DefaultDocsConfig()
	}

	return &DocsGenerator{
		config: config,
	}
}

// DefaultDocsConfig returns default documentation configuration
func DefaultDocsConfig() *DocsConfig {
	return &DocsConfig{
		OutputDir:      "./docs",
		TemplateDir:    "./templates",
		Format:         "markdown",
		IncludeSBOM:    true,
		IncludeVulns:   true,
		IncludeAPI:     true,
		CustomSections: make(map[string]string),
	}
}

// GenerateDocumentation generates comprehensive documentation for a package
func (dg *DocsGenerator) GenerateDocumentation(ctx context.Context, req *DocumentationRequest) (*DocumentationResult, error) {
	startTime := time.Now()

	// Validate request
	if req.Manifest == nil {
		return nil, fmt.Errorf("manifest is required")
	}

	// Determine output format
	format := req.Format
	if format == "" {
		format = dg.config.Format
	}

	// Determine sections to include
	sections := req.Sections
	if len(sections) == 0 {
		sections = dg.getDefaultSections()
	}

	// Generate documentation based on format
	var outputPath string
	var err error

	switch format {
	case "markdown":
		outputPath, err = dg.generateMarkdown(req, sections)
	case "html":
		outputPath, err = dg.generateHTML(req, sections)
	case "pdf":
		outputPath, err = dg.generatePDF(req, sections)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate documentation: %w", err)
	}

	// Calculate file size
	size, err := dg.getFileSize(outputPath)
	if err != nil {
		size = 0
	}

	result := &DocumentationResult{
		OutputPath:     outputPath,
		Format:         format,
		Size:           size,
		GeneratedAt:    time.Now(),
		GenerationTime: time.Since(startTime),
		Sections:       sections,
	}

	return result, nil
}

// GenerateDeploymentGuide generates a deployment guide
func (dg *DocsGenerator) GenerateDeploymentGuide(manifest *CapsuleManifest) (string, error) {
	return dg.renderTemplate("deployment_guide", manifest)
}

// GenerateAPIDocumentation generates API documentation from manifest
func (dg *DocsGenerator) GenerateAPIDocumentation(manifest *CapsuleManifest) (string, error) {
	return dg.renderTemplate("api_docs", manifest)
}

// GenerateSecurityReport generates a security report
func (dg *DocsGenerator) GenerateSecurityReport(manifest *CapsuleManifest) (string, error) {
	return dg.renderTemplate("security_report", manifest)
}

// Helper methods

func (dg *DocsGenerator) getDefaultSections() []string {
	sections := []string{
		"overview",
		"installation",
		"configuration",
		"deployment",
		"monitoring",
	}

	if dg.config.IncludeAPI {
		sections = append(sections, "api")
	}

	if dg.config.IncludeSBOM {
		sections = append(sections, "sbom")
	}

	if dg.config.IncludeVulns {
		sections = append(sections, "security")
	}

	return sections
}

func (dg *DocsGenerator) generateMarkdown(req *DocumentationRequest, sections []string) (string, error) {
	var content strings.Builder

	// Header
	content.WriteString(fmt.Sprintf("# %s\n\n", req.Manifest.Name))
	content.WriteString(fmt.Sprintf("%s\n\n", req.Manifest.Description))

	// Table of Contents
	content.WriteString("## Table of Contents\n\n")
	for _, section := range sections {
		content.WriteString(fmt.Sprintf("- [%s](#%s)\n", strings.Title(section), strings.ToLower(section)))
	}
	content.WriteString("\n")

	// Generate each section
	for _, section := range sections {
		sectionContent, err := dg.generateSection(section, req.Manifest)
		if err != nil {
			return "", fmt.Errorf("failed to generate section %s: %w", section, err)
		}
		content.WriteString(sectionContent)
		content.WriteString("\n")
	}

	// Write to file
	outputPath := filepath.Join(dg.config.OutputDir, req.Manifest.Name+".md")
	if req.OutputPath != "" {
		outputPath = req.OutputPath
	}

	return outputPath, dg.writeFile(outputPath, content.String())
}

func (dg *DocsGenerator) generateHTML(req *DocumentationRequest, sections []string) (string, error) {
	// Generate markdown first
	mdPath, err := dg.generateMarkdown(req, sections)
	if err != nil {
		return "", err
	}

	// Convert to HTML (simplified implementation)
	htmlPath := strings.Replace(mdPath, ".md", ".html", 1)

	htmlContent := `<!DOCTYPE html>
<html>
<head>
    <title>` + req.Manifest.Name + `</title>
    <style>
        body { font-family: Arial, sans-serif; margin: 40px; }
        h1 { color: #333; border-bottom: 2px solid #0066cc; }
        h2 { color: #666; border-bottom: 1px solid #ccc; }
        pre { background: #f5f5f5; padding: 10px; border-radius: 5px; }
        code { background: #f0f0f0; padding: 2px 4px; border-radius: 3px; }
        table { border-collapse: collapse; width: 100%; }
        th, td { border: 1px solid #ddd; padding: 8px; text-align: left; }
        th { background-color: #f2f2f2; }
    </style>
</head>
<body>`

	// Read markdown content and convert to HTML (simplified)
	mdContent, _ := dg.readFile(mdPath)
	htmlContent += "<div>" + dg.markdownToHTML(mdContent) + "</div>"

	htmlContent += `
</body>
</html>`

	return htmlPath, dg.writeFile(htmlPath, htmlContent)
}

func (dg *DocsGenerator) generatePDF(req *DocumentationRequest, sections []string) (string, error) {
	// For now, return an error as PDF generation requires additional dependencies
	return "", fmt.Errorf("PDF generation not implemented yet")
}

func (dg *DocsGenerator) generateSection(sectionName string, manifest *CapsuleManifest) (string, error) {
	switch sectionName {
	case "overview":
		return dg.generateOverviewSection(manifest)
	case "installation":
		return dg.generateInstallationSection(manifest)
	case "configuration":
		return dg.generateConfigurationSection(manifest)
	case "deployment":
		return dg.generateDeploymentSection(manifest)
	case "monitoring":
		return dg.generateMonitoringSection(manifest)
	case "api":
		return dg.generateAPISection(manifest)
	case "sbom":
		return dg.generateSBOMSection(manifest)
	case "security":
		return dg.generateSecuritySection(manifest)
	default:
		return fmt.Sprintf("## %s\n\nSection not implemented.\n\n", strings.Title(sectionName)), nil
	}
}

func (dg *DocsGenerator) generateOverviewSection(manifest *CapsuleManifest) (string, error) {
	var content strings.Builder

	content.WriteString("## Overview\n\n")
	content.WriteString(fmt.Sprintf("**Package:** %s\n", manifest.Name))
	content.WriteString(fmt.Sprintf("**Version:** %s\n", manifest.Version))
	content.WriteString(fmt.Sprintf("**Language:** %s\n", manifest.Language))
	content.WriteString(fmt.Sprintf("**Framework:** %s\n", manifest.Framework))
	if manifest.Author != "" {
		content.WriteString(fmt.Sprintf("**Author:** %s\n", manifest.Author))
	}
	if manifest.License != "" {
		content.WriteString(fmt.Sprintf("**License:** %s\n", manifest.License))
	}
	content.WriteString(fmt.Sprintf("**Created:** %s\n", manifest.CreatedAt.Format(time.RFC3339)))
	content.WriteString("\n")

	if len(manifest.Tags) > 0 {
		content.WriteString("**Tags:** ")
		content.WriteString(strings.Join(manifest.Tags, ", "))
		content.WriteString("\n\n")
	}

	return content.String(), nil
}

func (dg *DocsGenerator) generateInstallationSection(manifest *CapsuleManifest) (string, error) {
	var content strings.Builder

	content.WriteString("## Installation\n\n")
	content.WriteString("### Prerequisites\n\n")

	// Runtime requirements
	if manifest.Runtime.MinVersion != "" {
		content.WriteString(fmt.Sprintf("- **%s:** %s or later\n", manifest.Language, manifest.Runtime.MinVersion))
	}

	// Platform requirements
	if len(manifest.Runtime.Platform) > 0 {
		content.WriteString(fmt.Sprintf("- **Platform:** %s\n", strings.Join(manifest.Runtime.Platform, ", ")))
	}

	// Resource requirements
	if manifest.Runtime.Resources.CPU != "" || manifest.Runtime.Resources.Memory != "" {
		content.WriteString("- **Resources:**\n")
		if manifest.Runtime.Resources.CPU != "" {
			content.WriteString(fmt.Sprintf("  - CPU: %s\n", manifest.Runtime.Resources.CPU))
		}
		if manifest.Runtime.Resources.Memory != "" {
			content.WriteString(fmt.Sprintf("  - Memory: %s\n", manifest.Runtime.Resources.Memory))
		}
		if manifest.Runtime.Resources.Storage != "" {
			content.WriteString(fmt.Sprintf("  - Storage: %s\n", manifest.Runtime.Resources.Storage))
		}
	}

	content.WriteString("\n### Installation Steps\n\n")
	content.WriteString("1. Download the `.qlcapsule` package\n")
	content.WriteString("2. Extract the package using QLF CLI:\n")
	content.WriteString("   ```bash\n")
	content.WriteString("   qlf extract " + manifest.Name + ".qlcapsule\n")
	content.WriteString("   ```\n")
	content.WriteString("3. Deploy using the provided manifests\n\n")

	return content.String(), nil
}

func (dg *DocsGenerator) generateConfigurationSection(manifest *CapsuleManifest) (string, error) {
	var content strings.Builder

	content.WriteString("## Configuration\n\n")

	// Environment variables
	if len(manifest.Runtime.Environment) > 0 {
		content.WriteString("### Environment Variables\n\n")
		content.WriteString("| Variable | Description |\n")
		content.WriteString("|----------|-------------|\n")
		for key, value := range manifest.Runtime.Environment {
			content.WriteString(fmt.Sprintf("| `%s` | %s |\n", key, value))
		}
		content.WriteString("\n")
	}

	// Deployment configuration
	if manifest.Deployment.Type != "" {
		content.WriteString("### Deployment Configuration\n\n")
		content.WriteString(fmt.Sprintf("**Type:** %s\n\n", manifest.Deployment.Type))

		// Secrets
		if len(manifest.Deployment.Secrets) > 0 {
			content.WriteString("#### Required Secrets\n\n")
			for _, secret := range manifest.Deployment.Secrets {
				content.WriteString(fmt.Sprintf("- **%s**", secret.Name))
				if secret.Description != "" {
					content.WriteString(fmt.Sprintf(": %s", secret.Description))
				}
				content.WriteString("\n")
				if len(secret.Keys) > 0 {
					content.WriteString(fmt.Sprintf("  - Keys: %s\n", strings.Join(secret.Keys, ", ")))
				}
			}
			content.WriteString("\n")
		}

		// ConfigMaps
		if len(manifest.Deployment.ConfigMaps) > 0 {
			content.WriteString("#### Required ConfigMaps\n\n")
			for _, cm := range manifest.Deployment.ConfigMaps {
				content.WriteString(fmt.Sprintf("- **%s**", cm.Name))
				if cm.Description != "" {
					content.WriteString(fmt.Sprintf(": %s", cm.Description))
				}
				content.WriteString("\n")
			}
			content.WriteString("\n")
		}
	}

	return content.String(), nil
}

func (dg *DocsGenerator) generateDeploymentSection(manifest *CapsuleManifest) (string, error) {
	var content strings.Builder

	content.WriteString("## Deployment\n\n")

	if manifest.Deployment.Type != "" {
		content.WriteString(fmt.Sprintf("### %s Deployment\n\n", strings.Title(manifest.Deployment.Type)))

		// Manifests
		if len(manifest.Deployment.Manifests) > 0 {
			content.WriteString("#### Manifests\n\n")
			for _, manifest := range manifest.Deployment.Manifests {
				content.WriteString(fmt.Sprintf("- **%s** (%s)\n", manifest.Name, manifest.Type))
			}
			content.WriteString("\n")
		}

		// Networking
		if len(manifest.Deployment.Networking.Ports) > 0 {
			content.WriteString("#### Exposed Ports\n\n")
			content.WriteString("| Port | Name | Protocol | External |\n")
			content.WriteString("|------|------|----------|----------|\n")
			for _, port := range manifest.Deployment.Networking.Ports {
				external := "No"
				if port.External {
					external = "Yes"
				}
				protocol := port.Protocol
				if protocol == "" {
					protocol = "TCP"
				}
				content.WriteString(fmt.Sprintf("| %d | %s | %s | %s |\n", port.Port, port.Name, protocol, external))
			}
			content.WriteString("\n")
		}

		// Storage
		if len(manifest.Deployment.Storage) > 0 {
			content.WriteString("#### Storage Requirements\n\n")
			content.WriteString("| Name | Type | Size | Mount Path |\n")
			content.WriteString("|------|------|------|------------|\n")
			for _, storage := range manifest.Deployment.Storage {
				content.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", storage.Name, storage.Type, storage.Size, storage.MountPath))
			}
			content.WriteString("\n")
		}
	}

	return content.String(), nil
}

func (dg *DocsGenerator) generateMonitoringSection(manifest *CapsuleManifest) (string, error) {
	var content strings.Builder

	content.WriteString("## Monitoring\n\n")

	monitoring := manifest.Deployment.Monitoring

	// Health checks
	if monitoring.HealthCheck.Enabled {
		content.WriteString("### Health Checks\n\n")
		content.WriteString(fmt.Sprintf("- **Enabled:** Yes\n"))
		if monitoring.HealthCheck.Path != "" {
			content.WriteString(fmt.Sprintf("- **Path:** %s\n", monitoring.HealthCheck.Path))
		}
		if monitoring.HealthCheck.Port != 0 {
			content.WriteString(fmt.Sprintf("- **Port:** %d\n", monitoring.HealthCheck.Port))
		}
		if monitoring.HealthCheck.Interval != "" {
			content.WriteString(fmt.Sprintf("- **Interval:** %s\n", monitoring.HealthCheck.Interval))
		}
		content.WriteString("\n")
	}

	// Metrics
	if monitoring.Metrics.Enabled {
		content.WriteString("### Metrics\n\n")
		content.WriteString(fmt.Sprintf("- **Enabled:** Yes\n"))
		if monitoring.Metrics.Path != "" {
			content.WriteString(fmt.Sprintf("- **Path:** %s\n", monitoring.Metrics.Path))
		}
		if monitoring.Metrics.Port != 0 {
			content.WriteString(fmt.Sprintf("- **Port:** %d\n", monitoring.Metrics.Port))
		}
		if monitoring.Metrics.Format != "" {
			content.WriteString(fmt.Sprintf("- **Format:** %s\n", monitoring.Metrics.Format))
		}
		content.WriteString("\n")
	}

	// Logging
	if monitoring.Logging.Level != "" || monitoring.Logging.Format != "" {
		content.WriteString("### Logging\n\n")
		if monitoring.Logging.Level != "" {
			content.WriteString(fmt.Sprintf("- **Level:** %s\n", monitoring.Logging.Level))
		}
		if monitoring.Logging.Format != "" {
			content.WriteString(fmt.Sprintf("- **Format:** %s\n", monitoring.Logging.Format))
		}
		content.WriteString("\n")
	}

	return content.String(), nil
}

func (dg *DocsGenerator) generateAPISection(manifest *CapsuleManifest) (string, error) {
	var content strings.Builder

	content.WriteString("## API Documentation\n\n")
	content.WriteString("*API documentation would be generated from OpenAPI/Swagger specs*\n\n")

	return content.String(), nil
}

func (dg *DocsGenerator) generateSBOMSection(manifest *CapsuleManifest) (string, error) {
	var content strings.Builder

	content.WriteString("## Software Bill of Materials (SBOM)\n\n")

	if manifest.SBOM != nil {
		content.WriteString(fmt.Sprintf("**Format:** %s\n", manifest.SBOM.Format))
		content.WriteString(fmt.Sprintf("**Generated:** %s\n", manifest.SBOM.GeneratedAt.Format(time.RFC3339)))
		content.WriteString(fmt.Sprintf("**Tool:** %s\n", manifest.SBOM.Tool))
		content.WriteString("\n")

		if len(manifest.SBOM.Components) > 0 {
			content.WriteString("### Components\n\n")
			content.WriteString("| Name | Version | Type | License |\n")
			content.WriteString("|------|---------|------|--------|\n")
			for _, comp := range manifest.SBOM.Components {
				content.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", comp.Name, comp.Version, comp.Type, comp.License))
			}
			content.WriteString("\n")
		}
	} else {
		content.WriteString("No SBOM available.\n\n")
	}

	return content.String(), nil
}

func (dg *DocsGenerator) generateSecuritySection(manifest *CapsuleManifest) (string, error) {
	var content strings.Builder

	content.WriteString("## Security Report\n\n")

	if manifest.Attestation != nil {
		content.WriteString("### Build Attestation\n\n")
		content.WriteString(fmt.Sprintf("- **Builder:** %s\n", manifest.Attestation.Builder))
		content.WriteString(fmt.Sprintf("- **Build Platform:** %s\n", manifest.Attestation.BuildPlatform))
		content.WriteString(fmt.Sprintf("- **Build Time:** %s\n", manifest.Attestation.BuildTimestamp.Format(time.RFC3339)))
		content.WriteString(fmt.Sprintf("- **Reproducible:** %t\n", manifest.Attestation.Reproducible))
		if manifest.Attestation.SourceCommit != "" {
			content.WriteString(fmt.Sprintf("- **Source Commit:** %s\n", manifest.Attestation.SourceCommit))
		}
		content.WriteString("\n")
	}

	if len(manifest.Signatures) > 0 {
		content.WriteString("### Digital Signatures\n\n")
		content.WriteString("| Algorithm | Key ID | Signer | Timestamp |\n")
		content.WriteString("|-----------|--------|--------|----------|\n")
		for _, sig := range manifest.Signatures {
			content.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", sig.Algorithm, sig.KeyID, sig.Signer, sig.Timestamp.Format(time.RFC3339)))
		}
		content.WriteString("\n")
	}

	// Vulnerability information would be included here if available

	return content.String(), nil
}

// Utility methods

func (dg *DocsGenerator) renderTemplate(templateName string, data interface{}) (string, error) {
	// This is a simplified implementation
	// In production, would use actual template files
	return fmt.Sprintf("Template: %s\nData: %+v\n", templateName, data), nil
}

func (dg *DocsGenerator) writeFile(path, content string) error {
	// Create directory if it doesn't exist
	dir := filepath.Dir(path)
	if err := ensureDir(dir); err != nil {
		return err
	}

	// Write file
	return writeStringToFile(path, content)
}

func (dg *DocsGenerator) readFile(path string) (string, error) {
	// Read file content
	return readStringFromFile(path)
}

func (dg *DocsGenerator) getFileSize(path string) (int64, error) {
	// Get file size
	return getFileSize(path)
}

func (dg *DocsGenerator) markdownToHTML(markdown string) string {
	// Simplified markdown to HTML conversion
	// In production, would use a proper markdown parser
	html := strings.ReplaceAll(markdown, "\n## ", "\n<h2>")
	html = strings.ReplaceAll(html, "\n### ", "\n<h3>")
	html = strings.ReplaceAll(html, "\n# ", "\n<h1>")
	html = strings.ReplaceAll(html, "**", "<strong>")
	html = strings.ReplaceAll(html, "*", "<em>")
	html = strings.ReplaceAll(html, "\n", "<br>")
	return html
}

// Placeholder functions that would be implemented with actual file operations
func ensureDir(dir string) error {
	// Implementation would create directory
	return nil
}

func writeStringToFile(path, content string) error {
	// Implementation would write content to file
	return nil
}

func readStringFromFile(path string) (string, error) {
	// Implementation would read file content
	return "", nil
}

func getFileSize(path string) (int64, error) {
	// Implementation would get file size
	return 0, nil
}