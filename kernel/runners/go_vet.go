package runners

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/verifier"
)

// GoVetRunner implements static analysis using Go vet
type GoVetRunner struct {
	name    string
	version string
}

// NewGoVetRunner creates a new Go vet runner
func NewGoVetRunner() *GoVetRunner {
	return &GoVetRunner{
		name:    "go-vet",
		version: "1.0.0",
	}
}

// GetName returns the runner name
func (r *GoVetRunner) GetName() string {
	return r.name
}

// GetVersion returns the runner version
func (r *GoVetRunner) GetVersion() string {
	return r.version
}

// CanRun determines if this runner can process the given artifacts
func (r *GoVetRunner) CanRun(artifacts []verifier.Artifact) bool {
	for _, artifact := range artifacts {
		if artifact.Language == "go" && artifact.Type == verifier.ArtifactTypeSource {
			return true
		}
		// Also check file extension for .go files
		if strings.HasSuffix(artifact.Path, ".go") {
			return true
		}
	}
	return false
}

// Run executes go vet on the given artifacts
func (r *GoVetRunner) Run(ctx context.Context, artifacts []verifier.Artifact, config map[string]interface{}) (*verifier.RunnerResult, error) {
	startTime := time.Now()

	result := &verifier.RunnerResult{
		Success:   true,
		Issues:    []verifier.Issue{},
		Duration:  0,
		Artifacts: artifacts,
	}

	// Create temporary directory for Go files
	tempDir, err := os.MkdirTemp("", "go-vet-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp directory: %w", err)
	}
	defer os.RemoveAll(tempDir)

	// Write Go files to temp directory
	goFiles, err := r.writeGoFiles(tempDir, artifacts)
	if err != nil {
		return nil, fmt.Errorf("failed to write Go files: %w", err)
	}

	if len(goFiles) == 0 {
		result.Duration = time.Since(startTime)
		return result, nil
	}

	// Create a minimal go.mod file if it doesn't exist
	if err := r.ensureGoMod(tempDir); err != nil {
		return nil, fmt.Errorf("failed to create go.mod: %w", err)
	}

	// Run go vet
	cmd := exec.CommandContext(ctx, "go", "vet", "-json", "./...")
	cmd.Dir = tempDir

	// Set environment variables
	cmd.Env = append(os.Environ(),
		"GO111MODULE=on",
		"GOPROXY=direct",
		"GOSUMDB=off",
	)

	output, err := cmd.CombinedOutput()
	result.ExitCode = cmd.ProcessState.ExitCode()
	result.Duration = time.Since(startTime)

	if err != nil && result.ExitCode != 0 {
		// go vet returns non-zero exit code when issues are found
		result.Success = false
		result.Stderr = string(output)
	} else {
		result.Stdout = string(output)
	}

	// Parse go vet output
	issues, parseErr := r.parseGoVetOutput(string(output), tempDir)
	if parseErr != nil {
		// If parsing fails, create a generic issue
		result.Issues = []verifier.Issue{
			{
				Type:        verifier.IssueTypeSyntax,
				Severity:    verifier.SeverityError,
				Title:       "Go vet parsing error",
				Description: fmt.Sprintf("Failed to parse go vet output: %v", parseErr),
			},
		}
	} else {
		result.Issues = issues
	}

	// Map temp file paths back to original paths
	result.Issues = r.mapPathsToOriginal(result.Issues, tempDir, artifacts)

	return result, nil
}

// GetDefaultConfig returns the default configuration for go vet
func (r *GoVetRunner) GetDefaultConfig() map[string]interface{} {
	return map[string]interface{}{
		"checks": []string{
			"assign",
			"atomic",
			"bools",
			"buildtag",
			"cgocall",
			"composites",
			"copylocks",
			"errorsas",
			"httpresponse",
			"loopclosure",
			"lostcancel",
			"nilfunc",
			"printf",
			"shift",
			"stdmethods",
			"structtag",
			"tests",
			"unmarshal",
			"unreachable",
			"unsafeptr",
			"unusedresult",
		},
	}
}

// writeGoFiles writes Go source files to the temporary directory
func (r *GoVetRunner) writeGoFiles(tempDir string, artifacts []verifier.Artifact) ([]string, error) {
	var goFiles []string

	for _, artifact := range artifacts {
		if !r.isGoFile(artifact) {
			continue
		}

		// Create the file path in temp directory
		filePath := filepath.Join(tempDir, filepath.Base(artifact.Path))

		// Ensure unique names to avoid conflicts
		if _, err := os.Stat(filePath); err == nil {
			base := strings.TrimSuffix(filepath.Base(artifact.Path), ".go")
			filePath = filepath.Join(tempDir, fmt.Sprintf("%s_%d.go", base, len(goFiles)))
		}

		// Write the file
		if err := os.WriteFile(filePath, []byte(artifact.Content), 0644); err != nil {
			return nil, fmt.Errorf("failed to write file %s: %w", filePath, err)
		}

		goFiles = append(goFiles, filePath)
	}

	return goFiles, nil
}

