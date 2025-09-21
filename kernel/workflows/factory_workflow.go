package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
)

// FactoryWorkflowInput represents the input for the factory workflow
type FactoryWorkflowInput struct {
	Brief        string                 `json:"brief"`
	Config       map[string]interface{} `json:"config,omitempty"`
	RequesterID  string                 `json:"requester_id,omitempty"`
	ProjectID    string                 `json:"project_id,omitempty"`
	DryRun       bool                   `json:"dry_run,omitempty"`
	Verbose      bool                   `json:"verbose,omitempty"`
	OutputDir    string                 `json:"output_dir,omitempty"`
	Overlays     []string               `json:"overlays,omitempty"`
	Provider     string                 `json:"provider,omitempty"`
	Model        string                 `json:"model,omitempty"`
	Compare      bool                   `json:"compare,omitempty"`
}

// CLI-compatible aliases for easier integration
type FactoryInput = FactoryWorkflowInput

// FactoryWorkflowResult represents the result of the factory workflow
type FactoryWorkflowResult struct {
	Success       bool              `json:"success"`
	ProjectID     string            `json:"project_id"`
	IRSpec        *ir.IRSpec        `json:"ir_spec,omitempty"`
	GeneratedCode map[string]string `json:"generated_code,omitempty"`
	Artifacts     []string          `json:"artifacts"`
	Errors        []string          `json:"errors,omitempty"`
	Warnings      []string          `json:"warnings,omitempty"`
	Duration      time.Duration     `json:"duration"`
	// CLI-specific fields
	SOCPatch      string `json:"soc_patch,omitempty"`   // for dry-run output
	OutputPath    string `json:"output_path,omitempty"` // where files were written
	Summary       string `json:"summary,omitempty"`     // human-readable summary
}

// CLI-compatible aliases for easier integration
type FactoryResult = FactoryWorkflowResult

// FactoryWorkflow orchestrates the entire code generation process
func FactoryWorkflow(ctx workflow.Context, input FactoryWorkflowInput) (*FactoryWorkflowResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting FactoryWorkflow", "brief", input.Brief)

	startTime := workflow.Now(ctx)
	result := &FactoryWorkflowResult{
		Success:       true,
		ProjectID:     input.ProjectID,
		GeneratedCode: make(map[string]string),
		Artifacts:     []string{},
		Errors:        []string{},
		Warnings:      []string{},
	}

	// Set workflow timeout
	ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		HeartbeatTimeout:    30 * time.Second,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    100 * time.Second,
			MaximumAttempts:    3,
		},
	})

	// Step 1: Parse brief into IR (Intermediate Representation)
	logger.Info("Step 1: Parsing brief into IR")
	var irSpec *ir.IRSpec
	err := workflow.ExecuteActivity(ctx, ParseBriefActivity, input.Brief, input.Overlays, input.Config).Get(ctx, &irSpec)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, "Failed to parse brief: "+err.Error())
		result.Duration = workflow.Now(ctx).Sub(startTime)
		return result, nil
	}
	result.IRSpec = irSpec
	logger.Info("Successfully parsed brief into IR", "app_name", irSpec.App.Name)

	// Step 2: Validate IR specification
	logger.Info("Step 2: Validating IR specification")
	var validationResult ValidationResult
	err = workflow.ExecuteActivity(ctx, ValidateIRActivity, irSpec).Get(ctx, &validationResult)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, "Failed to validate IR: "+err.Error())
		result.Duration = workflow.Now(ctx).Sub(startTime)
		return result, nil
	}

	if !validationResult.Valid {
		result.Success = false
		result.Errors = append(result.Errors, validationResult.Errors...)
		result.Warnings = append(result.Warnings, validationResult.Warnings...)
		result.Duration = workflow.Now(ctx).Sub(startTime)
		return result, nil
	}
	result.Warnings = append(result.Warnings, validationResult.Warnings...)

	// Step 3: Generate code using agents
	logger.Info("Step 3: Generating code with agents", "provider", input.Provider, "model", input.Model)
	var codeGenResult CodeGenerationResult
	err = workflow.ExecuteActivity(ctx, GenerateCodeActivity, irSpec, input.Overlays, input.Config, input.Provider, input.Model).Get(ctx, &codeGenResult)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, "Failed to generate code: "+err.Error())
		result.Duration = workflow.Now(ctx).Sub(startTime)
		return result, nil
	}

	if !codeGenResult.Success {
		result.Success = false
		result.Errors = append(result.Errors, codeGenResult.Errors...)
	}
	result.GeneratedCode = codeGenResult.GeneratedCode
	result.Warnings = append(result.Warnings, codeGenResult.Warnings...)
	result.Artifacts = append(result.Artifacts, codeGenResult.Artifacts...)

	// Step 4: Verify generated code (static analysis)
	if result.Success && len(result.GeneratedCode) > 0 {
		logger.Info("Step 4: Verifying generated code")
		var verificationResult VerificationResult
		err = workflow.ExecuteActivity(ctx, VerifyCodeActivity, result.GeneratedCode, input.Config).Get(ctx, &verificationResult)
		if err != nil {
			result.Warnings = append(result.Warnings, "Code verification failed: "+err.Error())
		} else {
			if !verificationResult.Success {
				result.Warnings = append(result.Warnings, "Code verification found issues")
				result.Warnings = append(result.Warnings, verificationResult.Issues...)
			}
			result.Artifacts = append(result.Artifacts, verificationResult.Reports...)
		}
	}

	// Step 5: Package and store artifacts
	logger.Info("Step 5: Packaging artifacts")
	var packageResult PackageResult
	err = workflow.ExecuteActivity(ctx, PackageArtifactsActivity, result.GeneratedCode, result.IRSpec, input.ProjectID, input.DryRun, input.OutputDir).Get(ctx, &packageResult)
	if err != nil {
		result.Warnings = append(result.Warnings, "Failed to package artifacts: "+err.Error())
	} else {
		result.Artifacts = append(result.Artifacts, packageResult.ArtifactPaths...)
		result.OutputPath = packageResult.OutputPath
		result.SOCPatch = packageResult.SOCPatch
	}

	// Generate summary
	if input.DryRun {
		result.Summary = fmt.Sprintf("Dry run completed. Generated %d files for %s application.",
			len(result.GeneratedCode), result.IRSpec.App.Type)
	} else {
		result.Summary = fmt.Sprintf("Successfully generated %s application with %d files.",
			result.IRSpec.App.Type, len(result.GeneratedCode))
	}

	result.Duration = workflow.Now(ctx).Sub(startTime)
	logger.Info("FactoryWorkflow completed", "success", result.Success, "duration", result.Duration)

	return result, nil
}

// ValidationResult represents the result of IR validation
type ValidationResult struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

// CodeGenerationResult represents the result of code generation
type CodeGenerationResult struct {
	Success       bool              `json:"success"`
	GeneratedCode map[string]string `json:"generated_code"`
	Artifacts     []string          `json:"artifacts"`
	Errors        []string          `json:"errors"`
	Warnings      []string          `json:"warnings"`
}

// VerificationResult represents the result of code verification
type VerificationResult struct {
	Success bool     `json:"success"`
	Issues  []string `json:"issues"`
	Reports []string `json:"reports"`
}

// PackageResult represents the result of artifact packaging
type PackageResult struct {
	Success       bool     `json:"success"`
	ArtifactPaths []string `json:"artifact_paths"`
	OutputPath    string   `json:"output_path,omitempty"`  // directory where files were written
	SOCPatch      string   `json:"soc_patch,omitempty"`    // SOC-formatted patch for dry-run
}