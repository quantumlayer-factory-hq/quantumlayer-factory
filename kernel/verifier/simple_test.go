package verifier

import (
	"context"
	"fmt"
	"testing"
	"time"
)

// MockRunner is a simple mock runner for testing
type MockRunner struct {
	name       string
	version    string
	canRun     bool
	returnIssues []Issue
	returnError  error
}

func NewMockRunner(name string) *MockRunner {
	return &MockRunner{
		name:    name,
		version: "1.0.0",
		canRun:  true,
	}
}

func (r *MockRunner) GetName() string {
	return r.name
}

func (r *MockRunner) GetVersion() string {
	return r.version
}

func (r *MockRunner) CanRun(artifacts []Artifact) bool {
	return r.canRun
}

func (r *MockRunner) Run(ctx context.Context, artifacts []Artifact, config map[string]interface{}) (*RunnerResult, error) {
	if r.returnError != nil {
		return nil, r.returnError
	}

	return &RunnerResult{
		Success:   true,
		ExitCode:  0,
		Issues:    r.returnIssues,
		Duration:  100 * time.Millisecond,
		Artifacts: artifacts,
	}, nil
}

func (r *MockRunner) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"enabled": true,
	}
}

func (r *MockRunner) SetCanRun(canRun bool) {
	r.canRun = canRun
}

func (r *MockRunner) SetReturnIssues(issues []Issue) {
	r.returnIssues = issues
}

func (r *MockRunner) SetReturnError(err error) {
	r.returnError = err
}

func TestStaticGate_Basic(t *testing.T) {
	gate := NewStaticGate(GateConfig{
		Enabled: true,
		Timeout: 30 * time.Second,
	})

	if gate.GetType() != GateTypeStatic {
		t.Errorf("Expected gate type %s, got %s", GateTypeStatic, gate.GetType())
	}

	if gate.GetName() == "" {
		t.Error("Gate name should not be empty")
	}
}

func TestStaticGate_CanVerify(t *testing.T) {
	gate := NewStaticGate(GateConfig{Enabled: true})

	tests := []struct {
		name      string
		artifacts []Artifact
		expected  bool
	}{
		{
			name: "Source files with mock runner",
			artifacts: []Artifact{
				{
					Path:     "main.go",
					Type:     ArtifactTypeSource,
					Language: "go",
					Content:  "package main\n\nfunc main() {}\n",
				},
			},
			expected: true,
		},
		{
			name: "No source files",
			artifacts: []Artifact{
				{
					Path: "README.md",
					Type: ArtifactTypeDocumentation,
				},
			},
			expected: false,
		},
		{
			name:      "Empty artifacts",
			artifacts: []Artifact{},
			expected:  false,
		},
	}

	// Add a mock runner
	mockRunner := NewMockRunner("mock-runner")
	gate.AddRunner(mockRunner)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := gate.CanVerify(tt.artifacts)
			if result != tt.expected {
				t.Errorf("CanVerify() = %v, expected %v", result, tt.expected)
			}
		})
	}
}

func TestStaticGate_Verify_Success(t *testing.T) {
	gate := NewStaticGate(GateConfig{
		Enabled: true,
		Timeout: 30 * time.Second,
	})

	mockRunner := NewMockRunner("test-runner")
	gate.AddRunner(mockRunner)

	artifacts := []Artifact{
		{
			Path:     "main.go",
			Type:     ArtifactTypeSource,
			Language: "go",
			Content:  "package main\n\nfunc main() {}\n",
			Hash:     "test-hash",
		},
	}

	req := &VerificationRequest{
		RequestID: "test-success",
		Artifacts: artifacts,
		Config: GateConfig{
			Enabled: true,
		},
		Timeout: 30 * time.Second,
	}

	result, err := gate.Verify(context.Background(), req)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if !result.Success {
		t.Error("Verification should succeed")
	}

	if result.GateType != GateTypeStatic {
		t.Errorf("Expected gate type %s, got %s", GateTypeStatic, result.GateType)
	}

	if result.Metrics.FilesScanned == 0 {
		t.Error("Should have scanned at least one file")
	}
}

func TestStaticGate_Verify_WithIssues(t *testing.T) {
	gate := NewStaticGate(GateConfig{
		Enabled: true,
		Timeout: 30 * time.Second,
		Thresholds: map[string]float64{
			"max_errors": 0, // Fail on any error
		},
	})

	mockRunner := NewMockRunner("test-runner")
	mockRunner.SetReturnIssues([]Issue{
		{
			Type:        IssueTypeSyntax,
			Severity:    SeverityError,
			Title:       "Test error",
			Description: "This is a test error",
		},
	})
	gate.AddRunner(mockRunner)

	artifacts := []Artifact{
		{
			Path:     "main.go",
			Type:     ArtifactTypeSource,
			Language: "go",
			Content:  "package main\n\nfunc main() {}\n",
			Hash:     "test-hash",
		},
	}

	req := &VerificationRequest{
		RequestID: "test-with-issues",
		Artifacts: artifacts,
		Config: GateConfig{
			Enabled: true,
		},
		Timeout: 30 * time.Second,
	}

	result, err := gate.Verify(context.Background(), req)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if result.Success {
		t.Error("Verification should fail due to error threshold")
	}

	if len(result.Issues) == 0 {
		t.Error("Should have found issues")
	}

	if len(result.RepairHints) == 0 {
		t.Error("Should have generated repair hints")
	}
}

func TestStaticGate_AddRemoveRunner(t *testing.T) {
	gate := NewStaticGate(GateConfig{Enabled: true})

	mockRunner := NewMockRunner("test-runner")

	// Test adding runner
	err := gate.AddRunner(mockRunner)
	if err != nil {
		t.Fatalf("Failed to add runner: %v", err)
	}

	runners := gate.GetRunners()
	if len(runners) != 1 {
		t.Errorf("Expected 1 runner, got %d", len(runners))
	}

	// Test removing runner
	gate.RemoveRunner(mockRunner.GetName())
	runners = gate.GetRunners()
	if len(runners) != 0 {
		t.Errorf("Expected 0 runners after removal, got %d", len(runners))
	}
}

func TestStaticGate_AddRunner_Nil(t *testing.T) {
	gate := NewStaticGate(GateConfig{Enabled: true})

	err := gate.AddRunner(nil)
	if err == nil {
		t.Error("Expected error when adding nil runner")
	}
}

func TestStaticGate_Disabled(t *testing.T) {
	gate := NewStaticGate(GateConfig{Enabled: false})
	mockRunner := NewMockRunner("test-runner")
	gate.AddRunner(mockRunner)

	artifacts := []Artifact{
		{
			Path:     "main.go",
			Type:     ArtifactTypeSource,
			Language: "go",
			Content:  "package main\n\nfunc main() {}\n",
		},
	}

	if gate.CanVerify(artifacts) {
		t.Error("Disabled gate should not be able to verify")
	}
}

func TestStaticGate_CountLines(t *testing.T) {
	tests := []struct {
		content  string
		expected int
	}{
		{"", 0},
		{"single line", 1},
		{"line 1\nline 2", 2},
		{"line 1\nline 2\nline 3\n", 4},
		{"\n\n\n", 4},
	}

	for i, tt := range tests {
		t.Run(fmt.Sprintf("test-%d", i), func(t *testing.T) {
			result := countLines(tt.content)
			if result != tt.expected {
				t.Errorf("Expected %d lines, got %d for content %q",
					tt.expected, result, tt.content)
			}
		})
	}
}