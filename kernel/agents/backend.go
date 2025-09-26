package agents

import (
	"context"
	"fmt"
	"strings"
	"time"
	"math"

	"go.temporal.io/sdk/activity"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/llm"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/prompts"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/soc"
)

// BackendAgent specializes in generating backend code
type BackendAgent struct {
	*BaseAgent
	llmClient    llm.Client
	promptComposer *prompts.PromptComposer
}

// NewBackendAgent creates a new backend agent
func NewBackendAgent() *BackendAgent {
	capabilities := []string{
		"api_controllers",
		"service_layer",
		"data_models",
		"middleware",
		"authentication",
		"database_migrations",
		"error_handling",
		"logging",
		"configuration",
	}

	return &BackendAgent{
		BaseAgent: NewBaseAgent(AgentTypeBackend, "1.0.0", capabilities),
	}
}

// NewBackendAgentWithLLM creates a new backend agent with LLM capabilities
func NewBackendAgentWithLLM(llmClient llm.Client, promptComposer *prompts.PromptComposer) *BackendAgent {
	capabilities := []string{
		"api_controllers",
		"service_layer",
		"data_models",
		"middleware",
		"authentication",
		"database_migrations",
		"error_handling",
		"logging",
		"configuration",
	}

	return &BackendAgent{
		BaseAgent:      NewBaseAgent(AgentTypeBackend, "2.0.0", capabilities),
		llmClient:      llmClient,
		promptComposer: promptComposer,
	}
}

// CanHandle determines if this agent can handle the given specification
func (a *BackendAgent) CanHandle(spec *ir.IRSpec) bool {
	// Backend agent can handle API and web applications
	if spec.App.Type == "api" || spec.App.Type == "web" {
		return true
	}

	// Can handle if there are API endpoints defined
	if len(spec.API.Endpoints) > 0 {
		return true
	}

	// Can handle if there are backend-related features
	for _, feature := range spec.App.Features {
		if feature.Type == "auth" || feature.Type == "crud" || feature.Type == "payment" {
			return true
		}
	}

	return false
}

// Generate creates backend code from the specification
func (a *BackendAgent) Generate(ctx context.Context, req *GenerationRequest) (*GenerationResult, error) {
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

	// Determine the backend technology stack
	backend := req.Spec.App.Stack.Backend
	if backend.Language == "" {
		backend.Language = "python"
		backend.Framework = "fastapi"
	}

	// Generate based on LLM or fallback to templates
	if a.llmClient != nil && a.promptComposer != nil {
		err := a.generateWithLLM(ctx, req, result)
		if err != nil {
			return nil, fmt.Errorf("failed to generate backend with LLM: %w", err)
		}
	} else {
		// Fallback to template-based generation
		switch backend.Language {
		case "python":
			err := a.generatePythonBackend(req, result)
			if err != nil {
				return nil, fmt.Errorf("failed to generate Python backend: %w", err)
			}
		case "go":
			err := a.generateGoBackend(req, result)
			if err != nil {
				return nil, fmt.Errorf("failed to generate Go backend: %w", err)
			}
		case "nodejs":
			err := a.generateNodeBackend(req, result)
			if err != nil {
				return nil, fmt.Errorf("failed to generate Node.js backend: %w", err)
			}
		default:
			return nil, fmt.Errorf("unsupported backend language: %s", backend.Language)
		}
	}

	// Update metadata
	result.Metadata.Duration = time.Since(startTime)
	result.Metadata.FilesCreated = len(result.Files)
	result.Metadata.LinesOfCode = a.countLinesOfCode(result.Files)

	return result, nil
}

// Validate checks if the generated backend code meets requirements
func (a *BackendAgent) Validate(ctx context.Context, result *GenerationResult) (*ValidationResult, error) {
	validation := &ValidationResult{
		Valid: true,
		Metrics: ValidationMetrics{
			LinesOfCode: a.countLinesOfCode(result.Files),
			CodeQuality: 0.8, // Default quality score
		},
	}

	// Validate each generated file
	for _, file := range result.Files {
		err := a.validateFile(file, validation)
		if err != nil {
			validation.Errors = append(validation.Errors, ValidationError{
				File:     file.Path,
				Message:  err.Error(),
				Type:     "validation",
				Severity: "error",
			})
			validation.Valid = false
		}
	}

	return validation, nil
}

