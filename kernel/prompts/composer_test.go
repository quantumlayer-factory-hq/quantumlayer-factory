package prompts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/overlays"
)

func TestPromptComposer(t *testing.T) {
	config := DefaultComposerConfig()
	composer := NewPromptComposer(config)

	baseIR := createTestIRSpec()

	t.Run("ComposeBasicPrompt", func(t *testing.T) {
		req := ComposeRequest{
			AgentType: "backend",
			IRSpec:    baseIR,
		}

		result, err := composer.ComposePrompt(req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "backend", result.AgentType)
		assert.Greater(t, len(result.Prompt), 0)
		assert.Greater(t, result.TotalLength, 0)
	})

	t.Run("ComposeWithOverlays", func(t *testing.T) {
		overlayResult := createTestOverlayResult()

		req := ComposeRequest{
			AgentType:     "backend",
			IRSpec:        baseIR,
			OverlayResult: overlayResult,
		}

		result, err := composer.ComposePrompt(req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Contains(t, result.AppliedOverlays, "test-overlay")
		assert.Equal(t, 1, result.EnhancementsCount)
		assert.Contains(t, result.Prompt, "Test enhancement content")
	})

	t.Run("ComposeWithCustomPrompts", func(t *testing.T) {
		req := ComposeRequest{
			AgentType:     "backend",
			IRSpec:        baseIR,
			CustomPrompts: []string{"Custom instruction 1", "Custom instruction 2"},
		}

		result, err := composer.ComposePrompt(req)
		require.NoError(t, err)
		assert.Contains(t, result.Prompt, "Custom instruction 1")
		assert.Contains(t, result.Prompt, "Custom instruction 2")
	})

	t.Run("ComposeWithTemplateOverrides", func(t *testing.T) {
		req := ComposeRequest{
			AgentType: "backend",
			IRSpec:    baseIR,
			TemplateOverrides: map[string]string{
				"system": "Custom system prompt",
			},
		}

		result, err := composer.ComposePrompt(req)
		require.NoError(t, err)
		assert.NotNil(t, result)
		// Template overrides are applied (basic implementation for now)
	})

	t.Run("ComposeWithMetadata", func(t *testing.T) {
		config := DefaultComposerConfig()
		config.IncludeMetadata = true
		composer := NewPromptComposer(config)

		req := ComposeRequest{
			AgentType: "backend",
			IRSpec:    baseIR,
		}

		result, err := composer.ComposePrompt(req)
		require.NoError(t, err)
		assert.NotNil(t, result.Metadata)
		assert.Equal(t, "backend", result.Metadata["agent_type"])
		assert.Equal(t, "test-app", result.Metadata["app_name"])
	})

	t.Run("LengthWarning", func(t *testing.T) {
		config := DefaultComposerConfig()
		config.MaxPromptLength = 100 // Very small limit
		composer := NewPromptComposer(config)

		req := ComposeRequest{
			AgentType: "backend",
			IRSpec:    baseIR,
		}

		result, err := composer.ComposePrompt(req)
		require.NoError(t, err)
		assert.Len(t, result.Warnings, 1)
		assert.Contains(t, result.Warnings[0], "exceeds recommended maximum")
	})
}

func TestTemplateContext(t *testing.T) {
	config := DefaultComposerConfig()
	composer := NewPromptComposer(config)

	baseIR := createTestIRSpec()
	overlayResult := createTestOverlayResult()

	req := ComposeRequest{
		AgentType:     "backend",
		IRSpec:        baseIR,
		OverlayResult: overlayResult,
		Context: map[string]interface{}{
			"custom_key": "custom_value",
		},
	}

	ctx := composer.buildTemplateContext(req)

	t.Run("BasicContext", func(t *testing.T) {
		assert.Equal(t, "backend", ctx["AgentType"])
		assert.Equal(t, baseIR.App, ctx["App"])
		assert.Equal(t, baseIR.App.Features, ctx["Features"])
		assert.Equal(t, baseIR.App.Stack, ctx["Stack"])
		assert.Equal(t, "custom_value", ctx["Context"].(map[string]interface{})["custom_key"])
	})

	t.Run("OverlayContext", func(t *testing.T) {
		assert.NotNil(t, ctx["Overlays"])
		assert.NotNil(t, ctx["ValidationRules"])
		assert.NotNil(t, ctx["OverlayFeatures"])
	})
}

func TestOverlayEnhancements(t *testing.T) {
	config := DefaultComposerConfig()
	composer := NewPromptComposer(config)

	basePrompt := "## System\nBase system prompt\n\n## Context\nBase context"

	t.Run("ApplyEnhancements", func(t *testing.T) {
		overlayResult := createTestOverlayResult()
		req := ComposeRequest{
			AgentType:     "backend",
			OverlayResult: overlayResult,
		}

		enhanced, count := composer.applyOverlayEnhancements(basePrompt, req)
		assert.Greater(t, len(enhanced), len(basePrompt))
		assert.Equal(t, 1, count)
		assert.Contains(t, enhanced, "Test enhancement content")
	})

	t.Run("NoEnhancements", func(t *testing.T) {
		req := ComposeRequest{
			AgentType: "backend",
		}

		enhanced, count := composer.applyOverlayEnhancements(basePrompt, req)
		assert.Equal(t, basePrompt, enhanced)
		assert.Equal(t, 0, count)
	})

	t.Run("ConditionalEnhancements", func(t *testing.T) {
		overlayResult := &overlays.ResolverResult{
			PromptChanges: map[string][]overlays.PromptEnhancement{
				"backend": {
					{
						AgentType: "backend",
						Section:   "system",
						Content:   "Conditional enhancement",
						Position:  "before",
						Priority:  overlays.PriorityMedium,
						Conditions: map[string]string{
							"language": "go",
						},
					},
				},
			},
		}

		req := ComposeRequest{
			AgentType:     "backend",
			IRSpec:        createTestIRSpec(),
			OverlayResult: overlayResult,
		}

		enhanced, count := composer.applyOverlayEnhancements(basePrompt, req)
		assert.Greater(t, len(enhanced), len(basePrompt))
		assert.Equal(t, 1, count)
		assert.Contains(t, enhanced, "Conditional enhancement")
	})
}

func TestSectionParsing(t *testing.T) {
	config := DefaultComposerConfig()
	composer := NewPromptComposer(config)

	prompt := `## System
System content here

## Context
Context content here

## Examples
Example content here`

	sections := composer.parseSections(prompt)

	t.Run("ParseSections", func(t *testing.T) {
		assert.Len(t, sections, 3)
		assert.Contains(t, sections["system"], "System content here")
		assert.Contains(t, sections["context"], "Context content here")
		assert.Contains(t, sections["examples"], "Example content here")
	})

	t.Run("ReconstructPrompt", func(t *testing.T) {
		reconstructed := composer.reconstructPrompt(sections)
		assert.Contains(t, reconstructed, "## System")
		assert.Contains(t, reconstructed, "## Context")
		assert.Contains(t, reconstructed, "## Examples")
	})
}

func TestEnhancementPositions(t *testing.T) {
	config := DefaultComposerConfig()
	composer := NewPromptComposer(config)

	sections := map[string]string{
		"system": "Original system content",
	}

	t.Run("BeforePosition", func(t *testing.T) {
		enhancement := overlays.PromptEnhancement{
			Section:  "system",
			Content:  "Before content",
			Position: "before",
		}

		result := composer.applyEnhancementToSection(sections, enhancement)
		assert.Contains(t, result["system"], "Before content")
		assert.True(t, strings.Index(result["system"], "Before content") < strings.Index(result["system"], "Original system content"))
	})

	t.Run("AfterPosition", func(t *testing.T) {
		enhancement := overlays.PromptEnhancement{
			Section:  "system",
			Content:  "After content",
			Position: "after",
		}

		result := composer.applyEnhancementToSection(sections, enhancement)
		assert.Contains(t, result["system"], "After content")
		assert.True(t, strings.Index(result["system"], "Original system content") < strings.Index(result["system"], "After content"))
	})

	t.Run("ReplacePosition", func(t *testing.T) {
		enhancement := overlays.PromptEnhancement{
			Section:  "system",
			Content:  "Replacement content",
			Position: "replace",
		}

		result := composer.applyEnhancementToSection(sections, enhancement)
		assert.Equal(t, "Replacement content", result["system"])
		assert.NotContains(t, result["system"], "Original system content")
	})
}

func TestConditionMatching(t *testing.T) {
	config := DefaultComposerConfig()
	composer := NewPromptComposer(config)

	baseIR := createTestIRSpec()

	t.Run("LanguageCondition", func(t *testing.T) {
		enhancement := overlays.PromptEnhancement{
			Conditions: map[string]string{
				"language": "go",
			},
		}

		req := ComposeRequest{
			AgentType: "backend",
			IRSpec:    baseIR,
		}

		matches := composer.matchesConditions(enhancement, req)
		assert.True(t, matches)
	})

	t.Run("FailedCondition", func(t *testing.T) {
		enhancement := overlays.PromptEnhancement{
			Conditions: map[string]string{
				"language": "python",
			},
		}

		req := ComposeRequest{
			AgentType: "backend",
			IRSpec:    baseIR,
		}

		matches := composer.matchesConditions(enhancement, req)
		assert.False(t, matches)
	})

	t.Run("CustomContextCondition", func(t *testing.T) {
		enhancement := overlays.PromptEnhancement{
			Conditions: map[string]string{
				"custom_key": "custom_value",
			},
		}

		req := ComposeRequest{
			AgentType: "backend",
			IRSpec:    baseIR,
			Context: map[string]interface{}{
				"custom_key": "custom_value",
			},
		}

		matches := composer.matchesConditions(enhancement, req)
		assert.True(t, matches)
	})
}

func TestTemplateManager(t *testing.T) {
	tempDir := t.TempDir()

	// Create test template
	templateContent := `## System
Test template for {{.AgentType}}

App: {{.App.Name}}
Language: {{.Stack.Backend.Language}}`

	templateFile := filepath.Join(tempDir, "test.tmpl")
	err := os.WriteFile(templateFile, []byte(templateContent), 0644)
	require.NoError(t, err)

	tm := NewTemplateManager([]string{tempDir})

	t.Run("LoadExistingTemplate", func(t *testing.T) {
		tmpl, err := tm.GetTemplate("test")
		require.NoError(t, err)
		assert.NotNil(t, tmpl)
	})

	t.Run("LoadNonexistentTemplate", func(t *testing.T) {
		tmpl, err := tm.GetTemplate("nonexistent")
		require.NoError(t, err)
		assert.NotNil(t, tmpl) // Should return basic template
	})

	t.Run("ListAvailableTemplates", func(t *testing.T) {
		templates, err := tm.ListAvailableTemplates()
		require.NoError(t, err)
		assert.Contains(t, templates, "test")
	})

	t.Run("ValidateTemplate", func(t *testing.T) {
		err := tm.ValidateTemplate("test")
		assert.NoError(t, err)
	})

	t.Run("ReloadTemplates", func(t *testing.T) {
		// Load template first
		_, err := tm.GetTemplate("test")
		require.NoError(t, err)

		// Reload should clear cache
		tm.ReloadTemplates()

		// Should still work
		_, err = tm.GetTemplate("test")
		require.NoError(t, err)
	})
}

func TestTemplateFunctions(t *testing.T) {
	t.Run("FormatList", func(t *testing.T) {
		items := []interface{}{"item1", "item2", "item3"}
		result := formatList(items)
		assert.Contains(t, result, "- item1")
		assert.Contains(t, result, "- item2")
		assert.Contains(t, result, "- item3")
	})

	t.Run("FormatCode", func(t *testing.T) {
		code := "fmt.Println(\"hello\")"
		result := formatCode(code, "go")
		assert.Contains(t, result, "```go")
		assert.Contains(t, result, code)
		assert.Contains(t, result, "```")
	})

	t.Run("Indent", func(t *testing.T) {
		text := "line1\nline2\nline3"
		result := indent(text, 4)
		lines := strings.Split(result, "\n")
		for _, line := range lines {
			if strings.TrimSpace(line) != "" {
				assert.True(t, strings.HasPrefix(line, "    "))
			}
		}
	})
}

// Helper functions for tests

func createTestIRSpec() *ir.IRSpec {
	return &ir.IRSpec{
		App: ir.AppSpec{
			Name:   "test-app",
			Type:   "api",
			Domain: "testing",
			Features: []ir.Feature{
				{
					Name:        "authentication",
					Description: "User authentication system",
					Type:        "security",
					Priority:    "high",
				},
			},
			Stack: ir.TechStack{
				Backend: ir.BackendStack{
					Language:  "go",
					Framework: "gin",
				},
				Database: ir.DatabaseStack{
					Type: "postgresql",
				},
			},
		},
	}
}

func createTestOverlayResult() *overlays.ResolverResult {
	return &overlays.ResolverResult{
		ResolvedOverlays: []string{"test-overlay"},
		AppliedOrder:     []string{"test-overlay"},
		PromptChanges: map[string][]overlays.PromptEnhancement{
			"backend": {
				{
					AgentType: "backend",
					Section:   "system",
					Content:   "Test enhancement content",
					Position:  "before",
					Priority:  overlays.PriorityMedium,
				},
			},
		},
		ValidationRules: []overlays.ValidationRule{
			{
				Name:     "test_rule",
				Type:     "validation",
				Severity: "warning",
				Message:  "Test validation rule",
			},
		},
		Metadata: map[string]overlays.OverlayMetadata{
			"test-overlay": {
				Name:        "test-overlay",
				Version:     "1.0.0",
				Type:        overlays.OverlayTypeDomain,
				Description: "Test overlay",
			},
		},
	}
}