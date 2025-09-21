package verifier

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/llm"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/prompts"
)

// VerificationPipeline implements a complete verification pipeline
type VerificationPipeline struct {
	gates          []Gate
	repairLoop     *RepairLoop
	config         PipelineConfig
	metrics        PipelineMetrics
	mu             sync.RWMutex
}

// PipelineConfig configures the verification pipeline
type PipelineConfig struct {
	Name                string        `json:"name"`
	Parallel            bool          `json:"parallel"`           // Run gates in parallel
	StopOnFirstFailure  bool          `json:"stop_on_first_failure"`
	EnableRepairLoop    bool          `json:"enable_repair_loop"`
	MaxRepairIterations int           `json:"max_repair_iterations"`
	Timeout             time.Duration `json:"timeout"`
	Environment         string        `json:"environment"`        // "development", "staging", "production"
}

// NewVerificationPipeline creates a new verification pipeline
func NewVerificationPipeline(config PipelineConfig) *VerificationPipeline {
	// Set defaults
	if config.MaxRepairIterations == 0 {
		config.MaxRepairIterations = 3
	}
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Minute
	}
	if config.Environment == "" {
		config.Environment = "development"
	}

	return &VerificationPipeline{
		gates:   []Gate{},
		config:  config,
		metrics: PipelineMetrics{},
	}
}

// NewVerificationPipelineWithLLM creates a pipeline with LLM repair capabilities
func NewVerificationPipelineWithLLM(config PipelineConfig, llmClient llm.Client, promptComposer *prompts.PromptComposer) *VerificationPipeline {
	pipeline := NewVerificationPipeline(config)

	if config.EnableRepairLoop {
		repairConfig := RepairConfig{
			Enabled:             true,
			MaxAttempts:         3,
			MaxIterations:       config.MaxRepairIterations,
			Timeout:             10 * time.Minute,
			AutoApply:           true,
			ConfidenceThreshold: 0.8,
			SeverityThreshold:   SeverityWarning,
			AllowedIssueTypes: []IssueType{
				IssueTypeSyntax,
				IssueTypeStyle,
				IssueTypeMaintenance,
			},
			BackoffStrategy: "exponential",
			PreserveSafety:  true,
		}

		pipeline.repairLoop = NewRepairLoop(llmClient, promptComposer, repairConfig)
	}

	return pipeline
}

// AddGate adds a verification gate to the pipeline
func (p *VerificationPipeline) AddGate(gate Gate) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.gates = append(p.gates, gate)
	return nil
}

// RemoveGate removes a gate from the pipeline
func (p *VerificationPipeline) RemoveGate(gateType GateType) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	for i, gate := range p.gates {
		if gate.GetType() == gateType {
			p.gates = append(p.gates[:i], p.gates[i+1:]...)
			return nil
		}
	}

	return fmt.Errorf("gate type %s not found in pipeline", gateType)
}

// Execute runs the complete verification pipeline
func (p *VerificationPipeline) Execute(ctx context.Context, artifacts []Artifact) (*PipelineResult, error) {
	startTime := time.Now()

	result := &PipelineResult{
		RequestID:     fmt.Sprintf("pipeline_%d", time.Now().Unix()),
		GateResults:   []VerificationResult{},
		Artifacts:     artifacts,
		Summary:       PipelineSummary{},
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, p.config.Timeout)
	defer cancel()

	// Execute gates
	var gateResults []VerificationResult
	var err error

	if p.config.Parallel {
		gateResults, err = p.executeGatesParallel(timeoutCtx, artifacts)
	} else {
		gateResults, err = p.executeGatesSequential(timeoutCtx, artifacts)
	}

	if err != nil {
		result.Success = false
		result.Duration = time.Since(startTime)
		return result, err
	}

	result.GateResults = gateResults

	// Collect all issues for potential repair
	allIssues := []Issue{}
	for _, gateResult := range gateResults {
		allIssues = append(allIssues, gateResult.Issues...)
	}

	// Apply repair loop if enabled and there are issues
	if p.repairLoop != nil && len(allIssues) > 0 {
		repairResult, err := p.applyRepairLoop(timeoutCtx, allIssues, artifacts)
		if err != nil {
			// Store repair error in metadata since PipelineResult doesn't have Warnings
			if result.Summary.Overall.Benchmark == 0 {
				result.Summary.Overall.Benchmark = -1 // Indicate repair failed
			}
		} else if repairResult != nil {
			// Re-run verification after repairs
			repairedArtifacts := p.updateArtifactsWithRepairs(artifacts, repairResult.ModifiedFiles)

			var rerunResults []VerificationResult
			if p.config.Parallel {
				rerunResults, _ = p.executeGatesParallel(timeoutCtx, repairedArtifacts)
			} else {
				rerunResults, _ = p.executeGatesSequential(timeoutCtx, repairedArtifacts)
			}

			if len(rerunResults) > 0 {
				result.GateResults = rerunResults
				result.Artifacts = repairedArtifacts
			}

			// Store repair metadata
			if result.Summary.Overall.Benchmark == 0 {
				result.Summary.Overall.Benchmark = float64(len(repairResult.Fixes)) / float64(len(allIssues))
			}
		}
	}

	// Calculate final results
	p.calculatePipelineResults(result)
	result.Duration = time.Since(startTime)

	// Update metrics
	p.updateMetrics(result)

	return result, nil
}

