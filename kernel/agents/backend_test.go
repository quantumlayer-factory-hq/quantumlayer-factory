package agents

import (
	"context"
	"strings"
	"testing"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBackendAgent(t *testing.T) {
	agent := NewBackendAgent()
	assert.NotNil(t, agent)
	assert.Equal(t, AgentTypeBackend, agent.GetType())
	assert.Contains(t, agent.GetCapabilities(), "api_controllers")
	assert.Contains(t, agent.GetCapabilities(), "data_models")
}

func TestBackendAgent_CanHandle(t *testing.T) {
	agent := NewBackendAgent()

	tests := []struct {
		name     string
		spec     *ir.IRSpec
		expected bool
	}{
		{
			name: "API spec with backend language",
			spec: &ir.IRSpec{
				App: ir.AppSpec{
					Type: "api",
					Stack: ir.TechStack{
						Backend: ir.BackendStack{
							Language:  "go",
							Framework: "gin",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Web app with backend",
			spec: &ir.IRSpec{
				App: ir.AppSpec{
					Type: "web",
					Stack: ir.TechStack{
						Backend: ir.BackendStack{
							Language: "python",
						},
					},
				},
			},
			expected: true,
		},
		{
			name: "Frontend only app",
			spec: &ir.IRSpec{
				App: ir.AppSpec{
					Type: "spa",
					Stack: ir.TechStack{
						Frontend: ir.FrontendStack{
							Framework: "react",
						},
					},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.CanHandle(tt.spec)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBackendAgent_Generate(t *testing.T) {
	agent := NewBackendAgent()
	ctx := context.Background()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name:        "test-api",
			Type:        "api",
			Description: "Test API service for Python backend",
			Stack: ir.TechStack{
				Backend: ir.BackendStack{
					Language:  "python",
					Framework: "fastapi",
				},
			},
		},
		API: ir.APISpec{
			Type: "rest",
			Endpoints: []ir.Endpoint{
				{
					Path:        "/users",
					Method:      "GET",
					Description: "Get all users",
				},
			},
		},
		Data: ir.DataSpec{
			Entities: []ir.Entity{
				{
					Name: "User",
					Fields: []ir.Field{
						{Name: "ID", Type: "int", Required: true},
						{Name: "Name", Type: "string", Required: true},
					},
				},
			},
		},
	}

	req := &GenerationRequest{
		Spec: spec,
		Target: GenerationTarget{
			Language:  "python",
			Framework: "fastapi",
		},
		Context: map[string]interface{}{
			"output_dir": "test-output",
		},
	}

	result, err := agent.Generate(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Greater(t, len(result.Files), 0)

	// Check that some Python code is generated
	var pythonFileFound bool
	for _, file := range result.Files {
		if file.Language == "python" || strings.HasSuffix(file.Path, ".py") {
			pythonFileFound = true
			assert.Contains(t, file.Content, "FastAPI")
			break
		}
	}
	assert.True(t, pythonFileFound, "Python code should be generated")
}



func TestBackendAgent_Validate(t *testing.T) {
	agent := NewBackendAgent()
	ctx := context.Background()

	result := &GenerationResult{
		Success: true,
		Files: []GeneratedFile{
			{
				Path:     "main.go",
				Type:     "source",
				Language: "go",
				Content:  "package main\n\nfunc main() {\n\t// TODO: implement\n}",
			},
		},
	}

	validationResult, err := agent.Validate(ctx, result)
	require.NoError(t, err)
	assert.NotNil(t, validationResult)
	assert.True(t, validationResult.Valid)

	// Test validation with no files - this is actually valid (empty results are allowed)
	emptyResult := &GenerationResult{
		Success: true,
		Files:   []GeneratedFile{},
	}

	validationResult, err = agent.Validate(ctx, emptyResult)
	require.NoError(t, err)
	assert.True(t, validationResult.Valid)
	assert.Equal(t, 0, len(validationResult.Errors))
}

func TestBackendAgent_GetFileExtension(t *testing.T) {
	agent := NewBackendAgent()

	tests := []struct {
		language string
		expected string
	}{
		{"go", ".go"},
		{"python", ".py"},
		{"javascript", ".js"},
		{"typescript", ".ts"},
		{"java", ".java"},
		{"csharp", ".cs"},
		{"unknown", ".txt"},
	}

	for _, tt := range tests {
		t.Run(tt.language, func(t *testing.T) {
			result := agent.getFileExtension(tt.language)
			assert.Equal(t, tt.expected, result)
		})
	}
}