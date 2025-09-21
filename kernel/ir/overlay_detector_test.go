package ir

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestOverlayDetector(t *testing.T) {
	detector := NewOverlayDetector()

	t.Run("DetectFintechOverlay", func(t *testing.T) {
		brief := "Create a payment processing system with credit card handling and fraud detection"
		result := detector.DetectOverlays(brief)

		assert.NotEmpty(t, result.Suggestions)

		// Should detect fintech domain
		fintechFound := false
		for _, suggestion := range result.Suggestions {
			if suggestion.Name == "fintech" && suggestion.Type == "domain" {
				fintechFound = true
				assert.Greater(t, suggestion.Confidence, 0.5)
				assert.Contains(t, suggestion.Keywords, "payment")
				assert.Contains(t, suggestion.Reason, "financial services")
				break
			}
		}
		assert.True(t, fintechFound, "Should detect fintech overlay")
	})

	t.Run("DetectHealthcareOverlay", func(t *testing.T) {
		brief := "Build a patient management system for hospitals with medical records and doctor appointments"
		result := detector.DetectOverlays(brief)

		assert.NotEmpty(t, result.Suggestions)

		// Should detect healthcare domain
		healthcareFound := false
		for _, suggestion := range result.Suggestions {
			if suggestion.Name == "healthcare" && suggestion.Type == "domain" {
				healthcareFound = true
				assert.Greater(t, suggestion.Confidence, 0.5)
				assert.Contains(t, suggestion.Keywords, "patient")
				break
			}
		}
		assert.True(t, healthcareFound, "Should detect healthcare overlay")
	})

	t.Run("DetectHIPAACompliance", func(t *testing.T) {
		brief := "Create HIPAA-compliant medical system with protected health information handling"
		result := detector.DetectOverlays(brief)

		assert.NotEmpty(t, result.Suggestions)

		// Should detect both healthcare domain and HIPAA compliance
		healthcareFound := false
		hipaaFound := false

		for _, suggestion := range result.Suggestions {
			if suggestion.Name == "healthcare" && suggestion.Type == "domain" {
				healthcareFound = true
			}
			if suggestion.Name == "hipaa" && suggestion.Type == "compliance" {
				hipaaFound = true
				assert.Greater(t, suggestion.Confidence, 0.8) // High confidence for explicit HIPAA mention
				assert.Contains(t, suggestion.Keywords, "hipaa")
			}
		}

		assert.True(t, healthcareFound, "Should detect healthcare domain")
		assert.True(t, hipaaFound, "Should detect HIPAA compliance")

		// Should auto-apply HIPAA due to high confidence
		assert.Contains(t, result.AutoApply, "hipaa")
	})

	t.Run("DetectPCICompliance", func(t *testing.T) {
		brief := "Build PCI-DSS compliant payment gateway with card data protection"
		result := detector.DetectOverlays(brief)

		pciFound := false
		for _, suggestion := range result.Suggestions {
			if suggestion.Name == "pci" && suggestion.Type == "compliance" {
				pciFound = true
				assert.GreaterOrEqual(t, suggestion.Confidence, 0.6)
				break
			}
		}
		assert.True(t, pciFound, "Should detect PCI compliance")
	})

	t.Run("DetectGDPRCompliance", func(t *testing.T) {
		brief := "Create user management system with GDPR compliance and data protection rights"
		result := detector.DetectOverlays(brief)

		gdprFound := false
		for _, suggestion := range result.Suggestions {
			if suggestion.Name == "gdpr" && suggestion.Type == "compliance" {
				gdprFound = true
				assert.Greater(t, suggestion.Confidence, 0.8)
				assert.Contains(t, suggestion.Keywords, "gdpr")
				break
			}
		}
		assert.True(t, gdprFound, "Should detect GDPR compliance")
	})

	t.Run("DetectEcommerceOverlay", func(t *testing.T) {
		brief := "Build online shopping platform with product catalog, shopping cart, and checkout"
		result := detector.DetectOverlays(brief)

		ecommerceFound := false
		for _, suggestion := range result.Suggestions {
			if suggestion.Name == "ecommerce" && suggestion.Type == "domain" {
				ecommerceFound = true
				assert.Greater(t, suggestion.Confidence, 0.5)
				// Check for any e-commerce keywords (shopping, product, cart, checkout, etc.)
				hasEcommerceKeyword := false
				for _, keyword := range suggestion.Keywords {
					if keyword == "shopping" || keyword == "product" || keyword == "cart" || keyword == "checkout" {
						hasEcommerceKeyword = true
						break
					}
				}
				assert.True(t, hasEcommerceKeyword, "Should contain e-commerce keywords")
				break
			}
		}
		assert.True(t, ecommerceFound, "Should detect e-commerce overlay")
	})

	t.Run("NoOverlayDetection", func(t *testing.T) {
		brief := "Create a simple todo list application with basic CRUD operations"
		result := detector.DetectOverlays(brief)

		// Should have minimal or no suggestions
		assert.Empty(t, result.AutoApply)

		// Any suggestions should have low confidence
		for _, suggestion := range result.Suggestions {
			assert.Less(t, suggestion.Confidence, 0.5)
		}
	})

	t.Run("MultipleOverlayDetection", func(t *testing.T) {
		brief := "Create HIPAA-compliant healthcare payment system with credit card processing"
		result := detector.DetectOverlays(brief)

		// Should detect multiple overlays
		assert.Greater(t, len(result.Suggestions), 2)

		expectedOverlays := map[string]string{
			"healthcare": "domain",
			"fintech":    "domain",
			"hipaa":      "compliance",
		}

		for name, expectedType := range expectedOverlays {
			found := false
			for _, suggestion := range result.Suggestions {
				if suggestion.Name == name && suggestion.Type == expectedType {
					found = true
					break
				}
			}
			assert.True(t, found, "Should detect %s %s overlay", expectedType, name)
		}
	})

	t.Run("ConfidenceScoring", func(t *testing.T) {
		// High confidence test
		highConfidenceBrief := "Create PCI-DSS compliant payment card processing system"
		highResult := detector.DetectOverlays(highConfidenceBrief)

		// Low confidence test
		lowConfidenceBrief := "Create system with some payment features"
		lowResult := detector.DetectOverlays(lowConfidenceBrief)

		// Compare confidence levels
		var highPCIConfidence, lowPCIConfidence float64

		for _, suggestion := range highResult.Suggestions {
			if suggestion.Name == "pci" {
				highPCIConfidence = suggestion.Confidence
			}
		}

		for _, suggestion := range lowResult.Suggestions {
			if suggestion.Name == "pci" {
				lowPCIConfidence = suggestion.Confidence
			}
		}

		if highPCIConfidence > 0 && lowPCIConfidence > 0 {
			assert.Greater(t, highPCIConfidence, lowPCIConfidence, "Explicit PCI mention should have higher confidence")
		}
	})
}

