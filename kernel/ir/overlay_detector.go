package ir

import (
	"regexp"
	"strings"
)

// OverlayDetector analyzes briefs to suggest relevant overlays
type OverlayDetector struct {
	domainPatterns     map[string]*regexp.Regexp
	compliancePatterns map[string]*regexp.Regexp
}

// OverlaySuggestion represents a suggested overlay with confidence
type OverlaySuggestion struct {
	Name       string  `json:"name"`
	Type       string  `json:"type"`       // "domain" or "compliance"
	Confidence float64 `json:"confidence"` // 0.0 to 1.0
	Reason     string  `json:"reason"`     // Why this overlay was suggested
	Keywords   []string `json:"keywords"`  // Matching keywords found
}

// OverlayDetectionResult contains suggested overlays and metadata
type OverlayDetectionResult struct {
	Suggestions []OverlaySuggestion `json:"suggestions"`
	AutoApply   []string           `json:"auto_apply"`   // High-confidence overlays to auto-apply
	Warnings    []string           `json:"warnings"`     // Potential issues or conflicts
}

// NewOverlayDetector creates a new overlay detector with built-in patterns
func NewOverlayDetector() *OverlayDetector {
	return &OverlayDetector{
		domainPatterns:     initializeDomainPatterns(),
		compliancePatterns: initializeCompliancePatterns(),
	}
}

// DetectOverlays analyzes a brief and suggests relevant overlays
func (d *OverlayDetector) DetectOverlays(brief string) *OverlayDetectionResult {
	normalizedBrief := strings.ToLower(brief)
	result := &OverlayDetectionResult{
		Suggestions: []OverlaySuggestion{},
		AutoApply:   []string{},
		Warnings:    []string{},
	}

	// Detect domain overlays
	domainSuggestions := d.detectDomainOverlays(normalizedBrief)
	result.Suggestions = append(result.Suggestions, domainSuggestions...)

	// Detect compliance overlays
	complianceSuggestions := d.detectComplianceOverlays(normalizedBrief)
	result.Suggestions = append(result.Suggestions, complianceSuggestions...)

	// Determine auto-apply overlays (high confidence)
	for _, suggestion := range result.Suggestions {
		if suggestion.Confidence >= 0.8 {
			result.AutoApply = append(result.AutoApply, suggestion.Name)
		}
	}

	// Check for potential conflicts or warnings
	result.Warnings = d.checkForWarnings(result.Suggestions)

	return result
}

// detectDomainOverlays finds domain-specific overlay suggestions
func (d *OverlayDetector) detectDomainOverlays(brief string) []OverlaySuggestion {
	var suggestions []OverlaySuggestion

	for domain, pattern := range d.domainPatterns {
		matches := pattern.FindAllString(brief, -1)
		if len(matches) > 0 {
			confidence := d.calculateDomainConfidence(domain, matches, brief)
			if confidence > 0.3 { // Minimum threshold
				suggestion := OverlaySuggestion{
					Name:       domain,
					Type:       "domain",
					Confidence: confidence,
					Reason:     d.getDomainReason(domain, matches),
					Keywords:   d.deduplicateStrings(matches),
				}
				suggestions = append(suggestions, suggestion)
			}
		}
	}

	return suggestions
}

// detectComplianceOverlays finds compliance-specific overlay suggestions
func (d *OverlayDetector) detectComplianceOverlays(brief string) []OverlaySuggestion {
	var suggestions []OverlaySuggestion

	for compliance, pattern := range d.compliancePatterns {
		matches := pattern.FindAllString(brief, -1)
		if len(matches) > 0 {
			confidence := d.calculateComplianceConfidence(compliance, matches, brief)
			if confidence > 0.3 { // Lower threshold for testing
				suggestion := OverlaySuggestion{
					Name:       compliance,
					Type:       "compliance",
					Confidence: confidence,
					Reason:     d.getComplianceReason(compliance, matches),
					Keywords:   d.deduplicateStrings(matches),
				}
				suggestions = append(suggestions, suggestion)
			}
		}
	}

	return suggestions
}

// calculateDomainConfidence calculates confidence for domain overlays
func (d *OverlayDetector) calculateDomainConfidence(domain string, matches []string, brief string) float64 {
	baseConfidence := float64(len(matches)) * 0.3 // Increased base confidence

	// Boost confidence for multiple different keywords
	uniqueMatches := d.deduplicateStrings(matches)
	if len(uniqueMatches) > 1 {
		baseConfidence += 0.3
	}

	// Domain-specific adjustments
	switch domain {
	case "fintech":
		if strings.Contains(brief, "bank") || strings.Contains(brief, "financial") {
			baseConfidence += 0.2
		}
	case "healthcare":
		if strings.Contains(brief, "patient") || strings.Contains(brief, "medical") {
			baseConfidence += 0.2
		}
	case "ecommerce":
		if strings.Contains(brief, "shop") || strings.Contains(brief, "product") {
			baseConfidence += 0.2
		}
	}

	// Cap at 1.0
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}

	return baseConfidence
}

// calculateComplianceConfidence calculates confidence for compliance overlays
func (d *OverlayDetector) calculateComplianceConfidence(compliance string, matches []string, brief string) float64 {
	// Compliance keywords are more specific, so higher base confidence
	baseConfidence := float64(len(matches)) * 0.4

	// Explicit compliance mentions get very high confidence
	for _, match := range matches {
		if len(match) > 3 && strings.Contains(match, compliance) {
			baseConfidence = 0.9
			break
		}
	}

	// Cap at 1.0
	if baseConfidence > 1.0 {
		baseConfidence = 1.0
	}

	return baseConfidence
}

