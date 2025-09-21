package verifier

import (
	"context"
	"testing"
	"time"
)

// TestVerificationPipelineIntegration tests the complete verification pipeline
func TestVerificationPipelineIntegration(t *testing.T) {
	// Create pipeline configuration
	config := PipelineConfig{
		Name:                "integration_test_pipeline",
		Parallel:            false, // Sequential for predictable testing
		StopOnFirstFailure:  false,
		EnableRepairLoop:    false, // Disable LLM for basic test
		MaxRepairIterations: 3,
		Timeout:             5 * time.Minute,
		Environment:         "test",
	}

	// Create pipeline
	pipeline := NewVerificationPipeline(config)

	// Add verification gates
	unitGateConfig := GateConfig{
		Enabled: true,
		Timeout: 2 * time.Minute,
		Parallel: false,
	}
	unitGate := NewUnitTestGate(unitGateConfig)
	err := pipeline.AddGate(unitGate)
	if err != nil {
		t.Fatalf("Failed to add unit test gate: %v", err)
	}

	contractGateConfig := GateConfig{
		Enabled: true,
		Timeout: 2 * time.Minute,
		Parallel: false,
	}
	contractGate := NewContractTestGate(contractGateConfig)
	err = pipeline.AddGate(contractGate)
	if err != nil {
		t.Fatalf("Failed to add contract test gate: %v", err)
	}

	// Create test artifacts
	artifacts := []Artifact{
		{
			Path:     "main_test.go",
			Type:     ArtifactTypeTest,
			Language: "go",
			Content: `package main

import "testing"

func TestExample(t *testing.T) {
	if 2+2 != 4 {
		t.Error("Math is broken!")
	}
}
`,
			Size: 120,
			Hash: "test_hash_1",
		},
		{
			Path:     "api.yaml",
			Type:     ArtifactTypeSchema,
			Language: "yaml",
			Content: `openapi: 3.0.0
info:
  title: Test API
  version: 1.0.0
servers:
  - url: http://localhost:8080
paths:
  /health:
    get:
      responses:
        200:
          description: OK
`,
			Size: 200,
			Hash: "test_hash_2",
		},
		{
			Path:     "main.go",
			Type:     ArtifactTypeSource,
			Language: "go",
			Content: `package main

import (
	"fmt"
	"net/http"
)

func main() {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, "OK")
	})

	fmt.Println("Server starting on :8080")
	http.ListenAndServe(":8080", nil)
}
`,
			Size: 250,
			Hash: "test_hash_3",
		},
	}

	// Execute pipeline
	ctx := context.Background()
	result, err := pipeline.Execute(ctx, artifacts)
	if err != nil {
		t.Fatalf("Pipeline execution failed: %v", err)
	}

	// Verify results
	if result == nil {
		t.Fatal("Pipeline result is nil")
	}

	if result.RequestID == "" {
		t.Error("Pipeline result should have a request ID")
	}

	if len(result.GateResults) == 0 {
		t.Error("Pipeline should have gate results")
	}

	// Check that gates were executed
	expectedGates := 2 // Unit test gate + Contract test gate
	if len(result.GateResults) < expectedGates {
		t.Errorf("Expected at least %d gate results, got %d", expectedGates, len(result.GateResults))
	}

	// Verify gate types were executed
	gateTypes := make(map[GateType]bool)
	for _, gateResult := range result.GateResults {
		gateTypes[gateResult.GateType] = true
	}

	if !gateTypes[GateTypeUnit] {
		t.Error("Unit test gate should have been executed")
	}

	if !gateTypes[GateTypeContract] {
		t.Error("Contract test gate should have been executed")
	}

	// Check pipeline metrics
	metrics := pipeline.GetMetrics()
	if metrics.GatesExecuted == 0 {
		t.Error("Pipeline metrics should show gates were executed")
	}

	if metrics.TotalDuration == 0 {
		t.Error("Pipeline metrics should show non-zero duration")
	}

	// Verify artifacts were processed
	if len(result.Artifacts) != len(artifacts) {
		t.Errorf("Expected %d artifacts in result, got %d", len(artifacts), len(result.Artifacts))
	}

	t.Logf("Pipeline execution completed successfully:")
	t.Logf("  - Request ID: %s", result.RequestID)
	t.Logf("  - Gates executed: %d", len(result.GateResults))
	t.Logf("  - Total issues: %d", result.TotalIssues)
	t.Logf("  - Total warnings: %d", result.TotalWarnings)
	t.Logf("  - Success: %v", result.Success)
	t.Logf("  - Duration: %v", result.Duration)
	t.Logf("  - Overall quality score: %.2f (%s)", result.Summary.Overall.Score, result.Summary.Overall.Grade)
}

