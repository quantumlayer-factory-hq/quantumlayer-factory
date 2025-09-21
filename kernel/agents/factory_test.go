package agents

import (
	"context"
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

func TestFactory_GetSupportedTypes(t *testing.T) {
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

	types := factory.GetSupportedTypes()
	if len(types) != 2 {
		t.Errorf("Expected 2 supported types, got %d", len(types))
	}

	// Check that both types are present
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

func TestBackendAgent_CanHandle(t *testing.T) {
	agent := NewBackendAgent()

	tests := []struct {
		name     string
		spec     *ir.IRSpec
		expected bool
	}{
		{
			name: "API application",
			spec: &ir.IRSpec{
				App: ir.AppSpec{Type: "api"},
			},
			expected: true,
		},
		{
			name: "Web application",
			spec: &ir.IRSpec{
				App: ir.AppSpec{Type: "web"},
			},
			expected: true,
		},
		{
			name: "CLI application",
			spec: &ir.IRSpec{
				App: ir.AppSpec{Type: "cli"},
			},
			expected: false,
		},
		{
			name: "Application with API endpoints",
			spec: &ir.IRSpec{
				App: ir.AppSpec{Type: "mobile"},
				API: ir.APISpec{
					Endpoints: []ir.Endpoint{
						{Path: "/users", Method: "GET"},
					},
				},
			},
			expected: true,
		},
		{
			name: "Application with auth features",
			spec: &ir.IRSpec{
				App: ir.AppSpec{
					Type: "desktop",
					Features: []ir.Feature{
						{Type: "auth"},
					},
				},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.CanHandle(tt.spec)
			if result != tt.expected {
				t.Errorf("CanHandle() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestBackendAgent_Generate(t *testing.T) {
	agent := NewBackendAgent()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name:        "Test API",
			Description: "A test API application",
			Type:        "api",
			Stack: ir.TechStack{
				Backend: ir.BackendStack{
					Language:  "python",
					Framework: "fastapi",
				},
			},
		},
		API: ir.APISpec{
			Endpoints: []ir.Endpoint{
				{
					Path:    "/users",
					Method:  "GET",
					Summary: "Get users",
				},
			},
		},
		Data: ir.DataSpec{
			Entities: []ir.Entity{
				{
					Name: "User",
					Fields: []ir.Field{
						{Name: "id", Type: "uuid", Required: true},
						{Name: "email", Type: "string", Required: true},
					},
				},
			},
		},
	}

	req := &GenerationRequest{
		Spec: spec,
		Target: GenerationTarget{
			Type:     "backend",
			Language: "python",
		},
		Options: GenerationOptions{
			FormatCode:     true,
			ValidateOutput: true,
		},
	}

	result, err := agent.Generate(context.Background(), req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if !result.Success {
		t.Error("Generation should have succeeded")
	}

	if len(result.Files) == 0 {
		t.Error("Expected generated files, got none")
	}

	// Check that main.py was generated
	hasMain := false
	for _, file := range result.Files {
		if file.Path == "main.py" {
			hasMain = true
			if len(file.Content) == 0 {
				t.Error("main.py should have content")
			}
			break
		}
	}

	if !hasMain {
		t.Error("Expected main.py to be generated")
	}
}

func TestDefaultFactory(t *testing.T) {
	// Register default agents
	err := RegisterDefaultAgents()
	if err != nil {
		t.Fatalf("Failed to register default agents: %v", err)
	}

	factory := GetDefaultFactory()

	// Test that we can get supported types
	types := factory.GetSupportedTypes()
	if len(types) == 0 {
		t.Error("Expected some supported agent types")
	}

	// Test that we can create a backend agent
	agent, err := factory.GetAgent(AgentTypeBackend)
	if err != nil {
		t.Fatalf("Failed to get backend agent: %v", err)
	}

	if agent.GetType() != AgentTypeBackend {
		t.Errorf("Expected backend agent, got %s", agent.GetType())
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

	// Test with API spec (should prefer backend)
	apiSpec := &ir.IRSpec{
		App: ir.AppSpec{Type: "api"},
		API: ir.APISpec{
			Endpoints: []ir.Endpoint{
				{Path: "/test", Method: "GET"},
			},
		},
	}

	agent, err := factory.GetBestAgent(apiSpec)
	if err != nil {
		t.Fatalf("Failed to get best agent: %v", err)
	}

	if agent.GetType() != AgentTypeBackend {
		t.Errorf("Expected backend agent for API spec, got %s", agent.GetType())
	}
}