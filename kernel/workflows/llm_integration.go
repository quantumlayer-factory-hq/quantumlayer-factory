package workflows

import (
	"fmt"
	"os"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/agents"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/llm"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/prompts"
)

// createLLMEnabledFactory creates an agent factory with LLM support based on configuration
func createLLMEnabledFactory(provider, model string) (*agents.AgentFactory, error) {
	// Note: This function runs in activity context, not workflow context
	// Logging provider and model for debugging
	fmt.Printf("createLLMEnabledFactory called with provider=%s model=%s\n", provider, model)

	// If no provider specified, return regular factory with default agents
	if provider == "" || provider == "template" {
		fmt.Println("Using template-based factory - no provider specified")
		return createTemplateFactory(), nil
	}

	// Create LLM configuration from environment and parameters
	config := &llm.Config{
		DefaultProvider: llm.Provider(provider),
		Bedrock: llm.BedrockConfig{
			Region:       "eu-west-2", // UK London region
			AccessKeyID:  os.Getenv("AWS_ACCESS_KEY_ID"),
			SecretAccessKey: os.Getenv("AWS_SECRET_ACCESS_KEY"),
			DefaultModel: llm.ModelClaudeSonnet,
		},
		Azure: llm.AzureConfig{
			Endpoint:    os.Getenv("AZURE_OPENAI_ENDPOINT"),
			APIKey:      os.Getenv("AZURE_OPENAI_API_KEY"),
			APIVersion:  "2024-02-01",
			Location:    "uksouth",
		},
	}

	// Create LLM client factory
	clientFactory := llm.NewClientFactory(config)

	// Get the appropriate client for the provider
	var llmClient llm.Client
	var err error

	fmt.Printf("Creating LLM client for provider: %s\n", provider)

	switch provider {
	case "aws", "bedrock", "aws-bedrock":
		llmClient, err = clientFactory.CreateClient(llm.ProviderBedrock)
	case "azure", "azure-openai":
		llmClient, err = clientFactory.CreateClient(llm.ProviderAzure)
	default:
		fmt.Printf("Unsupported provider %s, falling back to template factory\n", provider)
		return createTemplateFactory(), nil
	}

	if err != nil {
		fmt.Printf("LLM client creation failed: %v, falling back to template factory\n", err)
		return createTemplateFactory(), nil
	}

	fmt.Printf("Successfully created LLM client for provider: %s\n", provider)

	// Create prompt composer with default config
	promptComposer := prompts.NewPromptComposer(prompts.ComposerConfig{
		TemplatePaths: []string{"kernel/prompts/templates"},
		EnableCaching: true,
		DefaultLanguage: "english",
		MaxPromptLength: 32000,
	})

	// Create factory with LLM support
	factory := agents.NewFactoryWithLLM(llmClient, promptComposer)
	fmt.Println("Created LLM-enabled factory successfully")

	return factory, nil
}

// createTemplateFactory creates a template-based agent factory
func createTemplateFactory() *agents.AgentFactory {
	factory := agents.NewFactory()
	// Register default agents
	factory.RegisterAgent(agents.AgentTypeBackend, func() agents.Agent {
		return agents.NewBackendAgent()
	})
	factory.RegisterAgent(agents.AgentTypeFrontend, func() agents.Agent {
		return agents.NewFrontendAgent()
	})
	factory.RegisterAgent(agents.AgentTypeDatabase, func() agents.Agent {
		return agents.NewDatabaseAgent()
	})
	factory.RegisterAgent(agents.AgentTypeAPI, func() agents.Agent {
		return agents.NewAPIAgent()
	})
	factory.RegisterAgent(agents.AgentTypeTest, func() agents.Agent {
		return agents.NewTestAgent()
	})
	return factory
}