// TestPipelineGateManagement tests adding and removing gates
func TestPipelineGateManagement(t *testing.T) {
	config := PipelineConfig{
		Name:        "gate_management_test",
		Parallel:    false,
		Timeout:     1 * time.Minute,
		Environment: "test",
	}

	pipeline := NewVerificationPipeline(config)

	// Initially no gates
	gates := pipeline.GetGates()
	if len(gates) != 0 {
		t.Errorf("Expected 0 gates initially, got %d", len(gates))
	}

	// Add a gate
	unitGateConfig := GateConfig{
		Enabled: true,
		Timeout: 30 * time.Second,
	}
	unitGate := NewUnitTestGate(unitGateConfig)
	err := pipeline.AddGate(unitGate)
	if err != nil {
		t.Fatalf("Failed to add unit test gate: %v", err)
	}

	// Verify gate was added
	gates = pipeline.GetGates()
	if len(gates) != 1 {
		t.Errorf("Expected 1 gate after adding, got %d", len(gates))
	}

	if gates[0].GetType() != GateTypeUnit {
		t.Errorf("Expected unit gate type, got %v", gates[0].GetType())
	}

	// Remove the gate
	err = pipeline.RemoveGate(GateTypeUnit)
	if err != nil {
		t.Fatalf("Failed to remove unit test gate: %v", err)
	}

	// Verify gate was removed
	gates = pipeline.GetGates()
	if len(gates) != 0 {
		t.Errorf("Expected 0 gates after removal, got %d", len(gates))
	}

	// Try to remove non-existent gate
	err = pipeline.RemoveGate(GateTypeContract)
	if err == nil {
		t.Error("Expected error when removing non-existent gate")
	}
}

