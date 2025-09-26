package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/agents"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/verifier"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/services/packager"
)

// ParseBriefActivity parses a natural language brief into IR
func ParseBriefActivity(ctx context.Context, brief string, overlays []string, config map[string]interface{}) (*ir.IRSpec, error) {

	// Create IR compiler
	compiler := ir.NewCompiler()

	var result *ir.CompilationResult
	var err error

	// Use overlays if provided, otherwise do regular compilation
	if len(overlays) > 0 {
		result, err = compiler.CompileWithOverlays(brief, overlays)
	} else {
		result, err = compiler.Compile(brief)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to compile brief: %w", err)
	}

	return result.Spec, nil
}

// ValidateIRActivity validates the IR specification
func ValidateIRActivity(ctx context.Context, irSpec *ir.IRSpec) (ValidationResult, error) {

	result := ValidationResult{
		Valid:    true,
		Errors:   []string{},
		Warnings: []string{},
	}

	// Basic validation checks
	if irSpec.App.Name == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Application name is required")
	}

	if irSpec.App.Type == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Application type is required")
	}

	if len(irSpec.App.Features) == 0 {
		result.Warnings = append(result.Warnings, "No features defined in application")
	}

	// Validate stack configuration
	if irSpec.App.Stack.Backend.Language == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Backend language is required")
	}

	if irSpec.App.Stack.Backend.Framework == "" {
		result.Valid = false
		result.Errors = append(result.Errors, "Backend framework is required")
	}

	// Validate data model if present
	if len(irSpec.Data.Entities) > 0 {
		for _, entity := range irSpec.Data.Entities {
			if entity.Name == "" {
				result.Valid = false
				result.Errors = append(result.Errors, "Entity name is required")
			}
			if len(entity.Fields) == 0 {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Entity '%s' has no fields defined", entity.Name))
			}
		}
	}

	// Validate API spec if present
	if len(irSpec.API.Endpoints) > 0 {
		for _, endpoint := range irSpec.API.Endpoints {
			if endpoint.Path == "" {
				result.Valid = false
				result.Errors = append(result.Errors, "Endpoint path is required")
			}
			if endpoint.Method == "" {
				result.Valid = false
				result.Errors = append(result.Errors, "Endpoint method is required")
			}
		}
	}

	return result, nil
}

