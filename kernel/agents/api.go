package agents

import (
	"context"
	"fmt"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/llm"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/prompts"
)

// APIAgent specializes in generating API specifications and documentation
type APIAgent struct {
	*BaseAgent
	llmClient      llm.Client
	promptComposer *prompts.PromptComposer
}

// NewAPIAgent creates a new API agent
func NewAPIAgent() *APIAgent {
	capabilities := []string{
		"openapi_specs",
		"graphql_schemas",
		"rest_documentation",
		"api_validation",
		"endpoint_testing",
		"swagger_ui",
		"postman_collections",
		"api_versioning",
		"rate_limiting",
		"authentication_specs",
	}

	return &APIAgent{
		BaseAgent: NewBaseAgent(AgentTypeAPI, "1.0.0", capabilities),
	}
}

// NewAPIAgentWithLLM creates a new API agent with LLM capabilities
func NewAPIAgentWithLLM(llmClient llm.Client, promptComposer *prompts.PromptComposer) *APIAgent {
	capabilities := []string{
		"openapi_specs",
		"graphql_schemas",
		"rest_documentation",
		"api_validation",
		"endpoint_testing",
		"swagger_ui",
		"postman_collections",
		"api_versioning",
		"rate_limiting",
		"authentication_specs",
	}

	return &APIAgent{
		BaseAgent:      NewBaseAgent(AgentTypeAPI, "2.0.0", capabilities),
		llmClient:      llmClient,
		promptComposer: promptComposer,
	}
}

// CanHandle determines if this agent can handle the given specification
func (a *APIAgent) CanHandle(spec *ir.IRSpec) bool {
	// API agent can handle specs with API endpoints or when API documentation is needed
	if len(spec.API.Endpoints) > 0 {
		return true
	}

	// Can handle GraphQL APIs
	if spec.API.Type == "graphql" {
		return true
	}

	// Can handle REST APIs
	if spec.API.Type == "rest" {
		return true
	}

	return false
}

// Generate creates API specifications and documentation from the specification
func (a *APIAgent) Generate(ctx context.Context, req *GenerationRequest) (*GenerationResult, error) {
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
			return nil, fmt.Errorf("failed to generate API specs with LLM: %w", err)
		}
	} else {
		// Fallback for non-LLM mode
		err := a.generateWithTemplates(req, result)
		if err != nil {
			return nil, fmt.Errorf("failed to generate API specs with templates: %w", err)
		}
	}

	// Update metadata
	result.Metadata.Duration = time.Since(startTime)
	result.Metadata.FilesCreated = len(result.Files)

	return result, nil
}

// generateWithLLM generates API specifications using LLM
func (a *APIAgent) generateWithLLM(ctx context.Context, req *GenerationRequest, result *GenerationResult) error {
	// Build prompt using the prompt composer
	composeReq := prompts.ComposeRequest{
		AgentType: "api",
		IRSpec:    req.Spec,
		Context:   req.Context,
	}

	promptResult, err := a.promptComposer.ComposePrompt(composeReq)
	if err != nil {
		return fmt.Errorf("failed to compose prompt: %w", err)
	}

	// Select model - API specs need precision
	model := llm.ModelClaudeSonnet

	// Call LLM to generate API specifications
	llmReq := &llm.GenerateRequest{
		Prompt:      promptResult.Prompt,
		Model:       model,
		MaxTokens:   8192,
		Temperature: 0.1, // Lower temperature for precise API specs
	}

	response, err := a.llmClient.Generate(ctx, llmReq)
	if err != nil {
		return fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse generated files
	files, err := a.parseGeneratedSpecs(response.Content, req.Spec)
	if err != nil {
		return fmt.Errorf("failed to parse generated API specs: %w", err)
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

// generateWithTemplates generates API specifications using templates (fallback)
func (a *APIAgent) generateWithTemplates(req *GenerationRequest, result *GenerationResult) error {
	// Generate basic OpenAPI spec
	if req.Spec.API.Type == "" || req.Spec.API.Type == "rest" {
		openAPISpec := a.generateOpenAPISpec(req.Spec)
		file := GeneratedFile{
			Path:     "api/openapi.yaml",
			Type:     "config",
			Language: "yaml",
			Template: "openapi_template",
			Content:  openAPISpec,
		}
		result.Files = append(result.Files, file)
	}

	// Generate GraphQL schema if specified
	if req.Spec.API.Type == "graphql" {
		graphQLSchema := a.generateGraphQLSchema(req.Spec)
		file := GeneratedFile{
			Path:     "api/schema.graphql",
			Type:     "config",
			Language: "graphql",
			Template: "graphql_template",
			Content:  graphQLSchema,
		}
		result.Files = append(result.Files, file)
	}

	return nil
}

// parseGeneratedSpecs parses LLM output into API specification files
func (a *APIAgent) parseGeneratedSpecs(content string, spec *ir.IRSpec) ([]GeneratedFile, error) {
	files := []GeneratedFile{}

	// For now, create a comprehensive API spec file
	// In a real implementation, this would parse the LLM output for different spec types

	if spec.API.Type == "" || spec.API.Type == "rest" {
		openAPIFile := GeneratedFile{
			Path:     "api/openapi.yaml",
			Type:     "config",
			Language: "yaml",
			Template: "llm_generated",
			Content:  content,
		}
		files = append(files, openAPIFile)
	}

	if spec.API.Type == "graphql" {
		graphQLFile := GeneratedFile{
			Path:     "api/schema.graphql",
			Type:     "config",
			Language: "graphql",
			Template: "llm_generated",
			Content:  content,
		}
		files = append(files, graphQLFile)
	}

	return files, nil
}

// generateOpenAPISpec generates a basic OpenAPI specification
func (a *APIAgent) generateOpenAPISpec(spec *ir.IRSpec) string {
	// Basic OpenAPI 3.0 template
	return fmt.Sprintf(`openapi: 3.0.0
info:
  title: %s
  description: %s
  version: 1.0.0
paths:
  /health:
    get:
      summary: Health check endpoint
      responses:
        '200':
          description: Service is healthy
          content:
            application/json:
              schema:
                type: object
                properties:
                  status:
                    type: string
                    example: healthy
`, spec.App.Name, spec.App.Description)
}

// generateGraphQLSchema generates a basic GraphQL schema
func (a *APIAgent) generateGraphQLSchema(spec *ir.IRSpec) string {
	return fmt.Sprintf(`# %s GraphQL Schema
# %s

type Query {
  health: HealthStatus
}

type HealthStatus {
  status: String!
  timestamp: String!
}
`, spec.App.Name, spec.App.Description)
}

// Validate checks if the generated API specifications meet requirements
func (a *APIAgent) Validate(ctx context.Context, result *GenerationResult) (*ValidationResult, error) {
	return &ValidationResult{
		Valid:    true,
		Warnings: []string{"API validation not yet implemented"},
	}, nil
}