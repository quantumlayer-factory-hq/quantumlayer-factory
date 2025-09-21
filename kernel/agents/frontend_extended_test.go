package agents

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFrontendAgent_GetFileExtension(t *testing.T) {
	agent := NewFrontendAgent()

	tests := []struct {
		framework string
		expected  string
	}{
		{"react", ".jsx"},
		{"vue", ".vue"},
		{"angular", ".ts"},
		{"unknown", ".js"},
		{"", ".js"},
	}

	for _, tt := range tests {
		t.Run(tt.framework, func(t *testing.T) {
			result := agent.getFileExtension(tt.framework)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestFrontendAgent_GetLanguage(t *testing.T) {
	agent := NewFrontendAgent()

	tests := []struct {
		framework string
		expected  string
	}{
		{"angular", "typescript"},
		{"react", "javascript"},
		{"vue", "javascript"},
		{"unknown", "javascript"},
		{"", "javascript"},
	}

	for _, tt := range tests {
		t.Run(tt.framework, func(t *testing.T) {
			result := agent.getLanguage(tt.framework)
			assert.Equal(t, tt.expected, result)
		})
	}
}