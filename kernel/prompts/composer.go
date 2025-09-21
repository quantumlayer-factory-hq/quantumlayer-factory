package prompts

import (
	"bytes"
	"fmt"
	"sort"
	"strings"
	"text/template"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/overlays"
)

// ComposerConfig configures the prompt composition behavior
type ComposerConfig struct {
	TemplatePaths     []string `yaml:"template_paths" json:"template_paths"`
	EnableCaching     bool     `yaml:"enable_caching" json:"enable_caching"`
	DefaultLanguage   string   `yaml:"default_language" json:"default_language"`
	MaxPromptLength   int      `yaml:"max_prompt_length" json:"max_prompt_length"`
	IncludeMetadata   bool     `yaml:"include_metadata" json:"include_metadata"`
	DebugMode         bool     `yaml:"debug_mode" json:"debug_mode"`
}

// DefaultComposerConfig returns sensible defaults
func DefaultComposerConfig() ComposerConfig {
	return ComposerConfig{
		TemplatePaths:   []string{"./kernel/prompts/templates"},
		EnableCaching:   true,
		DefaultLanguage: "en",
		MaxPromptLength: 32000, // Leave room for response
		IncludeMetadata: false,
		DebugMode:       false,
	}
}

// PromptComposer handles dynamic prompt generation with overlay integration
type PromptComposer struct {
	config      ComposerConfig
	templates   map[string]*template.Template
	cache       map[string]string
	templateMgr *TemplateManager
}

// NewPromptComposer creates a new prompt composer
func NewPromptComposer(config ComposerConfig) *PromptComposer {
	return &PromptComposer{
		config:      config,
		templates:   make(map[string]*template.Template),
		cache:       make(map[string]string),
		templateMgr: NewTemplateManager(config.TemplatePaths),
	}
}

// ComposeRequest represents a request to compose a prompt
type ComposeRequest struct {
	AgentType         string                        `json:"agent_type"`          // "backend", "frontend", "database"
	IRSpec            *ir.IRSpec                    `json:"ir_spec"`
	OverlayResult     *overlays.ResolverResult      `json:"overlay_result"`
	Context           map[string]interface{}        `json:"context,omitempty"`
	CustomPrompts     []string                      `json:"custom_prompts,omitempty"`
	IncludeSections   []string                      `json:"include_sections,omitempty"` // "system", "context", "examples"
	ExcludeSections   []string                      `json:"exclude_sections,omitempty"`
	TemplateOverrides map[string]string             `json:"template_overrides,omitempty"`
}

// ComposeResult contains the composed prompt and metadata
type ComposeResult struct {
	Prompt            string            `json:"prompt"`
	AgentType         string            `json:"agent_type"`
	TotalLength       int               `json:"total_length"`
	SectionLengths    map[string]int    `json:"section_lengths"`
	AppliedOverlays   []string          `json:"applied_overlays"`
	UsedTemplate      string            `json:"used_template,omitempty"`
	EnhancementsCount int               `json:"enhancements_count"`
	Metadata          map[string]string `json:"metadata,omitempty"`
	Warnings          []string          `json:"warnings,omitempty"`
}

// ComposePrompt creates a complete prompt for the specified agent
func (c *PromptComposer) ComposePrompt(req ComposeRequest) (*ComposeResult, error) {
	// Get base template for agent type
	baseTemplate, err := c.templateMgr.GetTemplate(req.AgentType)
	if err != nil {
		return nil, fmt.Errorf("failed to get template for %s: %w", req.AgentType, err)
	}

	// Prepare template context
	templateCtx := c.buildTemplateContext(req)

	// Apply template overrides if provided
	if len(req.TemplateOverrides) > 0 {
		baseTemplate = c.applyTemplateOverrides(baseTemplate, req.TemplateOverrides)
	}

	// Execute base template
	basePrompt, err := c.executeTemplate(baseTemplate, templateCtx)
	if err != nil {
		return nil, fmt.Errorf("failed to execute base template: %w", err)
	}

	// Apply overlay enhancements
	enhancedPrompt, enhancementCount := c.applyOverlayEnhancements(basePrompt, req)

	// Build final prompt
	finalPrompt := c.buildFinalPrompt(enhancedPrompt, req)

	// Calculate metrics
	result := &ComposeResult{
		Prompt:            finalPrompt,
		AgentType:         req.AgentType,
		TotalLength:       len(finalPrompt),
		SectionLengths:    c.calculateSectionLengths(finalPrompt),
		AppliedOverlays:   c.getAppliedOverlayNames(req.OverlayResult),
		UsedTemplate:      fmt.Sprintf("%s.tmpl", req.AgentType),
		EnhancementsCount: enhancementCount,
		Warnings:          []string{},
	}

	// Add metadata if requested
	if c.config.IncludeMetadata {
		result.Metadata = c.buildMetadata(req)
	}

	// Check for warnings
	if result.TotalLength > c.config.MaxPromptLength {
		result.Warnings = append(result.Warnings,
			fmt.Sprintf("Prompt length (%d) exceeds recommended maximum (%d)",
				result.TotalLength, c.config.MaxPromptLength))
	}

	return result, nil
}

