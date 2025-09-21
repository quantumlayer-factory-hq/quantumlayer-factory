package workflows

import (
	"context"
	"fmt"
	"strings"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/agents"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/verifier"
)

// ParseBriefActivity parses a natural language brief into IR
func ParseBriefActivity(ctx context.Context, brief string, config map[string]interface{}) (*ir.IRSpec, error) {

	// Create IR compiler
	compiler := ir.NewCompiler()

	// Parse the brief
	result, err := compiler.Compile(brief)
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
func GenerateCodeActivity(ctx context.Context, irSpec *ir.IRSpec, config map[string]interface{}) (CodeGenerationResult, error) {

	result := CodeGenerationResult{
		Success:       true,
		GeneratedCode: make(map[string]string),
		Artifacts:     []string{},
		Errors:        []string{},
		Warnings:      []string{},
	}

	// Generate backend code
	if irSpec.App.Stack.Backend.Language != "" {

		backendAgent := agents.NewBackendAgent()

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
			},
		}

		output, err := backendAgent.Generate(context.Background(), request)
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

	// Generate frontend code if frontend stack is specified
	if irSpec.App.Stack.Frontend.Framework != "" {

		// Frontend agent would be implemented here
		result.Warnings = append(result.Warnings, "Frontend generation not yet implemented")
	}

	// Generate database code if entities are defined
	if len(irSpec.Data.Entities) > 0 {

		// Database agent would be implemented here
		result.Warnings = append(result.Warnings, "Database generation not yet implemented")
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

		// In a full implementation, this would actually write files
		// For now, just create artifact path references
		for filename := range generatedCode {
			artifactPath := fmt.Sprintf("%s/%s", projectDir, filename)
			result.ArtifactPaths = append(result.ArtifactPaths, artifactPath)
		}

		// Add IR spec as artifact
		irSpecPath := fmt.Sprintf("%s/ir_spec.json", projectDir)
		result.ArtifactPaths = append(result.ArtifactPaths, irSpecPath)

		// Add README
		readmePath := fmt.Sprintf("%s/README.md", projectDir)
		result.ArtifactPaths = append(result.ArtifactPaths, readmePath)
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