// generateWithLLM generates backend code using LLM
func (a *BackendAgent) generateWithLLM(ctx context.Context, req *GenerationRequest, result *GenerationResult) error {
	// Build prompt using the prompt composer
	composeReq := prompts.ComposeRequest{
		AgentType:   "backend",
		IRSpec:      req.Spec,
		Context:     req.Context,
	}

	promptResult, err := a.promptComposer.ComposePrompt(composeReq)
	if err != nil {
		return fmt.Errorf("failed to compose prompt: %w", err)
	}

	// Select model based on task complexity
	model := a.selectModel(req.Spec)

	// Record heartbeat before LLM call
	if activity.HasHeartbeatDetails(ctx) {
		activity.RecordHeartbeat(ctx, "Calling LLM for backend code generation...")
	}

	// Call LLM with retry logic for rate limits
	llmReq := &llm.GenerateRequest{
		Prompt:       promptResult.Prompt,
		Model:        model,
		MaxTokens:    32768, // Large enough for all 9 FastAPI files
		Temperature:  0.2,  // Low temperature for consistent code generation
	}

	response, err := a.callLLMWithRetry(ctx, llmReq)

	// Record heartbeat after LLM call
	if activity.HasHeartbeatDetails(ctx) {
		activity.RecordHeartbeat(ctx, "LLM generation completed, validating SOC format...")
	}
	if err != nil {
		return fmt.Errorf("LLM generation failed: %w", err)
	}

	// Log LLM response summary for debugging
	fmt.Printf("[LLM] Response received: %d chars, provider=%s, model=%s, tokens=%d/%d\n",
		len(response.Content), response.Provider, response.Model,
		response.Usage.PromptTokens, response.Usage.CompletionTokens)

	// Temporarily log raw response for SOC debugging
	fmt.Printf("=== RAW LLM RESPONSE START ===\n%s\n=== RAW LLM RESPONSE END ===\n", response.Content)

	// Handle truncated responses from max_tokens limit
	content := response.Content
	if !strings.HasSuffix(strings.TrimSpace(content), "### END") {
		// Response appears truncated (missing ### END), attempting repair...
		content = strings.TrimSpace(content) + "\n### END"
	}

	// Try SOC parsing first, fallback to direct diff parsing
	var files []GeneratedFile
	var parseErr error

	// Filter out prose contamination before SOC validation
	content = a.filterProseFromSOC(content)

	socParser := soc.NewParser(nil) // Allow all file paths
	patch, err := socParser.Parse(content)
	if err != nil || !patch.Valid {
		// SOC parsing failed, try direct diff parsing
		fmt.Printf("[BACKEND] SOC parsing failed (err=%v, valid=%v), attempting direct diff parsing...\n", err, patch != nil && patch.Valid)

		diffFiles, diffErr := a.extractFilesFromDiff(response.Content)
		if diffErr != nil || len(diffFiles) == 0 {
			// Both approaches failed
			socErr := fmt.Sprintf("SOC validation failed: %v", err)
			if patch != nil && !patch.Valid {
				socErr = fmt.Sprintf("LLM generated invalid SOC format: %v", patch.Errors)
			}
			return fmt.Errorf("%s; diff parsing also failed: %v", socErr, diffErr)
		}

		// Convert diff files to GeneratedFile format
		files = a.convertDiffFilesToGenerated(diffFiles, req.Spec)
		fmt.Printf("[BACKEND] Successfully parsed %d files using direct diff parsing\n", len(files))
	} else {
		// SOC parsing succeeded
		// Convert SOC patch to generated files
		files, parseErr = a.convertSOCPatchToFiles(patch, req.Spec)
		if parseErr != nil {
			return fmt.Errorf("failed to parse generated code: %w", parseErr)
		}
		fmt.Printf("[BACKEND] Successfully parsed %d files using SOC format\n", len(files))
	}
	// Converted to %d generated files
	_ = len(files)

	// Post-process files to extract any leftover diff content
	files = a.postProcessFiles(files)
	fmt.Printf("[BACKEND] After post-processing: %d files\n", len(files))

	// Add generated files to result
	result.Files = append(result.Files, files...)

	// Add generation metadata
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

// selectModel chooses the appropriate LLM model based on task complexity
func (a *BackendAgent) selectModel(spec *ir.IRSpec) llm.Model {
	// Calculate complexity score
	complexity := 0
	complexity += len(spec.API.Endpoints) * 2
	complexity += len(spec.Data.Entities) * 3
	complexity += len(spec.App.Features) * 2
	complexity += len(spec.Data.Relationships) * 1

	// Select model based on complexity
	if complexity < 10 {
		// Simple backend - use fast model
		return llm.ModelClaudeHaiku
	} else if complexity < 25 {
		// Medium complexity - use balanced model
		return llm.ModelClaudeSonnet
	} else {
		// Complex backend - use advanced model
		return llm.ModelClaude37
	}
}

// convertSOCPatchToFiles converts SOC patch to GeneratedFile format
func (a *BackendAgent) convertSOCPatchToFiles(patch *soc.Patch, spec *ir.IRSpec) ([]GeneratedFile, error) {
	files := []GeneratedFile{}

	// Extract files from SOC patch content
	// SOC patch.Content contains the unified diff
	// We need to extract the actual file content from the diff
	fileContents, err := a.extractFilesFromDiff(patch.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract files from SOC patch: %w", err)
	}
	// Extracted files from diff content
	_ = len(fileContents)

	// Process all files from the SOC patch
	for _, filePath := range patch.Files {
		content, exists := fileContents[filePath]
		if !exists || content == "" {
			continue
		}

		file := GeneratedFile{
			Path:     filePath,
			Type:     a.determineFileType(filePath),
			Language: a.detectLanguage(filePath),
			Template: "soc_llm_generated",
			Content:  content,
		}
		files = append(files, file)
	}

	return files, nil
}

// convertDiffFilesToGenerated converts extracted diff files to GeneratedFile format
func (a *BackendAgent) convertDiffFilesToGenerated(diffFiles map[string]string, spec *ir.IRSpec) []GeneratedFile {
	files := []GeneratedFile{}

	for filePath, content := range diffFiles {
		if content == "" {
			continue
		}

		file := GeneratedFile{
			Path:     filePath,
			Type:     a.determineFileType(filePath),
			Language: a.detectLanguage(filePath),
			Template: "diff_llm_generated",
			Content:  content,
		}
		files = append(files, file)
	}

	return files
}

// postProcessFiles cleans up generated files by extracting leftover diff content
func (a *BackendAgent) postProcessFiles(files []GeneratedFile) []GeneratedFile {
	processedFiles := make([]GeneratedFile, 0, len(files))
	additionalFiles := make([]GeneratedFile, 0)

	for _, file := range files {
		// Check if file content contains leftover diff markers
		if strings.Contains(file.Content, "\n--- a/") {
			cleanContent, extractedFiles := a.extractLeftoverDiffs(file.Content)

			// Update the original file with clean content
			cleanFile := file
			cleanFile.Content = cleanContent
			processedFiles = append(processedFiles, cleanFile)

			// Add extracted files
			for filename, content := range extractedFiles {
				if content != "" {
					extractedFile := GeneratedFile{
						Path:     filename,
						Type:     a.determineFileType(filename),
						Language: a.detectLanguage(filename),
						Template: "post_processed_diff",
						Content:  content,
					}
					additionalFiles = append(additionalFiles, extractedFile)
				}
			}
		} else {
			// File is clean, keep as-is
			processedFiles = append(processedFiles, file)
		}
	}

	// Combine processed and additional files, avoiding duplicates
	allFiles := a.deduplicateFiles(append(processedFiles, additionalFiles...))

	if len(additionalFiles) > 0 {
		fmt.Printf("[BACKEND] Post-processing extracted %d additional files from leftover diff content\n", len(additionalFiles))
	}

	return allFiles
}

// extractLeftoverDiffs separates clean content from leftover diff sections
func (a *BackendAgent) extractLeftoverDiffs(content string) (string, map[string]string) {
	extractedFiles := make(map[string]string)

	// Find the first diff marker
	diffStart := strings.Index(content, "\n--- a/")
	if diffStart == -1 {
		return content, extractedFiles
	}

	// Split content: clean part + diff part
	cleanContent := strings.TrimSpace(content[:diffStart])
	diffContent := content[diffStart+1:] // Skip the leading \n

	// Extract files from the diff content
	diffFiles, err := a.extractFilesFromDiff(diffContent)
	if err != nil {
		fmt.Printf("[BACKEND] Warning: failed to extract leftover diff content: %v\n", err)
		return content, extractedFiles // Return original if extraction fails
	}

	// Merge extracted files
	for filename, fileContent := range diffFiles {
		extractedFiles[filename] = fileContent
	}

	return cleanContent, extractedFiles
}

// deduplicateFiles removes duplicate files, preferring the last occurrence (most complete)
func (a *BackendAgent) deduplicateFiles(files []GeneratedFile) []GeneratedFile {
	fileMap := make(map[string]GeneratedFile)
	var result []GeneratedFile

	// Build map to detect duplicates (last wins)
	for _, file := range files {
		fileMap[file.Path] = file
	}

	// Preserve original order where possible
	seen := make(map[string]bool)
	for _, file := range files {
		if !seen[file.Path] {
			result = append(result, fileMap[file.Path])
			seen[file.Path] = true
		}
	}

	if len(result) < len(files) {
		fmt.Printf("[BACKEND] Deduplication removed %d duplicate files\n", len(files)-len(result))
	}

	return result
}

// attemptDiffRepair repairs common diff format issues that LLMs create
func (a *BackendAgent) attemptDiffRepair(diffContent string) string {
	lines := strings.Split(diffContent, "\n")
	var repairedLines []string

	for i, line := range lines {
		// Add missing +++ header after --- header
		if strings.HasPrefix(line, "--- a/") && i+1 < len(lines) {
			repairedLines = append(repairedLines, line)
			nextLine := lines[i+1]
			if !strings.HasPrefix(nextLine, "+++ b/") {
				// Extract filename and add missing +++ header
				filename := strings.TrimPrefix(line, "--- a/")
				repairedLines = append(repairedLines, "+++ b/"+filename)
			}
			continue
		}

		// Add missing + prefix to code lines that look like content
		if !strings.HasPrefix(line, "---") && !strings.HasPrefix(line, "+++") &&
		   !strings.HasPrefix(line, "@@") && !strings.HasPrefix(line, "+") &&
		   !strings.HasPrefix(line, "-") && line != "" {
			// Heuristic: if it looks like code, add + prefix
			if a.looksLikeCode(line) {
				repairedLines = append(repairedLines, "+"+line)
				continue
			}
		}

		repairedLines = append(repairedLines, line)
	}

	return strings.Join(repairedLines, "\n")
}

// looksLikeCode uses heuristics to determine if a line looks like code content
func (a *BackendAgent) looksLikeCode(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}

	// Common code patterns
	codePatterns := []string{
		"import ", "from ", "def ", "class ", "function ", "const ", "let ", "var ",
		"if ", "for ", "while ", "try ", "catch ", "throw ", "return ", "export ",
		"interface ", "type ", "enum ", "async ", "await ", "public ", "private ",
		"protected ", "static ", "final ", "abstract ", "extends ", "implements ",
		"package ", "namespace ", "using ", "include ", "#include", "require(",
		"module.exports", "export default", "export {", "import {",
	}

	for _, pattern := range codePatterns {
		if strings.HasPrefix(line, pattern) {
			return true
		}
	}

	// Check for common code structures
	if strings.Contains(line, "() {") || strings.Contains(line, "): ") ||
	   strings.Contains(line, " = ") || strings.Contains(line, " => ") ||
	   strings.Contains(line, "://") || strings.HasSuffix(line, ";") ||
	   strings.HasSuffix(line, "{") || strings.HasSuffix(line, "}") ||
	   strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") {
		return true
	}

	return false
}