// buildTemplateContext creates the context for template execution
func (c *PromptComposer) buildTemplateContext(req ComposeRequest) map[string]interface{} {
	ctx := map[string]interface{}{
		"AgentType": req.AgentType,
		"App":       req.IRSpec.App,
		"Features":  req.IRSpec.App.Features,
		"Stack":     req.IRSpec.App.Stack,
		"API":       req.IRSpec.API,
		"Data":      req.IRSpec.Data,
		"Security":  req.IRSpec.NonFunctionals.Security,
		"Context":   req.Context,
	}

	// Add overlay-specific context
	if req.OverlayResult != nil {
		ctx["Overlays"] = req.OverlayResult.Metadata
		ctx["ValidationRules"] = req.OverlayResult.ValidationRules

		// Group features by overlay
		overlayFeatures := make(map[string][]ir.Feature)
		for _, overlay := range req.OverlayResult.AppliedOrder {
			overlayFeatures[overlay] = []ir.Feature{}
		}

		// Add features to their respective overlays (simplified)
		for _, feature := range req.IRSpec.App.Features {
			if feature.Type == "overlay" {
				// In a real implementation, we'd track which overlay added which feature
				overlayFeatures["unknown"] = append(overlayFeatures["unknown"], feature)
			}
		}
		ctx["OverlayFeatures"] = overlayFeatures
	}

	return ctx
}

// applyOverlayEnhancements injects overlay-specific prompt enhancements
func (c *PromptComposer) applyOverlayEnhancements(basePrompt string, req ComposeRequest) (string, int) {
	if req.OverlayResult == nil {
		return basePrompt, 0
	}

	enhancements := req.OverlayResult.PromptChanges[req.AgentType]
	if len(enhancements) == 0 {
		return basePrompt, 0
	}

	// Sort enhancements by priority (higher priority applied last)
	sort.Slice(enhancements, func(i, j int) bool {
		return enhancements[i].Priority < enhancements[j].Priority
	})

	result := basePrompt
	sections := c.parseSections(result)

	for _, enhancement := range enhancements {
		// Skip if enhancement doesn't match conditions
		if !c.matchesConditions(enhancement, req) {
			continue
		}

		// Apply enhancement to appropriate section
		sections = c.applyEnhancementToSection(sections, enhancement)
	}

	return c.reconstructPrompt(sections), len(enhancements)
}

// parseSections breaks down the prompt into sections
func (c *PromptComposer) parseSections(prompt string) map[string]string {
	sections := make(map[string]string)

	// Simple section parsing - look for section markers
	lines := strings.Split(prompt, "\n")
	currentSection := "system"
	currentContent := []string{}

	for _, line := range lines {
		if strings.HasPrefix(line, "## ") {
			// Save previous section
			if len(currentContent) > 0 {
				sections[currentSection] = strings.Join(currentContent, "\n")
			}
			// Start new section
			currentSection = strings.ToLower(strings.TrimSpace(strings.TrimPrefix(line, "## ")))
			currentContent = []string{}
		} else {
			currentContent = append(currentContent, line)
		}
	}

	// Save final section
	if len(currentContent) > 0 {
		sections[currentSection] = strings.Join(currentContent, "\n")
	}

	return sections
}