// getDomainReason provides human-readable reason for domain suggestion
func (d *OverlayDetector) getDomainReason(domain string, matches []string) string {
	keywords := strings.Join(d.deduplicateStrings(matches), ", ")

	switch domain {
	case "fintech":
		return "Detected financial services keywords: " + keywords
	case "healthcare":
		return "Detected healthcare-related keywords: " + keywords
	case "ecommerce":
		return "Detected e-commerce keywords: " + keywords
	default:
		return "Detected " + domain + " keywords: " + keywords
	}
}

// getComplianceReason provides human-readable reason for compliance suggestion
func (d *OverlayDetector) getComplianceReason(compliance string, matches []string) string {
	keywords := strings.Join(d.deduplicateStrings(matches), ", ")

	switch compliance {
	case "pci":
		return "Detected PCI DSS compliance requirements: " + keywords
	case "hipaa":
		return "Detected HIPAA compliance requirements: " + keywords
	case "gdpr":
		return "Detected GDPR compliance requirements: " + keywords
	default:
		return "Detected " + compliance + " compliance keywords: " + keywords
	}
}

// checkForWarnings identifies potential conflicts or issues
func (d *OverlayDetector) checkForWarnings(suggestions []OverlaySuggestion) []string {
	var warnings []string

	// Check for compliance conflicts (rare but possible)
	complianceCount := 0
	for _, suggestion := range suggestions {
		if suggestion.Type == "compliance" && suggestion.Confidence > 0.6 {
			complianceCount++
		}
	}

	if complianceCount > 2 {
		warnings = append(warnings, "Multiple compliance overlays detected - verify requirements are compatible")
	}

	// Check for missing domain when compliance is present
	hasCompliance := false
	hasDomain := false
	for _, suggestion := range suggestions {
		if suggestion.Type == "compliance" && suggestion.Confidence > 0.6 {
			hasCompliance = true
		}
		if suggestion.Type == "domain" && suggestion.Confidence > 0.5 {
			hasDomain = true
		}
	}

	if hasCompliance && !hasDomain {
		warnings = append(warnings, "Compliance overlay detected without domain context - consider adding relevant domain overlay")
	}

	return warnings
}

// deduplicateStrings removes duplicate strings from slice
func (d *OverlayDetector) deduplicateStrings(strings []string) []string {
	keys := make(map[string]bool)
	var result []string

	for _, str := range strings {
		if !keys[str] {
			keys[str] = true
			result = append(result, str)
		}
	}

	return result
}

// initializeDomainPatterns creates regex patterns for domain detection
func initializeDomainPatterns() map[string]*regexp.Regexp {
	patterns := map[string]string{
		"fintech": `(?i)\b(payment|banking|financial|fintech|transaction|credit|debit|card|wallet|investment|trading|loan|mortgage|insurance|crypto|blockchain|bitcoin)\b`,
		"healthcare": `(?i)\b(healthcare|medical|patient|hospital|clinic|doctor|physician|nurse|health|treatment|diagnosis|prescription|therapy|hipaa|phi|ehr|emr)\b`,
		"ecommerce": `(?i)\b(ecommerce|e-commerce|shop|store|product|inventory|cart|checkout|order|customer|retail|marketplace|catalog|purchase|sale)\b`,
	}

	compiled := make(map[string]*regexp.Regexp)
	for domain, pattern := range patterns {
		compiled[domain] = regexp.MustCompile(pattern)
	}

	return compiled
}

// initializeCompliancePatterns creates regex patterns for compliance detection
func initializeCompliancePatterns() map[string]*regexp.Regexp {
	patterns := map[string]string{
		"pci": `(?i)\b(pci|pci-dss|card data|payment card|cardholder|card security|payment security)\b`,
		"hipaa": `(?i)\b(hipaa|phi|protected health|health information|medical privacy|patient privacy)\b`,
		"gdpr": `(?i)\b(gdpr|data protection|privacy|personal data|consent|data subject|right to be forgotten|data portability)\b`,
	}

	compiled := make(map[string]*regexp.Regexp)
	for compliance, pattern := range patterns {
		compiled[compliance] = regexp.MustCompile(pattern)
	}

	return compiled
}

// ValidateOverlayCompatibility checks if overlays are compatible with each other and the tech stack
func (d *OverlayDetector) ValidateOverlayCompatibility(overlays []string, spec *IRSpec) []string {
	var warnings []string

	// Check for known incompatibilities
	hasFintech := false
	hasHealthcare := false

	for _, overlay := range overlays {
		switch overlay {
		case "fintech":
			hasFintech = true
		case "healthcare":
			hasHealthcare = true
		}
	}

	// Healthcare and fintech together might need special consideration
	if hasFintech && hasHealthcare {
		warnings = append(warnings, "Healthcare and fintech overlays both applied - ensure proper data separation")
	}

	// Technology stack compatibility
	if spec != nil {
		language := spec.App.Stack.Backend.Language

		// Some compliance features might need specific language support
		for _, overlay := range overlays {
			if overlay == "gdpr" && language == "javascript" {
				warnings = append(warnings, "GDPR compliance in JavaScript requires careful handling of data types")
			}
		}
	}

	return warnings
}