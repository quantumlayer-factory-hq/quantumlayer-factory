package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestBaseAgent_ConfigMethods(t *testing.T) {
	agent := NewBaseAgent(AgentTypeBackend, "1.0.0", []string{"test"})

	// Test SetConfig and GetConfig
	agent.SetConfig("key1", "value1")
	value, exists := agent.GetConfig("key1")
	assert.True(t, exists)
	assert.Equal(t, "value1", value)

	// Test non-existent key
	_, exists = agent.GetConfig("nonexistent")
	assert.False(t, exists)

	// Test overwriting config
	agent.SetConfig("key1", "new_value")
	value, exists = agent.GetConfig("key1")
	assert.True(t, exists)
	assert.Equal(t, "new_value", value)
}