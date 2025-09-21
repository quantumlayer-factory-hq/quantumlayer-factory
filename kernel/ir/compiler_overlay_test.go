package ir

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompilerOverlayIntegration(t *testing.T) {
	compiler := NewCompiler()

	t.Run("CompileWithOverlayDetection", func(t *testing.T) {
		brief := "Create a HIPAA-compliant patient management system with medical records"
		result, err := compiler.Compile(brief)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should have overlay detection results
		assert.NotNil(t, result.OverlayDetection)
		assert.NotEmpty(t, result.OverlayDetection.Suggestions)

		// Should detect healthcare and HIPAA
		suggestions := result.OverlayDetection.Suggestions
		healthcareFound := false
		hipaaFound := false

		for _, suggestion := range suggestions {
			if suggestion.Name == "healthcare" && suggestion.Type == "domain" {
				healthcareFound = true
			}
			if suggestion.Name == "hipaa" && suggestion.Type == "compliance" {
				hipaaFound = true
			}
		}

		assert.True(t, healthcareFound, "Should detect healthcare domain")
		assert.True(t, hipaaFound, "Should detect HIPAA compliance")

		// HIPAA should be in required overlays due to high confidence
		assert.Contains(t, result.RequiredOverlays, "hipaa")
	})

	t.Run("CompileWithExplicitOverlays", func(t *testing.T) {
		brief := "Create a payment processing system"
		overlays := []string{"fintech", "pci"}

		result, err := compiler.CompileWithOverlays(brief, overlays)
		require.NoError(t, err)
		assert.NotNil(t, result)

		// Should include the explicitly specified overlays
		assert.Contains(t, result.RequiredOverlays, "fintech")
		assert.Contains(t, result.RequiredOverlays, "pci")

		// Should not duplicate overlays between required and suggested
		for _, required := range result.RequiredOverlays {
			assert.NotContains(t, result.SuggestedOverlays, required)
		}
	})

	t.Run("SuggestOverlaysOnly", func(t *testing.T) {
		brief := "Build PCI-compliant e-commerce platform with payment processing"
		suggestions := compiler.SuggestOverlays(brief)

		assert.NotNil(t, suggestions)
		assert.NotEmpty(t, suggestions.Suggestions)

		// Should suggest relevant overlays
		expectedOverlays := []string{"ecommerce", "fintech", "pci"}
		foundOverlays := make(map[string]bool)

		for _, suggestion := range suggestions.Suggestions {
			foundOverlays[suggestion.Name] = true
		}

		for _, expected := range expectedOverlays {
			assert.True(t, foundOverlays[expected], "Should suggest %s overlay", expected)
		}
	})

	t.Run("OverlayCompatibilityWarnings", func(t *testing.T) {
		brief := "Create healthcare payment system"
		overlays := []string{"healthcare", "fintech", "hipaa", "pci"}

		result, err := compiler.CompileWithOverlays(brief, overlays)
		require.NoError(t, err)

		// Should include compatibility warning about healthcare + fintech
		warningFound := false
		for _, warning := range result.Warnings {
			if contains(warning, "data separation") {
				warningFound = true
				break
			}
		}
		assert.True(t, warningFound, "Should warn about healthcare + fintech combination")
	})

	t.Run("EmptyBriefHandling", func(t *testing.T) {
		_, err := compiler.Compile("")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "brief cannot be empty")
	})

	t.Run("LowConfidenceSuggestions", func(t *testing.T) {
		brief := "Create a simple user management system"
		result, err := compiler.Compile(brief)
		require.NoError(t, err)

		// Should have minimal overlay suggestions
		assert.Empty(t, result.RequiredOverlays)

		// Any suggestions should have low confidence
		for _, suggestion := range result.OverlayDetection.Suggestions {
			assert.Less(t, suggestion.Confidence, 0.5)
		}
	})

	t.Run("MultipleComplianceWarning", func(t *testing.T) {
		brief := "Create system with HIPAA, PCI, and GDPR compliance requirements"
		result, err := compiler.Compile(brief)
		require.NoError(t, err)

		// Should detect multiple compliance overlays
		complianceCount := 0
		for _, suggestion := range result.OverlayDetection.Suggestions {
			if suggestion.Type == "compliance" && suggestion.Confidence > 0.6 {
				complianceCount++
			}
		}
		assert.Greater(t, complianceCount, 1)

		// Should warn about multiple compliance overlays
		warningFound := false
		for _, warning := range result.Warnings {
			if contains(warning, "Multiple compliance") {
				warningFound = true
				break
			}
		}
		assert.True(t, warningFound, "Should warn about multiple compliance overlays")
	})

	t.Run("DomainDetectionAccuracy", func(t *testing.T) {
		testCases := map[string][]string{
			"Create online banking platform with account management":        {"fintech"},
			"Build patient portal for healthcare providers":                 {"healthcare"},
			"Develop e-commerce marketplace with product catalog":           {"ecommerce"},
			"Create HIPAA-compliant medical records system":                {"healthcare", "hipaa"},
			"Build PCI-DSS compliant payment gateway":                      {"fintech", "pci"},
			"Develop GDPR-compliant user data management system":           {"gdpr"},
		}

		for brief, expectedOverlays := range testCases {
			result, err := compiler.Compile(brief)
			require.NoError(t, err, "Failed to compile brief: %s", brief)

			allOverlays := append(result.RequiredOverlays, result.SuggestedOverlays...)

			for _, expected := range expectedOverlays {
				found := false
				for _, actual := range allOverlays {
					if actual == expected {
						found = true
						break
					}
				}
				// Also check in detection suggestions
				if !found {
					for _, suggestion := range result.OverlayDetection.Suggestions {
						if suggestion.Name == expected && suggestion.Confidence > 0.3 {
							found = true
							break
						}
					}
				}
				assert.True(t, found, "Should detect %s overlay for brief: %s", expected, brief)
			}
		}
	})

	t.Run("OverlayAutoApply", func(t *testing.T) {
		brief := "Create PCI-DSS compliant credit card processing system"
		result, err := compiler.Compile(brief)
		require.NoError(t, err)

		// High-confidence overlays should be auto-applied
		assert.NotEmpty(t, result.OverlayDetection.AutoApply)

		// PCI should be auto-applied due to explicit mention
		assert.Contains(t, result.OverlayDetection.AutoApply, "pci")
	})

	t.Run("OverlayDeduplication", func(t *testing.T) {
		brief := "Create fintech payment system"
		overlays := []string{"fintech"} // Explicitly specify what might be auto-detected

		result, err := compiler.CompileWithOverlays(brief, overlays)
		require.NoError(t, err)

		// Should not have duplicates between required and suggested
		overlayMap := make(map[string]int)

		for _, overlay := range result.RequiredOverlays {
			overlayMap[overlay]++
		}

		for _, overlay := range result.SuggestedOverlays {
			overlayMap[overlay]++
		}

		// No overlay should appear more than once across all lists
		for overlay, count := range overlayMap {
			assert.LessOrEqual(t, count, 1, "Overlay %s appears %d times (should be 1 or 0)", overlay, count)
		}
	})
}

