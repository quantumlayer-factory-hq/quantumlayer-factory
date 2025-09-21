package agents

import (
	"context"
	"testing"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewFrontendAgent(t *testing.T) {
	agent := NewFrontendAgent()
	assert.NotNil(t, agent)
	assert.Equal(t, AgentTypeFrontend, agent.GetType())
	assert.Contains(t, agent.GetCapabilities(), "react_components")
	assert.Contains(t, agent.GetCapabilities(), "react_components")
}

func TestFrontendAgent_CanHandle(t *testing.T) {
	agent := NewFrontendAgent()

	tests := []struct {
		name     string
		spec     *ir.IRSpec
		expected bool
	}{
		{
			name: "SPA application",
			spec: &ir.IRSpec{
				App: ir.AppSpec{Type: "spa"},
			},
			expected: true,
		},
		{
			name: "Web application with UI spec",
			spec: &ir.IRSpec{
				App: ir.AppSpec{Type: "web"},
				UI: ir.UISpec{
					Pages: []ir.Page{
						{Name: "Home", Path: "/"},
					},
				},
			},
			expected: true,
		},
		{
			name: "API only application",
			spec: &ir.IRSpec{
				App: ir.AppSpec{Type: "api"},
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

func TestFrontendAgent_Generate(t *testing.T) {
	agent := NewFrontendAgent()
	ctx := context.Background()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "test-spa",
			Type: "spa",
			Stack: ir.TechStack{
				Frontend: ir.FrontendStack{
					Framework: "react",
					Language:  "typescript",
				},
			},
		},
		UI: ir.UISpec{
			Pages: []ir.Page{
				{
					Name: "Home",
					Path: "/",
					Title: "Home Page",
				},
			},
		},
	}

	req := &GenerationRequest{
		Spec: spec,
		Target: GenerationTarget{
			Language:  "typescript",
			Framework: "react",
		},
	}

	result, err := agent.Generate(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// In template mode (no LLM), agent returns warning but no files
	if len(result.Files) == 0 {
		assert.Contains(t, result.Warnings, "Frontend agent running in template mode - limited functionality")
	} else {
		// If files are generated, check they're frontend-related
		var frontendFileFound bool
		for _, file := range result.Files {
			if file.Type == "source" && (file.Language == "typescript" || file.Language == "javascript") {
				frontendFileFound = true
				break
			}
		}
		assert.True(t, frontendFileFound, "Frontend code should be generated")
	}
}

func TestFrontendAgent_Validate(t *testing.T) {
	agent := NewFrontendAgent()
	ctx := context.Background()

	result := &GenerationResult{
		Success: true,
		Files: []GeneratedFile{
			{
				Path:     "src/App.tsx",
				Type:     "component",
				Language: "typescript",
				Content:  "import React from 'react';\n\nfunction App() {\n  return <div>Hello World</div>;\n}",
			},
		},
	}

	validation, err := agent.Validate(ctx, result)
	require.NoError(t, err)
	assert.NotNil(t, validation)
	// Frontend validation is not yet implemented, so expect false and warnings
	assert.False(t, validation.Valid)
	assert.Contains(t, validation.Warnings, "Frontend validation not yet implemented")
}