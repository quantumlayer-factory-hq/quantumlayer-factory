package overlays

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
)

func TestFileSystemResolver(t *testing.T) {
	// Create temporary directory for test overlays
	tempDir := t.TempDir()

	// Create test overlay files
	createTestOverlayFiles(t, tempDir)

	config := ResolverConfig{
		OverlayPaths:       []string{tempDir},
		ConflictResolution: "priority",
		EnableCaching:      true,
		CacheTTL:           300,
	}

	resolver := NewFileSystemResolver(config)

	t.Run("LoadOverlay", func(t *testing.T) {
		overlay, err := resolver.LoadOverlay("test-domain")
		require.NoError(t, err)
		assert.NotNil(t, overlay)

		metadata := overlay.GetMetadata()
		assert.Equal(t, "test-domain", metadata.Name)
		assert.Equal(t, "1.0.0", metadata.Version)
		assert.Equal(t, OverlayTypeDomain, metadata.Type)
	})

	t.Run("LoadNonexistentOverlay", func(t *testing.T) {
		_, err := resolver.LoadOverlay("nonexistent")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "overlay not found")
	})

	t.Run("ValidateOverlays", func(t *testing.T) {
		err := resolver.ValidateOverlays([]string{"test-domain"})
		assert.NoError(t, err)
	})

	t.Run("ValidateInvalidOverlays", func(t *testing.T) {
		err := resolver.ValidateOverlays([]string{"nonexistent"})
		assert.Error(t, err)
	})

	t.Run("ListAvailable", func(t *testing.T) {
		metadata, err := resolver.ListAvailable()
		require.NoError(t, err)
		assert.Len(t, metadata, 2) // test-domain and test-compliance

		found := false
		for _, meta := range metadata {
			if meta.Name == "test-domain" {
				found = true
				break
			}
		}
		assert.True(t, found, "test-domain overlay should be in available list")
	})
}

func TestOverlayResolution(t *testing.T) {
	tempDir := t.TempDir()
	createTestOverlayFiles(t, tempDir)

	config := ResolverConfig{
		OverlayPaths:       []string{tempDir},
		ConflictResolution: "priority",
		EnableCaching:      false,
	}

	resolver := NewFileSystemResolver(config)

	baseIR := &ir.IRSpec{
		App: ir.AppSpec{
			Name:     "test-app",
			Type:     "api",
			Features: []ir.Feature{},
			Stack: ir.TechStack{
				Backend: ir.BackendStack{
					Language: "go",
				},
				Database: ir.DatabaseStack{
					Type: "sqlite",
				},
			},
		},
	}

	t.Run("ResolveOverlays", func(t *testing.T) {
		result, err := resolver.ResolveOverlays([]string{"test-domain"}, baseIR)
		require.NoError(t, err)
		assert.NotNil(t, result)

		assert.Contains(t, result.ResolvedOverlays, "test-domain")
		assert.NotNil(t, result.IRSpec)

		// Check that the test_feature was added
		found := false
		for _, feature := range result.IRSpec.App.Features {
			if feature.Name == "test_feature" {
				found = true
				break
			}
		}
		assert.True(t, found, "test_feature should be in features list")
	})

	t.Run("ResolveMultipleOverlays", func(t *testing.T) {
		result, err := resolver.ResolveOverlays([]string{"test-domain", "test-compliance"}, baseIR)
		require.NoError(t, err)

		assert.Len(t, result.ResolvedOverlays, 2)
		assert.Contains(t, result.ResolvedOverlays, "test-domain")
		assert.Contains(t, result.ResolvedOverlays, "test-compliance")

		// Check that features from both overlays were added
		featureNames := make([]string, len(result.IRSpec.App.Features))
		for i, feature := range result.IRSpec.App.Features {
			featureNames[i] = feature.Name
		}
		assert.Contains(t, featureNames, "test_feature")
		assert.Contains(t, featureNames, "compliance_feature")
	})

	t.Run("ResolveDependencies", func(t *testing.T) {
		// Create overlay with dependency
		createDependentOverlay(t, tempDir)

		result, err := resolver.ResolveOverlays([]string{"dependent-overlay"}, baseIR)
		require.NoError(t, err)

		// Should resolve both dependent-overlay and its dependency (test-domain)
		assert.Len(t, result.ResolvedOverlays, 2)
		assert.Contains(t, result.ResolvedOverlays, "test-domain")
		assert.Contains(t, result.ResolvedOverlays, "dependent-overlay")

		// Dependencies should be applied first (test-domain before dependent-overlay)
		// Since test-domain is a dependency of dependent-overlay, it should come first
		assert.Equal(t, "test-domain", result.AppliedOrder[0])
		assert.Equal(t, "dependent-overlay", result.AppliedOrder[1])
	})
}