// extractFilesFromDiff extracts file contents from unified diff format (robust version)
func (a *BackendAgent) extractFilesFromDiff(diffContent string) (map[string]string, error) {
	// First attempt repair of common diff format issues
	diffContent = a.attemptDiffRepair(diffContent)

	files := make(map[string]string)
	currentFile := ""
	var currentContent strings.Builder
	inFileContent := false

	// Strip any prose before/after diff content
	if start := strings.Index(diffContent, "--- a/"); start > 0 {
		diffContent = diffContent[start:]
	}
	if start := strings.Index(diffContent, "diff --git"); start >= 0 && start < strings.Index(diffContent, "--- a/") {
		diffContent = diffContent[start:]
	}

	lines := strings.Split(diffContent, "\n")
	for i, line := range lines {
		line = strings.TrimSpace(line)

		// Detect new file start (more robust)
		if strings.HasPrefix(line, "--- a/") || strings.HasPrefix(line, "--- /dev/null") {
			// Save previous file if exists and has content
			if currentFile != "" && currentContent.Len() > 0 {
				content := strings.TrimSpace(currentContent.String())
				if content != "" {
					files[currentFile] = content
				}
			}

			// Extract filename from --- line
			if strings.HasPrefix(line, "--- a/") {
				currentFile = strings.TrimSpace(strings.TrimPrefix(line, "--- a/"))
			}
			currentContent.Reset()
			inFileContent = false
			continue
		}

		// Handle +++ line (use it to confirm/update filename)
		if strings.HasPrefix(line, "+++ b/") {
			newFile := strings.TrimSpace(strings.TrimPrefix(line, "+++ b/"))
			if newFile != "" && newFile != "/dev/null" {
				currentFile = newFile
			}
			inFileContent = true
			continue
		}

		// Skip hunk headers and git metadata
		if strings.HasPrefix(line, "@@") ||
		   strings.HasPrefix(line, "diff --git") ||
		   strings.HasPrefix(line, "index ") {
			inFileContent = true
			continue
		}

		// Collect file content
		if currentFile != "" {
			// Handle added lines in diff
			if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
				content := strings.TrimPrefix(line, "+")
				currentContent.WriteString(content)
				currentContent.WriteString("\n")
				inFileContent = true
			} else if !strings.HasPrefix(line, "-") &&
					  !strings.HasPrefix(line, "\\") &&
					  !strings.HasPrefix(line, "diff") &&
					  !strings.HasPrefix(line, "index") {
				// Context line or raw content (when LLM forgets + prefix)
				// Only add if we're clearly in file content or line looks like code
				if inFileContent ||
				   strings.Contains(line, "import ") ||
				   strings.Contains(line, "from ") ||
				   strings.Contains(line, "def ") ||
				   strings.Contains(line, "class ") ||
				   strings.Contains(line, "function ") ||
				   (i > 0 && currentContent.Len() > 0) {
					currentContent.WriteString(line)
					currentContent.WriteString("\n")
				}
			}
		}
	}

	// Save last file
	if currentFile != "" && currentContent.Len() > 0 {
		content := strings.TrimSpace(currentContent.String())
		if content != "" {
			files[currentFile] = content
		}
	}

	// Validate we extracted at least one file
	if len(files) == 0 {
		return nil, fmt.Errorf("no files could be extracted from diff content")
	}

	return files, nil
}