func TestOverlayDetectionEdgeCases(t *testing.T) {
	compiler := NewCompiler()

	t.Run("MixedCaseKeywords", func(t *testing.T) {
		brief := "Create PAYMENT system with Credit Card processing"
		result, err := compiler.Compile(brief)
		require.NoError(t, err)

		// Should detect fintech regardless of case
		fintechFound := false
		for _, suggestion := range result.OverlayDetection.Suggestions {
			if suggestion.Name == "fintech" {
				fintechFound = true
				break
			}
		}
		assert.True(t, fintechFound, "Should detect fintech overlay with mixed case keywords")
	})

	t.Run("MultipleKeywordInstances", func(t *testing.T) {
		brief := "Create payment system for payment processing with payment verification"
		result, err := compiler.Compile(brief)
		require.NoError(t, err)

		// Should still detect fintech with multiple instances of same keyword
		fintechFound := false
		for _, suggestion := range result.OverlayDetection.Suggestions {
			if suggestion.Name == "fintech" {
				fintechFound = true
				assert.Greater(t, suggestion.Confidence, 0.5)
				break
			}
		}
		assert.True(t, fintechFound, "Should detect fintech with multiple keyword instances")
	})

	t.Run("WeakSignals", func(t *testing.T) {
		brief := "Create system with some financial features"
		result, err := compiler.Compile(brief)
		require.NoError(t, err)

		// Should have low confidence for weak signals
		for _, suggestion := range result.OverlayDetection.Suggestions {
			if suggestion.Name == "fintech" {
				assert.Less(t, suggestion.Confidence, 0.7, "Weak signals should have lower confidence")
			}
		}
	})

	t.Run("IrrelevantContent", func(t *testing.T) {
		brief := "Create todo list application for personal task management"
		result, err := compiler.Compile(brief)
		require.NoError(t, err)

		// Should have minimal or no overlay suggestions
		assert.Empty(t, result.RequiredOverlays)
		assert.LessOrEqual(t, len(result.SuggestedOverlays), 1)
	})
}

// Helper function for string contains check
func contains(s, substr string) bool {
	return len(s) > 0 && len(substr) > 0 &&
		   (s == substr || len(s) > len(substr) &&
		   (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
		   findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}