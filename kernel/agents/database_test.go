package agents

import (
	"context"
	"testing"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewDatabaseAgent(t *testing.T) {
	agent := NewDatabaseAgent()
	assert.NotNil(t, agent)
	assert.Equal(t, AgentTypeDatabase, agent.GetType())
	assert.Contains(t, agent.GetCapabilities(), "schema_generation")
	assert.Contains(t, agent.GetCapabilities(), "migrations")
}

func TestDatabaseAgent_CanHandle(t *testing.T) {
	agent := NewDatabaseAgent()

	tests := []struct {
		name     string
		spec     *ir.IRSpec
		expected bool
	}{
		{
			name: "Spec with entities",
			spec: &ir.IRSpec{
				Data: ir.DataSpec{
					Entities: []ir.Entity{
						{Name: "User"},
					},
				},
			},
			expected: true,
		},
		{
			name: "Spec with relationships",
			spec: &ir.IRSpec{
				Data: ir.DataSpec{
					Relationships: []ir.Relationship{
						{From: "User", To: "Order"},
					},
				},
			},
			expected: true,
		},
		{
			name: "Empty spec",
			spec: &ir.IRSpec{},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := agent.CanHandle(tt.spec)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDatabaseAgent_Generate(t *testing.T) {
	agent := NewDatabaseAgent()
	ctx := context.Background()

	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "test-db",
			Stack: ir.TechStack{
				Database: ir.DatabaseStack{
					Type: "postgresql",
				},
			},
		},
		Data: ir.DataSpec{
			Entities: []ir.Entity{
				{
					Name: "User",
					Fields: []ir.Field{
						{Name: "ID", Type: "int", Required: true},
						{Name: "Email", Type: "string", Required: true},
					},
				},
			},
		},
	}

	req := &GenerationRequest{
		Spec: spec,
		Target: GenerationTarget{
			Type: "schema",
		},
	}

	result, err := agent.Generate(ctx, req)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)

	// In template mode (no LLM), agent returns warning but no files
	if len(result.Files) == 0 {
		assert.Contains(t, result.Warnings, "Database agent running in template mode - limited functionality")
	} else {
		// If files are generated, check they're database-related
		var dbFileFound bool
		for _, file := range result.Files {
			if file.Type == "source" && file.Language == "sql" {
				dbFileFound = true
				break
			}
		}
		assert.True(t, dbFileFound, "Database schema/migration should be generated")
	}
}

func TestDatabaseAgent_Validate(t *testing.T) {
	agent := NewDatabaseAgent()
	ctx := context.Background()

	result := &GenerationResult{
		Success: true,
		Files: []GeneratedFile{
			{
				Path:    "schema.sql",
				Type:    "schema",
				Content: "CREATE TABLE users (id INT PRIMARY KEY, email VARCHAR(255));",
			},
		},
	}

	validation, err := agent.Validate(ctx, result)
	require.NoError(t, err)
	assert.NotNil(t, validation)
	// Database validation is not yet implemented, so expect false and warnings
	assert.False(t, validation.Valid)
	assert.Contains(t, validation.Warnings, "Database validation not yet implemented")
}