// parseGeneratedCode parses LLM output into individual files (legacy fallback)
func (a *BackendAgent) parseGeneratedCode(content string, spec *ir.IRSpec) ([]GeneratedFile, error) {
	files := []GeneratedFile{}

	// Look for code blocks marked with file paths
	// Expected format: ```filename:path/to/file.ext
	lines := strings.Split(content, "\n")
	var currentFile *GeneratedFile
	var codeLines []string

	for _, line := range lines {
		// Check for file delimiter
		if strings.HasPrefix(line, "```") {
			if currentFile != nil {
				// End of current file
				currentFile.Content = strings.Join(codeLines, "\n")
				files = append(files, *currentFile)
				currentFile = nil
				codeLines = []string{}
			} else if strings.Contains(line, ":") {
				// Start of new file
				parts := strings.SplitN(line[3:], ":", 2)
				if len(parts) == 2 {
					language := strings.TrimSpace(parts[0])
					path := strings.TrimSpace(parts[1])

					currentFile = &GeneratedFile{
						Path:     path,
						Type:     a.determineFileType(path),
						Language: language,
						Template: "llm_generated",
					}
				}
			}
		} else if currentFile != nil {
			// Add line to current file
			codeLines = append(codeLines, line)
		}
	}

	// Handle last file if not closed
	if currentFile != nil {
		currentFile.Content = strings.Join(codeLines, "\n")
		files = append(files, *currentFile)
	}

	// If no structured files found, create a single main file
	if len(files) == 0 {
		backend := spec.App.Stack.Backend
		ext := a.getFileExtension(backend.Language)
		mainFile := GeneratedFile{
			Path:     fmt.Sprintf("main%s", ext),
			Type:     "source",
			Language: backend.Language,
			Template: "llm_generated",
			Content:  content,
		}
		files = append(files, mainFile)
	}

	return files, nil
}

// determineFileType determines the file type based on file extension
func (a *BackendAgent) determineFileType(path string) string {
	if strings.HasSuffix(path, ".txt") || strings.HasSuffix(path, ".md") {
		return "config"
	}
	if strings.HasSuffix(path, ".json") || strings.HasSuffix(path, ".yaml") || strings.HasSuffix(path, ".yml") {
		return "config"
	}
	if strings.HasSuffix(path, ".dockerfile") || strings.HasPrefix(path, "Dockerfile") {
		return "config"
	}
	return "source"
}

// detectLanguage detects programming language from file extension
func (a *BackendAgent) detectLanguage(filePath string) string {
	if strings.HasSuffix(filePath, ".py") {
		return "python"
	}
	if strings.HasSuffix(filePath, ".go") {
		return "go"
	}
	if strings.HasSuffix(filePath, ".js") || strings.HasSuffix(filePath, ".ts") {
		return "javascript"
	}
	if strings.HasSuffix(filePath, ".sql") {
		return "sql"
	}
	return "text"
}

// callLLMWithRetry calls LLM with exponential backoff retry logic
func (a *BackendAgent) callLLMWithRetry(ctx context.Context, req *llm.GenerateRequest) (*llm.GenerateResponse, error) {
	maxRetries := 3
	baseDelay := time.Second

	for attempt := 0; attempt <= maxRetries; attempt++ {
		// Record heartbeat for retry attempts
		if activity.HasHeartbeatDetails(ctx) {
			if attempt == 0 {
				activity.RecordHeartbeat(ctx, "Calling LLM...")
			} else {
				activity.RecordHeartbeat(ctx, fmt.Sprintf("Retrying LLM call (attempt %d/%d)...", attempt+1, maxRetries+1))
			}
		}

		fmt.Printf("[LLM] Request: model=%s, prompt=%d chars, max_tokens=%d, temp=%.1f\n",
			req.Model, len(req.Prompt), req.MaxTokens, req.Temperature)

		response, err := a.llmClient.Generate(ctx, req)
		if err == nil {
			fmt.Printf("[LLM] Success: %d tokens response\n", response.Usage.CompletionTokens)
			return response, nil
		}

		fmt.Printf("[LLM] Error: %v\n", err)

		// Check if this is a rate limit error
		if strings.Contains(err.Error(), "RATE_LIMIT") || strings.Contains(err.Error(), "rate limit") {
			if attempt < maxRetries {
				// Calculate exponential backoff delay
				delay := time.Duration(float64(baseDelay) * math.Pow(2, float64(attempt)))

				// Sleep with periodic heartbeats to prevent timeout
				if activity.HasHeartbeatDetails(ctx) {
					activity.RecordHeartbeat(ctx, fmt.Sprintf("Rate limited, waiting %v before retry...", delay))
				}

				// Sleep in 5-second intervals with heartbeats
				for elapsed := time.Duration(0); elapsed < delay; elapsed += 5*time.Second {
					select {
					case <-ctx.Done():
						return nil, ctx.Err()
					case <-time.After(min(5*time.Second, delay-elapsed)):
						if activity.HasHeartbeatDetails(ctx) && elapsed+5*time.Second < delay {
							remaining := delay - elapsed - 5*time.Second
							activity.RecordHeartbeat(ctx, fmt.Sprintf("Rate limited, %v remaining...", remaining))
						}
					}
				}
				continue
			}
		}

		// For non-rate-limit errors or max retries exceeded, return the error
		return nil, err
	}

	return nil, fmt.Errorf("max retries exceeded")
}

