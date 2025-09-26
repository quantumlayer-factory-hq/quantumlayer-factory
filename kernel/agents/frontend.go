package agents

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/llm"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/prompts"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/soc"
)

// FrontendAgent specializes in generating frontend code
type FrontendAgent struct {
	*BaseAgent
	llmClient      llm.Client
	promptComposer *prompts.PromptComposer
}

// NewFrontendAgent creates a new frontend agent
func NewFrontendAgent() *FrontendAgent {
	capabilities := []string{
		"react_components",
		"vue_components",
		"angular_components",
		"ui_layouts",
		"routing",
		"state_management",
		"styling",
		"forms",
		"authentication_ui",
	}

	return &FrontendAgent{
		BaseAgent: NewBaseAgent(AgentTypeFrontend, "1.0.0", capabilities),
	}
}

// NewFrontendAgentWithLLM creates a new frontend agent with LLM capabilities
func NewFrontendAgentWithLLM(llmClient llm.Client, promptComposer *prompts.PromptComposer) *FrontendAgent {
	fmt.Printf("[FRONTEND] Creating FrontendAgent with LLM - Client: %v, Composer: %v\n",
		llmClient != nil, promptComposer != nil)

	capabilities := []string{
		"react_components",
		"vue_components",
		"angular_components",
		"ui_layouts",
		"routing",
		"state_management",
		"styling",
		"forms",
		"authentication_ui",
	}

	agent := &FrontendAgent{
		BaseAgent:      NewBaseAgent(AgentTypeFrontend, "2.0.0", capabilities),
		llmClient:      llmClient,
		promptComposer: promptComposer,
	}

	// Debug: verify the fields were set correctly
	fmt.Printf("[FRONTEND] Created agent - LLMClient: %v, PromptComposer: %v\n",
		agent.llmClient != nil, agent.promptComposer != nil)

	return agent
}

// CanHandle determines if this agent can handle the given specification
func (a *FrontendAgent) CanHandle(spec *ir.IRSpec) bool {
	// Frontend agent can handle web applications with UI
	if spec.App.Type == "web" && len(spec.UI.Pages) > 0 {
		return true
	}

	// Can handle single page applications
	if spec.App.Type == "spa" {
		return true
	}

	// Can handle if UI type is spa (even without explicit pages)
	if spec.UI.Type == "spa" {
		return true
	}

	// Can handle if frontend stack is defined and we have UI-related features
	if spec.App.Stack.Frontend.Framework != "" {
		// Check for UI-related features or explicit frontend mention
		for _, feature := range spec.App.Features {
			if strings.Contains(strings.ToLower(feature.Name), "ui") ||
			   strings.Contains(strings.ToLower(feature.Name), "frontend") ||
			   strings.Contains(strings.ToLower(feature.Name), "interface") ||
			   strings.Contains(strings.ToLower(feature.Name), "web") ||
			   feature.Type == "ui" || feature.Type == "frontend" {
				return true
			}
		}

		// If no backend features but frontend is defined, assume frontend-focused
		hasBackendFeatures := false
		for _, feature := range spec.App.Features {
			if feature.Type == "api" || feature.Type == "backend" ||
			   strings.Contains(strings.ToLower(feature.Name), "api") ||
			   strings.Contains(strings.ToLower(feature.Name), "backend") {
				hasBackendFeatures = true
				break
			}
		}

		if !hasBackendFeatures && len(spec.UI.Pages) > 0 {
			return true
		}
	}

	return false
}

// Generate creates frontend code from the specification
func (a *FrontendAgent) Generate(ctx context.Context, req *GenerationRequest) (*GenerationResult, error) {
	startTime := time.Now()

	// Debug logging to trace the LLM check
	fmt.Printf("[FRONTEND] Generate called - LLMClient: %v, PromptComposer: %v\n",
		a.llmClient != nil, a.promptComposer != nil)

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
		fmt.Printf("[FRONTEND] Using LLM generation path\n")
		err := a.generateWithLLM(ctx, req, result)
		if err != nil {
			return nil, fmt.Errorf("failed to generate frontend with LLM: %w", err)
		}
	} else {
		// Fallback for non-LLM mode
		fmt.Printf("[FRONTEND] Falling back to template mode - LLMClient: %v, PromptComposer: %v\n",
			a.llmClient != nil, a.promptComposer != nil)
		result.Warnings = []string{"Frontend agent running in template mode - limited functionality"}
	}

	// Update metadata
	result.Metadata.Duration = time.Since(startTime)
	result.Metadata.FilesCreated = len(result.Files)

	return result, nil
}