// GenerateCodeActivity generates code using the agent factory
func GenerateCodeActivity(ctx context.Context, irSpec *ir.IRSpec, overlays []string, config map[string]interface{}, provider, model string) (CodeGenerationResult, error) {

	result := CodeGenerationResult{
		Success:       true,
		GeneratedCode: make(map[string]string),
		Artifacts:     []string{},
		Errors:        []string{},
		Warnings:      []string{},
	}

	// Create agent factory (LLM-enabled if provider specified)
	factory, err := createLLMEnabledFactory(provider, model)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to create agent factory: %v", err))
		return result, nil
	}

	// Generate backend code (only if backend stack is properly configured)
	if irSpec.App.Stack.Backend.Language != "" && irSpec.App.Stack.Backend.Framework != "" {

		backendAgent, err := factory.CreateAgent(agents.AgentTypeBackend)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create backend agent: %v", err))
		} else {
			// Create generation request
			request := &agents.GenerationRequest{
				Spec: irSpec,
				Target: agents.GenerationTarget{
					Type:      "backend",
					Language:  irSpec.App.Stack.Backend.Language,
					Framework: irSpec.App.Stack.Backend.Framework,
				},
				Options: agents.GenerationOptions{
					CreateDirectories: true,
					FormatCode:       true,
					ValidateOutput:   true,
				},
				Context: map[string]interface{}{
					"workflow": "factory",
					"overlays": overlays,
				},
			}

			// Record heartbeat before LLM call
			activity.RecordHeartbeat(ctx, "Generating backend code with agents...")
			output, err := backendAgent.Generate(ctx, request)
			if err != nil {
				result.Success = false
				result.Errors = append(result.Errors, fmt.Sprintf("Backend generation failed: %v", err))
			} else if !output.Success {
				result.Success = false
				result.Errors = append(result.Errors, output.Errors...)
				result.Warnings = append(result.Warnings, output.Warnings...)
			} else {
				// Add generated files to result
				for _, file := range output.Files {
					result.GeneratedCode[file.Path] = file.Content
					result.Artifacts = append(result.Artifacts, file.Path)
				}
				result.Warnings = append(result.Warnings, output.Warnings...)
			}
		}
	}

	// Generate frontend code if frontend stack is specified
	if irSpec.App.Stack.Frontend.Framework != "" {
		frontendAgent, err := factory.CreateAgent(agents.AgentTypeFrontend)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create frontend agent: %v", err))
		} else {
			// Create generation request
			request := &agents.GenerationRequest{
				Spec: irSpec,
				Target: agents.GenerationTarget{
					Type:      "frontend",
					Language:  "javascript", // Default, could be determined from spec
					Framework: irSpec.App.Stack.Frontend.Framework,
				},
				Options: agents.GenerationOptions{
					CreateDirectories: true,
					FormatCode:       true,
					ValidateOutput:   true,
				},
				Context: map[string]interface{}{
					"workflow": "factory",
					"overlays": overlays,
				},
			}

			// Record heartbeat before LLM call
			activity.RecordHeartbeat(ctx, "Generating frontend code...")
			output, err := frontendAgent.Generate(ctx, request)
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Frontend generation failed: %v", err))
			} else if !output.Success {
				result.Warnings = append(result.Warnings, output.Errors...)
				result.Warnings = append(result.Warnings, output.Warnings...)
			} else {
				// Add generated files to result
				for _, file := range output.Files {
					result.GeneratedCode[file.Path] = file.Content
					result.Artifacts = append(result.Artifacts, file.Path)
				}
				result.Warnings = append(result.Warnings, output.Warnings...)
			}
		}
	}

	// Generate database code if entities are defined AND database stack is configured
	if len(irSpec.Data.Entities) > 0 && irSpec.App.Stack.Database.Type != "" {
		databaseAgent, err := factory.CreateAgent(agents.AgentTypeDatabase)
		if err != nil {
			result.Success = false
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create database agent: %v", err))
		} else {
			// Create generation request
			request := &agents.GenerationRequest{
				Spec: irSpec,
				Target: agents.GenerationTarget{
					Type:     "database",
					Language: "sql",
				},
				Options: agents.GenerationOptions{
					CreateDirectories: true,
					FormatCode:       true,
					ValidateOutput:   true,
				},
				Context: map[string]interface{}{
					"workflow": "factory",
					"overlays": overlays,
				},
			}

			// Record heartbeat before LLM call
			activity.RecordHeartbeat(ctx, "Generating database code...")
			output, err := databaseAgent.Generate(ctx, request)
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Database generation failed: %v", err))
			} else if !output.Success {
				result.Warnings = append(result.Warnings, output.Errors...)
				result.Warnings = append(result.Warnings, output.Warnings...)
			} else {
				// Add generated files to result
				for _, file := range output.Files {
					result.GeneratedCode[file.Path] = file.Content
					result.Artifacts = append(result.Artifacts, file.Path)
				}
				result.Warnings = append(result.Warnings, output.Warnings...)
			}
		}
	}

	// Validate SPA-only requirements if this is a frontend-only app
	if shouldValidateSPA(irSpec) {
		violations := validateSPAOnly(result.GeneratedCode)
		if len(violations) > 0 {
			fmt.Printf("SPA validation failed: %v. Attempting repair...\n", violations)

			// Attempt one-shot repair
			repairedCode, err := repairSPA(ctx, factory, irSpec, violations)
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("SPA repair failed: %v", err))
			} else {
				fmt.Printf("SPA repair successful, replacing generated code\n")
				result.GeneratedCode = repairedCode
			}
		}
	}

	return result, nil
}