func TestOverlayCompatibility(t *testing.T) {
	detector := NewOverlayDetector()

	t.Run("ValidateCompatibility", func(t *testing.T) {
		spec := &IRSpec{
			App: AppSpec{
				Stack: TechStack{
					Backend: BackendStack{
						Language: "go",
					},
				},
			},
		}

		// Test compatible overlays
		overlays := []string{"fintech", "pci"}
		warnings := detector.ValidateOverlayCompatibility(overlays, spec)

		// Should not have major compatibility warnings for this combination
		assert.Empty(t, warnings)
	})

	t.Run("ConflictWarning", func(t *testing.T) {
		spec := &IRSpec{
			App: AppSpec{
				Stack: TechStack{
					Backend: BackendStack{
						Language: "go",
					},
				},
			},
		}

		// Test potentially conflicting overlays
		overlays := []string{"fintech", "healthcare"}
		warnings := detector.ValidateOverlayCompatibility(overlays, spec)

		// Should warn about potential data separation needs
		found := false
		for _, warning := range warnings {
			if strings.Contains(warning, "data separation") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should warn about fintech + healthcare combination")
	})

	t.Run("LanguageSpecificWarning", func(t *testing.T) {
		spec := &IRSpec{
			App: AppSpec{
				Stack: TechStack{
					Backend: BackendStack{
						Language: "javascript",
					},
				},
			},
		}

		overlays := []string{"gdpr"}
		warnings := detector.ValidateOverlayCompatibility(overlays, spec)

		// Should warn about GDPR in JavaScript
		found := false
		for _, warning := range warnings {
			if strings.Contains(warning, "JavaScript") && strings.Contains(warning, "GDPR") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should warn about GDPR compliance in JavaScript")
	})
}

func TestOverlayDetectionWarnings(t *testing.T) {
	detector := NewOverlayDetector()

	t.Run("MultipleComplianceWarning", func(t *testing.T) {
		// Create mock suggestions with multiple high-confidence compliance overlays
		suggestions := []OverlaySuggestion{
			{Name: "pci", Type: "compliance", Confidence: 0.9},
			{Name: "hipaa", Type: "compliance", Confidence: 0.8},
			{Name: "gdpr", Type: "compliance", Confidence: 0.7},
		}

		warnings := detector.checkForWarnings(suggestions)

		// Should warn about multiple compliance overlays
		found := false
		for _, warning := range warnings {
			if strings.Contains(warning, "Multiple compliance") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should warn about multiple compliance overlays")
	})

	t.Run("MissingDomainWarning", func(t *testing.T) {
		// Compliance without domain context
		suggestions := []OverlaySuggestion{
			{Name: "pci", Type: "compliance", Confidence: 0.9},
		}

		warnings := detector.checkForWarnings(suggestions)

		// Should warn about missing domain context
		found := false
		for _, warning := range warnings {
			if strings.Contains(warning, "domain context") {
				found = true
				break
			}
		}
		assert.True(t, found, "Should warn about compliance without domain context")
	})
}

func TestPatternMatching(t *testing.T) {
	detector := NewOverlayDetector()

	t.Run("CaseInsensitiveMatching", func(t *testing.T) {
		briefs := []string{
			"Create PAYMENT processing system",
			"create payment PROCESSING system",
			"Create Payment Processing System",
		}

		for _, brief := range briefs {
			result := detector.DetectOverlays(brief)

			fintechFound := false
			for _, suggestion := range result.Suggestions {
				if suggestion.Name == "fintech" {
					fintechFound = true
					break
				}
			}
			assert.True(t, fintechFound, "Should detect fintech regardless of case: %s", brief)
		}
	})

	t.Run("KeywordVariations", func(t *testing.T) {
		testCases := map[string]string{
			"banking system":          "fintech",
			"financial application":   "fintech",
			"patient portal":          "healthcare",
			"medical records":         "healthcare",
			"online store":           "ecommerce",
			"shopping cart":          "ecommerce",
		}

		for brief, expectedDomain := range testCases {
			result := detector.DetectOverlays(brief)

			found := false
			for _, suggestion := range result.Suggestions {
				if suggestion.Name == expectedDomain && suggestion.Type == "domain" {
					found = true
					break
				}
			}
			assert.True(t, found, "Should detect %s for brief: %s", expectedDomain, brief)
		}
	})
}