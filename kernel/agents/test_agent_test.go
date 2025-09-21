package agents

import (
	"context"
	"testing"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewTestAgent(t *testing.T) {
	agent := NewTestAgent()
	assert.NotNil(t, agent)
	assert.Equal(t, AgentTypeTest, agent.GetType())
	assert.Contains(t, agent.GetCapabilities(), "unit_tests")
	assert.Contains(t, agent.GetCapabilities(), "integration_tests")
	assert.Contains(t, agent.GetCapabilities(), "test_data_generation")
}

func TestTestAgent_CanHandle(t *testing.T) {
	agent := NewTestAgent()

	// Test agent can handle any specification - tests are always beneficial
	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "test-app",
			Type: "api",
		},
	}

	result := agent.CanHandle(spec)
	assert.True(t, result, "Test agent should be able to handle any specification")

	// Test with empty spec
	emptySpec := &ir.IRSpec{}
	result = agent.CanHandle(emptySpec)
	assert.True(t, result, "Test agent should handle empty specs too")
}

func TestTestAgent_Generate(t *testing.T) {
	agent := NewTestAgent()
	ctx := context.Background()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name:        "test-service",
			Type:        "api",
			Description: "Test service for unit testing",
			Stack: ir.TechStack{
				Backend: ir.BackendStack{
					Language:  "go",
					Framework: "gin",
				},
			},
		},
		API: ir.APISpec{
			Type: "rest",
			Endpoints: []ir.Endpoint{
				{
					Path:        "/health",
					Method:      "GET",
					Description: "Health check endpoint",
				},
			},
		},
	}

	req := &GenerationRequest{
		Spec: spec,
		Target: GenerationTarget{
			Language:  "go",
			Framework: "testing",
		},
		Context: map[string]interface{}{
			"test_type": "unit",
		},
	}

	result, err := agent.Generate(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Greater(t, len(result.Files), 0)

	// Check that a test file is generated
	var testFileFound bool
	for _, file := range result.Files {
		if file.Type == "test" {
			testFileFound = true
			assert.Contains(t, file.Content, "testing")
			assert.Contains(t, file.Content, "func Test")
			break
		}
	}
	assert.True(t, testFileFound, "A test file should be generated")
}

func TestTestAgent_GenerateWithTemplates(t *testing.T) {
	agent := NewTestAgent()

	tests := []struct {
		name     string
		language string
		expected bool
	}{
		{"Go tests", "go", true},
		{"Python tests", "python", true},
		{"JavaScript tests", "javascript", true},
		{"TypeScript tests", "typescript", true},
		{"Unknown language", "unknown", true}, // Should still generate with warnings
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &ir.IRSpec{
				App: ir.AppSpec{
					Name: "test-app",
					Stack: ir.TechStack{
						Backend: ir.BackendStack{
							Language: tt.language,
						},
					},
				},
			}

			req := &GenerationRequest{
				Spec: spec,
				Target: GenerationTarget{
					Language: tt.language,
				},
			}

			result := &GenerationResult{
				Files:    []GeneratedFile{},
				Warnings: []string{},
			}

			err := agent.generateWithTemplates(req, result)
			assert.NoError(t, err)

			if tt.language != "unknown" {
				assert.Greater(t, len(result.Files), 0, "Should generate test files for known languages")
			}
		})
	}
}

func TestTestAgent_GenerateGoTests(t *testing.T) {
	agent := NewTestAgent()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "TestApp",
		},
	}

	req := &GenerationRequest{
		Spec: spec,
	}

	result := &GenerationResult{
		Files: []GeneratedFile{},
	}

	agent.generateGoTests(req, result)

	assert.Greater(t, len(result.Files), 0)

	testFile := result.Files[0]
	assert.Equal(t, "tests/main_test.go", testFile.Path)
	assert.Equal(t, "test", testFile.Type)
	assert.Equal(t, "go", testFile.Language)
	assert.Contains(t, testFile.Content, "package main")
	assert.Contains(t, testFile.Content, "import")
	assert.Contains(t, testFile.Content, "testing")
	assert.Contains(t, testFile.Content, "TestTestApp")
}