// min returns the smaller of two durations
func min(a, b time.Duration) time.Duration {
	if a < b {
		return a
	}
	return b
}

// getFileExtension returns the appropriate file extension for a language
func (a *BackendAgent) getFileExtension(language string) string {
	switch language {
	case "python":
		return ".py"
	case "go":
		return ".go"
	case "nodejs", "javascript":
		return ".js"
	case "typescript":
		return ".ts"
	case "java":
		return ".java"
	case "csharp":
		return ".cs"
	default:
		return ".txt"
	}
}

// generatePythonBackend generates Python backend code
func (a *BackendAgent) generatePythonBackend(req *GenerationRequest, result *GenerationResult) error {
	framework := req.Spec.App.Stack.Backend.Framework
	if framework == "" {
		framework = "fastapi"
	}

	switch framework {
	case "fastapi":
		return a.generateFastAPIBackend(req, result)
	case "django":
		return a.generateDjangoBackend(req, result)
	case "flask":
		return a.generateFlaskBackend(req, result)
	default:
		return fmt.Errorf("unsupported Python framework: %s", framework)
	}
}

// generateFastAPIBackend generates FastAPI-specific code
func (a *BackendAgent) generateFastAPIBackend(req *GenerationRequest, result *GenerationResult) error {
	// Generate main application file
	mainFile := GeneratedFile{
		Path:     "main.py",
		Type:     "source",
		Language: "python",
		Template: "fastapi_main",
		Content:  a.generateFastAPIMain(req.Spec),
	}
	result.Files = append(result.Files, mainFile)

	// Generate models
	if len(req.Spec.Data.Entities) > 0 {
		modelsFile := GeneratedFile{
			Path:     "models.py",
			Type:     "source",
			Language: "python",
			Template: "fastapi_models",
			Content:  a.generateFastAPIModels(req.Spec),
		}
		result.Files = append(result.Files, modelsFile)
	}

	// Generate API routers
	if len(req.Spec.API.Endpoints) > 0 {
		routersFile := GeneratedFile{
			Path:     "routers.py",
			Type:     "source",
			Language: "python",
			Template: "fastapi_routers",
			Content:  a.generateFastAPIRouters(req.Spec),
		}
		result.Files = append(result.Files, routersFile)
	}

	// Generate requirements.txt
	reqFile := GeneratedFile{
		Path:     "requirements.txt",
		Type:     "config",
		Language: "text",
		Template: "python_requirements",
		Content:  a.generatePythonRequirements(req.Spec),
	}
	result.Files = append(result.Files, reqFile)

	return nil
}

// generateFastAPIMain generates the main FastAPI application file
func (a *BackendAgent) generateFastAPIMain(spec *ir.IRSpec) string {
	return fmt.Sprintf(`"""
%s - FastAPI Application

Generated by QuantumLayer Factory
"""

from fastapi import FastAPI, HTTPException
from fastapi.middleware.cors import CORSMiddleware
import uvicorn

app = FastAPI(
    title="%s",
    description="%s",
    version="1.0.0"
)

# Configure CORS
app.add_middleware(
    CORSMiddleware,
    allow_origins=["*"],
    allow_credentials=True,
    allow_methods=["*"],
    allow_headers=["*"],
)

@app.get("/")
async def root():
    return {"message": "Welcome to %s API"}

@app.get("/health")
async def health_check():
    return {"status": "healthy"}

if __name__ == "__main__":
    uvicorn.run(app, host="0.0.0.0", port=8000)
`, spec.App.Name, spec.App.Name, spec.App.Description, spec.App.Name)
}

// generateFastAPIModels generates Pydantic models for FastAPI
func (a *BackendAgent) generateFastAPIModels(spec *ir.IRSpec) string {
	var models strings.Builder

	models.WriteString(`"""
Data Models

Generated by QuantumLayer Factory
"""

from pydantic import BaseModel, Field
from typing import Optional, List
from datetime import datetime
import uuid

`)

	for _, entity := range spec.Data.Entities {
		models.WriteString(fmt.Sprintf("class %s(BaseModel):\n", entity.Name))
		for _, field := range entity.Fields {
			pythonType := a.convertToPythonType(field.Type)
			fieldDef := fmt.Sprintf("    %s: %s", field.Name, pythonType)

			if !field.Required {
				fieldDef = fmt.Sprintf("    %s: Optional[%s] = None", field.Name, pythonType)
			}

			if field.Default != "" {
				fieldDef += fmt.Sprintf(" = %s", field.Default)
			}

			models.WriteString(fieldDef + "\n")
		}
		models.WriteString("\n")
	}

	return models.String()
}

// generateFastAPIRouters generates API route handlers
func (a *BackendAgent) generateFastAPIRouters(spec *ir.IRSpec) string {
	var routers strings.Builder

	routers.WriteString(`"""
API Routers

Generated by QuantumLayer Factory
"""

from fastapi import APIRouter, HTTPException, Depends
from typing import List
import uuid

router = APIRouter()

`)

	for _, endpoint := range spec.API.Endpoints {
		method := strings.ToLower(endpoint.Method)
		funcName := a.generateFunctionName(endpoint.Path, endpoint.Method)

		routers.WriteString(fmt.Sprintf(`@router.%s("%s")
async def %s():
    """
    %s
    """
    # TODO: Implement %s logic
    return {"message": "Not implemented"}

`, method, endpoint.Path, funcName, endpoint.Summary, endpoint.Summary))
	}

	return routers.String()
}

