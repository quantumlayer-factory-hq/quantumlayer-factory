package agents

import (
	"testing"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
)

func TestFactory_CreateAgent(t *testing.T) {
	factory := NewFactory()

	// Register a test agent
	err := factory.RegisterAgent(AgentTypeBackend, func() Agent {
		return NewBackendAgent()
	})
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	// Create agent
	agent, err := factory.CreateAgent(AgentTypeBackend)
	if err != nil {
		t.Fatalf("Failed to create agent: %v", err)
	}

	if agent.GetType() != AgentTypeBackend {
		t.Errorf("Expected agent type %s, got %s", AgentTypeBackend, agent.GetType())
	}
}

func TestFactory_RegisterAgent(t *testing.T) {
	factory := NewFactory()

	// Register an agent
	err := factory.RegisterAgent(AgentTypeBackend, func() Agent {
		return NewBackendAgent()
	})
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	// Try to register the same agent type again (should fail)
	err = factory.RegisterAgent(AgentTypeBackend, func() Agent {
		return NewBackendAgent()
	})
	if err == nil {
		t.Error("Expected error when registering duplicate agent type")
	}
}

func TestFactory_GetSupportedTypes(t *testing.T) {
	factory := NewFactory()

	// Initially no types
	types := factory.GetSupportedTypes()
	if len(types) != 0 {
		t.Errorf("Expected 0 supported types, got %d", len(types))
	}

	// Register agents
	factory.RegisterAgent(AgentTypeBackend, func() Agent {
		return NewBackendAgent()
	})
	factory.RegisterAgent(AgentTypeFrontend, func() Agent {
		return NewFrontendAgent()
	})

	types = factory.GetSupportedTypes()
	if len(types) != 2 {
		t.Errorf("Expected 2 supported types, got %d", len(types))
	}

	// Check that backend and frontend are in the list
	hasBackend := false
	hasFrontend := false
	for _, agentType := range types {
		if agentType == AgentTypeBackend {
			hasBackend = true
		}
		if agentType == AgentTypeFrontend {
			hasFrontend = true
		}
	}

	if !hasBackend {
		t.Error("Backend agent type not found in supported types")
	}
	if !hasFrontend {
		t.Error("Frontend agent type not found in supported types")
	}
}

func TestFactory_GetBestAgent(t *testing.T) {
	factory := NewFactory()

	// Register test agents
	err := factory.RegisterAgent(AgentTypeBackend, func() Agent {
		return NewBackendAgent()
	})
	if err != nil {
		t.Fatalf("Failed to register backend agent: %v", err)
	}

	err = factory.RegisterAgent(AgentTypeFrontend, func() Agent {
		return NewFrontendAgent()
	})
	if err != nil {
		t.Fatalf("Failed to register frontend agent: %v", err)
	}

	// Test API spec - should get backend agent
	apiSpec := &ir.IRSpec{
		App: ir.AppSpec{Type: "api"},
		API: ir.APISpec{
			Type: "rest",
		},
	}

	agent, err := factory.GetBestAgent(apiSpec)
	if err != nil {
		t.Fatalf("Failed to get best agent: %v", err)
	}

	if agent.GetType() != AgentTypeBackend {
		t.Errorf("Expected backend agent for API spec, got %s", agent.GetType())
	}

	// Test SPA spec - should get frontend agent
	spaSpec := &ir.IRSpec{
		App: ir.AppSpec{Type: "spa"},
		UI: ir.UISpec{
			Pages: []ir.Page{
				{Name: "Home", Path: "/"},
			},
		},
	}

	agent, err = factory.GetBestAgent(spaSpec)
	if err != nil {
		t.Fatalf("Failed to get best agent: %v", err)
	}

	if agent.GetType() != AgentTypeFrontend {
		t.Errorf("Expected frontend agent for SPA spec, got %s", agent.GetType())
	}
}

func TestFactory_GetAgent(t *testing.T) {
	factory := NewFactory()

	// Register an agent
	err := factory.RegisterAgent(AgentTypeBackend, func() Agent {
		return NewBackendAgent()
	})
	if err != nil {
		t.Fatalf("Failed to register agent: %v", err)
	}

	// Get existing agent
	agent, err := factory.GetAgent(AgentTypeBackend)
	if err != nil {
		t.Fatalf("Failed to get agent: %v", err)
	}

	if agent.GetType() != AgentTypeBackend {
		t.Errorf("Expected backend agent, got %s", agent.GetType())
	}

	// Get same agent again (should return cached instance)
	agent2, err := factory.GetAgent(AgentTypeBackend)
	if err != nil {
		t.Fatalf("Failed to get cached agent: %v", err)
	}

	if agent != agent2 {
		t.Error("Expected same agent instance from cache")
	}

	// Try to get non-existent agent
	_, err = factory.GetAgent(AgentTypeTest)
	if err == nil {
		t.Error("Expected error when getting non-registered agent")
	}
}