// applyEnhancementToSection applies an enhancement to the appropriate section
func (c *PromptComposer) applyEnhancementToSection(sections map[string]string, enhancement overlays.PromptEnhancement) map[string]string {
	targetSection := enhancement.Section
	if _, exists := sections[targetSection]; !exists {
		sections[targetSection] = ""
	}

	content := enhancement.Content
	switch enhancement.Position {
	case "before":
		sections[targetSection] = content + "\n\n" + sections[targetSection]
	case "after":
		sections[targetSection] = sections[targetSection] + "\n\n" + content
	case "replace":
		sections[targetSection] = content
	default:
		// Default to append
		sections[targetSection] = sections[targetSection] + "\n\n" + content
	}

	return sections
}

// reconstructPrompt rebuilds the prompt from sections
func (c *PromptComposer) reconstructPrompt(sections map[string]string) string {
	var result strings.Builder

	// Define section order
	sectionOrder := []string{"system", "context", "requirements", "examples", "instructions"}

	for _, section := range sectionOrder {
		if content, exists := sections[section]; exists && strings.TrimSpace(content) != "" {
			result.WriteString(fmt.Sprintf("## %s\n", strings.Title(section)))
			result.WriteString(content)
			result.WriteString("\n\n")
		}
	}

	// Add any remaining sections
	for section, content := range sections {
		if !contains(sectionOrder, section) && strings.TrimSpace(content) != "" {
			result.WriteString(fmt.Sprintf("## %s\n", strings.Title(section)))
			result.WriteString(content)
			result.WriteString("\n\n")
		}
	}

	return strings.TrimSpace(result.String())
}

// matchesConditions checks if an enhancement should be applied based on conditions
func (c *PromptComposer) matchesConditions(enhancement overlays.PromptEnhancement, req ComposeRequest) bool {
	if len(enhancement.Conditions) == 0 {
		return true
	}

	// Check conditions against IR spec and context
	for key, expectedValue := range enhancement.Conditions {
		switch key {
		case "language":
			if req.IRSpec.App.Stack.Backend.Language != expectedValue {
				return false
			}
		case "framework":
			if req.IRSpec.App.Stack.Backend.Framework != expectedValue {
				return false
			}
		case "app_type":
			if req.IRSpec.App.Type != expectedValue {
				return false
			}
		case "domain":
			if req.IRSpec.App.Domain != expectedValue {
				return false
			}
		default:
			// Check custom context conditions
			if ctxValue, exists := req.Context[key]; exists {
				if fmt.Sprintf("%v", ctxValue) != expectedValue {
					return false
				}
			}
		}
	}

	return true
}

// Helper functions
func (c *PromptComposer) executeTemplate(tmpl *template.Template, ctx map[string]interface{}) (string, error) {
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, ctx); err != nil {
		return "", err
	}
	return buf.String(), nil
}

func (c *PromptComposer) applyTemplateOverrides(tmpl *template.Template, overrides map[string]string) *template.Template {
	// In a full implementation, this would parse and modify the template
	// For now, return the original template
	return tmpl
}

func (c *PromptComposer) buildFinalPrompt(prompt string, req ComposeRequest) string {
	var result strings.Builder

	// Add custom prompts if provided
	for _, customPrompt := range req.CustomPrompts {
		result.WriteString(customPrompt)
		result.WriteString("\n\n")
	}

	result.WriteString(prompt)
	return result.String()
}

func (c *PromptComposer) calculateSectionLengths(prompt string) map[string]int {
	sections := c.parseSections(prompt)
	lengths := make(map[string]int)
	for section, content := range sections {
		lengths[section] = len(content)
	}
	return lengths
}

func (c *PromptComposer) getAppliedOverlayNames(result *overlays.ResolverResult) []string {
	if result == nil {
		return []string{}
	}
	return result.AppliedOrder
}

func (c *PromptComposer) buildMetadata(req ComposeRequest) map[string]string {
	metadata := map[string]string{
		"agent_type": req.AgentType,
		"app_name":   req.IRSpec.App.Name,
		"app_type":   req.IRSpec.App.Type,
		"language":   req.IRSpec.App.Stack.Backend.Language,
	}

	if req.IRSpec.App.Stack.Backend.Framework != "" {
		metadata["framework"] = req.IRSpec.App.Stack.Backend.Framework
	}

	return metadata
}

// Utility functions
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}