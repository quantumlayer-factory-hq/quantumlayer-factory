package agents

import (
	"fmt"
	"sync"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/llm"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/prompts"
)

// AgentFactory implements the Factory interface
type AgentFactory struct {
	mu             sync.RWMutex
	creators       map[AgentType]AgentCreator
	agents         map[AgentType]Agent
	llmClient      llm.Client
	promptComposer *prompts.PromptComposer
}

// NewFactory creates a new agent factory
func NewFactory() *AgentFactory {
	return &AgentFactory{
		creators: make(map[AgentType]AgentCreator),
		agents:   make(map[AgentType]Agent),
	}
}

// NewFactoryWithLLM creates a new agent factory with LLM capabilities
func NewFactoryWithLLM(llmClient llm.Client, promptComposer *prompts.PromptComposer) *AgentFactory {
	return &AgentFactory{
		creators:       make(map[AgentType]AgentCreator),
		agents:         make(map[AgentType]Agent),
		llmClient:      llmClient,
		promptComposer: promptComposer,
	}
}

// CreateAgent creates an agent of the specified type
func (f *AgentFactory) CreateAgent(agentType AgentType) (Agent, error) {
	f.mu.Lock()
	defer f.mu.Unlock()

	// If LLM is available, create LLM-enabled agents
	if f.llmClient != nil && f.promptComposer != nil {
		agent := f.createLLMAgent(agentType)
		if agent != nil {
			f.agents[agentType] = agent
			return agent, nil
		}
	}

	// Fallback to standard creator-based agents
	creator, exists := f.creators[agentType]
	if !exists {
		return nil, fmt.Errorf("agent type %s not registered", agentType)
	}

	agent := creator()
	f.agents[agentType] = agent
	return agent, nil
}

// createLLMAgent creates an LLM-enabled agent for the given type
func (f *AgentFactory) createLLMAgent(agentType AgentType) Agent {
	switch agentType {
	case AgentTypeBackend:
		return NewBackendAgentWithLLM(f.llmClient, f.promptComposer)
	case AgentTypeFrontend:
		return NewFrontendAgentWithLLM(f.llmClient, f.promptComposer)
	case AgentTypeDatabase:
		return NewDatabaseAgentWithLLM(f.llmClient, f.promptComposer)
	case AgentTypeAPI:
		return NewAPIAgentWithLLM(f.llmClient, f.promptComposer)
	case AgentTypeTest:
		return NewTestAgentWithLLM(f.llmClient, f.promptComposer)
	default:
		return nil
	}
}

// GetSupportedTypes returns all supported agent types
func (f *AgentFactory) GetSupportedTypes() []AgentType {
	f.mu.RLock()
	defer f.mu.RUnlock()

	types := make([]AgentType, 0, len(f.creators))
	for agentType := range f.creators {
		types = append(types, agentType)
	}
	return types
}

// GetAgent returns an existing agent instance or creates one
func (f *AgentFactory) GetAgent(agentType AgentType) (Agent, error) {
	f.mu.RLock()
	agent, exists := f.agents[agentType]
	f.mu.RUnlock()

	if exists {
		return agent, nil
	}

	return f.CreateAgent(agentType)
}

// GetBestAgent returns the best agent for handling the given spec
func (f *AgentFactory) GetBestAgent(spec *ir.IRSpec) (Agent, error) {
	// Get available agent types first
	f.mu.RLock()
	agentTypes := make([]AgentType, 0, len(f.creators))
	for agentType := range f.creators {
		agentTypes = append(agentTypes, agentType)
	}
	f.mu.RUnlock()

	// Score agents based on their ability to handle the spec
	bestAgent := Agent(nil)
	bestScore := 0.0

	for _, agentType := range agentTypes {
		agent, err := f.GetAgent(agentType)
		if err != nil {
			continue
		}

		if agent.CanHandle(spec) {
			score := f.calculateAgentScore(agent, spec)
			if score > bestScore {
				bestScore = score
				bestAgent = agent
			}
		}
	}

	if bestAgent == nil {
		return nil, fmt.Errorf("no suitable agent found for specification")
	}

	return bestAgent, nil
}