// isGoFile checks if the artifact is a Go source file
func (r *GoVetRunner) isGoFile(artifact verifier.Artifact) bool {
	return artifact.Language == "go" &&
		   artifact.Type == verifier.ArtifactTypeSource &&
		   strings.HasSuffix(artifact.Path, ".go") &&
		   artifact.Content != ""
}

// ensureGoMod creates a minimal go.mod file if it doesn't exist
func (r *GoVetRunner) ensureGoMod(tempDir string) error {
	goModPath := filepath.Join(tempDir, "go.mod")

	if _, err := os.Stat(goModPath); os.IsNotExist(err) {
		goModContent := `module temp/verification

go 1.21
`
		return os.WriteFile(goModPath, []byte(goModContent), 0644)
	}

	return nil
}

// parseGoVetOutput parses the JSON output from go vet
func (r *GoVetRunner) parseGoVetOutput(output, tempDir string) ([]verifier.Issue, error) {
	var issues []verifier.Issue

	if strings.TrimSpace(output) == "" {
		return issues, nil
	}

	// go vet outputs one JSON object per line
	lines := strings.Split(output, "\n")

	for i, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		var vetIssue GoVetIssue
		if err := json.Unmarshal([]byte(line), &vetIssue); err != nil {
			// If it's not JSON, treat as stderr message
			if strings.Contains(line, "vet:") {
				issues = append(issues, verifier.Issue{
					ID:          fmt.Sprintf("go-vet-%d", i),
					Type:        verifier.IssueTypeSyntax,
					Severity:    verifier.SeverityError,
					Title:       "Go vet error",
					Description: line,
				})
			}
			continue
		}

		// Convert to our issue format
		issue := verifier.Issue{
			ID:          fmt.Sprintf("go-vet-%s-%d", vetIssue.Category, i),
			Type:        r.mapVetCategoryToIssueType(vetIssue.Category),
			Severity:    verifier.SeverityWarning, // go vet issues are typically warnings
			Title:       fmt.Sprintf("Go vet: %s", vetIssue.Category),
			Description: vetIssue.Text,
			File:        r.relativePath(vetIssue.Posn.Filename, tempDir),
			Line:        vetIssue.Posn.Line,
			Column:      vetIssue.Posn.Column,
			Rule:        vetIssue.Category,
			Category:    "static-analysis",
		}

		issues = append(issues, issue)
	}

	return issues, nil
}

// GoVetIssue represents a go vet issue in JSON format
type GoVetIssue struct {
	Category string `json:"category"`
	Text     string `json:"text"`
	Posn     struct {
		Filename string `json:"filename"`
		Line     int    `json:"line"`
		Column   int    `json:"column"`
	} `json:"posn"`
}

// mapVetCategoryToIssueType maps go vet categories to our issue types
func (r *GoVetRunner) mapVetCategoryToIssueType(category string) verifier.IssueType {
	switch category {
	case "printf", "structtag", "tests":
		return verifier.IssueTypeSyntax
	case "assign", "atomic", "copylocks", "loopclosure", "lostcancel":
		return verifier.IssueTypeSemantic
	case "bools", "buildtag", "nilfunc", "unreachable":
		return verifier.IssueTypeMaintenance
	case "cgocall", "unsafeptr":
		return verifier.IssueTypeSecurity
	case "httpresponse", "errorsas", "unmarshal", "unusedresult":
		return verifier.IssueTypePerformance
	default:
		return verifier.IssueTypeSemantic
	}
}

// relativePath converts absolute path to relative path from tempDir
func (r *GoVetRunner) relativePath(filePath, tempDir string) string {
	if rel, err := filepath.Rel(tempDir, filePath); err == nil {
		return rel
	}
	return filePath
}

// mapPathsToOriginal maps temp file paths back to original artifact paths
func (r *GoVetRunner) mapPathsToOriginal(issues []verifier.Issue, tempDir string, artifacts []verifier.Artifact) []verifier.Issue {
	// Create mapping from temp file names to original paths
	nameToPath := make(map[string]string)
	for _, artifact := range artifacts {
		if r.isGoFile(artifact) {
			tempName := filepath.Base(artifact.Path)
			nameToPath[tempName] = artifact.Path
		}
	}

	// Update issue file paths
	for i := range issues {
		tempName := filepath.Base(issues[i].File)
		if originalPath, exists := nameToPath[tempName]; exists {
			issues[i].File = originalPath
		}
	}

	return issues
}