func TestYAMLOverlay(t *testing.T) {
	spec := OverlaySpec{
		Metadata: OverlayMetadata{
			Name:     "test-overlay",
			Version:  "1.0.0",
			Type:     OverlayTypeDomain,
			Priority: PriorityMedium,
		},
		IRModifications: []IRModification{
			{
				Path:      "app.features",
				Operation: "add",
				Value:     []interface{}{"new_feature"},
			},
		},
		PromptEnhancements: []PromptEnhancement{
			{
				AgentType: "backend",
				Section:   "system",
				Content:   "Test prompt enhancement",
				Position:  "before",
				Priority:  PriorityHigh,
			},
		},
		ValidationRules: []ValidationRule{
			{
				Name:     "test_rule",
				Type:     "validation",
				Severity: "error",
				Message:  "Test validation rule",
			},
		},
	}

	overlay := NewYAMLOverlay(spec)

	t.Run("GetMetadata", func(t *testing.T) {
		metadata := overlay.GetMetadata()
		assert.Equal(t, "test-overlay", metadata.Name)
		assert.Equal(t, "1.0.0", metadata.Version)
		assert.Equal(t, OverlayTypeDomain, metadata.Type)
		assert.Equal(t, PriorityMedium, metadata.Priority)
	})

	t.Run("GetPromptEnhancements", func(t *testing.T) {
		enhancements := overlay.GetPromptEnhancements()
		assert.Len(t, enhancements, 1)
		assert.Equal(t, "backend", enhancements[0].AgentType)
		assert.Equal(t, "Test prompt enhancement", enhancements[0].Content)
	})

	t.Run("GetValidationRules", func(t *testing.T) {
		rules := overlay.GetValidationRules()
		assert.Len(t, rules, 1)
		assert.Equal(t, "test_rule", rules[0].Name)
		assert.Equal(t, "error", rules[0].Severity)
	})

	t.Run("GetDependencies", func(t *testing.T) {
		deps := overlay.GetDependencies()
		assert.Empty(t, deps)
	})

	t.Run("Validate", func(t *testing.T) {
		err := overlay.Validate()
		assert.NoError(t, err)
	})

	t.Run("ValidateInvalid", func(t *testing.T) {
		invalidSpec := OverlaySpec{
			Metadata: OverlayMetadata{
				Name: "", // Missing required field
			},
		}
		invalidOverlay := NewYAMLOverlay(invalidSpec)
		err := invalidOverlay.Validate()
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "overlay name is required")
	})
}

func TestIRModification(t *testing.T) {
	spec := OverlaySpec{
		Metadata: OverlayMetadata{
			Name:     "test-overlay",
			Version:  "1.0.0",
			Type:     OverlayTypeDomain,
			Priority: PriorityMedium,
		},
		IRModifications: []IRModification{
			{
				Path:      "app.features",
				Operation: "add",
				Value:     []interface{}{"feature1", "feature2"},
			},
			{
				Path:      "app.stack.database.type",
				Operation: "replace",
				Value:     "postgresql",
			},
		},
	}

	overlay := NewYAMLOverlay(spec)

	baseIR := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "test-app",
			Features: []ir.Feature{
				{Name: "existing_feature", Description: "Existing feature"},
			},
			Stack: ir.TechStack{
				Backend: ir.BackendStack{
					Language: "go",
				},
				Database: ir.DatabaseStack{
					Type: "sqlite",
				},
			},
		},
	}

	t.Run("ApplyToIR", func(t *testing.T) {
		modifiedIR, err := overlay.ApplyToIR(baseIR)
		require.NoError(t, err)

		// Check that features were added
		featureNames := make([]string, len(modifiedIR.App.Features))
		for i, feature := range modifiedIR.App.Features {
			featureNames[i] = feature.Name
		}
		assert.Contains(t, featureNames, "existing_feature")
		assert.Contains(t, featureNames, "feature1")
		assert.Contains(t, featureNames, "feature2")

		// Check that database was replaced
		assert.Equal(t, "postgresql", modifiedIR.App.Stack.Database.Type)
	})
}

func TestCaching(t *testing.T) {
	tempDir := t.TempDir()
	createTestOverlayFiles(t, tempDir)

	config := ResolverConfig{
		OverlayPaths:       []string{tempDir},
		ConflictResolution: "priority",
		EnableCaching:      true,
		CacheTTL:           300,
	}

	resolver := NewFileSystemResolver(config)

	baseIR := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "test-app",
			Type: "api",
		},
	}

	t.Run("CacheResults", func(t *testing.T) {
		// First call should populate cache
		result1, err := resolver.ResolveOverlays([]string{"test-domain"}, baseIR)
		require.NoError(t, err)

		// Second call should use cache (hard to test directly, but should not error)
		result2, err := resolver.ResolveOverlays([]string{"test-domain"}, baseIR)
		require.NoError(t, err)

		assert.Equal(t, len(result1.ResolvedOverlays), len(result2.ResolvedOverlays))
	})
}