// generatePythonRequirements generates requirements.txt for Python
func (a *BackendAgent) generatePythonRequirements(spec *ir.IRSpec) string {
	requirements := []string{
		"fastapi>=0.104.1",
		"uvicorn[standard]>=0.24.0",
		"pydantic>=2.4.2",
	}

	// Add database dependencies
	dbType := spec.App.Stack.Database.Type
	switch dbType {
	case "postgresql":
		requirements = append(requirements, "asyncpg>=0.29.0", "databases[postgresql]>=0.8.0")
	case "mysql":
		requirements = append(requirements, "aiomysql>=0.2.0", "databases[mysql]>=0.8.0")
	case "sqlite":
		requirements = append(requirements, "databases[sqlite]>=0.8.0")
	case "mongodb":
		requirements = append(requirements, "motor>=3.3.2")
	}

	// Add authentication dependencies
	for _, feature := range spec.App.Features {
		if feature.Type == "auth" {
			requirements = append(requirements, "python-jose[cryptography]>=3.3.0", "passlib[bcrypt]>=1.7.4")
			break
		}
	}

	return strings.Join(requirements, "\n") + "\n"
}

// generateGoBackend generates Go backend code
func (a *BackendAgent) generateGoBackend(req *GenerationRequest, result *GenerationResult) error {
	framework := req.Spec.App.Stack.Backend.Framework
	if framework == "" {
		framework = "gin"
	}

	switch framework {
	case "gin":
		return a.generateGinBackend(req, result)
	case "echo":
		return a.generateEchoBackend(req, result)
	case "fiber":
		return a.generateFiberBackend(req, result)
	default:
		return fmt.Errorf("unsupported Go framework: %s", framework)
	}
}

// generateNodeBackend generates Node.js backend code
func (a *BackendAgent) generateNodeBackend(req *GenerationRequest, result *GenerationResult) error {
	// TODO: Implement Node.js backend generation
	result.Warnings = append(result.Warnings, "Node.js backend generation not yet implemented")
	return nil
}

// generateDjangoBackend generates Django-specific code
func (a *BackendAgent) generateDjangoBackend(req *GenerationRequest, result *GenerationResult) error {
	// TODO: Implement Django backend generation
	result.Warnings = append(result.Warnings, "Django backend generation not yet implemented")
	return nil
}

// generateFlaskBackend generates Flask-specific code
func (a *BackendAgent) generateFlaskBackend(req *GenerationRequest, result *GenerationResult) error {
	// TODO: Implement Flask backend generation
	result.Warnings = append(result.Warnings, "Flask backend generation not yet implemented")
	return nil
}

// Helper functions

func (a *BackendAgent) convertToPythonType(fieldType string) string {
	switch fieldType {
	case "uuid":
		return "str"
	case "string":
		return "str"
	case "integer", "int":
		return "int"
	case "decimal", "float":
		return "float"
	case "boolean", "bool":
		return "bool"
	case "timestamp", "datetime":
		return "datetime"
	case "date":
		return "datetime"
	case "text":
		return "str"
	default:
		return "str"
	}
}

func (a *BackendAgent) generateFunctionName(path, method string) string {
	// Convert path like "/users/{id}" and method "GET" to "get_user_by_id"
	cleanPath := strings.ReplaceAll(path, "/", "_")
	cleanPath = strings.ReplaceAll(cleanPath, "{", "")
	cleanPath = strings.ReplaceAll(cleanPath, "}", "")
	cleanPath = strings.Trim(cleanPath, "_")

	if cleanPath == "" {
		cleanPath = "root"
	}

	return strings.ToLower(method) + "_" + strings.ToLower(cleanPath)
}

func (a *BackendAgent) countLinesOfCode(files []GeneratedFile) int {
	total := 0
	for _, file := range files {
		if file.Type == "source" {
			lines := strings.Split(file.Content, "\n")
			total += len(lines)
		}
	}
	return total
}

func (a *BackendAgent) validateFile(file GeneratedFile, validation *ValidationResult) error {
	// Basic validation - check if file has content
	if len(file.Content) == 0 {
		return fmt.Errorf("file %s is empty", file.Path)
	}

	// Language-specific validation
	switch file.Language {
	case "python":
		return a.validatePythonFile(file, validation)
	case "go":
		return a.validateGoFile(file, validation)
	default:
		// No specific validation for this language
		return nil
	}
}

func (a *BackendAgent) validatePythonFile(file GeneratedFile, validation *ValidationResult) error {
	// Basic Python syntax checks
	if !strings.Contains(file.Content, "def ") && !strings.Contains(file.Content, "class ") {
		validation.Warnings = append(validation.Warnings,
			fmt.Sprintf("File %s may not contain valid Python code", file.Path))
	}
	return nil
}

func (a *BackendAgent) validateGoFile(file GeneratedFile, validation *ValidationResult) error {
	// Basic Go syntax checks
	if !strings.Contains(file.Content, "package ") {
		return fmt.Errorf("Go file %s missing package declaration", file.Path)
	}
	return nil
}

