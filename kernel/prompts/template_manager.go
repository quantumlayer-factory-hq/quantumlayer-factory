package prompts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"text/template"
)

// TemplateManager handles loading and managing prompt templates
type TemplateManager struct {
	templatePaths []string
	templates     map[string]*template.Template
	funcMap       template.FuncMap
}

// NewTemplateManager creates a new template manager
func NewTemplateManager(templatePaths []string) *TemplateManager {
	if len(templatePaths) == 0 {
		templatePaths = []string{"./kernel/prompts/templates"}
	}

	funcMap := template.FuncMap{
		"join":        strings.Join,
		"toUpper":     strings.ToUpper,
		"toLower":     strings.ToLower,
		"hasPrefix":   strings.HasPrefix,
		"hasSuffix":   strings.HasSuffix,
		"contains":    strings.Contains,
		"formatList":  formatList,
		"formatCode":  formatCode,
		"indent":      indent,
	}

	return &TemplateManager{
		templatePaths: templatePaths,
		templates:     make(map[string]*template.Template),
		funcMap:       funcMap,
	}
}

// GetTemplate loads and returns a template for the specified agent type
func (tm *TemplateManager) GetTemplate(agentType string) (*template.Template, error) {
	// Check cache first
	if tmpl, exists := tm.templates[agentType]; exists {
		return tmpl, nil
	}

	// Try to load template from file system
	templateFile := fmt.Sprintf("%s.tmpl", agentType)

	for _, templatePath := range tm.templatePaths {
		fullPath := filepath.Join(templatePath, templateFile)
		if _, err := os.Stat(fullPath); err == nil {
			tmpl, err := tm.loadTemplate(fullPath, agentType)
			if err != nil {
				return nil, fmt.Errorf("failed to load template %s: %w", fullPath, err)
			}

			// Cache the template
			tm.templates[agentType] = tmpl
			return tmpl, nil
		}
	}

	// If no template file found, create a basic template
	return tm.createBasicTemplate(agentType)
}

// loadTemplate loads a template from file
func (tm *TemplateManager) loadTemplate(filePath, name string) (*template.Template, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	tmpl, err := template.New(name).Funcs(tm.funcMap).Parse(string(content))
	if err != nil {
		return nil, fmt.Errorf("failed to parse template: %w", err)
	}

	return tmpl, nil
}

// createBasicTemplate creates a fallback template when no file is found
func (tm *TemplateManager) createBasicTemplate(agentType string) (*template.Template, error) {
	basicTemplate := `## System

You are a {{.AgentType}} code generation agent specialized in creating production-ready {{.AgentType}} code.

## Context

Application: {{.App.Name}}
Type: {{.App.Type}}
Language: {{.Stack.Backend.Language}}
{{- if .Stack.Backend.Framework}}
Framework: {{.Stack.Backend.Framework}}
{{- end}}

## Requirements

Generate clean, maintainable, and well-documented {{.AgentType}} code that follows best practices for:
{{- range .Features}}
- {{.Name}}: {{.Description}}
{{- end}}

## Instructions

1. Follow the specified technology stack and patterns
2. Include appropriate error handling and logging
3. Add comprehensive comments and documentation
4. Ensure code is production-ready and secure
5. Follow industry best practices and conventions
6. Include necessary imports and dependencies

Generate the complete {{.AgentType}} implementation.`

	tmpl, err := template.New(agentType).Funcs(tm.funcMap).Parse(basicTemplate)
	if err != nil {
		return nil, fmt.Errorf("failed to create basic template: %w", err)
	}

	// Cache the basic template
	tm.templates[agentType] = tmpl
	return tmpl, nil
}

// ReloadTemplates clears the cache and forces reload of all templates
func (tm *TemplateManager) ReloadTemplates() {
	tm.templates = make(map[string]*template.Template)
}

// ListAvailableTemplates returns a list of available template files
func (tm *TemplateManager) ListAvailableTemplates() ([]string, error) {
	var templates []string

	for _, templatePath := range tm.templatePaths {
		files, err := filepath.Glob(filepath.Join(templatePath, "*.tmpl"))
		if err != nil {
			continue
		}

		for _, file := range files {
			name := strings.TrimSuffix(filepath.Base(file), ".tmpl")
			templates = append(templates, name)
		}
	}

	return templates, nil
}

// ValidateTemplate checks if a template is valid
func (tm *TemplateManager) ValidateTemplate(agentType string) error {
	_, err := tm.GetTemplate(agentType)
	return err
}

// Helper functions for templates

// formatList formats a slice of items as a bulleted list
func formatList(items []interface{}) string {
	var result strings.Builder
	for _, item := range items {
		result.WriteString(fmt.Sprintf("- %v\n", item))
	}
	return result.String()
}

// formatCode wraps text in code blocks
func formatCode(code string, language string) string {
	if language == "" {
		language = "text"
	}
	return fmt.Sprintf("```%s\n%s\n```", language, code)
}

// indent adds indentation to each line of text
func indent(text string, spaces int) string {
	indentStr := strings.Repeat(" ", spaces)
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		if strings.TrimSpace(line) != "" {
			lines[i] = indentStr + line
		}
	}
	return strings.Join(lines, "\n")
}