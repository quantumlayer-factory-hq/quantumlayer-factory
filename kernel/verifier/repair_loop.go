package verifier

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/llm"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/prompts"
)

// RepairLoop provides automatic error fixing using LLM
type RepairLoop struct {
	llmClient      llm.Client
	promptComposer *prompts.PromptComposer
	config         RepairConfig
	metrics        RepairMetrics
}

// RepairConfig configures the repair loop behavior
type RepairConfig struct {
	Enabled              bool          `json:"enabled"`
	MaxAttempts          int           `json:"max_attempts"`          // Maximum repair attempts per issue
	MaxIterations        int           `json:"max_iterations"`        // Maximum iterations for a single repair session
	Timeout              time.Duration `json:"timeout"`               // Timeout for repair operations
	AutoApply            bool          `json:"auto_apply"`            // Automatically apply fixes
	ConfidenceThreshold  float64       `json:"confidence_threshold"`  // Minimum confidence to apply fix
	SeverityThreshold    Severity      `json:"severity_threshold"`    // Only fix issues above this severity
	AllowedIssueTypes    []IssueType   `json:"allowed_issue_types"`   // Issue types that can be auto-fixed
	BackoffStrategy      string        `json:"backoff_strategy"`      // "linear", "exponential"
	PreserveSafety       bool          `json:"preserve_safety"`       // Preserve safety-critical code sections
}

// RepairMetrics tracks repair operation metrics
type RepairMetrics struct {
	TotalIssues        int           `json:"total_issues"`
	IssuesFixed        int           `json:"issues_fixed"`
	IssuesFailed       int           `json:"issues_failed"`
	IssuesSkipped      int           `json:"issues_skipped"`
	TotalIterations    int           `json:"total_iterations"`
	SuccessRate        float64       `json:"success_rate"`
	AverageConfidence  float64       `json:"average_confidence"`
	TotalDuration      time.Duration `json:"total_duration"`
	LLMTokensUsed      int           `json:"llm_tokens_used"`
	LLMCost            float64       `json:"llm_cost"`
}

// RepairRequest represents a request to fix issues
type RepairRequest struct {
	RequestID     string            `json:"request_id"`
	Issues        []Issue           `json:"issues"`
	SourceCode    map[string]string `json:"source_code"`    // file_path -> content
	Context       RepairContext     `json:"context"`
	Preferences   RepairPreferences `json:"preferences"`
}

// RepairContext provides additional context for repairs
type RepairContext struct {
	Language        string                 `json:"language"`
	Framework       string                 `json:"framework"`
	ProjectType     string                 `json:"project_type"`
	Dependencies    []string               `json:"dependencies"`
	CodeStyle       map[string]interface{} `json:"code_style"`
	SafetyCritical  []string               `json:"safety_critical"`  // Files/sections that should not be modified
	TestFiles       []string               `json:"test_files"`
	Documentation   string                 `json:"documentation"`
}

// RepairPreferences allows customization of repair behavior
type RepairPreferences struct {
	PreferSafety       bool     `json:"prefer_safety"`        // Prefer safer, more conservative fixes
	PreferPerformance  bool     `json:"prefer_performance"`   // Prefer performance-oriented fixes
	PreferReadability  bool     `json:"prefer_readability"`   // Prefer more readable solutions
	AvoidPatterns      []string `json:"avoid_patterns"`       // Code patterns to avoid
	PreferPatterns     []string `json:"prefer_patterns"`      // Code patterns to prefer
	MinimalChanges     bool     `json:"minimal_changes"`      // Make minimal changes to existing code
}