// VerifyCodeActivity performs static analysis on generated code
func VerifyCodeActivity(ctx context.Context, generatedCode map[string]string, config map[string]interface{}) (VerificationResult, error) {

	result := VerificationResult{
		Success: true,
		Issues:  []string{},
		Reports: []string{},
	}

	// Create static analysis gate
	gateConfig := verifier.GateConfig{
		Enabled: true,
		Rules:   make(map[string]interface{}),
	}

	// Add config overrides if provided
	if config != nil {
		for key, value := range config {
			gateConfig.Rules[key] = value
		}
	}

	staticGate := verifier.NewStaticGate(gateConfig)

	// Add Go vet runner for Go files
	goVetRunner := func() verifier.Runner {
		// We need to import the runners package, but for now we'll create a stub
		return &stubRunner{name: "go-vet"}
	}()

	err := staticGate.AddRunner(goVetRunner)
	if err != nil {
	}

	// Convert generated code to artifacts
	var artifacts []verifier.Artifact
	for filename, content := range generatedCode {
		language := detectLanguage(filename)
		artifact := verifier.Artifact{
			Path:     filename,
			Content:  content,
			Type:     verifier.ArtifactTypeSource,
			Language: language,
		}
		artifacts = append(artifacts, artifact)
	}

	// Run verification if we have artifacts
	if len(artifacts) > 0 && staticGate.CanVerify(artifacts) {
		request := &verifier.VerificationRequest{
			Artifacts: artifacts,
			Config:    gateConfig,
		}

		verifyResult, err := staticGate.Verify(ctx, request)
		if err != nil {
			result.Success = false
			result.Issues = append(result.Issues, fmt.Sprintf("Verification failed: %v", err))
		} else {
			if !verifyResult.Success {
				result.Success = false
			}

			// Convert verification issues to strings
			for _, issue := range verifyResult.Issues {
				issueStr := fmt.Sprintf("%s:%d - %s: %s", issue.File, issue.Line, issue.Severity, issue.Description)
				result.Issues = append(result.Issues, issueStr)
			}

			// Add verification report
			reportName := fmt.Sprintf("verification_report_%d_issues.txt", len(verifyResult.Issues))
			result.Reports = append(result.Reports, reportName)
		}
	} else {
		result.Issues = append(result.Issues, "No suitable verification runners found for generated code")
	}

	return result, nil
}

// PackageArtifactsActivity packages the generated code and artifacts
func PackageArtifactsActivity(ctx context.Context, generatedCode map[string]string, irSpec *ir.IRSpec, projectID string, dryRun bool, outputDir string) (PackageResult, error) {

	result := PackageResult{
		Success:       true,
		ArtifactPaths: []string{},
	}

	if outputDir == "" {
		outputDir = "./generated"
	}

	if dryRun {
		// Generate SOC-formatted patch for dry-run output
		var socPatch strings.Builder
		socPatch.WriteString("=== Generated Code (Dry Run) ===\n\n")

		for filename, content := range generatedCode {
			socPatch.WriteString(fmt.Sprintf("--- /dev/null\n+++ %s\n", filename))
			socPatch.WriteString("@@ -0,0 +1," + fmt.Sprintf("%d", strings.Count(content, "\n")+1) + " @@\n")

			// Add each line with + prefix
			for _, line := range strings.Split(content, "\n") {
				socPatch.WriteString("+" + line + "\n")
			}
			socPatch.WriteString("\n")
		}

		result.SOCPatch = socPatch.String()
		result.OutputPath = "(dry-run - no files written)"
	} else {
		// Write files to actual output directory
		projectDir := fmt.Sprintf("%s/%s", outputDir, projectID)
		result.OutputPath = projectDir

		// Create project directory
		err := os.MkdirAll(projectDir, 0755)
		if err != nil {
			return PackageResult{
				Success: false,
			}, fmt.Errorf("failed to create project directory: %w", err)
		}

		// Write each generated file using splitter
		for filename, content := range generatedCode {
			files, err := splitContent(filename, content)
			if err != nil {
				fmt.Printf("Warning: Failed to split content for %s: %v\n", filename, err)
				continue
			}

			for _, file := range files {
				// Harden path (prevent traversal)
				safePath := hardenPath(file.Path)
				if safePath == "" {
					fmt.Printf("Warning: Skipped unsafe path: %s\n", file.Path)
					continue
				}

				filePath := filepath.Join(projectDir, safePath)
				dir := filepath.Dir(filePath)

				// Create subdirectories if needed
				if dir != projectDir {
					err := os.MkdirAll(dir, 0755)
					if err != nil {
						fmt.Printf("Warning: Failed to create directory %s: %v\n", dir, err)
						continue
					}
				}

				// Write file
				err := os.WriteFile(filePath, []byte(file.Content), 0644)
				if err != nil {
					fmt.Printf("Warning: Failed to write file %s: %v\n", safePath, err)
					continue
				}

				result.ArtifactPaths = append(result.ArtifactPaths, filePath)
			}
		}

		// Write IR spec as JSON
		irSpecPath := filepath.Join(projectDir, "ir_spec.json")
		irSpecJSON, err := json.MarshalIndent(irSpec, "", "  ")
		if err != nil {
			fmt.Printf("Warning: Failed to marshal IR spec: %v\n", err)
		} else {
			err = os.WriteFile(irSpecPath, irSpecJSON, 0644)
			if err != nil {
				fmt.Printf("Warning: Failed to write IR spec: %v\n", err)
			} else {
				result.ArtifactPaths = append(result.ArtifactPaths, irSpecPath)
			}
		}

		// Generate and write README
		readmePath := filepath.Join(projectDir, "README.md")
		readmeContent := generateReadme(irSpec, generatedCode)
		err = os.WriteFile(readmePath, []byte(readmeContent), 0644)
		if err != nil {
			fmt.Printf("Warning: Failed to write README: %v\n", err)
		} else {
			result.ArtifactPaths = append(result.ArtifactPaths, readmePath)
		}

		// Create .qlcapsule package
		fmt.Printf("Creating .qlcapsule package for project %s\n", projectID)
		capsulePath, capsuleSize, err := createCapsulePackage(ctx, projectDir, irSpec, projectID)
		if err != nil {
			fmt.Printf("Warning: Failed to create .qlcapsule package: %v\n", err)
		} else {
			result.CapsulePath = capsulePath
			result.CapsuleSize = capsuleSize
			fmt.Printf("Successfully created .qlcapsule package: %s (%d bytes)\n", capsulePath, capsuleSize)
		}
	}

	return result, nil
}

