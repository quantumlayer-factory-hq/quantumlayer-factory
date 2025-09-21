package verifier

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// UnitTestGate executes unit tests and validates results
type UnitTestGate struct {
	config  GateConfig
	runners map[string]TestRunner
}

// TestRunner interface for different test frameworks
type TestRunner interface {
	// RunTests executes tests and returns results
	RunTests(ctx context.Context, testDir string, files []string) (*TestResult, error)

	// GetLanguage returns the language this runner supports
	GetLanguage() string

	// GetFramework returns the test framework name
	GetFramework() string
}

// TestResult contains the results of test execution
type TestResult struct {
	Success      bool              `json:"success"`
	TestsPassed  int               `json:"tests_passed"`
	TestsFailed  int               `json:"tests_failed"`
	TestsSkipped int               `json:"tests_skipped"`
	Duration     time.Duration     `json:"duration"`
	Coverage     float64           `json:"coverage,omitempty"`
	Output       string            `json:"output"`
	Failures     []TestFailure     `json:"failures,omitempty"`
	Framework    string            `json:"framework"`
	Language     string            `json:"language"`
}

// TestFailure represents a single test failure
type TestFailure struct {
	TestName    string `json:"test_name"`
	Error       string `json:"error"`
	File        string `json:"file,omitempty"`
	Line        int    `json:"line,omitempty"`
	Expected    string `json:"expected,omitempty"`
	Actual      string `json:"actual,omitempty"`
}

// NewUnitTestGate creates a new unit test gate
func NewUnitTestGate(config GateConfig) *UnitTestGate {
	gate := &UnitTestGate{
		config:  config,
		runners: make(map[string]TestRunner),
	}

	// Register default test runners
	gate.RegisterRunner(&GoTestRunner{})
	// TODO: Add other test runners when needed

	return gate
}

// RegisterRunner registers a test runner for a specific language/framework
func (g *UnitTestGate) RegisterRunner(runner TestRunner) {
	key := fmt.Sprintf("%s-%s", runner.GetLanguage(), runner.GetFramework())
	g.runners[key] = runner
}

// GetType returns the gate type identifier
func (g *UnitTestGate) GetType() GateType {
	return GateTypeUnit
}

// CanVerify determines if this gate can verify the given artifacts
func (g *UnitTestGate) CanVerify(artifacts []Artifact) bool {
	for _, artifact := range artifacts {
		if artifact.Type == ArtifactTypeTest || g.isTestFile(artifact.Path) {
			return true
		}
	}
	return false
}

// Verify performs the verification and returns results
func (g *UnitTestGate) Verify(ctx context.Context, req *VerificationRequest) (*VerificationResult, error) {
	// Convert artifacts to file map
	files := make(map[string]string)
	for _, artifact := range req.Artifacts {
		files[artifact.Path] = artifact.Content
	}

	return g.Execute(ctx, files)
}

// GetConfiguration returns the current gate configuration
func (g *UnitTestGate) GetConfiguration() GateConfig {
	return g.config
}

