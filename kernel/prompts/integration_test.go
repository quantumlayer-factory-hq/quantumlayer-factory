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

func TestPromptComposerIntegration(t *testing.T) {
	// Create resolver with test overlays
	tempDir := t.TempDir()
	createTestOverlayFiles(t, tempDir)

	resolverConfig := overlays.ResolverConfig{
		OverlayPaths:       []string{tempDir},
		ConflictResolution: "priority",
		EnableCaching:      false,
	}
	resolver := overlays.NewFileSystemResolver(resolverConfig)

	// Create prompt composer
	composerConfig := DefaultComposerConfig()
	composer := NewPromptComposer(composerConfig)

	// Create test IR spec
	baseIR := &ir.IRSpec{
		App: ir.AppSpec{
			Name:   "fintech-api",
			Type:   "api",
			Domain: "fintech",
			Features: []ir.Feature{
				{Name: "user-auth", Description: "User authentication", Type: "security", Priority: "high"},
				{Name: "payments", Description: "Payment processing", Type: "business", Priority: "high"},
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

	t.Run("EndToEndPromptGeneration", func(t *testing.T) {
		// Resolve overlays
		overlayResult, err := resolver.ResolveOverlays([]string{"test-fintech", "test-pci"}, baseIR)
		require.NoError(t, err)

		// Compose prompt
		req := ComposeRequest{
			AgentType:     "backend",
			IRSpec:        overlayResult.IRSpec,
			OverlayResult: overlayResult,
			Context: map[string]interface{}{
				"compliance_level": "strict",
			},
		}

		result, err := composer.ComposePrompt(req)
		require.NoError(t, err)

		// Verify basic structure
		assert.Equal(t, "backend", result.AgentType)
		assert.Greater(t, result.TotalLength, 1000) // Should be substantial prompt
		assert.Contains(t, result.AppliedOverlays, "test-fintech")
		assert.Contains(t, result.AppliedOverlays, "test-pci")

		// Verify fintech overlay features were added
		featureNames := []string{}
		for _, feature := range overlayResult.IRSpec.App.Features {
			featureNames = append(featureNames, feature.Name)
		}
		assert.Contains(t, featureNames, "payment_processing")
		assert.Contains(t, featureNames, "fraud_detection")
		assert.Contains(t, featureNames, "card_data_protection")

		// Verify prompt enhancements were applied
		assert.Greater(t, result.EnhancementsCount, 0)
		assert.Contains(t, result.Prompt, "financial services")
		assert.Contains(t, result.Prompt, "PCI DSS")

		// Verify sections are properly structured
		assert.Contains(t, result.Prompt, "## System")
		assert.Contains(t, result.Prompt, "## Context")

		// Check section lengths are calculated
		assert.Greater(t, len(result.SectionLengths), 0)
	})

	t.Run("MultipleAgentTypes", func(t *testing.T) {
		// Resolve overlays once
		overlayResult, err := resolver.ResolveOverlays([]string{"test-fintech"}, baseIR)
		require.NoError(t, err)

		agentTypes := []string{"backend", "frontend", "database"}

		for _, agentType := range agentTypes {
			t.Run(agentType, func(t *testing.T) {
				req := ComposeRequest{
					AgentType:     agentType,
					IRSpec:        overlayResult.IRSpec,
					OverlayResult: overlayResult,
				}

				result, err := composer.ComposePrompt(req)
				require.NoError(t, err)
				assert.Equal(t, agentType, result.AgentType)
				assert.Greater(t, result.TotalLength, 500)
				assert.Contains(t, result.UsedTemplate, agentType)
			})
		}
	})

	t.Run("ConditionalEnhancements", func(t *testing.T) {
		// Create overlay with conditional enhancement
		_ = createConditionalOverlay(t, tempDir)

		overlayResult, err := resolver.ResolveOverlays([]string{"conditional-overlay"}, baseIR)
		require.NoError(t, err)

		req := ComposeRequest{
			AgentType:     "backend",
			IRSpec:        overlayResult.IRSpec,
			OverlayResult: overlayResult,
		}

		result, err := composer.ComposePrompt(req)
		require.NoError(t, err)

		// Should include conditional enhancement since language is "go"
		assert.Contains(t, result.Prompt, "Go-specific enhancement")
	})

	t.Run("PriorityOrdering", func(t *testing.T) {
		// Test that higher priority overlays override lower priority ones
		overlayResult, err := resolver.ResolveOverlays([]string{"test-fintech", "test-pci"}, baseIR)
		require.NoError(t, err)

		req := ComposeRequest{
			AgentType:     "backend",
			IRSpec:        overlayResult.IRSpec,
			OverlayResult: overlayResult,
		}

		result, err := composer.ComposePrompt(req)
		require.NoError(t, err)

		// PCI has higher priority (10) than fintech (5), so should be applied last
		assert.Equal(t, "test-fintech", result.AppliedOverlays[0])
		assert.Equal(t, "test-pci", result.AppliedOverlays[1])

		// Should see both enhancements in the prompt
		assert.Contains(t, result.Prompt, "payment processing")
		assert.Contains(t, result.Prompt, "PCI DSS")
	})

	t.Run("TemplateWithOverlayData", func(t *testing.T) {
		overlayResult, err := resolver.ResolveOverlays([]string{"test-fintech"}, baseIR)
		require.NoError(t, err)

		req := ComposeRequest{
			AgentType:     "backend",
			IRSpec:        overlayResult.IRSpec,
			OverlayResult: overlayResult,
		}

		result, err := composer.ComposePrompt(req)
		require.NoError(t, err)

		// Template should include overlay-added features
		for _, feature := range overlayResult.IRSpec.App.Features {
			if feature.Type == "overlay" {
				assert.Contains(t, result.Prompt, feature.Name)
			}
		}
	})
}

func TestPromptComposerPerformance(t *testing.T) {
	tempDir := t.TempDir()
	createTestOverlayFiles(t, tempDir)

	resolverConfig := overlays.ResolverConfig{
		OverlayPaths:       []string{tempDir},
		ConflictResolution: "priority",
		EnableCaching:      true,
	}
	resolver := overlays.NewFileSystemResolver(resolverConfig)

	composerConfig := DefaultComposerConfig()
	composerConfig.EnableCaching = true
	composer := NewPromptComposer(composerConfig)

	baseIR := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "test-app",
			Type: "api",
			Stack: ir.TechStack{
				Backend: ir.BackendStack{Language: "go"},
			},
		},
	}

	t.Run("CachedComposition", func(t *testing.T) {
		overlayResult, err := resolver.ResolveOverlays([]string{"test-fintech"}, baseIR)
		require.NoError(t, err)

		req := ComposeRequest{
			AgentType:     "backend",
			IRSpec:        overlayResult.IRSpec,
			OverlayResult: overlayResult,
		}

		// First call should populate cache
		result1, err := composer.ComposePrompt(req)
		require.NoError(t, err)

		// Second call should use cache
		result2, err := composer.ComposePrompt(req)
		require.NoError(t, err)

		// Results should be identical
		assert.Equal(t, result1.TotalLength, result2.TotalLength)
		assert.Equal(t, result1.EnhancementsCount, result2.EnhancementsCount)
	})

	t.Run("LargePromptHandling", func(t *testing.T) {
		// Create composer with small max length for testing
		smallConfig := DefaultComposerConfig()
		smallConfig.MaxPromptLength = 5000 // Small limit to trigger warning
		smallComposer := NewPromptComposer(smallConfig)

		// Create request with many custom prompts
		overlayResult, err := resolver.ResolveOverlays([]string{"test-fintech", "test-pci"}, baseIR)
		require.NoError(t, err)

		customPrompts := make([]string, 10)
		for i := range customPrompts {
			customPrompts[i] = strings.Repeat("Custom instruction content. ", 100)
		}

		req := ComposeRequest{
			AgentType:     "backend",
			IRSpec:        overlayResult.IRSpec,
			OverlayResult: overlayResult,
			CustomPrompts: customPrompts,
		}

		result, err := smallComposer.ComposePrompt(req)
		require.NoError(t, err)

		// Should handle large prompts gracefully
		assert.Greater(t, result.TotalLength, 5000)
		assert.Len(t, result.Warnings, 1) // Should warn about length
		assert.Contains(t, result.Warnings[0], "exceeds recommended maximum")
	})
}

// Helper functions

func createConditionalOverlay(t *testing.T, tempDir string) string {
	conditionalOverlay := `
metadata:
  name: conditional-overlay
  version: "1.0.0"
  type: domain
  priority: 5
  description: Test conditional overlay
  created_at: 2025-01-01T00:00:00Z
  updated_at: 2025-01-01T00:00:00Z

dependencies: []

ir_modifications:
  - path: "app.features"
    operation: "add"
    value:
      - "conditional_feature"

prompt_enhancements:
  - agent_type: "backend"
    section: "system"
    content: "Go-specific enhancement for high-performance applications"
    position: "before"
    priority: 5
    conditions:
      language: "go"

validation_rules: []
`

	err := os.WriteFile(filepath.Join(tempDir, "conditional-overlay.yaml"), []byte(conditionalOverlay), 0644)
	require.NoError(t, err)
	return "conditional-overlay"
}

func createTestOverlayFiles(t *testing.T, tempDir string) {
	// Create fintech overlay
	fintechOverlay := `
metadata:
  name: test-fintech
  version: "1.0.0"
  type: domain
  priority: 5
  description: Test fintech overlay
  created_at: 2025-01-01T00:00:00Z
  updated_at: 2025-01-01T00:00:00Z

dependencies: []

ir_modifications:
  - path: "app.features"
    operation: "add"
    value:
      - "payment_processing"
      - "fraud_detection"
      - "kyc_verification"

prompt_enhancements:
  - agent_type: "backend"
    section: "system"
    content: "You are building a financial services application. Ensure all payment processing follows industry standards and includes proper fraud detection mechanisms."
    position: "before"
    priority: 5
  - agent_type: "database"
    section: "context"
    content: "Implement proper transaction handling for financial data with ACID compliance."
    position: "before"
    priority: 5

validation_rules:
  - name: "payment_validation"
    type: "security"
    severity: "error"
    message: "Payment processing must include fraud detection"
`

	// Create PCI overlay
	pciOverlay := `
metadata:
  name: test-pci
  version: "1.0.0"
  type: compliance
  priority: 10
  description: Test PCI compliance overlay
  created_at: 2025-01-01T00:00:00Z
  updated_at: 2025-01-01T00:00:00Z

dependencies: []

ir_modifications:
  - path: "app.features"
    operation: "add"
    value:
      - "card_data_protection"
      - "access_logging"
      - "encryption_at_rest"

prompt_enhancements:
  - agent_type: "backend"
    section: "system"
    content: "This application must comply with PCI DSS requirements. Implement proper card data encryption, access controls, and audit logging."
    position: "before"
    priority: 10

validation_rules:
  - name: "pci_compliance"
    type: "compliance"
    severity: "error"
    message: "Card data must be encrypted and access logged"
`

	err := os.WriteFile(filepath.Join(tempDir, "test-fintech.yaml"), []byte(fintechOverlay), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "test-pci.yaml"), []byte(pciOverlay), 0644)
	require.NoError(t, err)
}