// RepairResult contains the results of a repair operation
type RepairResult struct {
	Success        bool                   `json:"success"`
	RequestID      string                 `json:"request_id"`
	Fixes          []AppliedFix           `json:"fixes"`
	FailedIssues   []FailedRepair         `json:"failed_issues"`
	SkippedIssues  []SkippedRepair        `json:"skipped_issues"`
	Metrics        RepairMetrics          `json:"metrics"`
	Iterations     int                    `json:"iterations"`
	FinalResult    *VerificationResult    `json:"final_result,omitempty"`
	ModifiedFiles  map[string]string      `json:"modified_files"`  // file_path -> new_content
	Warnings       []string               `json:"warnings"`
	Duration       time.Duration          `json:"duration"`
}

// AppliedFix represents a successfully applied fix
type AppliedFix struct {
	IssueID      string    `json:"issue_id"`
	File         string    `json:"file"`
	Line         int       `json:"line"`
	FixType      string    `json:"fix_type"`
	Description  string    `json:"description"`
	Changes      []Change  `json:"changes"`
	Confidence   float64   `json:"confidence"`
	LLMModel     string    `json:"llm_model"`
	TokensUsed   int       `json:"tokens_used"`
	AppliedAt    time.Time `json:"applied_at"`
}

// Change represents a single code change
type Change struct {
	Type        string `json:"type"`         // "replace", "insert", "delete"
	StartLine   int    `json:"start_line"`
	EndLine     int    `json:"end_line"`
	StartColumn int    `json:"start_column,omitempty"`
	EndColumn   int    `json:"end_column,omitempty"`
	OldContent  string `json:"old_content"`
	NewContent  string `json:"new_content"`
}

// FailedRepair represents a repair attempt that failed
type FailedRepair struct {
	IssueID     string    `json:"issue_id"`
	Attempts    int       `json:"attempts"`
	LastError   string    `json:"last_error"`
	Suggestions []string  `json:"suggestions"`
	FailedAt    time.Time `json:"failed_at"`
}

// SkippedRepair represents an issue that was skipped
type SkippedRepair struct {
	IssueID    string `json:"issue_id"`
	Reason     string `json:"reason"`
	Severity   string `json:"severity"`
	SkippedAt  time.Time `json:"skipped_at"`
}

// NewRepairLoop creates a new repair loop instance
func NewRepairLoop(llmClient llm.Client, promptComposer *prompts.PromptComposer, config RepairConfig) *RepairLoop {
	// Set default config values
	if config.MaxAttempts == 0 {
		config.MaxAttempts = 3
	}
	if config.MaxIterations == 0 {
		config.MaxIterations = 5
	}
	if config.Timeout == 0 {
		config.Timeout = 10 * time.Minute
	}
	if config.ConfidenceThreshold == 0 {
		config.ConfidenceThreshold = 0.7
	}
	if config.SeverityThreshold == "" {
		config.SeverityThreshold = SeverityWarning
	}
	if len(config.AllowedIssueTypes) == 0 {
		config.AllowedIssueTypes = []IssueType{
			IssueTypeSyntax,
			IssueTypeStyle,
			IssueTypeMaintenance,
		}
	}
	if config.BackoffStrategy == "" {
		config.BackoffStrategy = "exponential"
	}

	return &RepairLoop{
		llmClient:      llmClient,
		promptComposer: promptComposer,
		config:         config,
		metrics:        RepairMetrics{},
	}
}

