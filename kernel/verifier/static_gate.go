package verifier

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// StaticGate performs static analysis verification
type StaticGate struct {
	name     string
	config   GateConfig
	runners  map[string]Runner
}

// NewStaticGate creates a new static analysis gate
func NewStaticGate(config GateConfig) *StaticGate {
	if config.Timeout == 0 {
		config.Timeout = 5 * time.Minute
	}

	return &StaticGate{
		name:    "Static Analysis Gate",
		config:  config,
		runners: make(map[string]Runner),
	}
}

// GetType returns the gate type
func (g *StaticGate) GetType() GateType {
	return GateTypeStatic
}

// GetName returns the gate name
func (g *StaticGate) GetName() string {
	return g.name
}

// GetConfiguration returns the gate configuration
func (g *StaticGate) GetConfiguration() GateConfig {
	return g.config
}

// AddRunner adds a verification tool runner to this gate
func (g *StaticGate) AddRunner(runner Runner) error {
	if runner == nil {
		return fmt.Errorf("runner cannot be nil")
	}

	g.runners[runner.GetName()] = runner
	return nil
}

// RemoveRunner removes a runner from this gate
func (g *StaticGate) RemoveRunner(name string) {
	delete(g.runners, name)
}

// GetRunners returns all registered runners
func (g *StaticGate) GetRunners() map[string]Runner {
	return g.runners
}

// CanVerify determines if this gate can verify the given artifacts
func (g *StaticGate) CanVerify(artifacts []Artifact) bool {
	if !g.config.Enabled {
		return false
	}

	// Check if any runner can handle any of the artifacts
	for _, artifact := range artifacts {
		if artifact.Type == ArtifactTypeSource || artifact.Type == ArtifactTypeConfig {
			for _, runner := range g.runners {
				if runner.CanRun([]Artifact{artifact}) {
					return true
				}
			}
		}
	}

	return false
}

// Verify performs static analysis verification
func (g *StaticGate) Verify(ctx context.Context, req *VerificationRequest) (*VerificationResult, error) {
	startTime := time.Now()

	result := &VerificationResult{
		Success:   true,
		GateType:  g.GetType(),
		GateName:  g.GetName(),
		Artifacts: req.Artifacts,
		Issues:    []Issue{},
		Warnings:  []string{},
		Timestamp: startTime,
		Metadata:  make(map[string]interface{}),
	}

	// Create a context with timeout
	timeoutCtx, cancel := context.WithTimeout(ctx, g.config.Timeout)
	defer cancel()

	// Group artifacts by language/framework for efficient processing
	artifactGroups := g.groupArtifacts(req.Artifacts)

	var allIssues []Issue
	var allWarnings []string
	var totalFilesScanned int
	var totalLinesScanned int

	// Run verification for each group
	for groupKey, artifacts := range artifactGroups {
		// Find suitable runners for this group
		suitableRunners := g.findSuitableRunners(artifacts)

		if len(suitableRunners) == 0 {
			warning := fmt.Sprintf("No suitable runners found for %s artifacts", groupKey)
			allWarnings = append(allWarnings, warning)
			continue
		}

		// Execute runners for this group
		for _, runner := range suitableRunners {
			runnerResult, err := g.executeRunner(timeoutCtx, runner, artifacts, req.Config)
			if err != nil {
				result.Success = false
				allWarnings = append(allWarnings, fmt.Sprintf("Runner %s failed: %v", runner.GetName(), err))
				continue
			}

			// Aggregate results
			allIssues = append(allIssues, runnerResult.Issues...)
			totalFilesScanned += len(artifacts)

			// Count lines scanned
			for _, artifact := range artifacts {
				if artifact.Content != "" {
					totalLinesScanned += countLines(artifact.Content)
				}
			}

			// Store runner-specific metadata
			result.Metadata[runner.GetName()] = map[string]interface{}{
				"exit_code": runnerResult.ExitCode,
				"duration":  runnerResult.Duration,
				"issues":    len(runnerResult.Issues),
			}
		}
	}

	// Process and categorize issues
	result.Issues = g.processIssues(allIssues)
	result.Warnings = allWarnings

	// Generate repair hints
	result.RepairHints = g.generateRepairHints(result.Issues)

	// Calculate metrics
	result.Metrics = VerificationMetrics{
		FilesScanned: totalFilesScanned,
		LinesScanned: totalLinesScanned,
		IssuesFound:  len(result.Issues),
		Duration:     time.Since(startTime),
	}

	// Determine overall success
	result.Success = g.determineSuccess(result.Issues)
	result.Duration = time.Since(startTime)

	return result, nil
}