// generateWithLLM generates frontend code using LLM with robust parsing
func (a *FrontendAgent) generateWithLLM(ctx context.Context, req *GenerationRequest, result *GenerationResult) error {
	// Build prompt using the prompt composer
	composeReq := prompts.ComposeRequest{
		AgentType: "frontend",
		IRSpec:    req.Spec,
		Context:   req.Context,
	}

	promptResult, err := a.promptComposer.ComposePrompt(composeReq)
	if err != nil {
		return fmt.Errorf("failed to compose prompt: %w", err)
	}

	// Select model - frontend typically needs balanced capabilities
	model := llm.ModelClaudeSonnet

	// Call LLM to generate code
	llmReq := &llm.GenerateRequest{
		Prompt:      promptResult.Prompt,
		Model:       model,
		MaxTokens:   16384, // Increased for multiple React files
		Temperature: 0.2,
	}

	response, err := a.llmClient.Generate(ctx, llmReq)
	if err != nil {
		return fmt.Errorf("LLM generation failed: %w", err)
	}

	// Log LLM response summary for debugging
	fmt.Printf("[FRONTEND] Response received: %d chars, provider=%s, model=%s, tokens=%d/%d\n",
		len(response.Content), response.Provider, response.Model,
		response.Usage.PromptTokens, response.Usage.CompletionTokens)

	// Handle truncated responses from max_tokens limit
	content := response.Content
	if !strings.HasSuffix(strings.TrimSpace(content), "### END") {
		content = strings.TrimSpace(content) + "\n### END"
	}

	// Try SOC parsing first, fallback to robust diff parsing with all fallback mechanisms
	var files []GeneratedFile
	var parseErr error

	// Filter out prose contamination before SOC validation
	content = a.filterProseFromSOC(content)

	socParser := soc.NewParser(nil)
	patch, err := socParser.Parse(content)
	if err != nil || !patch.Valid {
		// SOC parsing failed, try ROBUST diff parsing with all fallback mechanisms
		fmt.Printf("[FRONTEND] SOC parsing failed (err=%v, valid=%v), attempting robust diff parsing...\n", err, patch != nil && patch.Valid)

		// Use the enhanced extractFilesFromDiff with repair and post-processing
		diffFiles, diffErr := a.extractFilesFromDiff(response.Content)
		if diffErr != nil || len(diffFiles) == 0 {
			// Both approaches failed
			socErr := fmt.Sprintf("SOC validation failed: %v", err)
			if patch != nil && !patch.Valid {
				socErr = fmt.Sprintf("LLM generated invalid SOC format: %v", patch.Errors)
			}
			return fmt.Errorf("%s; robust diff parsing also failed: %v", socErr, diffErr)
		}

		// Convert diff files to GeneratedFile format
		files = a.convertDiffFilesToGenerated(diffFiles, req.Spec)
		fmt.Printf("[FRONTEND] Successfully parsed %d files using robust diff parsing\n", len(files))
	} else {
		// SOC parsing succeeded
		files, parseErr = a.convertSOCPatchToFiles(patch, req.Spec)
		if parseErr != nil {
			return fmt.Errorf("failed to parse generated code: %w", parseErr)
		}
		fmt.Printf("[FRONTEND] Successfully parsed %d files using SOC format\n", len(files))
	}

	// Post-process files to extract any leftover diff content (CRITICAL FOR ROBUST PARSING)
	files = a.postProcessFiles(files)

	// Deduplicate files to prevent issues
	files = a.deduplicateFiles(files)

	fmt.Printf("[FRONTEND] After post-processing and deduplication: %d files\n", len(files))

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

// convertSOCPatchToFiles converts SOC patch to GeneratedFile format for frontend
func (a *FrontendAgent) convertSOCPatchToFiles(patch *soc.Patch, spec *ir.IRSpec) ([]GeneratedFile, error) {
	files := []GeneratedFile{}

	// Extract files from SOC patch content
	fileContents, err := a.extractFilesFromDiff(patch.Content)
	if err != nil {
		return nil, fmt.Errorf("failed to extract files from SOC patch: %w", err)
	}

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
			Template: "soc_frontend_generated",
			Content:  content,
		}
		files = append(files, file)
	}

	return files, nil
}