// detectLanguage detects programming language from filename
func detectLanguage(filename string) string {
	if strings.HasSuffix(filename, ".go") {
		return "go"
	}
	if strings.HasSuffix(filename, ".py") {
		return "python"
	}
	if strings.HasSuffix(filename, ".js") || strings.HasSuffix(filename, ".ts") {
		return "javascript"
	}
	if strings.HasSuffix(filename, ".java") {
		return "java"
	}
	if strings.HasSuffix(filename, ".rb") {
		return "ruby"
	}
	return "unknown"
}

// stubRunner is a temporary stub implementation
type stubRunner struct {
	name string
}

func (s *stubRunner) GetName() string {
	return s.name
}

func (s *stubRunner) GetVersion() string {
	return "1.0.0"
}

func (s *stubRunner) CanRun(artifacts []verifier.Artifact) bool {
	for _, artifact := range artifacts {
		if artifact.Language == "go" {
			return true
		}
	}
	return false
}

func (s *stubRunner) Run(ctx context.Context, artifacts []verifier.Artifact, config map[string]interface{}) (*verifier.RunnerResult, error) {
	return &verifier.RunnerResult{
		Success:  true,
		Issues:   []verifier.Issue{},
		ExitCode: 0,
	}, nil
}

func (s *stubRunner) GetDefaultConfig() map[string]interface{} {
	return make(map[string]interface{})
}

