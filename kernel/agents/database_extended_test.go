package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDatabaseAgent_HasCapability(t *testing.T) {
	agent := NewDatabaseAgent()

	tests := []struct {
		capability string
		expected   bool
	}{
		{"schema_generation", true},
		{"migrations", true},
		{"seeds", true},
		{"unknown_capability", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.capability, func(t *testing.T) {
			result := agent.hasCapability(tt.capability)
			assert.Equal(t, tt.expected, result)
		})
	}
}