// convertDiffFilesToGenerated converts extracted diff files to GeneratedFile format for frontend
func (a *FrontendAgent) convertDiffFilesToGenerated(diffFiles map[string]string, spec *ir.IRSpec) []GeneratedFile {
	files := []GeneratedFile{}

	for filePath, content := range diffFiles {
		if content == "" {
			continue
		}

		file := GeneratedFile{
			Path:     filePath,
			Type:     a.determineFileType(filePath),
			Language: a.detectLanguage(filePath),
			Template: "diff_frontend_generated",
			Content:  content,
		}
		files = append(files, file)
	}

	return files
}

// postProcessFiles cleans up generated files by extracting leftover diff content (frontend version)
func (a *FrontendAgent) postProcessFiles(files []GeneratedFile) []GeneratedFile {
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
						Template: "post_processed_frontend",
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
		fmt.Printf("[FRONTEND] Post-processing extracted %d additional files from leftover diff content\n", len(additionalFiles))
	}

	return allFiles
}

// extractLeftoverDiffs separates clean content from leftover diff sections (frontend version)
func (a *FrontendAgent) extractLeftoverDiffs(content string) (string, map[string]string) {
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
		fmt.Printf("[FRONTEND] Warning: failed to extract leftover diff content: %v\n", err)
		return content, extractedFiles // Return original if extraction fails
	}

	// Merge extracted files
	for filename, fileContent := range diffFiles {
		extractedFiles[filename] = fileContent
	}

	return cleanContent, extractedFiles
}

// deduplicateFiles removes duplicate files, preferring the last occurrence (frontend version)
func (a *FrontendAgent) deduplicateFiles(files []GeneratedFile) []GeneratedFile {
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
		fmt.Printf("[FRONTEND] Deduplication removed %d duplicate files\n", len(files)-len(result))
	}

	return result
}

// extractFilesFromDiff extracts file contents from unified diff format (frontend version)
func (a *FrontendAgent) extractFilesFromDiff(diffContent string) (map[string]string, error) {
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
				// Only add if we're clearly in file content or line looks like frontend code
				if inFileContent || a.looksLikeFrontendCode(line) ||
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

// attemptDiffRepair repairs common diff format issues that LLMs create (frontend version)
func (a *FrontendAgent) attemptDiffRepair(diffContent string) string {
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
			// Heuristic: if it looks like frontend code, add + prefix
			if a.looksLikeFrontendCode(line) {
				repairedLines = append(repairedLines, "+"+line)
				continue
			}
		}

		repairedLines = append(repairedLines, line)
	}

	return strings.Join(repairedLines, "\n")
}

// looksLikeFrontendCode uses heuristics to determine if a line looks like frontend code
func (a *FrontendAgent) looksLikeFrontendCode(line string) bool {
	line = strings.TrimSpace(line)
	if line == "" {
		return false
	}

	// Common frontend code patterns
	frontendPatterns := []string{
		"import ", "export ", "from ", "const ", "let ", "var ", "function ",
		"class ", "interface ", "type ", "enum ", "async ", "await ",
		"React", "useState", "useEffect", "useCallback", "useMemo", "useContext",
		"Component", "render(", "return (", "props.", "state.",
		"<div", "<span", "<h1", "<h2", "<h3", "<p>", "<a ", "<img ", "<button",
		"onClick", "onChange", "onSubmit", "className", "style=",
		"npm ", "yarn ", "package.json", "webpack", "vite", "next",
		".jsx", ".tsx", ".vue", ".css", ".scss", ".less",
		"@media", "@keyframes", "flex", "grid", "display:",
		"margin:", "padding:", "width:", "height:", "color:",
	}

	for _, pattern := range frontendPatterns {
		if strings.Contains(line, pattern) {
			return true
		}
	}

	// Check for common frontend structures
	if strings.Contains(line, "() => ") || strings.Contains(line, "= () => ") ||
	   strings.Contains(line, "className=") || strings.Contains(line, "style={{") ||
	   strings.HasPrefix(line, "  ") || strings.HasPrefix(line, "\t") ||
	   strings.HasSuffix(line, ";") || strings.HasSuffix(line, ",") ||
	   strings.HasSuffix(line, "{") || strings.HasSuffix(line, "}") ||
	   strings.Contains(line, "/>") || strings.Contains(line, "</") {
		return true
	}

	return false
}