// executeGatesSequential runs gates one by one
func (p *VerificationPipeline) executeGatesSequential(ctx context.Context, artifacts []Artifact) ([]VerificationResult, error) {
	results := []VerificationResult{}

	for _, gate := range p.gates {
		if !gate.CanVerify(artifacts) {
			continue
		}

		req := &VerificationRequest{
			RequestID:   fmt.Sprintf("gate_%s_%d", gate.GetType(), time.Now().Unix()),
			Artifacts:   artifacts,
			Config:      gate.GetConfiguration(),
			Timeout:     p.config.Timeout,
			Environment: p.config.Environment,
		}

		result, err := gate.Verify(ctx, req)
		if err != nil {
			return results, fmt.Errorf("gate %s failed: %w", gate.GetName(), err)
		}

		results = append(results, *result)

		// Stop on first failure if configured
		if p.config.StopOnFirstFailure && !result.Success {
			break
		}
	}

	return results, nil
}

// executeGatesParallel runs gates in parallel
func (p *VerificationPipeline) executeGatesParallel(ctx context.Context, artifacts []Artifact) ([]VerificationResult, error) {
	type gateResult struct {
		result *VerificationResult
		err    error
		gate   Gate
	}

	// Find applicable gates
	applicableGates := []Gate{}
	for _, gate := range p.gates {
		if gate.CanVerify(artifacts) {
			applicableGates = append(applicableGates, gate)
		}
	}

	if len(applicableGates) == 0 {
		return []VerificationResult{}, nil
	}

	// Run gates in parallel
	resultChan := make(chan gateResult, len(applicableGates))
	var wg sync.WaitGroup

	for _, gate := range applicableGates {
		wg.Add(1)
		go func(g Gate) {
			defer wg.Done()

			req := &VerificationRequest{
				RequestID:   fmt.Sprintf("gate_%s_%d", g.GetType(), time.Now().Unix()),
				Artifacts:   artifacts,
				Config:      g.GetConfiguration(),
				Timeout:     p.config.Timeout,
				Environment: p.config.Environment,
			}

			result, err := g.Verify(ctx, req)
			resultChan <- gateResult{
				result: result,
				err:    err,
				gate:   g,
			}
		}(gate)
	}

	// Wait for all gates to complete
	go func() {
		wg.Wait()
		close(resultChan)
	}()

	// Collect results
	results := []VerificationResult{}
	for gateRes := range resultChan {
		if gateRes.err != nil {
			return results, fmt.Errorf("gate %s failed: %w", gateRes.gate.GetName(), gateRes.err)
		}
		results = append(results, *gateRes.result)
	}

	return results, nil
}

// applyRepairLoop applies the repair loop to fix issues
func (p *VerificationPipeline) applyRepairLoop(ctx context.Context, issues []Issue, artifacts []Artifact) (*RepairResult, error) {
	// Convert artifacts to source code map
	sourceCode := make(map[string]string)
	for _, artifact := range artifacts {
		sourceCode[artifact.Path] = artifact.Content
	}

	// Create repair request
	repairReq := &RepairRequest{
		RequestID:  fmt.Sprintf("repair_%d", time.Now().Unix()),
		Issues:     issues,
		SourceCode: sourceCode,
		Context: RepairContext{
			Language:    "go", // Default, should be detected from artifacts
			ProjectType: "web_service",
		},
		Preferences: RepairPreferences{
			PreferSafety:     true,
			MinimalChanges:   true,
			PreferReadability: true,
		},
	}

	return p.repairLoop.RepairIssues(ctx, repairReq)
}

