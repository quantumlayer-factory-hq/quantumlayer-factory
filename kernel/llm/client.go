package llm

import (
	"fmt"
	"strings"
)

// ClientFactory creates LLM clients based on configuration
type ClientFactory struct {
	config *Config
}

// NewClientFactory creates a new client factory
func NewClientFactory(config *Config) *ClientFactory {
	return &ClientFactory{
		config: config,
	}
}

// CreateClient creates a client for the specified provider
func (f *ClientFactory) CreateClient(provider Provider) (Client, error) {
	switch provider {
	case ProviderBedrock:
		return NewBedrockClient(f.config.Bedrock)
	case ProviderAzure:
		return NewAzureOpenAIClient(f.config.Azure)
	default:
		return nil, fmt.Errorf("unsupported provider: %s", provider)
	}
}

// CreateDefaultClient creates a client using the default provider
func (f *ClientFactory) CreateDefaultClient() (Client, error) {
	return f.CreateClient(f.config.DefaultProvider)
}

// CreateAllClients creates clients for all configured providers
func (f *ClientFactory) CreateAllClients() (map[Provider]Client, error) {
	clients := make(map[Provider]Client)

	// Create Bedrock client if configured
	if f.config.Bedrock.Region != "" {
		client, err := f.CreateClient(ProviderBedrock)
		if err == nil {
			clients[ProviderBedrock] = client
		}
	}

	// Create Azure client if configured
	if f.config.Azure.Endpoint != "" {
		client, err := f.CreateClient(ProviderAzure)
		if err == nil {
			clients[ProviderAzure] = client
		}
	}

	if len(clients) == 0 {
		return nil, fmt.Errorf("no LLM providers configured")
	}

	return clients, nil
}

// GetDefaultModel returns the default model for a provider
func GetDefaultModel(provider Provider) Model {
	switch provider {
	case ProviderBedrock:
		return ModelClaudeSonnet
	case ProviderAzure:
		return ModelGPT4Turbo
	default:
		return ModelClaudeSonnet
	}
}

// GetFastModel returns the fastest/cheapest model for a provider
func GetFastModel(provider Provider) Model {
	switch provider {
	case ProviderBedrock:
		return ModelClaudeHaiku
	case ProviderAzure:
		return ModelGPT35
	default:
		return ModelClaudeHaiku
	}
}

// GetAdvancedModel returns the most capable model for a provider
func GetAdvancedModel(provider Provider) Model {
	switch provider {
	case ProviderBedrock:
		return ModelClaude35
	case ProviderAzure:
		return ModelGPT4Turbo
	default:
		return ModelClaude35
	}
}

// ValidateModel checks if a model is valid for a provider
func ValidateModel(provider Provider, model Model) bool {
	validModels := GetValidModels(provider)
	for _, validModel := range validModels {
		if validModel == model {
			return true
		}
	}
	return false
}

// GetValidModels returns all valid models for a provider
func GetValidModels(provider Provider) []Model {
	switch provider {
	case ProviderBedrock:
		return []Model{
			ModelClaudeHaiku,
			ModelClaudeSonnet,
			ModelClaude35,
			ModelClaude37,
		}
	case ProviderAzure:
		return []Model{
			ModelGPT35,
			ModelGPT4,
			ModelGPT4Turbo,
			ModelGPT41,
			ModelGPT41Mini,
			ModelGPT41Nano,
			ModelGPT5,
			ModelO4Mini,
		}
	default:
		return []Model{}
	}
}

// EstimateCost estimates the cost for a request
func EstimateCost(provider Provider, model Model, inputTokens, outputTokens int) float64 {
	// Rough cost estimates (in USD per 1K tokens)
	// These should be updated with actual pricing
	costs := map[Provider]map[Model]struct {
		Input  float64
		Output float64
	}{
		ProviderBedrock: {
			ModelClaudeHaiku:  {Input: 0.00025, Output: 0.00125},
			ModelClaudeSonnet: {Input: 0.003, Output: 0.015},
			ModelClaude35:     {Input: 0.003, Output: 0.015},
		},
		ProviderAzure: {
			ModelGPT4Turbo: {Input: 0.01, Output: 0.03},
			ModelGPT35:     {Input: 0.0005, Output: 0.0015},
		},
	}

	if providerCosts, ok := costs[provider]; ok {
		if modelCost, ok := providerCosts[model]; ok {
			inputCost := float64(inputTokens) / 1000.0 * modelCost.Input
			outputCost := float64(outputTokens) / 1000.0 * modelCost.Output
			return inputCost + outputCost
		}
	}

	// Default fallback cost
	return float64(inputTokens+outputTokens) / 1000.0 * 0.002
}

// SelectOptimalModel selects the best model based on task complexity
func SelectOptimalModel(provider Provider, taskComplexity TaskComplexity) Model {
	switch taskComplexity {
	case TaskComplexitySimple:
		return GetFastModel(provider)
	case TaskComplexityMedium:
		return GetDefaultModel(provider)
	case TaskComplexityComplex:
		return GetAdvancedModel(provider)
	default:
		return GetDefaultModel(provider)
	}
}

// TaskComplexity represents the complexity of a generation task
type TaskComplexity int

const (
	TaskComplexitySimple TaskComplexity = iota
	TaskComplexityMedium
	TaskComplexityComplex
)

// AnalyzeTaskComplexity analyzes a prompt to determine task complexity
func AnalyzeTaskComplexity(prompt string) TaskComplexity {
	// Simple heuristics for now - could be enhanced with ML
	promptLen := len(prompt)

	// Check for complexity indicators
	complexKeywords := []string{
		"architecture", "design pattern", "complex", "advanced",
		"microservices", "distributed", "scalable", "enterprise",
		"security", "compliance", "integration", "optimization",
	}

	simpleKeywords := []string{
		"simple", "basic", "crud", "hello world", "quick",
		"minimal", "prototype", "demo", "example",
	}

	complexCount := 0
	simpleCount := 0

	for _, keyword := range complexKeywords {
		if containsIgnoreCase(prompt, keyword) {
			complexCount++
		}
	}

	for _, keyword := range simpleKeywords {
		if containsIgnoreCase(prompt, keyword) {
			simpleCount++
		}
	}

	// Decision logic
	if complexCount > simpleCount || promptLen > 500 {
		return TaskComplexityComplex
	} else if simpleCount > 0 && promptLen < 100 {
		return TaskComplexitySimple
	}

	return TaskComplexityMedium
}

// containsIgnoreCase checks if a string contains a substring (case insensitive)
func containsIgnoreCase(s, substr string) bool {
	return len(s) >= len(substr) &&
		   strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}