// filterProseFromSOC removes prose contamination from SOC content (frontend version)
func (a *FrontendAgent) filterProseFromSOC(content string) string {
	lines := strings.Split(content, "\n")
	var filteredLines []string
	inDiff := false
	inRawDiff := false
	fileListDone := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		lowerLine := strings.ToLower(line)

		// Track if we're in diff sections
		if strings.HasPrefix(trimmed, "```diff") || strings.HasPrefix(trimmed, "--- a/") {
			inDiff = true
			inRawDiff = strings.HasPrefix(trimmed, "--- a/")
		}
		if strings.HasPrefix(trimmed, "```") && inDiff && !strings.HasPrefix(trimmed, "```diff") {
			inDiff = false
			inRawDiff = false
		}

		// Mark file list as done when we see first diff content
		if (strings.HasPrefix(trimmed, "```diff") || strings.HasPrefix(trimmed, "--- a/")) && !fileListDone {
			fileListDone = true
		}

		// Skip prose lines that shouldn't be in SOC
		if !inDiff && !inRawDiff && fileListDone &&
		   (strings.Contains(lowerLine, "i'll create") ||
		    strings.Contains(lowerLine, "here's the") ||
		    strings.Contains(lowerLine, "let me") ||
		    strings.Contains(lowerLine, "this will") ||
		    strings.Contains(lowerLine, "the frontend")) {
			continue
		}

		// Keep other content (including empty lines before file list completion)
		if !fileListDone || trimmed != "" {
			filteredLines = append(filteredLines, line)
		}
	}

	result := strings.Join(filteredLines, "\n")
	return result
}

// determineFileType determines the type of a frontend file
func (a *FrontendAgent) determineFileType(filePath string) string {
	switch {
	case strings.HasSuffix(filePath, ".json"):
		return "config"
	case strings.HasSuffix(filePath, ".html"):
		return "template"
	case strings.HasSuffix(filePath, ".css") || strings.HasSuffix(filePath, ".scss") || strings.HasSuffix(filePath, ".less"):
		return "stylesheet"
	case strings.HasSuffix(filePath, ".js") || strings.HasSuffix(filePath, ".jsx") ||
		 strings.HasSuffix(filePath, ".ts") || strings.HasSuffix(filePath, ".tsx") ||
		 strings.HasSuffix(filePath, ".vue"):
		return "source"
	case strings.HasSuffix(filePath, ".md"):
		return "documentation"
	default:
		return "source"
	}
}

// detectLanguage detects the programming language from file path (frontend version)
func (a *FrontendAgent) detectLanguage(filePath string) string {
	switch {
	case strings.HasSuffix(filePath, ".js") || strings.HasSuffix(filePath, ".jsx"):
		return "javascript"
	case strings.HasSuffix(filePath, ".ts") || strings.HasSuffix(filePath, ".tsx"):
		return "typescript"
	case strings.HasSuffix(filePath, ".vue"):
		return "vue"
	case strings.HasSuffix(filePath, ".css"):
		return "css"
	case strings.HasSuffix(filePath, ".scss"):
		return "scss"
	case strings.HasSuffix(filePath, ".less"):
		return "less"
	case strings.HasSuffix(filePath, ".html"):
		return "html"
	case strings.HasSuffix(filePath, ".json"):
		return "json"
	case strings.HasSuffix(filePath, ".md"):
		return "markdown"
	default:
		return "javascript"
	}
}