// updateArtifactsWithRepairs updates artifacts with repaired code
func (p *VerificationPipeline) updateArtifactsWithRepairs(artifacts []Artifact, modifiedFiles map[string]string) []Artifact {
	updatedArtifacts := make([]Artifact, 0, len(artifacts))

	for _, artifact := range artifacts {
		if newContent, exists := modifiedFiles[artifact.Path]; exists {
			// Update artifact with repaired content
			updatedArtifact := artifact
			updatedArtifact.Content = newContent
			updatedArtifact.Size = int64(len(newContent))
			// TODO: Update hash
			updatedArtifacts = append(updatedArtifacts, updatedArtifact)
		} else {
			updatedArtifacts = append(updatedArtifacts, artifact)
		}
	}

	return updatedArtifacts
}

// calculatePipelineResults calculates final pipeline results
func (p *VerificationPipeline) calculatePipelineResults(result *PipelineResult) {
	totalIssues := 0
	totalWarnings := 0
	allPassed := true

	for _, gateResult := range result.GateResults {
		totalIssues += len(gateResult.Issues)
		totalWarnings += len(gateResult.Warnings)
		if !gateResult.Success {
			allPassed = false
		}
	}

	result.TotalIssues = totalIssues
	result.TotalWarnings = totalWarnings
	result.Success = allPassed

	// Calculate quality scores
	result.Summary = p.calculateQualityScores(result.GateResults)
}

// calculateQualityScores calculates quality scores for the pipeline
func (p *VerificationPipeline) calculateQualityScores(gateResults []VerificationResult) PipelineSummary {
	summary := PipelineSummary{}

	// Simple scoring based on issues found
	totalIssues := 0
	criticalIssues := 0
	errorIssues := 0
	warningIssues := 0

	for _, result := range gateResults {
		for _, issue := range result.Issues {
			totalIssues++
			switch issue.Severity {
			case SeverityCritical, SeverityBlocking:
				criticalIssues++
			case SeverityError:
				errorIssues++
			case SeverityWarning:
				warningIssues++
			}
		}
	}

	// Calculate overall score (1.0 = perfect, 0.0 = terrible)
	overallScore := 1.0
	if totalIssues > 0 {
		// Penalize based on severity
		penalty := float64(criticalIssues)*0.5 + float64(errorIssues)*0.3 + float64(warningIssues)*0.1
		overallScore = max(0.0, 1.0-penalty/10.0) // Normalize to reasonable scale
	}

	// Set all scores to overall for simplicity
	qualityScore := QualityScore{
		Score:  overallScore,
		Grade:  getGradeFromScore(overallScore),
		Issues: totalIssues,
		Trend:  "stable",
	}

	summary.Quality = qualityScore
	summary.Security = qualityScore
	summary.Maintainability = qualityScore
	summary.Performance = qualityScore
	summary.Compliance = qualityScore
	summary.Overall = qualityScore

	return summary
}

// getGradeFromScore converts a score to a letter grade
func getGradeFromScore(score float64) string {
	if score >= 0.9 {
		return "A"
	} else if score >= 0.8 {
		return "B"
	} else if score >= 0.7 {
		return "C"
	} else if score >= 0.6 {
		return "D"
	}
	return "F"
}

// updateMetrics updates pipeline metrics
func (p *VerificationPipeline) updateMetrics(result *PipelineResult) {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.metrics.GatesExecuted = len(result.GateResults)
	p.metrics.TotalDuration = result.Duration

	for _, gateResult := range result.GateResults {
		if gateResult.Success {
			p.metrics.GatesPassed++
		} else {
			p.metrics.GatesFailed++
		}
	}

	if p.config.Parallel {
		p.metrics.ParallelGates = len(result.GateResults)
	} else {
		p.metrics.SequentialGates = len(result.GateResults)
	}
}

// GetGates returns all gates in the pipeline
func (p *VerificationPipeline) GetGates() []Gate {
	p.mu.RLock()
	defer p.mu.RUnlock()

	gates := make([]Gate, len(p.gates))
	copy(gates, p.gates)
	return gates
}

// GetMetrics returns current pipeline metrics
func (p *VerificationPipeline) GetMetrics() PipelineMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	return p.metrics
}

// max returns the maximum of two float64 values
func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}