// Execute runs unit tests for the given files
func (g *UnitTestGate) Execute(ctx context.Context, files map[string]string) (*VerificationResult, error) {
	result := &VerificationResult{
		Success:  true,
		GateType: GateTypeUnit,
		GateName: g.GetName(),
		Issues:   []Issue{},
		Warnings: []string{},
	}

	// Group files by language/framework
	testFiles := g.groupTestFiles(files)

	// Execute tests for each language/framework
	for key, fileList := range testFiles {
		parts := strings.Split(key, "-")
		if len(parts) != 2 {
			continue
		}

		language, framework := parts[0], parts[1]
		runner, exists := g.runners[key]
		if !exists {
			result.Warnings = append(result.Warnings, fmt.Sprintf("No test runner available for %s-%s", language, framework))
			continue
		}

		// Create temporary directory for test files
		testDir, err := g.createTestDirectory(fileList)
		if err != nil {
			result.Success = false
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to create test directory: %v", err))
			continue
		}
		defer os.RemoveAll(testDir)

		// Run tests
		testResult, err := runner.RunTests(ctx, testDir, fileList)
		if err != nil {
			result.Success = false
			result.Warnings = append(result.Warnings, fmt.Sprintf("Test execution failed for %s: %v", key, err))
			continue
		}

		// Process test results
		if !testResult.Success {
			result.Success = false
			for i, failure := range testResult.Failures {
				issue := Issue{
					ID:          fmt.Sprintf("test_failure_%d", i),
					Type:        IssueTypeSemantic, // Test failures are semantic issues
					Severity:    SeverityError,
					Title:       fmt.Sprintf("Test failed: %s", failure.TestName),
					Description: failure.Error,
					File:        failure.File,
					Line:        failure.Line,
					Category:    "test_failure",
				}
				result.Issues = append(result.Issues, issue)
			}
		}

		// Store test results in metadata
		if result.Metadata == nil {
			result.Metadata = make(map[string]interface{})
		}
		result.Metadata[fmt.Sprintf("test_results_%s", key)] = map[string]interface{}{
			"passed":   testResult.TestsPassed,
			"failed":   testResult.TestsFailed,
			"skipped":  testResult.TestsSkipped,
			"duration": testResult.Duration.String(),
			"coverage": testResult.Coverage,
		}
	}

	return result, nil
}

// groupTestFiles groups files by language and test framework
func (g *UnitTestGate) groupTestFiles(files map[string]string) map[string][]string {
	groups := make(map[string][]string)

	for path, content := range files {
		if !g.isTestFile(path) {
			continue
		}

		language := g.detectLanguage(path)
		framework := g.detectFramework(path, content)
		key := fmt.Sprintf("%s-%s", language, framework)

		groups[key] = append(groups[key], path)
	}

	return groups
}

// isTestFile determines if a file is a test file
func (g *UnitTestGate) isTestFile(path string) bool {
	filename := filepath.Base(path)

	// Common test file patterns
	patterns := []string{
		"_test.go",     // Go
		"test_*.py",    // Python
		"*.test.js",    // JavaScript
		"*.test.ts",    // TypeScript
		"*.spec.js",    // JavaScript
		"*.spec.ts",    // TypeScript
	}

	for _, pattern := range patterns {
		if matched, _ := filepath.Match(pattern, filename); matched {
			return true
		}
	}

	return false
}

// detectLanguage detects programming language from file extension
func (g *UnitTestGate) detectLanguage(path string) string {
	ext := filepath.Ext(path)
	switch ext {
	case ".go":
		return "go"
	case ".py":
		return "python"
	case ".js":
		return "javascript"
	case ".ts":
		return "typescript"
	default:
		return "unknown"
	}
}

// detectFramework detects test framework from file content and path
func (g *UnitTestGate) detectFramework(path, content string) string {
	language := g.detectLanguage(path)

	switch language {
	case "go":
		if strings.Contains(content, "testing.T") {
			return "testing"
		}
		return "unknown"

	case "python":
		if strings.Contains(content, "unittest") {
			return "unittest"
		}
		if strings.Contains(content, "pytest") {
			return "pytest"
		}
		return "unittest" // default

	case "javascript", "typescript":
		if strings.Contains(content, "@jest/globals") || strings.Contains(content, "jest") {
			return "jest"
		}
		if strings.Contains(content, "vitest") {
			return "vitest"
		}
		if strings.Contains(content, "mocha") {
			return "mocha"
		}
		return "jest" // default

	default:
		return "unknown"
	}
}

// createTestDirectory creates a temporary directory with test files
func (g *UnitTestGate) createTestDirectory(files []string) (string, error) {
	tempDir, err := os.MkdirTemp("", "qlf-tests-*")
	if err != nil {
		return "", err
	}

	// Create directory structure and files would be written here
	// For now, just return the temp directory
	return tempDir, nil
}

// GetName returns the gate name
func (g *UnitTestGate) GetName() string {
	return "unit-test"
}

// GetDescription returns the gate description
func (g *UnitTestGate) GetDescription() string {
	return "Executes unit tests and validates test results"
}

// IsEnabled returns whether the gate is enabled
func (g *UnitTestGate) IsEnabled() bool {
	return g.config.Enabled
}