// RegisterAgent registers a new agent type
func (f *AgentFactory) RegisterAgent(agentType AgentType, creator AgentCreator) error {
	f.mu.Lock()
	defer f.mu.Unlock()

	if _, exists := f.creators[agentType]; exists {
		return fmt.Errorf("agent type %s already registered", agentType)
	}

	f.creators[agentType] = creator
	return nil
}

// calculateAgentScore determines how well an agent can handle a specification
func (f *AgentFactory) calculateAgentScore(agent Agent, spec *ir.IRSpec) float64 {
	score := 0.0

	// Base score for being able to handle the spec
	if agent.CanHandle(spec) {
		score += 0.5
	}

	// Bonus points based on agent type and spec requirements
	switch agent.GetType() {
	case AgentTypeBackend:
		if spec.App.Type == "api" || spec.App.Type == "web" {
			score += 0.3
		}
		if len(spec.API.Endpoints) > 0 {
			score += 0.2
		}

	case AgentTypeFrontend:
		if spec.App.Type == "web" || spec.App.Type == "spa" {
			score += 0.3
		}
		if len(spec.UI.Pages) > 0 {
			score += 0.2
		}

	case AgentTypeDatabase:
		if len(spec.Data.Entities) > 0 {
			score += 0.3
		}
		if len(spec.Data.Relationships) > 0 {
			score += 0.2
		}

	case AgentTypeAPI:
		if len(spec.API.Endpoints) > 0 {
			score += 0.4
		}
		if spec.API.Type == "rest" || spec.API.Type == "graphql" {
			score += 0.1
		}

	case AgentTypeDevOps:
		if len(spec.Ops.Environment) > 0 {
			score += 0.3
		}
		if spec.Ops.CI_CD.Provider != "" {
			score += 0.2
		}

	case AgentTypeTest:
		// Tests are always beneficial
		score += 0.2
		if len(spec.Acceptance) > 0 {
			score += 0.2
		}

	case AgentTypeDocumentation:
		// Documentation is always beneficial
		score += 0.1
		if len(spec.Questions) > 0 {
			score += 0.1
		}
	}

	return score
}

// DefaultFactory is a pre-configured factory with common agent types
var DefaultFactory = NewFactory()

// RegisterDefaultAgents registers the built-in agent types
func RegisterDefaultAgents() error {
	// Register backend agent
	err := DefaultFactory.RegisterAgent(AgentTypeBackend, func() Agent {
		return NewBackendAgent()
	})
	if err != nil {
		return fmt.Errorf("failed to register backend agent: %w", err)
	}

	// Register frontend agent
	err = DefaultFactory.RegisterAgent(AgentTypeFrontend, func() Agent {
		return NewFrontendAgent()
	})
	if err != nil {
		return fmt.Errorf("failed to register frontend agent: %w", err)
	}

	// Register database agent
	err = DefaultFactory.RegisterAgent(AgentTypeDatabase, func() Agent {
		return NewDatabaseAgent()
	})
	if err != nil {
		return fmt.Errorf("failed to register database agent: %w", err)
	}

	// Register API agent
	err = DefaultFactory.RegisterAgent(AgentTypeAPI, func() Agent {
		return NewAPIAgent()
	})
	if err != nil {
		return fmt.Errorf("failed to register API agent: %w", err)
	}

	// Register Test agent
	err = DefaultFactory.RegisterAgent(AgentTypeTest, func() Agent {
		return NewTestAgent()
	})
	if err != nil {
		return fmt.Errorf("failed to register test agent: %w", err)
	}

	return nil
}

// GetDefaultFactory returns the default factory instance
func GetDefaultFactory() Factory {
	return DefaultFactory
}