func TestTestAgent_GeneratePythonTests(t *testing.T) {
	agent := NewTestAgent()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "test_app",
		},
	}

	req := &GenerationRequest{
		Spec: spec,
	}

	result := &GenerationResult{
		Files: []GeneratedFile{},
	}

	agent.generatePythonTests(req, result)

	assert.Greater(t, len(result.Files), 0)

	testFile := result.Files[0]
	assert.Equal(t, "tests/test_main.py", testFile.Path)
	assert.Equal(t, "test", testFile.Type)
	assert.Equal(t, "python", testFile.Language)
	assert.Contains(t, testFile.Content, "import pytest")
	assert.Contains(t, testFile.Content, "import unittest")
	assert.Contains(t, testFile.Content, "class TestTest_app")
}

func TestTestAgent_GenerateJavaScriptTests(t *testing.T) {
	agent := NewTestAgent()

	tests := []struct {
		name     string
		language string
		expected string
	}{
		{"JavaScript tests", "javascript", ".js"},
		{"TypeScript tests", "typescript", ".ts"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &ir.IRSpec{
				App: ir.AppSpec{
					Name: "test-app",
					Stack: ir.TechStack{
						Backend: ir.BackendStack{
							Language: tt.language,
						},
					},
				},
			}

			req := &GenerationRequest{
				Spec: spec,
			}

			result := &GenerationResult{
				Files: []GeneratedFile{},
			}

			agent.generateJavaScriptTests(req, result)

			assert.Greater(t, len(result.Files), 0)

			testFile := result.Files[0]
			assert.Contains(t, testFile.Path, "tests/main.test"+tt.expected)
			assert.Equal(t, "test", testFile.Type)
			assert.Equal(t, tt.language, testFile.Language)
			assert.Contains(t, testFile.Content, "import { describe, it, expect")
			assert.Contains(t, testFile.Content, "describe('test-app'")
		})
	}
}

func TestTestAgent_ParseGeneratedTests(t *testing.T) {
	agent := NewTestAgent()

	tests := []struct {
		name     string
		content  string
		language string
		expected string
	}{
		{
			name:     "Go test content",
			content:  "package main\n\nimport \"testing\"\n\nfunc TestExample(t *testing.T) {\n\t// test code\n}",
			language: "go",
			expected: "tests/unit_test.go",
		},
		{
			name:     "Python test content",
			content:  "import unittest\n\nclass TestExample(unittest.TestCase):\n\tpass",
			language: "python",
			expected: "tests/test_unit.py",
		},
		{
			name:     "JavaScript test content",
			content:  "describe('test', () => {\n\tit('should work', () => {\n\t\texpect(true).toBe(true);\n\t});\n});",
			language: "javascript",
			expected: "tests/unit.test.js",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			spec := &ir.IRSpec{
				App: ir.AppSpec{
					Stack: ir.TechStack{
						Backend: ir.BackendStack{
							Language: tt.language,
						},
					},
				},
			}

			files, err := agent.parseGeneratedTests(tt.content, spec)
			require.NoError(t, err)
			assert.Greater(t, len(files), 0)

			testFile := files[0]
			assert.Equal(t, tt.expected, testFile.Path)
			assert.Equal(t, "test", testFile.Type)
			assert.Equal(t, tt.language, testFile.Language)
			assert.Equal(t, tt.content, testFile.Content)
		})
	}
}

func TestTestAgent_Validate(t *testing.T) {
	agent := NewTestAgent()
	ctx := context.Background()

	t.Run("Valid result with test files", func(t *testing.T) {
		result := &GenerationResult{
			Success: true,
			Files: []GeneratedFile{
				{
					Path: "test_main.py",
					Type: "test",
					Content: "import unittest\n\nclass TestExample(unittest.TestCase):\n\tpass",
				},
			},
		}

		validation, err := agent.Validate(ctx, result)
		require.NoError(t, err)
		assert.True(t, validation.Valid)
		assert.Equal(t, 0.8, validation.Metrics.CodeQuality)
	})

	t.Run("Invalid result with no test files", func(t *testing.T) {
		result := &GenerationResult{
			Success: true,
			Files: []GeneratedFile{
				{
					Path: "main.py",
					Type: "source",
					Content: "print('hello world')",
				},
			},
		}

		validation, err := agent.Validate(ctx, result)
		require.NoError(t, err)
		assert.False(t, validation.Valid)
		assert.Greater(t, len(validation.Errors), 0)
		assert.Equal(t, "missing_tests", validation.Errors[0].Type)
	})
}