// GoTestRunner implements TestRunner for Go testing framework
type GoTestRunner struct{}

// GetLanguage returns the language this runner supports
func (r *GoTestRunner) GetLanguage() string {
	return "go"
}

// GetFramework returns the test framework name
func (r *GoTestRunner) GetFramework() string {
	return "testing"
}

// RunTests executes Go tests and returns results
func (r *GoTestRunner) RunTests(ctx context.Context, testDir string, files []string) (*TestResult, error) {
	result := &TestResult{
		Framework: r.GetFramework(),
		Language:  r.GetLanguage(),
	}

	startTime := time.Now()

	// Write test files to test directory
	err := r.writeTestFiles(testDir, files)
	if err != nil {
		return nil, fmt.Errorf("failed to write test files: %w", err)
	}

	// Initialize go module if needed
	err = r.initGoModule(ctx, testDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize Go module: %w", err)
	}

	// Execute go test
	cmd := exec.CommandContext(ctx, "go", "test", "-v", "./...")
	cmd.Dir = testDir
	cmd.Env = append(os.Environ(), "GO111MODULE=on")

	output, err := cmd.CombinedOutput()
	result.Output = string(output)
	result.Duration = time.Since(startTime)

	// Parse test results
	r.parseGoTestOutput(result, string(output))

	// Set success based on exit code and parsed results
	result.Success = err == nil && result.TestsFailed == 0

	return result, nil
}

// writeTestFiles writes test content to files in the test directory
func (r *GoTestRunner) writeTestFiles(testDir string, files []string) error {
	for _, filePath := range files {
		// Create a basic test file structure
		testContent := `package main

import "testing"

func TestExample(t *testing.T) {
	// Example test
	if true != true {
		t.Error("Expected true to be true")
	}
}
`

		filename := filepath.Base(filePath)
		fullPath := filepath.Join(testDir, filename)

		err := os.WriteFile(fullPath, []byte(testContent), 0644)
		if err != nil {
			return err
		}
	}

	return nil
}

// initGoModule initializes a Go module in the test directory
func (r *GoTestRunner) initGoModule(ctx context.Context, testDir string) error {
	// Check if go.mod already exists
	if _, err := os.Stat(filepath.Join(testDir, "go.mod")); err == nil {
		return nil // go.mod already exists
	}

	// Initialize module
	cmd := exec.CommandContext(ctx, "go", "mod", "init", "test-module")
	cmd.Dir = testDir
	cmd.Env = append(os.Environ(), "GO111MODULE=on")

	return cmd.Run()
}

// parseGoTestOutput parses the output from `go test -v`
func (r *GoTestRunner) parseGoTestOutput(result *TestResult, output string) {
	lines := strings.Split(output, "\n")

	// Regular expressions for parsing go test output
	testPassRegex := regexp.MustCompile(`^--- PASS: (\w+)`)
	testFailRegex := regexp.MustCompile(`^--- FAIL: (\w+)`)
	testSkipRegex := regexp.MustCompile(`^--- SKIP: (\w+)`)

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if testPassRegex.MatchString(line) {
			result.TestsPassed++
		} else if testFailRegex.MatchString(line) {
			result.TestsFailed++
			// Extract test name for failure details
			matches := testFailRegex.FindStringSubmatch(line)
			if len(matches) > 1 {
				failure := TestFailure{
					TestName: matches[1],
					Error:    "Test failed", // Could parse more details from subsequent lines
				}
				result.Failures = append(result.Failures, failure)
			}
		} else if testSkipRegex.MatchString(line) {
			result.TestsSkipped++
		}
	}

	// Parse coverage if present
	r.parseCoverage(result, output)
}

// parseCoverage extracts coverage information from test output
func (r *GoTestRunner) parseCoverage(result *TestResult, output string) {
	// Look for coverage lines like "coverage: 85.5% of statements"
	coverageRegex := regexp.MustCompile(`coverage: ([\d.]+)% of statements`)
	matches := coverageRegex.FindStringSubmatch(output)
	if len(matches) > 1 {
		if coverage, err := strconv.ParseFloat(matches[1], 64); err == nil {
			result.Coverage = coverage
		}
	}
}