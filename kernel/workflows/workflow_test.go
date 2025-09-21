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

	result, err := ParseBriefActivity(ctx, brief, config)

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

	result, err := GenerateCodeActivity(ctx, irSpec, map[string]interface{}{})

	require.NoError(t, err)
	assert.True(t, result.Success)
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

	result, err := PackageArtifactsActivity(ctx, generatedCode, irSpec, projectID)

	require.NoError(t, err)
	assert.True(t, result.Success)
	assert.NotEmpty(t, result.ArtifactPaths)

	// Should include generated files + IR spec + README
	assert.GreaterOrEqual(t, len(result.ArtifactPaths), 4)

	// Check that project ID is included in paths
	for _, path := range result.ArtifactPaths {
		assert.Contains(t, path, projectID)
	}
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