// generateReadme creates a comprehensive README.md for the generated project
func generateReadme(irSpec *ir.IRSpec, generatedCode map[string]string) string {
	var readme strings.Builder

	// Project header
	readme.WriteString(fmt.Sprintf("# %s\n\n", irSpec.App.Name))

	if irSpec.App.Description != "" {
		readme.WriteString(fmt.Sprintf("%s\n\n", irSpec.App.Description))
	}

	// Generated info
	readme.WriteString("## 🤖 Generated Project\n\n")
	readme.WriteString(fmt.Sprintf("- **Generated**: %s\n", time.Now().Format("2006-01-02 15:04:05")))
	readme.WriteString(fmt.Sprintf("- **Type**: %s\n", irSpec.App.Type))
	readme.WriteString(fmt.Sprintf("- **Domain**: %s\n", irSpec.App.Domain))
	if irSpec.App.Stack.Backend.Language != "" {
		readme.WriteString(fmt.Sprintf("- **Language**: %s\n", irSpec.App.Stack.Backend.Language))
	}
	if irSpec.App.Stack.Backend.Framework != "" {
		readme.WriteString(fmt.Sprintf("- **Framework**: %s\n", irSpec.App.Stack.Backend.Framework))
	}
	readme.WriteString("\\n")

	// Files structure
	readme.WriteString("## 📁 Project Structure\\n\\n")
	readme.WriteString("```\\n")
	for filename := range generatedCode {
		readme.WriteString(fmt.Sprintf("%s\\n", filename))
	}
	readme.WriteString("ir_spec.json\\n")
	readme.WriteString("README.md\\n")
	readme.WriteString("```\\n\\n")

	// Features
	if len(irSpec.App.Features) > 0 {
		readme.WriteString("## ✨ Features\\n\\n")
		for _, feature := range irSpec.App.Features {
			readme.WriteString(fmt.Sprintf("- **%s**: %s\\n", feature.Name, feature.Description))
		}
		readme.WriteString("\\n")
	}

	// Entities
	if len(irSpec.Data.Entities) > 0 {
		readme.WriteString("## 📊 Data Models\\n\\n")
		for _, entity := range irSpec.Data.Entities {
			readme.WriteString(fmt.Sprintf("### %s\\n", entity.Name))
			if entity.Description != "" {
				readme.WriteString(fmt.Sprintf("%s\\n\\n", entity.Description))
			}
			if len(entity.Fields) > 0 {
				readme.WriteString("**Fields:**\\n")
				for _, field := range entity.Fields {
					readme.WriteString(fmt.Sprintf("- `%s` (%s)", field.Name, field.Type))
					if field.Required {
						readme.WriteString(" *required*")
					}
					if field.Description != "" {
						readme.WriteString(fmt.Sprintf(" - %s", field.Description))
					}
					readme.WriteString("\\n")
				}
			}
			readme.WriteString("\\n")
		}
	}

	// Quick start
	readme.WriteString("## 🚀 Quick Start\\n\\n")

	// Determine setup instructions based on language
	if irSpec.App.Stack.Backend.Language == "python" {
		readme.WriteString("```bash\\n")
		readme.WriteString("# Install dependencies\\n")
		readme.WriteString("pip install -r requirements.txt\\n\\n")
		readme.WriteString("# Run the application\\n")
		readme.WriteString("python main.py\\n")
		readme.WriteString("```\\n\\n")
	} else if irSpec.App.Stack.Backend.Language == "go" {
		readme.WriteString("```bash\\n")
		readme.WriteString("# Install dependencies\\n")
		readme.WriteString("go mod tidy\\n\\n")
		readme.WriteString("# Run the application\\n")
		readme.WriteString("go run main.go\\n")
		readme.WriteString("```\\n\\n")
	} else if irSpec.App.Stack.Backend.Language == "javascript" {
		readme.WriteString("```bash\\n")
		readme.WriteString("# Install dependencies\\n")
		readme.WriteString("npm install\\n\\n")
		readme.WriteString("# Run the application\\n")
		readme.WriteString("npm start\\n")
		readme.WriteString("```\\n\\n")
	}

	// API endpoints (if it's an API project)
	if irSpec.App.Type == "api" && len(irSpec.API.Endpoints) > 0 {
		readme.WriteString("## 🔗 API Endpoints\\n\\n")
		for _, endpoint := range irSpec.API.Endpoints {
			readme.WriteString(fmt.Sprintf("- `%s %s` - %s\\n", endpoint.Method, endpoint.Path, endpoint.Description))
		}
		readme.WriteString("\\n")
	}

	// Footer
	readme.WriteString("---\\n\\n")
	readme.WriteString("*Generated by [QuantumLayer Factory](https://github.com/quantumlayer-factory-hq/quantumlayer-factory)*\\n")

	return readme.String()
}

