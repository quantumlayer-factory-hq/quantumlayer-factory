package agents

import (
	"testing"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/stretchr/testify/assert"
)

func TestFactory_RegisterDefaultAgents(t *testing.T) {
	err := RegisterDefaultAgents()
	assert.NoError(t, err)

	// Check that default factory has agent types registered
	factory := GetDefaultFactory()
	supportedTypes := factory.GetSupportedTypes()
	assert.Contains(t, supportedTypes, AgentTypeBackend)
	assert.Contains(t, supportedTypes, AgentTypeFrontend)
	assert.Contains(t, supportedTypes, AgentTypeDatabase)
	assert.Contains(t, supportedTypes, AgentTypeAPI)
	assert.Contains(t, supportedTypes, AgentTypeTest)
}

func TestFactory_GetDefaultFactory(t *testing.T) {
	factory1 := GetDefaultFactory()
	factory2 := GetDefaultFactory()

	// Should return the same instance (singleton)
	assert.Equal(t, factory1, factory2)

	// Should have default agents registered
	supportedTypes := factory1.GetSupportedTypes()
	assert.Greater(t, len(supportedTypes), 0)
}

func TestFactory_CalculateAgentScore(t *testing.T) {
	factory := NewFactory()
	agent := NewBackendAgent()

	tests := []struct {
		name     string
		spec     *ir.IRSpec
		expected float64
	}{
		{
			name: "Perfect match - API with backend",
			spec: &ir.IRSpec{
				App: ir.AppSpec{
					Type: "api",
					Stack: ir.TechStack{
						Backend: ir.BackendStack{
							Language: "python",
						},
					},
				},
			},
			expected: 0.8, // 0.5 + 0.3 (can handle + api type)
		},
		{
			name: "No match - frontend only",
			spec: &ir.IRSpec{
				App: ir.AppSpec{
					Type: "spa",
					Stack: ir.TechStack{
						Frontend: ir.FrontendStack{
							Framework: "react",
						},
					},
				},
			},
			expected: 0.0, // Backend agent can't handle SPA
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := factory.calculateAgentScore(agent, tt.spec)
			assert.Equal(t, tt.expected, score)
		})
	}
}

func TestFactory_CreateLLMAgent(t *testing.T) {
	factory := &AgentFactory{}

	// Test creating LLM agent without LLM client returns basic agent
	agent := factory.createLLMAgent(AgentTypeBackend)
	assert.NotNil(t, agent)
	assert.Equal(t, AgentTypeBackend, agent.GetType())
}