package workflows

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
)

func TestParseActivityDirectly(t *testing.T) {
	ctx := context.Background()
	brief := "Create a Go API with user authentication"
	config := map[string]interface{}{}

	result, err := ParseBriefActivity(ctx, brief, []string{}, config)

	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.NotEmpty(t, result.App.Name)
	assert.Equal(t, "go", result.App.Stack.Backend.Language)
}

func TestValidateActivityDirectly(t *testing.T) {
	ctx := context.Background()

	// Test validation with valid IR
	irSpec := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "test-app",
			Type: "api",
			Stack: ir.TechStack{
				Backend: ir.BackendStack{
					Language:  "go",
					Framework: "gin",
				},
			},
		},
	}

	result, err := ValidateIRActivity(ctx, irSpec)

	require.NoError(t, err)
	assert.True(t, result.Valid)
	assert.Empty(t, result.Errors)
}

func TestValidateActivityWithErrors(t *testing.T) {
	ctx := context.Background()

	// Test validation with invalid IR
	irSpec := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "test-app",
			// Missing required fields
		},
	}

	result, err := ValidateIRActivity(ctx, irSpec)

	require.NoError(t, err)
	assert.False(t, result.Valid)
	assert.NotEmpty(t, result.Errors)
	assert.Contains(t, result.Errors, "Application type is required")
	assert.Contains(t, result.Errors, "Backend language is required")
}

func TestGenerateCodeActivityDirectly(t *testing.T) {
	ctx := context.Background()

	// Test code generation
	irSpec := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "test-app",
			Type: "api",
			Stack: ir.TechStack{
				Backend: ir.BackendStack{
					Language:  "go",
					Framework: "gin",
				},
			},
		},
	}

	result, err := GenerateCodeActivity(ctx, irSpec, []string{}, map[string]interface{}{}, "template", "")

	require.NoError(t, err)
	if !result.Success {
		t.Logf("Generation failed. Errors: %v, Warnings: %v", result.Errors, result.Warnings)
	}
	assert.True(t, result.Success, "Generation should succeed")
	// Backend agent will generate code when fully implemented
	// For now we just verify the activity completes successfully
}

func TestVerifyCodeActivityDirectly(t *testing.T) {
	ctx := context.Background()

	// Test code verification with simple Go code
	generatedCode := map[string]string{
		"main.go": `package main

import "fmt"

func main() {
	fmt.Println("Hello, World!")
}`,
	}

	result, err := VerifyCodeActivity(ctx, generatedCode, map[string]interface{}{})

	require.NoError(t, err)
	// Verification should succeed for basic code
	assert.True(t, result.Success)
}

func TestPackageArtifactsActivityDirectly(t *testing.T) {
	ctx := context.Background()

	generatedCode := map[string]string{
		"main.go":    "package main\n\nfunc main() {}",
		"config.yml": "port: 8080",
	}

	irSpec := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "test-app",
		},
	}

	projectID := "test-project-123"

	// Test normal mode
	result, err := PackageArtifactsActivity(ctx, generatedCode, irSpec, projectID, false, "./generated")
	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.ArtifactPaths)
	assert.Contains(t, result.OutputPath, projectID)

	// Test dry-run mode
	dryResult, err := PackageArtifactsActivity(ctx, generatedCode, irSpec, projectID, true, "./generated")
	require.NoError(t, err)
	assert.True(t, dryResult.Success)
	assert.NotEmpty(t, dryResult.SOCPatch)
	assert.Contains(t, dryResult.SOCPatch, "=== Generated Code (Dry Run) ===")
	assert.Equal(t, "(dry-run - no files written)", dryResult.OutputPath)
}

func TestLanguageDetection(t *testing.T) {
	tests := []struct {
		filename string
		expected string
	}{
		{"main.go", "go"},
		{"app.py", "python"},
		{"index.js", "javascript"},
		{"component.ts", "javascript"},
		{"App.java", "java"},
		{"script.rb", "ruby"},
		{"unknown.xyz", "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			result := detectLanguage(tt.filename)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestWorkflowTypes(t *testing.T) {
	// Test that workflow input/output types work correctly
	input := FactoryWorkflowInput{
		Brief:       "Create a simple API",
		ProjectID:   "test-123",
		RequesterID: "user-456",
		Config: map[string]interface{}{
			"dry_run": true,
		},
	}

	assert.Equal(t, "Create a simple API", input.Brief)
	assert.Equal(t, "test-123", input.ProjectID)
	assert.Equal(t, "user-456", input.RequesterID)
	assert.True(t, input.Config["dry_run"].(bool))

	// Test result type
	result := FactoryWorkflowResult{
		Success:   true,
		ProjectID: input.ProjectID,
		GeneratedCode: map[string]string{
			"main.go": "package main",
		},
		Artifacts: []string{"main.go"},
	}

	assert.True(t, result.Success)
	assert.Equal(t, input.ProjectID, result.ProjectID)
	assert.Len(t, result.GeneratedCode, 1)
	assert.Len(t, result.Artifacts, 1)
}