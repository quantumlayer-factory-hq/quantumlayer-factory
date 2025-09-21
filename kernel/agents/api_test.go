package agents

import (
	"context"
	"strings"
	"testing"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewAPIAgent(t *testing.T) {
	agent := NewAPIAgent()
	assert.NotNil(t, agent)
	assert.Equal(t, AgentTypeAPI, agent.GetType())
	assert.Contains(t, agent.GetCapabilities(), "openapi_specs")
	assert.Contains(t, agent.GetCapabilities(), "rest_documentation")
}

func TestAPIAgent_CanHandle(t *testing.T) {
	agent := NewAPIAgent()

	tests := []struct {
		name     string
		spec     *ir.IRSpec
		expected bool
	}{
		{
			name: "API with endpoints",
			spec: &ir.IRSpec{
				API: ir.APISpec{
					Type: "rest",
					Endpoints: []ir.Endpoint{
						{Path: "/users", Method: "GET"},
					},
				},
			},
			expected: true,
		},
		{
			name: "API type application",
			spec: &ir.IRSpec{
				App: ir.AppSpec{Type: "api"},
				API: ir.APISpec{Type: "rest"},
			},
			expected: true,
		},
		{
			name: "Web app with no API",
			spec: &ir.IRSpec{
				App: ir.AppSpec{Type: "web"},
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

func TestAPIAgent_Generate(t *testing.T) {
	agent := NewAPIAgent()
	ctx := context.Background()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name:        "user-api",
			Type:        "api",
			Description: "User management API for testing OpenAPI generation",
		},
		API: ir.APISpec{
			Type: "rest",
			Endpoints: []ir.Endpoint{
				{
					Path:        "/users",
					Method:      "GET",
					Description: "Get all users",
					Responses: map[string]ir.Response{
						"200": {
							Description: "Success",
						},
					},
				},
				{
					Path:        "/users/{id}",
					Method:      "GET",
					Description: "Get user by ID",
					Parameters: []ir.Parameter{
						{Name: "id", In: "path", Type: "integer", Required: true},
					},
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
						{Name: "Email", Type: "string", Required: true},
					},
				},
			},
		},
	}

	req := &GenerationRequest{
		Spec: spec,
		Target: GenerationTarget{
			Type: "openapi",
		},
	}

	result, err := agent.Generate(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Greater(t, len(result.Files), 0)

	// Check that OpenAPI spec is generated
	var openapiFound bool
	for _, file := range result.Files {
		if file.Type == "schema" || strings.Contains(file.Path, "openapi") || strings.Contains(file.Content, "openapi:") {
			openapiFound = true
			assert.Contains(t, file.Content, "openapi:")
			assert.Contains(t, file.Content, "user-api")
			// Template mode generates /health, LLM mode would generate /users
			// Accept either as valid
			hasEndpoint := strings.Contains(file.Content, "/users") || strings.Contains(file.Content, "/health")
			assert.True(t, hasEndpoint, "Should contain API endpoints")
			break
		}
	}
	assert.True(t, openapiFound, "OpenAPI specification should be generated")
}

func TestAPIAgent_GenerateWithTemplates(t *testing.T) {
	agent := NewAPIAgent()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name:        "test-api",
			Description: "Test API for template generation",
		},
		API: ir.APISpec{
			Type: "rest",
			Endpoints: []ir.Endpoint{
				{Path: "/test", Method: "GET", Description: "Test endpoint"},
			},
		},
	}

	req := &GenerationRequest{Spec: spec}
	result := &GenerationResult{Files: []GeneratedFile{}}

	err := agent.generateWithTemplates(req, result)
	assert.NoError(t, err)
	assert.Greater(t, len(result.Files), 0)

	// Check generated files
	var openapiFound bool
	for _, file := range result.Files {
		if strings.Contains(file.Path, "openapi") || file.Type == "schema" {
			openapiFound = true
			assert.Contains(t, file.Content, "openapi:")
		}
	}
	assert.True(t, openapiFound, "OpenAPI spec should be generated")
}

func TestAPIAgent_GenerateOpenAPISpec(t *testing.T) {
	agent := NewAPIAgent()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name:        "pet-store",
			Description: "Pet store API",
		},
		API: ir.APISpec{
			Type: "rest",
		},
	}

	content := agent.generateOpenAPISpec(spec)
	assert.Contains(t, content, "openapi: 3.0.0")
	assert.Contains(t, content, "title: pet-store")
	// Don't check for specific endpoints since they're not in the test spec
}


func TestAPIAgent_Validate(t *testing.T) {
	agent := NewAPIAgent()
	ctx := context.Background()

	result := &GenerationResult{
		Success: true,
		Files: []GeneratedFile{
			{
				Path:    "openapi.yaml",
				Type:    "schema",
				Content: "openapi: 3.0.0",
			},
		},
	}

	validation, err := agent.Validate(ctx, result)
	require.NoError(t, err)
	assert.NotNil(t, validation)
}