// filterProseFromSOC removes conversational prose from LLM responses before SOC validation
func (a *BackendAgent) filterProseFromSOC(content string) string {
	lines := strings.Split(content, "\n")
	var filteredLines []string
	inDiff := false
	inRawDiff := false
	fileListDone := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)

		// Track if we're inside a diff block (```diff format)
		if strings.HasPrefix(trimmed, "```diff") {
			inDiff = true
			fileListDone = true
			filteredLines = append(filteredLines, line)
			continue
		}
		if strings.HasPrefix(trimmed, "```") && inDiff {
			inDiff = false
			filteredLines = append(filteredLines, line)
			continue
		}

		// Track if we're in raw diff format (--- a/ format)
		if strings.HasPrefix(trimmed, "--- a/") || strings.HasPrefix(trimmed, "+++ b/") {
			inRawDiff = true
			fileListDone = true
			filteredLines = append(filteredLines, line)
			continue
		}

		// Keep SOC structure lines
		if trimmed == "### FACTORY/1 PATCH" ||
		   trimmed == "### END" ||
		   strings.HasPrefix(trimmed, "- file:") {
			filteredLines = append(filteredLines, line)
			continue
		}

		// Keep diff content when inside any diff block
		if inDiff || inRawDiff {
			filteredLines = append(filteredLines, line)
			continue
		}

		// After file list is done, skip empty lines until we hit diff content
		if fileListDone && trimmed == "" {
			continue
		}

		// Filter out conversational lines outside diff blocks
		lowerLine := strings.ToLower(trimmed)
		if strings.HasPrefix(lowerLine, "human:") ||
		   strings.HasPrefix(lowerLine, "assistant:") ||
		   strings.Contains(lowerLine, "thank you") ||
		   strings.Contains(lowerLine, "you're welcome") ||
		   strings.Contains(lowerLine, "i'll help") ||
		   strings.Contains(lowerLine, "let me") ||
		   strings.Contains(lowerLine, "here's") ||
		   strings.Contains(lowerLine, "sure, i") ||
		   strings.Contains(lowerLine, "of course") {
			// Filtering prose line
			continue
		}

		// Convert prose-like lines inside diff blocks to proper comments
		if (inDiff || inRawDiff) && strings.HasPrefix(line, "+") {
			content := strings.TrimPrefix(line, "+")
			content = strings.TrimSpace(content)

			// Check if it looks like prose (contains spaces and common English words)
			if !strings.HasPrefix(content, "#") &&
			   !strings.HasPrefix(content, "//") &&
			   !strings.HasPrefix(content, "--") &&
			   strings.Contains(content, " ") &&
			   (strings.Contains(lowerLine, "get the current user") ||
			    strings.Contains(lowerLine, "return the") ||
			    strings.Contains(lowerLine, "based on") ||
			    strings.Contains(lowerLine, "jwt token")) {

				// Detect language from context to use proper comment syntax
				commentPrefix := a.detectCommentPrefix(strings.Join(lines, "\n"))
				filteredLines = append(filteredLines, "+    " + commentPrefix + " " + content)
				continue
			}
		}

		// Keep other content (including empty lines before file list completion)
		if !fileListDone || trimmed != "" {
			filteredLines = append(filteredLines, line)
		}
	}

	result := strings.Join(filteredLines, "\n")
	// Prose filtering completed
	_ = len(lines)
	_ = len(filteredLines)
	return result
}

// detectCommentPrefix detects the appropriate comment prefix for the current language context
func (a *BackendAgent) detectCommentPrefix(content string) string {
	// Extract file paths from the entire SOC content to detect language
	// Look at the full SOC content to determine the primary language being generated
	if strings.Contains(content, "main.go") || strings.Contains(content, ".go") {
		return "//"
	}
	if strings.Contains(content, "main.py") || strings.Contains(content, ".py") {
		return "#"
	}
	if strings.Contains(content, ".js") || strings.Contains(content, ".ts") {
		return "//"
	}
	if strings.Contains(content, ".sql") {
		return "--"
	}
	// Default to Python-style comments if unsure
	return "#"
}

// generateGinBackend generates Gin-specific Go backend code
func (a *BackendAgent) generateGinBackend(req *GenerationRequest, result *GenerationResult) error {
	// Generate main application file
	mainFile := GeneratedFile{
		Path:     "main.go",
		Type:     "source",
		Language: "go",
		Template: "gin_main",
		Content:  a.generateGinMain(req.Spec),
	}
	result.Files = append(result.Files, mainFile)

	// Generate models
	if len(req.Spec.Data.Entities) > 0 {
		modelsFile := GeneratedFile{
			Path:     "models/models.go",
			Type:     "source",
			Language: "go",
			Template: "gin_models",
			Content:  a.generateGinModels(req.Spec),
		}
		result.Files = append(result.Files, modelsFile)
	}

	// Generate API handlers
	if len(req.Spec.API.Endpoints) > 0 {
		handlersFile := GeneratedFile{
			Path:     "handlers/handlers.go",
			Type:     "source",
			Language: "go",
			Template: "gin_handlers",
			Content:  a.generateGinHandlers(req.Spec),
		}
		result.Files = append(result.Files, handlersFile)
	}

	// Generate go.mod
	goModFile := GeneratedFile{
		Path:     "go.mod",
		Type:     "config",
		Language: "text",
		Template: "go_mod",
		Content:  a.generateGoMod(req.Spec),
	}
	result.Files = append(result.Files, goModFile)

	// Generate README
	readmeFile := GeneratedFile{
		Path:     "README.md",
		Type:     "documentation",
		Language: "markdown",
		Template: "gin_readme",
		Content:  a.generateGinReadme(req.Spec),
	}
	result.Files = append(result.Files, readmeFile)

	return nil
}

// generateEchoBackend generates Echo-specific Go backend code
func (a *BackendAgent) generateEchoBackend(req *GenerationRequest, result *GenerationResult) error {
	result.Warnings = append(result.Warnings, "Echo backend generation not yet implemented")
	return nil
}

// generateFiberBackend generates Fiber-specific Go backend code
func (a *BackendAgent) generateFiberBackend(req *GenerationRequest, result *GenerationResult) error {
	result.Warnings = append(result.Warnings, "Fiber backend generation not yet implemented")
	return nil
}

// generateGinMain generates the main Go application file for Gin
func (a *BackendAgent) generateGinMain(spec *ir.IRSpec) string {
	return fmt.Sprintf(`package main

import (
	"net/http"
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/gin-contrib/cors"
)

func main() {
	// Create Gin router
	r := gin.Default()

	// Add CORS middleware
	config := cors.DefaultConfig()
	config.AllowAllOrigins = true
	config.AllowHeaders = []string{"Origin", "Content-Length", "Content-Type", "Authorization"}
	r.Use(cors.New(config))

	// Health check endpoint
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status":  "healthy",
			"service": "%s",
			"version": "1.0.0",
		})
	})

	// API routes
	api := r.Group("/api/v1")
	{
		// TODO: Add your API routes here
		api.GET("/", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{
				"message": "%s API",
				"version": "1.0.0",
			})
		})
	}

	// Get port from environment or default to 8080
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting %s on port %%s", port)
	log.Fatal(r.Run(":" + port))
}
`, spec.App.Name, spec.App.Name, spec.App.Name)
}