// TestPipelineParallelExecution tests parallel gate execution
func TestPipelineParallelExecution(t *testing.T) {
	config := PipelineConfig{
		Name:        "parallel_test",
		Parallel:    true, // Enable parallel execution
		Timeout:     2 * time.Minute,
		Environment: "test",
	}

	pipeline := NewVerificationPipeline(config)

	// Add multiple gates
	unitGate := NewUnitTestGate(GateConfig{Enabled: true, Timeout: 30 * time.Second})
	contractGate := NewContractTestGate(GateConfig{Enabled: true, Timeout: 30 * time.Second})

	err := pipeline.AddGate(unitGate)
	if err != nil {
		t.Fatalf("Failed to add unit test gate: %v", err)
	}

	err = pipeline.AddGate(contractGate)
	if err != nil {
		t.Fatalf("Failed to add contract test gate: %v", err)
	}

	// Create test artifacts that both gates can verify
	artifacts := []Artifact{
		{
			Path:     "test_file.go",
			Type:     ArtifactTypeTest,
			Language: "go",
			Content:  "package main\n\nimport \"testing\"\n\nfunc TestDummy(t *testing.T) {}\n",
			Size:     60,
			Hash:     "parallel_test_hash",
		},
		{
			Path:     "openapi.yaml",
			Type:     ArtifactTypeSchema,
			Language: "yaml",
			Content: `openapi: 3.0.0
info:
  title: Parallel Test API
  version: 1.0.0
paths:
  /test:
    get:
      responses:
        200:
          description: OK
`,
			Size: 150,
			Hash: "parallel_contract_hash",
		},
	}

	// Execute pipeline
	ctx := context.Background()
	start := time.Now()
	result, err := pipeline.Execute(ctx, artifacts)
	duration := time.Since(start)

	if err != nil {
		t.Fatalf("Parallel pipeline execution failed: %v", err)
	}

	// Verify results
	if result == nil {
		t.Fatal("Pipeline result is nil")
	}

	if !result.Success {
		t.Error("Pipeline should succeed with valid artifacts")
	}

	// Parallel execution should be faster than sequential (though hard to test reliably)
	t.Logf("Parallel execution completed in %v", duration)

	// Verify both gates were executed
	gateTypes := make(map[GateType]bool)
	for _, gateResult := range result.GateResults {
		gateTypes[gateResult.GateType] = true
	}

	expectedGateCount := 2
	if len(gateTypes) != expectedGateCount {
		t.Errorf("Expected %d unique gate types executed, got %d", expectedGateCount, len(gateTypes))
	}

	// Check metrics show parallel execution
	metrics := pipeline.GetMetrics()
	if metrics.ParallelGates == 0 {
		t.Error("Metrics should show parallel gates were executed")
	}
}

// TestGateCanVerifyLogic tests gate verification logic
func TestGateCanVerifyLogic(t *testing.T) {
	unitGate := NewUnitTestGate(GateConfig{Enabled: true})
	contractGate := NewContractTestGate(GateConfig{Enabled: true})

	// Test unit gate verification
	testArtifacts := []Artifact{
		{Path: "main_test.go", Type: ArtifactTypeTest},
		{Path: "helper_test.go", Type: ArtifactTypeTest},
	}

	if !unitGate.CanVerify(testArtifacts) {
		t.Error("Unit gate should be able to verify test artifacts")
	}

	sourceArtifacts := []Artifact{
		{Path: "main.go", Type: ArtifactTypeSource},
		{Path: "config.json", Type: ArtifactTypeConfig},
	}

	if unitGate.CanVerify(sourceArtifacts) {
		t.Error("Unit gate should not verify non-test artifacts without test files")
	}

	// Test contract gate verification
	contractArtifacts := []Artifact{
		{Path: "api.yaml", Type: ArtifactTypeSchema},
		{Path: "openapi.yml", Type: ArtifactTypeSchema},
	}

	if !contractGate.CanVerify(contractArtifacts) {
		t.Error("Contract gate should be able to verify schema artifacts")
	}

	if contractGate.CanVerify(sourceArtifacts) {
		t.Error("Contract gate should not verify source artifacts without schemas")
	}
}

// BenchmarkPipelineExecution benchmarks pipeline execution performance
func BenchmarkPipelineExecution(b *testing.B) {
	config := PipelineConfig{
		Name:        "benchmark_test",
		Parallel:    true,
		Timeout:     1 * time.Minute,
		Environment: "test",
	}

	pipeline := NewVerificationPipeline(config)

	// Add gates
	unitGate := NewUnitTestGate(GateConfig{Enabled: true, Timeout: 10 * time.Second})
	err := pipeline.AddGate(unitGate)
	if err != nil {
		b.Fatalf("Failed to add gate: %v", err)
	}

	// Create test artifacts
	artifacts := []Artifact{
		{
			Path:     "bench_test.go",
			Type:     ArtifactTypeTest,
			Language: "go",
			Content:  "package main\n\nimport \"testing\"\n\nfunc TestBench(t *testing.T) {}\n",
			Size:     65,
			Hash:     "bench_hash",
		},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		_, err := pipeline.Execute(ctx, artifacts)
		if err != nil {
			b.Fatalf("Pipeline execution failed: %v", err)
		}
	}
}