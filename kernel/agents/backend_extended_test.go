package agents

import (
	"testing"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/stretchr/testify/assert"
)

func TestBackendAgent_ConvertToPythonType(t *testing.T) {
	agent := NewBackendAgent()

	tests := []struct {
		input    string
		expected string
	}{
		{"string", "str"},
		{"int", "int"},
		{"float", "float"},
		{"bool", "bool"},
		{"boolean", "bool"},
		{"unknown", "str"},
		{"", "str"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := agent.convertToPythonType(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBackendAgent_GenerateFunctionName(t *testing.T) {
	agent := NewBackendAgent()

	tests := []struct {
		path     string
		method   string
		expected string
	}{
		{"/users", "GET", "get_users"},
		{"/users/{id}", "GET", "get_users_id"},
		{"/api/v1/products", "POST", "post_api_v1_products"},
		{"/health", "GET", "get_health"},
	}

	for _, tt := range tests {
		t.Run(tt.path+"_"+tt.method, func(t *testing.T) {
			result := agent.generateFunctionName(tt.path, tt.method)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBackendAgent_CountLinesOfCode(t *testing.T) {
	agent := NewBackendAgent()

	tests := []struct {
		files    []GeneratedFile
		expected int
	}{
		{[]GeneratedFile{}, 0},
		{[]GeneratedFile{{Type: "source", Content: "line1"}}, 1},
		{[]GeneratedFile{{Type: "source", Content: "line1\nline2"}}, 2},
		{[]GeneratedFile{{Type: "source", Content: "line1"}, {Type: "source", Content: "line2\nline3"}}, 3},
	}

	for _, tt := range tests {
		t.Run("", func(t *testing.T) {
			result := agent.countLinesOfCode(tt.files)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestBackendAgent_ValidateFile(t *testing.T) {
	agent := NewBackendAgent()

	tests := []struct {
		name        string
		file        GeneratedFile
		expectedErr bool
	}{
		{
			name: "Valid Python file",
			file: GeneratedFile{
				Language: "python",
				Content:  "def hello(): pass",
			},
			expectedErr: false,
		},
		{
			name: "Valid Go file",
			file: GeneratedFile{
				Language: "go",
				Content:  "package main\nfunc main() {}",
			},
			expectedErr: false,
		},
		{
			name: "Empty content",
			file: GeneratedFile{
				Language: "python",
				Content:  "",
			},
			expectedErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			validation := &ValidationResult{}
			err := agent.validateFile(tt.file, validation)
			if tt.expectedErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestBackendAgent_GeneratePythonRequirements(t *testing.T) {
	agent := NewBackendAgent()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Stack: ir.TechStack{
				Backend: ir.BackendStack{
					Framework: "fastapi",
				},
			},
		},
	}

	content := agent.generatePythonRequirements(spec)
	assert.Contains(t, content, "fastapi")
	assert.Contains(t, content, "uvicorn")
}