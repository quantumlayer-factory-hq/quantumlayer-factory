package llm

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// LoadConfig loads LLM configuration from file and environment
func LoadConfig(configPath string) (*Config, error) {
	config := &Config{
		DefaultProvider: ProviderBedrock, // Default to Bedrock
		Bedrock: BedrockConfig{
			Region:       "eu-west-2", // London region
			DefaultModel: ModelClaudeSonnet,
			Models: map[string]Model{
				"fast":     ModelClaudeHaiku,
				"default":  ModelClaudeSonnet,
				"advanced": ModelClaude37,
			},
		},
		Azure: AzureConfig{
			APIVersion: "2024-02-01",
			Location:   "uksouth",
			Deployments: map[string]string{
				string(ModelGPT35):     "gpt-35-turbo",
				string(ModelGPT4):      "gpt-4",
				string(ModelGPT4Turbo): "gpt-4-turbo",
				string(ModelGPT41):     "gpt-4.1",
				string(ModelGPT41Mini): "gpt-4.1-mini",
				string(ModelGPT41Nano): "gpt-4.1-nano",
				string(ModelGPT5):      "gpt-5",
				string(ModelO4Mini):    "o4-mini",
			},
		},
		Cache: CacheConfig{
			Enabled: true,
			TTL:     1 * time.Hour,
			RedisURL: "redis://localhost:6379",
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: 60,
			TokensPerMinute:   90000,
		},
		Budget: BudgetConfig{
			MonthlyLimit:    500.00,
			AlertThreshold:  0.8,
			TrackingEnabled: true,
		},
	}

	// Load from file if it exists
	if configPath != "" {
		if data, err := os.ReadFile(configPath); err == nil {
			if err := yaml.Unmarshal(data, config); err != nil {
				return nil, fmt.Errorf("failed to parse config file: %w", err)
			}
		}
	}

	// Override with environment variables
	applyEnvironmentOverrides(config)

	// Validate configuration
	if err := validateConfig(config); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return config, nil
}

// applyEnvironmentOverrides applies environment variable overrides
func applyEnvironmentOverrides(config *Config) {
	// AWS Bedrock configuration
	if v := os.Getenv("AWS_ACCESS_KEY_ID"); v != "" {
		config.Bedrock.AccessKeyID = v
	}
	if v := os.Getenv("AWS_SECRET_ACCESS_KEY"); v != "" {
		config.Bedrock.SecretAccessKey = v
	}
	if v := os.Getenv("AWS_REGION"); v != "" {
		config.Bedrock.Region = v
	}

	// Azure OpenAI configuration
	if v := os.Getenv("AZURE_OPENAI_KEY"); v != "" {
		config.Azure.APIKey = v
	}
	if v := os.Getenv("AZURE_OPENAI_ENDPOINT"); v != "" {
		config.Azure.Endpoint = v
	}

	// Provider selection
	if v := os.Getenv("QLF_LLM_PROVIDER"); v != "" {
		config.DefaultProvider = Provider(v)
	}

	// Cache configuration
	if v := os.Getenv("QLF_LLM_CACHE_ENABLED"); v == "false" {
		config.Cache.Enabled = false
	}
	if v := os.Getenv("REDIS_URL"); v != "" {
		config.Cache.RedisURL = v
	}

	// Budget configuration
	if v := os.Getenv("QLF_LLM_BUDGET_LIMIT"); v != "" {
		if limit, err := parseFloat(v); err == nil {
			config.Budget.MonthlyLimit = limit
		}
	}
}

// validateConfig validates the configuration
func validateConfig(config *Config) error {
	// Check that at least one provider is configured
	hasProvider := false

	if config.Bedrock.Region != "" {
		hasProvider = true
		if config.Bedrock.AccessKeyID == "" || config.Bedrock.SecretAccessKey == "" {
			// Allow for default AWS credential chain
		}
	}

	if config.Azure.Endpoint != "" && config.Azure.APIKey != "" {
		hasProvider = true
	}

	if !hasProvider {
		return fmt.Errorf("at least one LLM provider must be configured")
	}

	// Validate default provider
	switch config.DefaultProvider {
	case ProviderBedrock:
		if config.Bedrock.Region == "" {
			return fmt.Errorf("Bedrock region is required when using Bedrock as default provider")
		}
	case ProviderAzure:
		if config.Azure.Endpoint == "" || config.Azure.APIKey == "" {
			return fmt.Errorf("Azure OpenAI endpoint and API key are required when using Azure as default provider")
		}
	default:
		return fmt.Errorf("invalid default provider: %s", config.DefaultProvider)
	}

	// Validate budget limits
	if config.Budget.MonthlyLimit < 0 {
		return fmt.Errorf("monthly budget limit cannot be negative")
	}
	if config.Budget.AlertThreshold < 0 || config.Budget.AlertThreshold > 1 {
		return fmt.Errorf("alert threshold must be between 0 and 1")
	}

	return nil
}

// GetExampleConfig returns an example configuration
func GetExampleConfig() *Config {
	return &Config{
		DefaultProvider: ProviderBedrock,
		Bedrock: BedrockConfig{
			Region:          "eu-west-2",
			AccessKeyID:     "${AWS_ACCESS_KEY_ID}",
			SecretAccessKey: "${AWS_SECRET_ACCESS_KEY}",
			DefaultModel:    ModelClaudeSonnet,
			Models: map[string]Model{
				"fast":     ModelClaudeHaiku,
				"default":  ModelClaudeSonnet,
				"advanced": ModelClaude37,
			},
		},
		Azure: AzureConfig{
			Endpoint:   "${AZURE_OPENAI_ENDPOINT}",
			APIKey:     "${AZURE_OPENAI_KEY}",
			APIVersion: "2024-02-01",
			Location:   "uksouth",
			Deployments: map[string]string{
				string(ModelGPT35):     "gpt-35-turbo",
				string(ModelGPT4):      "gpt-4",
				string(ModelGPT4Turbo): "gpt-4-turbo",
				string(ModelGPT41):     "gpt-4.1",
				string(ModelGPT41Mini): "gpt-4.1-mini",
				string(ModelGPT41Nano): "gpt-4.1-nano",
				string(ModelGPT5):      "gpt-5",
				string(ModelO4Mini):    "o4-mini",
			},
		},
		Cache: CacheConfig{
			Enabled:  true,
			TTL:      1 * time.Hour,
			RedisURL: "redis://localhost:6379",
		},
		RateLimit: RateLimitConfig{
			RequestsPerMinute: 60,
			TokensPerMinute:   90000,
		},
		Budget: BudgetConfig{
			MonthlyLimit:    500.00,
			AlertThreshold:  0.8,
			TrackingEnabled: true,
		},
	}
}

// SaveConfig saves configuration to file
func SaveConfig(config *Config, path string) error {
	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	return os.WriteFile(path, data, 0644)
}

// parseFloat parses a string to float64
func parseFloat(s string) (float64, error) {
	var f float64
	if _, err := fmt.Sscanf(s, "%f", &f); err != nil {
		return 0, err
	}
	return f, nil
}