// RepairIssues attempts to fix the given issues automatically
func (r *RepairLoop) RepairIssues(ctx context.Context, req *RepairRequest) (*RepairResult, error) {
	if !r.config.Enabled {
		return &RepairResult{
			Success:   false,
			RequestID: req.RequestID,
			Warnings:  []string{"Repair loop is disabled"},
		}, nil
	}

	startTime := time.Now()
	result := &RepairResult{
		RequestID:     req.RequestID,
		Fixes:         []AppliedFix{},
		FailedIssues:  []FailedRepair{},
		SkippedIssues: []SkippedRepair{},
		ModifiedFiles: make(map[string]string),
		Warnings:      []string{},
	}

	// Copy source code for modifications
	for path, content := range req.SourceCode {
		result.ModifiedFiles[path] = content
	}

	// Create timeout context
	timeoutCtx, cancel := context.WithTimeout(ctx, r.config.Timeout)
	defer cancel()

	// Start repair iterations
	iteration := 0
	remainingIssues := r.filterRepairableIssues(req.Issues)

	for iteration < r.config.MaxIterations && len(remainingIssues) > 0 {
		iteration++
		r.metrics.TotalIterations++

		iterationFixes := []AppliedFix{}
		newRemainingIssues := []Issue{}

		for _, issue := range remainingIssues {
			select {
			case <-timeoutCtx.Done():
				result.Warnings = append(result.Warnings, "Repair operation timed out")
				goto FinishRepair
			default:
			}

			fix, err := r.repairSingleIssue(timeoutCtx, issue, result.ModifiedFiles, req.Context, req.Preferences)
			if err != nil {
				result.FailedIssues = append(result.FailedIssues, FailedRepair{
					IssueID:   issue.ID,
					Attempts:  1,
					LastError: err.Error(),
					FailedAt:  time.Now(),
				})
				r.metrics.IssuesFailed++
				continue
			}

			if fix != nil {
				// Apply the fix to the code
				err = r.applyFix(fix, result.ModifiedFiles)
				if err != nil {
					result.FailedIssues = append(result.FailedIssues, FailedRepair{
						IssueID:   issue.ID,
						Attempts:  1,
						LastError: fmt.Sprintf("Failed to apply fix: %v", err),
						FailedAt:  time.Now(),
					})
					r.metrics.IssuesFailed++
					continue
				}

				iterationFixes = append(iterationFixes, *fix)
				result.Fixes = append(result.Fixes, *fix)
				r.metrics.IssuesFixed++
			} else {
				// Issue couldn't be fixed, add back to remaining
				newRemainingIssues = append(newRemainingIssues, issue)
			}
		}

		remainingIssues = newRemainingIssues

		// If no fixes were made in this iteration, break to avoid infinite loop
		if len(iterationFixes) == 0 {
			break
		}
	}

FinishRepair:
	result.Iterations = iteration
	result.Duration = time.Since(startTime)
	result.Success = len(result.Fixes) > 0

	// Calculate metrics
	r.metrics.TotalIssues = len(req.Issues)
	r.metrics.TotalDuration = result.Duration
	if r.metrics.TotalIssues > 0 {
		r.metrics.SuccessRate = float64(r.metrics.IssuesFixed) / float64(r.metrics.TotalIssues)
	}

	// Calculate average confidence
	totalConfidence := 0.0
	for _, fix := range result.Fixes {
		totalConfidence += fix.Confidence
	}
	if len(result.Fixes) > 0 {
		r.metrics.AverageConfidence = totalConfidence / float64(len(result.Fixes))
	}

	result.Metrics = r.metrics

	return result, nil
}

// filterRepairableIssues filters issues that can be repaired
func (r *RepairLoop) filterRepairableIssues(issues []Issue) []Issue {
	var repairable []Issue

	for _, issue := range issues {
		// Check severity threshold
		if !r.meetsSeveityThreshold(issue.Severity) {
			continue
		}

		// Check if issue type is allowed
		if !r.isAllowedIssueType(issue.Type) {
			continue
		}

		// Check if file is safety-critical (this would need context)
		// For now, allow all issues
		repairable = append(repairable, issue)
	}

	return repairable
}

// meetsSeveityThreshold checks if issue severity meets threshold
func (r *RepairLoop) meetsSeveityThreshold(severity Severity) bool {
	severityLevels := map[Severity]int{
		SeverityInfo:     1,
		SeverityWarning:  2,
		SeverityError:    3,
		SeverityCritical: 4,
		SeverityBlocking: 5,
	}

	issueSeverity := severityLevels[severity]
	thresholdSeverity := severityLevels[r.config.SeverityThreshold]

	return issueSeverity >= thresholdSeverity
}

