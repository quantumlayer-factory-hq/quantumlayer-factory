package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/llm"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/prompts"
)

// TestAgent specializes in generating test code and test data
type TestAgent struct {
	*BaseAgent
	llmClient      llm.Client
	promptComposer *prompts.PromptComposer
}

// NewTestAgent creates a new test agent
func NewTestAgent() *TestAgent {
	capabilities := []string{
		"unit_tests",
		"integration_tests",
		"e2e_tests",
		"test_data_generation",
		"test_fixtures",
		"mock_generation",
		"test_coverage",
		"performance_tests",
		"security_tests",
		"api_tests",
	}

	return &TestAgent{
		BaseAgent: NewBaseAgent(AgentTypeTest, "1.0.0", capabilities),
	}
}

// NewTestAgentWithLLM creates a new test agent with LLM capabilities
func NewTestAgentWithLLM(llmClient llm.Client, promptComposer *prompts.PromptComposer) *TestAgent {
	capabilities := []string{
		"unit_tests",
		"integration_tests",
		"e2e_tests",
		"test_data_generation",
		"test_fixtures",
		"mock_generation",
		"test_coverage",
		"performance_tests",
		"security_tests",
		"api_tests",
	}

	return &TestAgent{
		BaseAgent:      NewBaseAgent(AgentTypeTest, "2.0.0", capabilities),
		llmClient:      llmClient,
		promptComposer: promptComposer,
	}
}

// CanHandle determines if this agent can handle the given specification
func (a *TestAgent) CanHandle(spec *ir.IRSpec) bool {
	// Test agent can handle any specification - tests are always beneficial
	return true
}

// Generate creates test code from the specification
func (a *TestAgent) Generate(ctx context.Context, req *GenerationRequest) (*GenerationResult, error) {
	startTime := time.Now()

	result := &GenerationResult{
		Success: true,
		Files:   []GeneratedFile{},
		Metadata: GenerationMetadata{
			AgentType:    a.GetType(),
			AgentVersion: a.version,
			GeneratedAt:  startTime,
		},
	}

	// Generate using LLM if available
	if a.llmClient != nil && a.promptComposer != nil {
		err := a.generateWithLLM(ctx, req, result)
		if err != nil {
			return nil, fmt.Errorf("failed to generate tests with LLM: %w", err)
		}
	} else {
		// Fallback for non-LLM mode
		err := a.generateWithTemplates(req, result)
		if err != nil {
			return nil, fmt.Errorf("failed to generate tests with templates: %w", err)
		}
	}

	// Update metadata
	result.Metadata.Duration = time.Since(startTime)
	result.Metadata.FilesCreated = len(result.Files)

	return result, nil
}

