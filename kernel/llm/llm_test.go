package llm

import (
	"testing"
	"time"
)

func TestProviderTypes(t *testing.T) {
	// Test provider constants
	if ProviderBedrock != "bedrock" {
		t.Errorf("Expected ProviderBedrock to be 'bedrock', got %s", ProviderBedrock)
	}
	if ProviderAzure != "azure" {
		t.Errorf("Expected ProviderAzure to be 'azure', got %s", ProviderAzure)
	}
}

func TestModelValidation(t *testing.T) {
	// Test Bedrock model validation
	if !ValidateModel(ProviderBedrock, ModelClaudeSonnet) {
		t.Error("Claude Sonnet should be valid for Bedrock")
	}
	if ValidateModel(ProviderBedrock, ModelGPT4Turbo) {
		t.Error("GPT-4 should not be valid for Bedrock")
	}

	// Test Azure model validation
	if !ValidateModel(ProviderAzure, ModelGPT4Turbo) {
		t.Error("GPT-4 should be valid for Azure")
	}
	if ValidateModel(ProviderAzure, ModelClaudeSonnet) {
		t.Error("Claude should not be valid for Azure")
	}
}

func TestTaskComplexityAnalysis(t *testing.T) {
	tests := []struct {
		prompt   string
		expected TaskComplexity
	}{
		{"Hello world", TaskComplexitySimple},
		{"Create a simple CRUD API", TaskComplexitySimple},
		{"Build a user management system", TaskComplexityMedium},
		{"Design a scalable microservices architecture with advanced security", TaskComplexityComplex},
		{"Enterprise-grade distributed system with compliance requirements", TaskComplexityComplex},
	}

	for _, test := range tests {
		result := AnalyzeTaskComplexity(test.prompt)
		if result != test.expected {
			t.Errorf("For prompt '%s', expected complexity %d, got %d", test.prompt, test.expected, result)
		}
	}
}

func TestCostEstimation(t *testing.T) {
	// Test cost estimation for different providers
	bedrockCost := EstimateCost(ProviderBedrock, ModelClaudeHaiku, 1000, 500)
	azureCost := EstimateCost(ProviderAzure, ModelGPT4Turbo, 1000, 500)

	if bedrockCost <= 0 {
		t.Error("Bedrock cost should be positive")
	}
	if azureCost <= 0 {
		t.Error("Azure cost should be positive")
	}

	// GPT-4 should generally be more expensive than Claude Haiku
	if azureCost <= bedrockCost {
		t.Logf("Note: Azure GPT-4 ($%.4f) is not more expensive than Bedrock Haiku ($%.4f) as expected", azureCost, bedrockCost)
	}
}

func TestModelSelection(t *testing.T) {
	// Test optimal model selection
	simpleModel := SelectOptimalModel(ProviderBedrock, TaskComplexitySimple)
	complexModel := SelectOptimalModel(ProviderBedrock, TaskComplexityComplex)

	if simpleModel == complexModel {
		t.Error("Simple and complex tasks should use different models")
	}

	// Fast model should be different from advanced model
	fastModel := GetFastModel(ProviderBedrock)
	advancedModel := GetAdvancedModel(ProviderBedrock)

	if fastModel == advancedModel {
		t.Error("Fast and advanced models should be different")
	}
}

func TestCacheKeyGeneration(t *testing.T) {
	req1 := &GenerateRequest{
		Prompt:      "Test prompt",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	req2 := &GenerateRequest{
		Prompt:      "Test prompt",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	req3 := &GenerateRequest{
		Prompt:      "Different prompt",
		MaxTokens:   100,
		Temperature: 0.7,
	}

	key1 := GenerateCacheKey(req1)
	key2 := GenerateCacheKey(req2)
	key3 := GenerateCacheKey(req3)

	// Same requests should generate same keys
	if key1 != key2 {
		t.Error("Identical requests should generate identical cache keys")
	}

	// Different requests should generate different keys
	if key1 == key3 {
		t.Error("Different requests should generate different cache keys")
	}
}

func TestConfigDefaults(t *testing.T) {
	config := GetExampleConfig()

	// Test default values
	if config.DefaultProvider != ProviderBedrock {
		t.Error("Default provider should be Bedrock")
	}

	if config.Bedrock.Region != "eu-west-2" {
		t.Error("Default Bedrock region should be eu-west-2 (London)")
	}

	if config.Azure.Location != "uksouth" {
		t.Error("Default Azure location should be uksouth")
	}

	if config.Cache.TTL != 1*time.Hour {
		t.Error("Default cache TTL should be 1 hour")
	}

	if config.Budget.MonthlyLimit != 500.00 {
		t.Error("Default monthly budget should be $500")
	}
}

func TestErrorTypes(t *testing.T) {
	err := &Error{
		Code:     ErrorCodeRateLimit,
		Message:  "Rate limit exceeded",
		Provider: ProviderBedrock,
		Retry:    true,
	}

	if err.Error() == "" {
		t.Error("Error should have a string representation")
	}

	if !err.Retry {
		t.Error("Rate limit errors should be retryable")
	}
}

func TestClientFactory(t *testing.T) {
	config := GetExampleConfig()

	// Set minimal required config for testing
	config.Bedrock.AccessKeyID = "test"
	config.Bedrock.SecretAccessKey = "test"
	config.Azure.APIKey = "test"
	config.Azure.Endpoint = "https://test.openai.azure.com"

	factory := NewClientFactory(config)
	if factory == nil {
		t.Error("Factory should not be nil")
	}

	// Test default model selection
	defaultModel := GetDefaultModel(ProviderBedrock)
	if defaultModel == "" {
		t.Error("Default model should not be empty")
	}

	fastModel := GetFastModel(ProviderAzure)
	if fastModel == "" {
		t.Error("Fast model should not be empty")
	}
}