// isAllowedIssueType checks if issue type is allowed for auto-repair
func (r *RepairLoop) isAllowedIssueType(issueType IssueType) bool {
	for _, allowed := range r.config.AllowedIssueTypes {
		if issueType == allowed {
			return true
		}
	}
	return false
}

// repairSingleIssue attempts to repair a single issue using LLM
func (r *RepairLoop) repairSingleIssue(ctx context.Context, issue Issue, sourceCode map[string]string, repairContext RepairContext, preferences RepairPreferences) (*AppliedFix, error) {
	// Build repair prompt
	prompt, err := r.buildRepairPrompt(issue, sourceCode, repairContext, preferences)
	if err != nil {
		return nil, fmt.Errorf("failed to build repair prompt: %w", err)
	}

	// Select appropriate model for repair
	model := llm.ModelClaudeHaiku // Fast model for simple fixes
	if issue.Severity == SeverityCritical || issue.Severity == SeverityBlocking {
		model = llm.ModelClaudeSonnet // More capable model for critical issues
	}

	// Call LLM to generate fix
	llmReq := &llm.GenerateRequest{
		Prompt:      prompt,
		Model:       model,
		MaxTokens:   2048,
		Temperature: 0.1, // Low temperature for consistent, focused repairs
	}

	response, err := r.llmClient.Generate(ctx, llmReq)
	if err != nil {
		return nil, fmt.Errorf("LLM generation failed: %w", err)
	}

	// Parse LLM response into fix
	fix, err := r.parseRepairResponse(issue, response.Content, string(response.Model))
	if err != nil {
		return nil, fmt.Errorf("failed to parse repair response: %w", err)
	}

	// Update metrics
	r.metrics.LLMTokensUsed += response.Usage.TotalTokens
	r.metrics.LLMCost += response.Usage.Cost

	// Check confidence threshold
	if fix.Confidence < r.config.ConfidenceThreshold {
		return nil, nil // Don't apply low confidence fixes
	}

	return fix, nil
}

// buildRepairPrompt creates a prompt for LLM to generate a fix
func (r *RepairLoop) buildRepairPrompt(issue Issue, sourceCode map[string]string, repairContext RepairContext, preferences RepairPreferences) (string, error) {
	var prompt strings.Builder

	prompt.WriteString("You are an expert code repair assistant. Your task is to fix the following issue:\n\n")

	// Issue details
	prompt.WriteString(fmt.Sprintf("Issue ID: %s\n", issue.ID))
	prompt.WriteString(fmt.Sprintf("Type: %s\n", issue.Type))
	prompt.WriteString(fmt.Sprintf("Severity: %s\n", issue.Severity))
	prompt.WriteString(fmt.Sprintf("Title: %s\n", issue.Title))
	prompt.WriteString(fmt.Sprintf("Description: %s\n", issue.Description))

	if issue.File != "" {
		prompt.WriteString(fmt.Sprintf("File: %s\n", issue.File))
		if issue.Line > 0 {
			prompt.WriteString(fmt.Sprintf("Line: %d\n", issue.Line))
		}
	}

	// Context
	prompt.WriteString(fmt.Sprintf("\nLanguage: %s\n", repairContext.Language))
	if repairContext.Framework != "" {
		prompt.WriteString(fmt.Sprintf("Framework: %s\n", repairContext.Framework))
	}

	// Source code context
	if issue.File != "" && sourceCode[issue.File] != "" {
		prompt.WriteString(fmt.Sprintf("\nRelevant source code from %s:\n```\n%s\n```\n", issue.File, sourceCode[issue.File]))
	}

	// Preferences
	if preferences.PreferSafety {
		prompt.WriteString("\nPreference: Prioritize safe, conservative fixes.\n")
	}
	if preferences.MinimalChanges {
		prompt.WriteString("Preference: Make minimal changes to existing code.\n")
	}
	if len(preferences.AvoidPatterns) > 0 {
		prompt.WriteString(fmt.Sprintf("Avoid these patterns: %s\n", strings.Join(preferences.AvoidPatterns, ", ")))
	}

	// Instructions
	prompt.WriteString(`
Please provide a fix for this issue with the following format:

CONFIDENCE: <float between 0.0 and 1.0>
FIX_TYPE: <brief description of fix type>
DESCRIPTION: <detailed description of the fix>

CHANGES:
FILE: <file_path>
REPLACE:
<original code to replace>
WITH:
<new code>

Only provide fixes you are confident will resolve the issue without introducing new problems.
If you cannot provide a confident fix, respond with "CONFIDENCE: 0.0".
`)

	return prompt.String(), nil
}