// groupArtifacts groups artifacts by language and framework for efficient processing
func (g *StaticGate) groupArtifacts(artifacts []Artifact) map[string][]Artifact {
	groups := make(map[string][]Artifact)

	for _, artifact := range artifacts {
		// Only process source and config files
		if artifact.Type != ArtifactTypeSource && artifact.Type != ArtifactTypeConfig {
			continue
		}

		// Create group key from language and framework
		groupKey := artifact.Language
		if artifact.Framework != "" {
			groupKey = fmt.Sprintf("%s-%s", artifact.Language, artifact.Framework)
		}
		if groupKey == "" {
			groupKey = "unknown"
		}

		groups[groupKey] = append(groups[groupKey], artifact)
	}

	return groups
}

// findSuitableRunners finds runners that can process the given artifacts
func (g *StaticGate) findSuitableRunners(artifacts []Artifact) []Runner {
	var suitable []Runner

	for _, runner := range g.runners {
		if runner.CanRun(artifacts) {
			suitable = append(suitable, runner)
		}
	}

	return suitable
}

// executeRunner executes a specific runner with the given artifacts
func (g *StaticGate) executeRunner(ctx context.Context, runner Runner, artifacts []Artifact, config GateConfig) (*RunnerResult, error) {
	// Merge runner default config with gate config
	runnerConfig := runner.GetDefaultConfig()
	if config.Rules != nil {
		for key, value := range config.Rules {
			runnerConfig[key] = value
		}
	}

	return runner.Run(ctx, artifacts, runnerConfig)
}

// processIssues processes and enhances issues from runners
func (g *StaticGate) processIssues(issues []Issue) []Issue {
	var processed []Issue

	for i, issue := range issues {
		// Generate unique ID if not provided
		if issue.ID == "" {
			issue.ID = fmt.Sprintf("static-%d-%s", i, strings.ToLower(string(issue.Type)))
		}

		// Ensure severity is set
		if issue.Severity == "" {
			issue.Severity = g.inferSeverity(issue)
		}

		// Enhance description if needed
		if issue.Description == "" && issue.Title != "" {
			issue.Description = issue.Title
		}

		processed = append(processed, issue)
	}

	return processed
}

// inferSeverity infers severity from issue type and content
func (g *StaticGate) inferSeverity(issue Issue) Severity {
	switch issue.Type {
	case IssueTypeSecurity:
		return SeverityError
	case IssueTypeSyntax:
		return SeverityError
	case IssueTypeCompliance:
		return SeverityWarning
	case IssueTypeStyle:
		return SeverityInfo
	case IssueTypePerformance:
		return SeverityWarning
	case IssueTypeMaintenance:
		return SeverityInfo
	default:
		return SeverityWarning
	}
}

// generateRepairHints generates hints for fixing common issues
func (g *StaticGate) generateRepairHints(issues []Issue) []RepairHint {
	var hints []RepairHint

	for _, issue := range issues {
		hint := RepairHint{
			IssueID:    issue.ID,
			Confidence: 0.7, // Default confidence
		}

		switch issue.Type {
		case IssueTypeSyntax:
			hint.Suggestion = "Check syntax according to language specification"
			hint.Automated = false

		case IssueTypeSecurity:
			hint.Suggestion = "Review security best practices and apply appropriate fixes"
			hint.Automated = false
			hint.References = []string{
				"OWASP Top 10",
				"Language-specific security guidelines",
			}

		case IssueTypeStyle:
			hint.Suggestion = "Apply automatic code formatting"
			hint.Automated = true
			hint.Confidence = 0.9

		case IssueTypeCompliance:
			hint.Suggestion = "Review compliance requirements and update code accordingly"
			hint.Automated = false
		}

		if hint.Suggestion != "" {
			hints = append(hints, hint)
		}
	}

	return hints
}

// determineSuccess determines if the verification was successful based on issues
func (g *StaticGate) determineSuccess(issues []Issue) bool {
	// Check thresholds from configuration
	if thresholds := g.config.Thresholds; thresholds != nil {
		errorCount := 0
		criticalCount := 0
		blockingCount := 0

		for _, issue := range issues {
			switch issue.Severity {
			case SeverityError:
				errorCount++
			case SeverityCritical:
				criticalCount++
			case SeverityBlocking:
				blockingCount++
			}
		}

		// Check blocking issues first
		if blockingCount > 0 {
			return false
		}

		// Check critical threshold
		if maxCritical, ok := thresholds["max_critical"]; ok {
			if float64(criticalCount) > maxCritical {
				return false
			}
		}

		// Check error threshold
		if maxErrors, ok := thresholds["max_errors"]; ok {
			if float64(errorCount) > maxErrors {
				return false
			}
		}
	}

	// Default: fail if there are any blocking or critical issues
	for _, issue := range issues {
		if issue.Severity == SeverityBlocking || issue.Severity == SeverityCritical {
			return false
		}
	}

	return true
}

// countLines counts the number of lines in the given content
func countLines(content string) int {
	if content == "" {
		return 0
	}
	return strings.Count(content, "\n") + 1
}