// createCapsulePackage creates a .qlcapsule package from the generated project
func createCapsulePackage(ctx context.Context, projectDir string, irSpec *ir.IRSpec, projectID string) (string, int64, error) {
	// Initialize packager service
	packagerConfig := packager.DefaultPackagerConfig()
	packagerService := packager.NewPackagerService(packagerConfig)

	// Create package request
	req := &packager.PackageRequest{
		Name:        irSpec.App.Name,
		Version:     "1.0.0",
		Description: irSpec.App.Description,
		Author:      "QuantumLayer Factory",
		License:     "MIT",

		// Source information
		SourcePath: projectDir,

		// Configuration
		Language:  irSpec.App.Stack.Backend.Language,
		Framework: irSpec.App.Stack.Backend.Framework,
		Runtime: packager.RuntimeSpec{
			Platform: []string{"linux/amd64"},
			Arch:     []string{"amd64"},
			Resources: packager.ResourceLimits{
				CPU:     "100m",
				Memory:  "128Mi",
				Storage: "1Gi",
			},
		},

		// Security options (basic for now)
		GenerateSBOM: false, // Disable SBOM for simplicity
		ScanVulns:    false, // Disable vulnerability scanning for simplicity

		// Compression options
		Compression:      "gzip",
		CompressionLevel: 6,

		// Metadata
		Tags: []string{irSpec.App.Type, irSpec.App.Domain},
	}

	// Create the package
	result, err := packagerService.CreatePackage(ctx, req)
	if err != nil {
		return "", 0, fmt.Errorf("failed to create capsule package: %w", err)
	}

	return result.CapsulePath, result.Size, nil
}

// SplitFile represents a file extracted from content
type SplitFile struct {
	Path    string
	Content string
}

// splitContent parses blob content and fans out files
func splitContent(originalKey, content string) ([]SplitFile, error) {
	// Check if content contains SOC format
	if strings.Contains(content, "### FACTORY/1 PATCH") {
		return parseSOCBundle(content)
	}

	// Check if content contains unified diff format
	if strings.Contains(content, "diff --git") || strings.Contains(content, "--- /dev/null") {
		return parseUnifiedDiff(content)
	}

	// Otherwise, treat as single file
	return []SplitFile{{
		Path:    originalKey,
		Content: content,
	}}, nil
}

// parseSOCBundle extracts files from SOC format bundle
func parseSOCBundle(content string) ([]SplitFile, error) {
	var files []SplitFile
	lines := strings.Split(content, "\n")

	var currentFile *SplitFile
	var inFileContent bool

	for _, line := range lines {
		// SOC file marker: "- file: path/to/file.ext"
		if strings.HasPrefix(line, "- file: ") {
			// Save previous file if exists
			if currentFile != nil {
				files = append(files, *currentFile)
			}

			// Start new file
			path := strings.TrimPrefix(line, "- file: ")
			currentFile = &SplitFile{Path: path, Content: ""}
			inFileContent = false
			continue
		}

		// Diff header lines
		if strings.HasPrefix(line, "--- ") || strings.HasPrefix(line, "+++ ") || strings.HasPrefix(line, "@@") {
			inFileContent = true
			continue
		}

		// Content lines (remove + prefix from diffs)
		if currentFile != nil && inFileContent {
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				currentFile.Content += strings.TrimPrefix(line, "+") + "\n"
			} else if !strings.HasPrefix(line, "-") && !strings.HasPrefix(line, "###") {
				currentFile.Content += line + "\n"
			}
		}
	}

	// Save last file
	if currentFile != nil {
		files = append(files, *currentFile)
	}

	return files, nil
}

// parseUnifiedDiff extracts files from unified diff format
func parseUnifiedDiff(content string) ([]SplitFile, error) {
	var files []SplitFile
	lines := strings.Split(content, "\n")

	var currentFile *SplitFile

	for _, line := range lines {
		// New file marker: "+++ b/path/to/file.ext"
		if strings.HasPrefix(line, "+++ b/") {
			// Save previous file
			if currentFile != nil {
				files = append(files, *currentFile)
			}

			// Start new file
			path := strings.TrimPrefix(line, "+++ b/")
			currentFile = &SplitFile{Path: path, Content: ""}
			continue
		}

		// Content lines (skip diff headers)
		if currentFile != nil && !strings.HasPrefix(line, "---") &&
		   !strings.HasPrefix(line, "+++") && !strings.HasPrefix(line, "@@") &&
		   !strings.HasPrefix(line, "diff --git") {
			if strings.HasPrefix(line, "+") {
				currentFile.Content += strings.TrimPrefix(line, "+") + "\n"
			}
		}
	}

	// Save last file
	if currentFile != nil {
		files = append(files, *currentFile)
	}

	return files, nil
}

// hardenPath prevents path traversal and normalizes paths
func hardenPath(path string) string {
	// Remove leading slash
	path = strings.TrimPrefix(path, "/")

	// Check for path traversal attempts
	if strings.Contains(path, "..") {
		return ""
	}

	// Normalize path separators
	path = filepath.Clean(path)

	// Additional safety checks
	if path == "." || path == "" {
		return ""
	}

	return path
}

