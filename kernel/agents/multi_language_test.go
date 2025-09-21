package agents

import (
	"context"
	"strings"
	"testing"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestMultiLanguageBackendSupport tests that the backend agent supports multiple languages
func TestMultiLanguageBackendSupport(t *testing.T) {
	agent := NewBackendAgent()
	ctx := context.Background()

	languages := []struct {
		name      string
		language  string
		framework string
		expected  bool // whether generation should succeed
	}{
		{"Python FastAPI", "python", "fastapi", true},
		{"Python Django", "python", "django", false}, // not implemented
		{"Python Flask", "python", "flask", false},   // not implemented
		{"Go Gin", "go", "gin", false},               // not implemented
		{"Node.js Express", "nodejs", "express", false}, // not implemented
	}

	for _, lang := range languages {
		t.Run(lang.name, func(t *testing.T) {
			spec := &ir.IRSpec{
				App: ir.AppSpec{
					Name: "multi-lang-test",
					Type: "api",
					Stack: ir.TechStack{
						Backend: ir.BackendStack{
							Language:  lang.language,
							Framework: lang.framework,
						},
					},
				},
				API: ir.APISpec{
					Endpoints: []ir.Endpoint{
						{Path: "/health", Method: "GET", Description: "Health check"},
					},
				},
			}

			req := &GenerationRequest{Spec: spec}
			result, err := agent.Generate(ctx, req)

			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.True(t, result.Success)

			if lang.expected {
				// Should generate actual files
				assert.Greater(t, len(result.Files), 0, "Expected files to be generated for %s", lang.name)

				// Check file extensions match the language
				var correctExtension bool
				expectedExt := agent.getFileExtension(lang.language)
				for _, file := range result.Files {
					if strings.HasSuffix(file.Path, expectedExt) {
						correctExtension = true
						break
					}
				}
				assert.True(t, correctExtension, "Expected files with extension %s for %s", expectedExt, lang.name)
			} else {
				// Should have warnings about not being implemented
				assert.Greater(t, len(result.Warnings), 0, "Expected warnings for unimplemented %s", lang.name)
			}
		})
	}
}

// TestMultiLanguageTestSupport tests that the test agent supports multiple languages
func TestMultiLanguageTestSupport(t *testing.T) {
	agent := NewTestAgent()
	ctx := context.Background()

	languages := []struct {
		name      string
		language  string
		extension string
	}{
		{"Go", "go", ".go"},
		{"Python", "python", ".py"},
		{"JavaScript", "javascript", ".js"},
		{"TypeScript", "typescript", ".ts"},
	}

	for _, lang := range languages {
		t.Run(lang.name, func(t *testing.T) {
			spec := &ir.IRSpec{
				App: ir.AppSpec{
					Name: "test-app",
					Stack: ir.TechStack{
						Backend: ir.BackendStack{
							Language: lang.language,
						},
					},
				},
			}

			req := &GenerationRequest{
				Spec: spec,
				Target: GenerationTarget{
					Language: lang.language,
				},
			}

			result, err := agent.Generate(ctx, req)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.True(t, result.Success)
			assert.Greater(t, len(result.Files), 0)

			// Check that test files have correct extension
			var testFileFound bool
			for _, file := range result.Files {
				if file.Type == "test" && strings.HasSuffix(file.Path, lang.extension) {
					testFileFound = true
					break
				}
			}
			assert.True(t, testFileFound, "Expected test file with %s extension for %s", lang.extension, lang.name)
		})
	}
}

// TestMultiLanguageFrontendSupport tests frontend framework support
func TestMultiLanguageFrontendSupport(t *testing.T) {
	agent := NewFrontendAgent()

	frameworks := []struct {
		name      string
		framework string
		extension string
		language  string
	}{
		{"React", "react", ".jsx", "javascript"},
		{"Vue", "vue", ".vue", "javascript"},
		{"Angular", "angular", ".ts", "typescript"},
	}

	for _, fw := range frameworks {
		t.Run(fw.name, func(t *testing.T) {
			// Test file extension mapping
			ext := agent.getFileExtension(fw.framework)
			assert.Equal(t, fw.extension, ext, "Expected extension %s for %s", fw.extension, fw.name)

			// Test language mapping
			lang := agent.getLanguage(fw.framework)
			assert.Equal(t, fw.language, lang, "Expected language %s for %s", fw.language, fw.name)
		})
	}
}

// TestBackendFileExtensions tests that all supported languages have correct file extensions
func TestBackendFileExtensions(t *testing.T) {
	agent := NewBackendAgent()

	extensions := map[string]string{
		"python":     ".py",
		"go":         ".go",
		"javascript": ".js",
		"nodejs":     ".js",
		"typescript": ".ts",
		"java":       ".java",
		"csharp":     ".cs",
		"unknown":    ".txt",
	}

	for language, expectedExt := range extensions {
		t.Run(language, func(t *testing.T) {
			ext := agent.getFileExtension(language)
			assert.Equal(t, expectedExt, ext, "Expected extension %s for language %s", expectedExt, language)
		})
	}
}

// TestUniversalAgentCapabilities tests that agents declare universal capabilities
func TestUniversalAgentCapabilities(t *testing.T) {
	tests := []struct {
		name         string
		agent        Agent
		capabilities []string
	}{
		{
			name:  "Backend Agent",
			agent: NewBackendAgent(),
			capabilities: []string{
				"api_controllers",
				"service_layer",
				"data_models",
				"middleware",
				"authentication",
			},
		},
		{
			name:  "Frontend Agent",
			agent: NewFrontendAgent(),
			capabilities: []string{
				"react_components",
				"vue_components",
				"angular_components",
				"ui_layouts",
				"routing",
				"state_management",
			},
		},
		{
			name:  "Database Agent",
			agent: NewDatabaseAgent(),
			capabilities: []string{
				"schema_generation",
				"migrations",
				"seeds",
				"indexes",
				"relationships",
			},
		},
		{
			name:  "API Agent",
			agent: NewAPIAgent(),
			capabilities: []string{
				"openapi_specs",
				"rest_documentation",
				"graphql_schemas",
			},
		},
		{
			name:  "Test Agent",
			agent: NewTestAgent(),
			capabilities: []string{
				"unit_tests",
				"integration_tests",
				"test_fixtures",
				"mock_generation",
				"test_coverage",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			agentCaps := tt.agent.GetCapabilities()
			assert.Greater(t, len(agentCaps), 0, "Agent should have capabilities")

			// Check that agent has expected universal capabilities
			for _, expectedCap := range tt.capabilities {
				assert.Contains(t, agentCaps, expectedCap, "Agent should have capability: %s", expectedCap)
			}
		})
	}
}

// TestCrossLanguageCompatibility tests that different agents can work together
func TestCrossLanguageCompatibility(t *testing.T) {
	factory := NewFactory()

	// Register agents manually
	factory.RegisterAgent(AgentTypeBackend, func() Agent { return NewBackendAgent() })
	factory.RegisterAgent(AgentTypeFrontend, func() Agent { return NewFrontendAgent() })
	factory.RegisterAgent(AgentTypeDatabase, func() Agent { return NewDatabaseAgent() })
	factory.RegisterAgent(AgentTypeAPI, func() Agent { return NewAPIAgent() })
	factory.RegisterAgent(AgentTypeTest, func() Agent { return NewTestAgent() })

	ctx := context.Background()

	// Test Python backend + React frontend + PostgreSQL database
	spec := &ir.IRSpec{
		App: ir.AppSpec{
			Name: "full-stack-app",
			Type: "web",
			Stack: ir.TechStack{
				Backend: ir.BackendStack{
					Language:  "python",
					Framework: "fastapi",
				},
				Frontend: ir.FrontendStack{
					Framework: "react",
					Language:  "typescript",
				},
				Database: ir.DatabaseStack{
					Type: "postgresql",
				},
			},
		},
		API: ir.APISpec{
			Endpoints: []ir.Endpoint{
				{Path: "/api/users", Method: "GET"},
			},
		},
		UI: ir.UISpec{
			Pages: []ir.Page{
				{Name: "Home", Path: "/"},
			},
		},
		Data: ir.DataSpec{
			Entities: []ir.Entity{
				{Name: "User", Fields: []ir.Field{{Name: "ID", Type: "int"}}},
			},
		},
	}

	// Test that multiple agents can handle this spec
	agents := []struct {
		name      string
		agentType AgentType
		canHandle bool
	}{
		{"Backend", AgentTypeBackend, true},
		{"Frontend", AgentTypeFrontend, true},
		{"Database", AgentTypeDatabase, true},
		{"API", AgentTypeAPI, true},
		{"Test", AgentTypeTest, true},
	}

	for _, agentTest := range agents {
		t.Run(agentTest.name, func(t *testing.T) {
			agent, err := factory.CreateAgent(agentTest.agentType)
			require.NoError(t, err)
			assert.NotNil(t, agent)

			canHandle := agent.CanHandle(spec)
			assert.Equal(t, agentTest.canHandle, canHandle, "Agent %s should handle the spec: %v", agentTest.name, agentTest.canHandle)

			if canHandle {
				req := &GenerationRequest{Spec: spec}
				result, err := agent.Generate(ctx, req)
				require.NoError(t, err)
				assert.True(t, result.Success)
			}
		})
	}
}

// TestLanguageSpecificValidation tests that validation works across languages
func TestLanguageSpecificValidation(t *testing.T) {
	agent := NewBackendAgent()

	testCases := []struct {
		name     string
		file     GeneratedFile
		hasError bool
	}{
		{
			name: "Valid Python",
			file: GeneratedFile{
				Language: "python",
				Content:  "def hello_world():\n    return 'Hello, World!'",
			},
			hasError: false,
		},
		{
			name: "Invalid Python",
			file: GeneratedFile{
				Language: "python",
				Content:  "print('hello')", // no def or class, should generate warning
			},
			hasError: false, // Python validation only warns, doesn't error
		},
		{
			name: "Valid Go",
			file: GeneratedFile{
				Language: "go",
				Content:  "package main\n\nfunc main() {\n    fmt.Println(\"Hello, World!\")\n}",
			},
			hasError: false,
		},
		{
			name: "Invalid Go",
			file: GeneratedFile{
				Language: "go",
				Content:  "func main() {\n    fmt.Println(\"Hello, World!\")\n}", // missing package
			},
			hasError: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			validation := &ValidationResult{}
			err := agent.validateFile(tc.file, validation)

			if tc.hasError {
				assert.Error(t, err, "Expected validation error for %s", tc.name)
			} else {
				assert.NoError(t, err, "Expected no validation error for %s", tc.name)
			}
		})
	}
}