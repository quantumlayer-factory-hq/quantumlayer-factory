package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/llm"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/prompts"
)

// DatabaseAgent specializes in generating database-related code
type DatabaseAgent struct {
	*BaseAgent
	llmClient      llm.Client
	promptComposer *prompts.PromptComposer
}

// NewDatabaseAgent creates a new database agent
func NewDatabaseAgent() *DatabaseAgent {
	capabilities := []string{
		"schema_generation",
		"migrations",
		"seeds",
		"indexes",
		"relationships",
		"orm_models",
		"queries",
		"stored_procedures",
		"backup_scripts",
	}

	return &DatabaseAgent{
		BaseAgent: NewBaseAgent(AgentTypeDatabase, "1.0.0", capabilities),
	}
}

// NewDatabaseAgentWithLLM creates a new database agent with LLM capabilities
func NewDatabaseAgentWithLLM(llmClient llm.Client, promptComposer *prompts.PromptComposer) *DatabaseAgent {
	capabilities := []string{
		"schema_generation",
		"migrations",
		"seeds",
		"indexes",
		"relationships",
		"orm_models",
		"queries",
		"stored_procedures",
		"backup_scripts",
	}

	return &DatabaseAgent{
		BaseAgent:      NewBaseAgent(AgentTypeDatabase, "2.0.0", capabilities),
		llmClient:      llmClient,
		promptComposer: promptComposer,
	}
}

// CanHandle determines if this agent can handle the given specification
func (a *DatabaseAgent) CanHandle(spec *ir.IRSpec) bool {
	// Database agent can handle specs with entities or data requirements
	if len(spec.Data.Entities) > 0 {
		return true
	}

	if len(spec.Data.Relationships) > 0 {
		return true
	}

	return false
}

// Generate creates database code from the specification
func (a *DatabaseAgent) Generate(ctx context.Context, req *GenerationRequest) (*GenerationResult, error) {
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
			return nil, fmt.Errorf("failed to generate database with LLM: %w", err)
		}
	} else {
		// Fallback for non-LLM mode
		result.Warnings = []string{"Database agent running in template mode - limited functionality"}
	}

	// Update metadata
	result.Metadata.Duration = time.Since(startTime)
	result.Metadata.FilesCreated = len(result.Files)

	return result, nil
}

// generateWithLLM generates database code using LLM
func (a *DatabaseAgent) generateWithLLM(ctx context.Context, req *GenerationRequest, result *GenerationResult) error {
	// Build prompt using the prompt composer
	composeReq := prompts.ComposeRequest{
		AgentType: "database",
		IRSpec:    req.Spec,
		Context:   req.Context,
	}

	promptResult, err := a.promptComposer.ComposePrompt(composeReq)
	if err != nil {
		return fmt.Errorf("failed to compose prompt: %w", err)
	}

	// Select model - database operations need high accuracy
	model := llm.ModelClaudeSonnet

	// Call LLM to generate code
	llmReq := &llm.GenerateRequest{
		Prompt:      promptResult.Prompt,
		Model:       model,
		MaxTokens:   8192,
		Temperature: 0.1, // Lower temperature for database schemas
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

// parseGeneratedCode parses LLM output into database files
func (a *DatabaseAgent) parseGeneratedCode(content string, spec *ir.IRSpec) ([]GeneratedFile, error) {
	files := []GeneratedFile{}

	// Create main schema file
	schemaFile := GeneratedFile{
		Path:     fmt.Sprintf("database/schema.sql"),
		Type:     "source",
		Language: "sql",
		Template: "llm_generated",
		Content:  content,
	}
	files = append(files, schemaFile)

	// If migrations are enabled, create migration file
	if a.hasCapability("migrations") {
		migrationFile := GeneratedFile{
			Path:     fmt.Sprintf("database/migrations/001_initial_schema.sql"),
			Type:     "source",
			Language: "sql",
			Template: "llm_generated",
			Content:  content,
		}
		files = append(files, migrationFile)
	}

	return files, nil
}

// hasCapability checks if the agent has a specific capability
func (a *DatabaseAgent) hasCapability(capability string) bool {
	for _, cap := range a.capabilities {
		if cap == capability {
			return true
		}
	}
	return false
}

// Validate checks if the generated database code meets requirements
func (a *DatabaseAgent) Validate(ctx context.Context, result *GenerationResult) (*ValidationResult, error) {
	return &ValidationResult{
		Valid:    false,
		Warnings: []string{"Database validation not yet implemented"},
	}, nil
}