// generateGinModels generates Go struct models for Gin
func (a *BackendAgent) generateGinModels(spec *ir.IRSpec) string {
	var models strings.Builder

	models.WriteString(`package models

import (
	"time"
	"gorm.io/gorm"
)

`)

	for _, entity := range spec.Data.Entities {
		models.WriteString(fmt.Sprintf("type %s struct {\n", entity.Name))
		models.WriteString("\tgorm.Model\n")

		for _, field := range entity.Fields {
			goType := a.convertToGoType(field.Type)
			tag := fmt.Sprintf("`json:\"%s\" gorm:\"%s\"`", field.Name, field.Name)

			if !field.Required && goType != "string" {
				goType = "*" + goType
			}

			models.WriteString(fmt.Sprintf("\t%s %s %s\n",
				strings.Title(field.Name), goType, tag))
		}
		models.WriteString("}\n\n")
	}

	return models.String()
}

// generateGinHandlers generates API handlers for Gin
func (a *BackendAgent) generateGinHandlers(spec *ir.IRSpec) string {
	var handlers strings.Builder

	handlers.WriteString(`package handlers

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

`)

	// Generate CRUD handlers for each entity
	for _, entity := range spec.Data.Entities {
		entityLower := strings.ToLower(entity.Name)

		handlers.WriteString(fmt.Sprintf("// Get%s handles GET /%s/:id\nfunc Get%s(c *gin.Context) {\n\tid := c.Param(\"id\")\n\t// TODO: Implement get %s by ID\n\tc.JSON(http.StatusOK, gin.H{\n\t\t\"id\": id,\n\t\t\"message\": \"Get %s by ID\",\n\t})\n}\n\n", entity.Name, entityLower, entity.Name, entityLower, entity.Name))
		handlers.WriteString(fmt.Sprintf("// List%s handles GET /%s\nfunc List%s(c *gin.Context) {\n\t// TODO: Implement list %s\n\tc.JSON(http.StatusOK, gin.H{\n\t\t\"data\": []interface{}{},\n\t\t\"message\": \"List %s\",\n\t})\n}\n\n", entity.Name, entityLower, entity.Name, entityLower, entity.Name))
		handlers.WriteString(fmt.Sprintf("// Create%s handles POST /%s\nfunc Create%s(c *gin.Context) {\n\t// TODO: Implement create %s\n\tc.JSON(http.StatusCreated, gin.H{\n\t\t\"message\": \"Create %s\",\n\t})\n}\n\n", entity.Name, entityLower, entity.Name, entityLower, entity.Name))
		handlers.WriteString(fmt.Sprintf("// Update%s handles PUT /%s/:id\nfunc Update%s(c *gin.Context) {\n\tid := c.Param(\"id\")\n\t// TODO: Implement update %s\n\tc.JSON(http.StatusOK, gin.H{\n\t\t\"id\": id,\n\t\t\"message\": \"Update %s\",\n\t})\n}\n\n", entity.Name, entityLower, entity.Name, entityLower, entity.Name))
		handlers.WriteString(fmt.Sprintf("// Delete%s handles DELETE /%s/:id\nfunc Delete%s(c *gin.Context) {\n\tid := c.Param(\"id\")\n\t// TODO: Implement delete %s\n\tc.JSON(http.StatusOK, gin.H{\n\t\t\"id\": id,\n\t\t\"message\": \"Delete %s\",\n\t})\n}\n\n", entity.Name, entityLower, entity.Name, entityLower, entity.Name))
	}

	return handlers.String()
}

// generateGoMod generates go.mod file
func (a *BackendAgent) generateGoMod(spec *ir.IRSpec) string {
	moduleName := strings.ToLower(strings.ReplaceAll(spec.App.Name, " ", "-"))
	return fmt.Sprintf("module %s\n\ngo 1.21\n\nrequire (\n\tgithub.com/gin-contrib/cors v1.4.0\n\tgithub.com/gin-gonic/gin v1.9.1\n\tgorm.io/gorm v1.25.4\n\tgorm.io/driver/postgres v1.5.2\n)\n", moduleName)
}

// generateGinReadme generates README.md for Gin backend
func (a *BackendAgent) generateGinReadme(spec *ir.IRSpec) string {
	return fmt.Sprintf("# %s\n\n%s\n\n## Getting Started\n\n### Prerequisites\n\n- Go 1.21 or higher\n- PostgreSQL (optional)\n\n### Installation\n\n1. Install dependencies:\n   ```bash\n   go mod download\n   ```\n\n2. Run the application:\n   ```bash\n   go run main.go\n   ```\n\nThe server will start on http://localhost:8080\n\n### API Endpoints\n\n- `GET /health` - Health check\n- `GET /api/v1/` - API information\n\n### Environment Variables\n\n- `PORT` - Server port (default: 8080)\n- `DATABASE_URL` - PostgreSQL connection string (optional)\n\n## Generated by QuantumLayer Factory\n\nThis backend was automatically generated using QuantumLayer Factory.\n", spec.App.Name, spec.App.Description)
}

// convertToGoType converts IR types to Go types
func (a *BackendAgent) convertToGoType(irType string) string {
	switch irType {
	case "string", "text":
		return "string"
	case "int", "integer":
		return "int"
	case "float", "number":
		return "float64"
	case "bool", "boolean":
		return "bool"
	case "datetime", "timestamp":
		return "time.Time"
	case "uuid":
		return "string"
	case "email":
		return "string"
	case "url":
		return "string"
	case "json":
		return "interface{}"
	default:
		return "string"
	}
}