package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/llm"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/prompts"
)

// FrontendAgent specializes in generating frontend code
type FrontendAgent struct {
	*BaseAgent
	llmClient      llm.Client
	promptComposer *prompts.PromptComposer
}

// NewFrontendAgent creates a new frontend agent
func NewFrontendAgent() *FrontendAgent {
	capabilities := []string{
		"react_components",
		"vue_components",
		"angular_components",
		"ui_layouts",
		"routing",
		"state_management",
		"styling",
		"forms",
		"authentication_ui",
	}

	return &FrontendAgent{
		BaseAgent: NewBaseAgent(AgentTypeFrontend, "1.0.0", capabilities),
	}
}

// NewFrontendAgentWithLLM creates a new frontend agent with LLM capabilities
func NewFrontendAgentWithLLM(llmClient llm.Client, promptComposer *prompts.PromptComposer) *FrontendAgent {
	capabilities := []string{
		"react_components",
		"vue_components",
		"angular_components",
		"ui_layouts",
		"routing",
		"state_management",
		"styling",
		"forms",
		"authentication_ui",
	}

	return &FrontendAgent{
		BaseAgent:      NewBaseAgent(AgentTypeFrontend, "2.0.0", capabilities),
		llmClient:      llmClient,
		promptComposer: promptComposer,
	}
}

// CanHandle determines if this agent can handle the given specification
func (a *FrontendAgent) CanHandle(spec *ir.IRSpec) bool {
	// Frontend agent can handle web applications with UI
	if spec.App.Type == "web" && len(spec.UI.Pages) > 0 {
		return true
	}

	// Can handle single page applications
	if spec.App.Type == "spa" {
		return true
	}

	return false
}

// Generate creates frontend code from the specification
func (a *FrontendAgent) Generate(ctx context.Context, req *GenerationRequest) (*GenerationResult, error) {
	startTime := time.Now()

	result := &GenerationResult{
		Success: true,
		Files:   []GeneratedFile{},
		Metadata: GenerationMetadata{
			AgentType:    a.GetType(),
			AgentVersion: a.version,
			GeneratedAt:  startTime,
		},
	}

	// Generate using LLM if available
	if a.llmClient != nil && a.promptComposer != nil {
		err := a.generateWithLLM(ctx, req, result)
		if err != nil {
			return nil, fmt.Errorf("failed to generate frontend with LLM: %w", err)
		}
	} else {
		// Fallback for non-LLM mode
		result.Warnings = []string{"Frontend agent running in template mode - limited functionality"}
	}

	// Update metadata
	result.Metadata.Duration = time.Since(startTime)
	result.Metadata.FilesCreated = len(result.Files)

	return result, nil
}

// generateWithLLM generates frontend code using LLM
func (a *FrontendAgent) generateWithLLM(ctx context.Context, req *GenerationRequest, result *GenerationResult) error {
	// Build prompt using the prompt composer
	composeReq := prompts.ComposeRequest{
		AgentType: "frontend",
		IRSpec:    req.Spec,
		Context:   req.Context,
	}

	promptResult, err := a.promptComposer.ComposePrompt(composeReq)
	if err != nil {
		return fmt.Errorf("failed to compose prompt: %w", err)
	}

	// Select model - frontend typically needs balanced capabilities
	model := llm.ModelClaudeSonnet

	// Call LLM to generate code
	llmReq := &llm.GenerateRequest{
		Prompt:      promptResult.Prompt,
		Model:       model,
		MaxTokens:   8192,
		Temperature: 0.2,
	}

	response, err := a.llmClient.Generate(ctx, llmReq)
	if err != nil {
		return fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse generated files
	files, err := a.parseGeneratedCode(response.Content, req.Spec)
	if err != nil {
		return fmt.Errorf("failed to parse generated code: %w", err)
	}

	result.Files = append(result.Files, files...)

	// Add LLM usage metadata
	result.Metadata.LLMUsage = &LLMUsageMetadata{
		Provider:         string(response.Provider),
		Model:           string(response.Model),
		PromptTokens:    response.Usage.PromptTokens,
		CompletionTokens: response.Usage.CompletionTokens,
		TotalTokens:     response.Usage.TotalTokens,
		Cost:            response.Usage.Cost,
	}

	return nil
}

// parseGeneratedCode parses LLM output into frontend files
func (a *FrontendAgent) parseGeneratedCode(content string, spec *ir.IRSpec) ([]GeneratedFile, error) {
	// Similar to backend agent parsing logic
	// For now, create a simple structure
	files := []GeneratedFile{}

	framework := spec.App.Stack.Frontend.Framework
	if framework == "" {
		framework = "react"
	}

	ext := a.getFileExtension(framework)

	mainFile := GeneratedFile{
		Path:     fmt.Sprintf("src/App%s", ext),
		Type:     "source",
		Language: a.getLanguage(framework),
		Template: "llm_generated",
		Content:  content,
	}
	files = append(files, mainFile)

	return files, nil
}

// getFileExtension returns appropriate file extension for frontend framework
func (a *FrontendAgent) getFileExtension(framework string) string {
	switch framework {
	case "react":
		return ".jsx"
	case "vue":
		return ".vue"
	case "angular":
		return ".ts"
	default:
		return ".js"
	}
}

// getLanguage returns the primary language for the framework
func (a *FrontendAgent) getLanguage(framework string) string {
	switch framework {
	case "angular":
		return "typescript"
	case "react", "vue":
		return "javascript"
	default:
		return "javascript"
	}
}

// Validate checks if the generated frontend code meets requirements
func (a *FrontendAgent) Validate(ctx context.Context, result *GenerationResult) (*ValidationResult, error) {
	return &ValidationResult{
		Valid:    false,
		Warnings: []string{"Frontend validation not yet implemented"},
	}, nil
}