// parseRepairResponse parses LLM response into an AppliedFix
func (r *RepairLoop) parseRepairResponse(issue Issue, response string, model string) (*AppliedFix, error) {
	lines := strings.Split(response, "\n")

	fix := &AppliedFix{
		IssueID:    issue.ID,
		File:       issue.File,
		Line:       issue.Line,
		LLMModel:   model,
		AppliedAt:  time.Now(),
		Changes:    []Change{},
	}

	// Parse response sections
	var currentSection string
	var currentChange *Change
	var buffer strings.Builder

	for _, line := range lines {
		line = strings.TrimSpace(line)

		if strings.HasPrefix(line, "CONFIDENCE:") {
			confidence := strings.TrimSpace(strings.TrimPrefix(line, "CONFIDENCE:"))
			if conf, err := fmt.Sscanf(confidence, "%f", &fix.Confidence); err != nil || conf != 1 {
				fix.Confidence = 0.0
			}
		} else if strings.HasPrefix(line, "FIX_TYPE:") {
			fix.FixType = strings.TrimSpace(strings.TrimPrefix(line, "FIX_TYPE:"))
		} else if strings.HasPrefix(line, "DESCRIPTION:") {
			fix.Description = strings.TrimSpace(strings.TrimPrefix(line, "DESCRIPTION:"))
		} else if strings.HasPrefix(line, "FILE:") {
			if currentChange != nil {
				// Finish previous change
				if currentSection == "WITH" {
					currentChange.NewContent = strings.TrimSpace(buffer.String())
				}
				fix.Changes = append(fix.Changes, *currentChange)
			}

			currentChange = &Change{
				Type: "replace",
			}
			buffer.Reset()
		} else if line == "REPLACE:" {
			currentSection = "REPLACE"
			buffer.Reset()
		} else if line == "WITH:" {
			if currentChange != nil && currentSection == "REPLACE" {
				currentChange.OldContent = strings.TrimSpace(buffer.String())
			}
			currentSection = "WITH"
			buffer.Reset()
		} else if line != "" && (currentSection == "REPLACE" || currentSection == "WITH") {
			if buffer.Len() > 0 {
				buffer.WriteString("\n")
			}
			buffer.WriteString(line)
		}
	}

	// Finish last change
	if currentChange != nil && currentSection == "WITH" {
		currentChange.NewContent = strings.TrimSpace(buffer.String())
		fix.Changes = append(fix.Changes, *currentChange)
	}

	return fix, nil
}

// applyFix applies a fix to the source code
func (r *RepairLoop) applyFix(fix *AppliedFix, sourceCode map[string]string) error {
	for _, change := range fix.Changes {
		if change.Type == "replace" && change.OldContent != "" && change.NewContent != "" {
			file := fix.File
			if file == "" {
				continue
			}

			content, exists := sourceCode[file]
			if !exists {
				return fmt.Errorf("file %s not found in source code", file)
			}

			// Simple string replacement
			newContent := strings.ReplaceAll(content, change.OldContent, change.NewContent)
			sourceCode[file] = newContent
		}
	}

	return nil
}

// GetMetrics returns current repair metrics
func (r *RepairLoop) GetMetrics() RepairMetrics {
	return r.metrics
}

// ResetMetrics resets the repair metrics
func (r *RepairLoop) ResetMetrics() {
	r.metrics = RepairMetrics{}
}