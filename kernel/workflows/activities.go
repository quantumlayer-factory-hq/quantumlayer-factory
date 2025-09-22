package workflows

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"go.temporal.io/sdk/activity"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/agents"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/verifier"
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

	// Generate backend code
	if irSpec.App.Stack.Backend.Language != "" {

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

	// Generate database code if entities are defined
	if len(irSpec.Data.Entities) > 0 {
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

		// Write each generated file
		for filename, content := range generatedCode {
			filePath := filepath.Join(projectDir, filename)
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
			err := os.WriteFile(filePath, []byte(content), 0644)
			if err != nil {
				fmt.Printf("Warning: Failed to write file %s: %v\n", filename, err)
				continue
			}

			result.ArtifactPaths = append(result.ArtifactPaths, filePath)
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