// Helper functions for test setup

func createTestOverlayFiles(t *testing.T, tempDir string) {
	domainOverlay := `
metadata:
  name: test-domain
  version: "1.0.0"
  type: domain
  priority: 5
  description: Test domain overlay
  created_at: 2025-01-01T00:00:00Z
  updated_at: 2025-01-01T00:00:00Z

dependencies: []

ir_modifications:
  - path: "app.features"
    operation: "add"
    value:
      - "test_feature"

prompt_enhancements:
  - agent_type: "backend"
    section: "system"
    content: "Test domain prompt"
    position: "before"
    priority: 5

validation_rules:
  - name: "test_domain_rule"
    type: "validation"
    severity: "warning"
    message: "Test domain validation rule"
`

	complianceOverlay := `
metadata:
  name: test-compliance
  version: "1.0.0"
  type: compliance
  priority: 10
  description: Test compliance overlay
  created_at: 2025-01-01T00:00:00Z
  updated_at: 2025-01-01T00:00:00Z

dependencies: []

ir_modifications:
  - path: "app.features"
    operation: "add"
    value:
      - "compliance_feature"

prompt_enhancements:
  - agent_type: "backend"
    section: "system"
    content: "Test compliance prompt"
    position: "before"
    priority: 10

validation_rules:
  - name: "test_compliance_rule"
    type: "compliance"
    severity: "error"
    message: "Test compliance validation rule"
`

	err := os.WriteFile(filepath.Join(tempDir, "test-domain.yaml"), []byte(domainOverlay), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "test-compliance.yaml"), []byte(complianceOverlay), 0644)
	require.NoError(t, err)
}

func createDependentOverlay(t *testing.T, tempDir string) {
	dependentOverlay := `
metadata:
  name: dependent-overlay
  version: "1.0.0"
  type: domain
  priority: 3
  description: Test overlay with dependencies
  created_at: 2025-01-01T00:00:00Z
  updated_at: 2025-01-01T00:00:00Z

dependencies:
  - test-domain

ir_modifications:
  - path: "app.features"
    operation: "add"
    value:
      - "dependent_feature"

prompt_enhancements: []
validation_rules: []
`

	err := os.WriteFile(filepath.Join(tempDir, "dependent-overlay.yaml"), []byte(dependentOverlay), 0644)
	require.NoError(t, err)
}

func TestPriorityOrdering(t *testing.T) {
	tempDir := t.TempDir()

	// Create overlays with different priorities
	lowPriorityOverlay := `
metadata:
  name: low-priority
  version: "1.0.0"
  type: domain
  priority: 1
  description: Low priority overlay
  created_at: 2025-01-01T00:00:00Z
  updated_at: 2025-01-01T00:00:00Z

dependencies: []
ir_modifications: []
prompt_enhancements: []
validation_rules: []
`

	highPriorityOverlay := `
metadata:
  name: high-priority
  version: "1.0.0"
  type: compliance
  priority: 10
  description: High priority overlay
  created_at: 2025-01-01T00:00:00Z
  updated_at: 2025-01-01T00:00:00Z

dependencies: []
ir_modifications: []
prompt_enhancements: []
validation_rules: []
`

	err := os.WriteFile(filepath.Join(tempDir, "low-priority.yaml"), []byte(lowPriorityOverlay), 0644)
	require.NoError(t, err)

	err = os.WriteFile(filepath.Join(tempDir, "high-priority.yaml"), []byte(highPriorityOverlay), 0644)
	require.NoError(t, err)

	config := ResolverConfig{
		OverlayPaths:       []string{tempDir},
		ConflictResolution: "priority",
		EnableCaching:      false,
	}

	resolver := NewFileSystemResolver(config)

	baseIR := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "test-app",
			Type: "api",
		},
	}

	t.Run("PriorityBasedOrdering", func(t *testing.T) {
		result, err := resolver.ResolveOverlays([]string{"high-priority", "low-priority"}, baseIR)
		require.NoError(t, err)

		// Lower priority should be applied first (index 0), higher priority last (index 1)
		assert.Equal(t, "low-priority", result.AppliedOrder[0])
		assert.Equal(t, "high-priority", result.AppliedOrder[1])
	})
}