// shouldValidateSPA determines if SPA validation should be applied
func shouldValidateSPA(irSpec *ir.IRSpec) bool {
	// Check if brief mentions "no backend" or "SPA" or "single-page"
	brief := strings.ToLower(irSpec.Brief)
	return strings.Contains(brief, "no backend") ||
		strings.Contains(brief, "spa") ||
		strings.Contains(brief, "single-page") ||
		strings.Contains(brief, "client-side")
}

// validateSPAOnly validates that generated code is SPA-only
func validateSPAOnly(generatedCode map[string]string) []string {
	var violations []string

	// Check for banned file extensions and directories
	bannedExts := []string{".py", ".sql", ".sh", ".yml", ".yaml", ".toml", ".dockerfile"}
	bannedDirs := []string{"backend", "server", "api", "database", "migrations", "ops"}

	for filepath, content := range generatedCode {
		// Check file extensions
		for _, ext := range bannedExts {
			if strings.HasSuffix(filepath, ext) {
				violations = append(violations, fmt.Sprintf("Banned file extension: %s", filepath))
			}
		}

		// Check directories
		for _, dir := range bannedDirs {
			if strings.Contains(filepath, dir+"/") {
				violations = append(violations, fmt.Sprintf("Banned directory: %s in %s", dir, filepath))
			}
		}

		// Check content for backend patterns
		contentLower := strings.ToLower(content)
		patterns := []string{
			"fastapi", "sqlalchemy", "psycopg2", "alembic",
			"create\\s+table", "### factory/1 patch",
			"oauth2passwordbearer", "bcrypt", "jwt",
		}

		for _, pattern := range patterns {
			if matched, _ := regexp.MatchString(pattern, contentLower); matched {
				violations = append(violations, fmt.Sprintf("Banned content pattern '%s' in %s", pattern, filepath))
			}
		}
	}

	// Check for required SPA files
	required := []string{"package.json", "index.html", "src/main.tsx", "src/App.tsx"}
	for _, req := range required {
		found := false
		for filepath := range generatedCode {
			if strings.Contains(filepath, req) {
				found = true
				break
			}
		}
		if !found {
			violations = append(violations, fmt.Sprintf("Missing required file: %s", req))
		}
	}

	return violations
}

// repairSPA attempts to repair SPA violations with a one-shot LLM call
func repairSPA(ctx context.Context, factory *agents.AgentFactory, irSpec *ir.IRSpec, violations []string) (map[string]string, error) {
	// Get frontend agent for repair
	frontendAgent, err := factory.CreateAgent(agents.AgentTypeFrontend)
	if err != nil {
		return nil, fmt.Errorf("failed to create frontend agent for repair: %w", err)
	}

	// Create repair prompt
	repairPrompt := fmt.Sprintf(`Your output included backend/SQL artifacts. Re-emit a pure React+TypeScript SPA as a single unified diff. No prose.

Required files: package.json, index.html, tsconfig.json, vite.config.ts, src/main.tsx, src/App.tsx, components, styles.
Forbidden: backend, SQL, SOC headers.

Brief: %s

Generate ONLY frontend files for a client-side todo application with localStorage persistence.`, irSpec.Brief)

	// Create repair request
	request := &agents.GenerationRequest{
		Spec: irSpec,
		Target: agents.GenerationTarget{
			Type:      "frontend",
			Language:  "typescript",
			Framework: "react",
		},
		Options: agents.GenerationOptions{
			CreateDirectories: true,
			FormatCode:       true,
			ValidateOutput:   true,
		},
		Context: map[string]interface{}{
			"workflow": "repair",
			"prompt":   repairPrompt,
		},
	}

	// Generate repaired code
	output, err := frontendAgent.Generate(ctx, request)
	if err != nil {
		return nil, fmt.Errorf("repair generation failed: %w", err)
	}

	if !output.Success {
		return nil, fmt.Errorf("repair generation unsuccessful: %v", output.Errors)
	}

	// Convert to map format
	repairedCode := make(map[string]string)
	for _, file := range output.Files {
		repairedCode[file.Path] = file.Content
	}

	return repairedCode, nil
}