// generateWithLLM generates test code using LLM
func (a *TestAgent) generateWithLLM(ctx context.Context, req *GenerationRequest, result *GenerationResult) error {
	// Build prompt using the prompt composer
	composeReq := prompts.ComposeRequest{
		AgentType: "test",
		IRSpec:    req.Spec,
		Context:   req.Context,
	}

	promptResult, err := a.promptComposer.ComposePrompt(composeReq)
	if err != nil {
		return fmt.Errorf("failed to compose prompt: %w", err)
	}

	// Select model - test generation needs good reasoning
	model := llm.ModelClaudeSonnet

	// Call LLM to generate test code
	llmReq := &llm.GenerateRequest{
		Prompt:      promptResult.Prompt,
		Model:       model,
		MaxTokens:   8192,
		Temperature: 0.2, // Moderate temperature for diverse test cases
	}

	response, err := a.llmClient.Generate(ctx, llmReq)
	if err != nil {
		return fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse generated files
	files, err := a.parseGeneratedTests(response.Content, req.Spec)
	if err != nil {
		return fmt.Errorf("failed to parse generated tests: %w", err)
	}

	result.Files = append(result.Files, files...)

	// Add LLM usage metadata
	result.Metadata.LLMUsage = &LLMUsageMetadata{
		Provider:         string(response.Provider),
		Model:           string(response.Model),
		PromptTokens:    response.Usage.PromptTokens,
		CompletionTokens: response.Usage.CompletionTokens,
		TotalTokens:     response.Usage.TotalTokens,
		Cost:            response.Usage.Cost,
	}

	return nil
}

// generateWithTemplates generates test code using templates (fallback)
func (a *TestAgent) generateWithTemplates(req *GenerationRequest, result *GenerationResult) error {
	language := req.Spec.App.Stack.Backend.Language
	if language == "" {
		language = "go" // Default
	}

	// Generate unit tests based on language
	switch strings.ToLower(language) {
	case "go":
		a.generateGoTests(req, result)
	case "python":
		a.generatePythonTests(req, result)
	case "javascript", "typescript":
		a.generateJavaScriptTests(req, result)
	default:
		// Generic test file
		result.Warnings = append(result.Warnings, fmt.Sprintf("No specific test template for language: %s", language))
	}

	return nil
}

// parseGeneratedTests parses LLM output into test files
func (a *TestAgent) parseGeneratedTests(content string, spec *ir.IRSpec) ([]GeneratedFile, error) {
	files := []GeneratedFile{}

	language := spec.App.Stack.Backend.Language
	if language == "" {
		language = "go" // Default
	}

	// Determine test path based on language
	var testPath string
	switch strings.ToLower(language) {
	case "go":
		testPath = "tests/unit_test.go"
	case "python":
		testPath = "tests/test_unit.py"
	case "javascript":
		testPath = "tests/unit.test.js"
	case "typescript":
		testPath = "tests/unit.test.ts"
	default:
		testPath = "tests/unit_test.txt"
	}

	// Create main test file
	testFile := GeneratedFile{
		Path:     testPath,
		Type:     "test",
		Language: language,
		Template: "llm_generated",
		Content:  content,
	}
	files = append(files, testFile)

	return files, nil
}

// generateGoTests generates Go test templates
func (a *TestAgent) generateGoTests(req *GenerationRequest, result *GenerationResult) {
	testContent := fmt.Sprintf(`package main

import (
	"testing"
)

// Test%s tests the main functionality
func Test%s(t *testing.T) {
	// TODO: Implement test cases
	t.Skip("Test not yet implemented")
}

// TestHealthCheck tests the health check endpoint
func TestHealthCheck(t *testing.T) {
	// TODO: Implement health check test
	t.Skip("Health check test not yet implemented")
}
`, strings.Title(req.Spec.App.Name), strings.Title(req.Spec.App.Name))

	file := GeneratedFile{
		Path:     "tests/main_test.go",
		Type:     "test",
		Language: "go",
		Template: "go_test_template",
		Content:  testContent,
	}
	result.Files = append(result.Files, file)
}

// generatePythonTests generates Python test templates
func (a *TestAgent) generatePythonTests(req *GenerationRequest, result *GenerationResult) {
	testContent := fmt.Sprintf(`import pytest
import unittest

class Test%s(unittest.TestCase):
	"""Test cases for %s"""

	def setUp(self):
		"""Set up test fixtures"""
		pass

	def test_main_functionality(self):
		"""Test main functionality"""
		self.skipTest("Test not yet implemented")

	def test_health_check(self):
		"""Test health check endpoint"""
		self.skipTest("Health check test not yet implemented")

if __name__ == '__main__':
	unittest.main()
`, strings.Title(req.Spec.App.Name), req.Spec.App.Name)

	file := GeneratedFile{
		Path:     "tests/test_main.py",
		Type:     "test",
		Language: "python",
		Template: "python_test_template",
		Content:  testContent,
	}
	result.Files = append(result.Files, file)
}

// generateJavaScriptTests generates JavaScript/TypeScript test templates
func (a *TestAgent) generateJavaScriptTests(req *GenerationRequest, result *GenerationResult) {
	language := req.Spec.App.Stack.Backend.Language
	ext := ".js"
	if language == "typescript" {
		ext = ".ts"
	}

	testContent := fmt.Sprintf(`import { describe, it, expect, beforeEach } from '@jest/globals';

describe('%s', () => {
  beforeEach(() => {
    // Setup test fixtures
  });

  it('should test main functionality', () => {
    // TODO: Implement test cases
    expect(true).toBe(true);
  });

  it('should test health check endpoint', () => {
    // TODO: Implement health check test
    expect(true).toBe(true);
  });
});
`, req.Spec.App.Name)

	file := GeneratedFile{
		Path:     fmt.Sprintf("tests/main.test%s", ext),
		Type:     "test",
		Language: language,
		Template: "js_test_template",
		Content:  testContent,
	}
	result.Files = append(result.Files, file)
}

// Validate checks if the generated test code meets requirements
func (a *TestAgent) Validate(ctx context.Context, result *GenerationResult) (*ValidationResult, error) {
	validationResult := &ValidationResult{
		Valid:    true,
		Warnings: []string{},
		Metrics: ValidationMetrics{
			CodeQuality: 0.8, // Tests generally improve code quality
		},
	}

	// Check if test files were generated
	hasTests := false
	for _, file := range result.Files {
		if file.Type == "test" {
			hasTests = true
			break
		}
	}

	if !hasTests {
		validationResult.Valid = false
		validationResult.Errors = []ValidationError{
			{
				Message:  "No test files generated",
				Type:     "missing_tests",
				Severity: "error",
			},
		}
	}

	return validationResult, nil
}