// getFileExtension returns appropriate file extension for frontend framework
func (a *FrontendAgent) getFileExtension(framework string) string {
	switch framework {
	case "react":
		return ".jsx"
	case "vue":
		return ".vue"
	case "angular":
		return ".ts"
	default:
		return ".js"
	}
}

// getLanguage returns the primary language for the framework
func (a *FrontendAgent) getLanguage(framework string) string {
	switch framework {
	case "angular":
		return "typescript"
	case "react", "vue":
		return "javascript"
	default:
		return "javascript"
	}
}

// Validate checks if the generated frontend code meets requirements
func (a *FrontendAgent) Validate(ctx context.Context, result *GenerationResult) (*ValidationResult, error) {
	validation := &ValidationResult{
		Valid:    true,
		Warnings: []string{},
	}

	// Check if we have any files generated
	if len(result.Files) == 0 {
		validation.Valid = false
		validation.Warnings = append(validation.Warnings, "No frontend files were generated")
		return validation, nil
	}

	// Track required files for a complete frontend
	hasPackageJson := false
	hasMainComponent := false
	hasIndexFile := false
	fileTypeCount := make(map[string]int)

	for _, file := range result.Files {
		// Count file types
		fileTypeCount[file.Type]++

		// Check for critical files
		if file.Path == "package.json" {
			hasPackageJson = true
			// Validate package.json structure
			if !strings.Contains(file.Content, "\"name\":") {
				validation.Warnings = append(validation.Warnings, "package.json missing name field")
			}
			if !strings.Contains(file.Content, "\"scripts\":") {
				validation.Warnings = append(validation.Warnings, "package.json missing scripts section")
			}
		}

		if strings.Contains(file.Path, "App.") &&
		   (strings.HasSuffix(file.Path, ".jsx") || strings.HasSuffix(file.Path, ".tsx") || strings.HasSuffix(file.Path, ".vue")) {
			hasMainComponent = true
		}

		if strings.Contains(file.Path, "index.") &&
		   (strings.HasSuffix(file.Path, ".js") || strings.HasSuffix(file.Path, ".tsx") || strings.HasSuffix(file.Path, ".html")) {
			hasIndexFile = true
		}

		// Check for leftover diff markers (should be cleaned by post-processing)
		if strings.Contains(file.Content, "--- a/") || strings.Contains(file.Content, "+++ b/") {
			validation.Valid = false
			validation.Warnings = append(validation.Warnings, fmt.Sprintf("File %s contains leftover diff markers", file.Path))
		}

		// Check for placeholder content
		if strings.Contains(file.Content, "[COMPLETE RUNNABLE CODE HERE]") ||
		   strings.Contains(file.Content, "TODO: Implement") {
			validation.Warnings = append(validation.Warnings, fmt.Sprintf("File %s contains placeholder content", file.Path))
		}

		// Validate file content is not empty
		if strings.TrimSpace(file.Content) == "" {
			validation.Warnings = append(validation.Warnings, fmt.Sprintf("File %s is empty", file.Path))
		}
	}

	// Check for essential files
	if !hasPackageJson {
		validation.Warnings = append(validation.Warnings, "Missing package.json file - required for frontend projects")
	}
	if !hasMainComponent {
		validation.Warnings = append(validation.Warnings, "Missing main application component (App.jsx/tsx)")
	}
	if !hasIndexFile {
		validation.Warnings = append(validation.Warnings, "Missing index entry point file")
	}

	// Check file diversity
	if fileTypeCount["source"] == 0 {
		validation.Valid = false
		validation.Warnings = append(validation.Warnings, "No source files generated")
	}

	// Suggest improvements
	if fileTypeCount["stylesheet"] == 0 {
		validation.Warnings = append(validation.Warnings, "Consider adding CSS/SCSS files for styling")
	}

	fmt.Printf("[FRONTEND] Validation completed: %d files, %d warnings, valid=%v\n",
		len(result.Files), len(validation.Warnings), validation.Valid)

	return validation, nil
}