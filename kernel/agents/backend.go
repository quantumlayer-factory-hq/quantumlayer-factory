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
		MaxTokens:    8192, // Large enough for backend code
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
	socParser := soc.NewParser(nil) // Allow all file paths
	patch, err := socParser.Parse(response.Content)
	if err != nil {
		return fmt.Errorf("SOC validation failed: %w", err)
	}
	if !patch.Valid {
		return fmt.Errorf("LLM generated invalid SOC format: %v", patch.Errors)
	}

	// Convert SOC patch to generated files
	fmt.Printf("[SOC] Patch valid=%v, files=%d, content=%d bytes\n",
		patch.Valid, len(patch.Files), len(patch.Content))
	files, err := a.convertSOCPatchToFiles(patch, req.Spec)
	if err != nil {
		return fmt.Errorf("failed to parse generated code: %w", err)
	}
	fmt.Printf("[SOC] Converted to %d generated files\n", len(files))

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

	for _, filePath := range patch.Files {
		content, exists := fileContents[filePath]
		if !exists {
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

// extractFilesFromDiff extracts file contents from unified diff format
func (a *BackendAgent) extractFilesFromDiff(diffContent string) (map[string]string, error) {
	files := make(map[string]string)
	currentFile := ""
	var currentContent strings.Builder

	lines := strings.Split(diffContent, "\n")
	for _, line := range lines {
		if strings.HasPrefix(line, "--- a/") {
			// Save previous file if exists
			if currentFile != "" {
				files[currentFile] = currentContent.String()
			}
			// Start new file
			currentFile = strings.TrimPrefix(line, "--- a/")
			currentContent.Reset()
		} else if strings.HasPrefix(line, "+") && !strings.HasPrefix(line, "+++") {
			// Add content line (remove + prefix)
			currentContent.WriteString(strings.TrimPrefix(line, "+"))
			currentContent.WriteString("\n")
		}
	}

	// Save last file
	if currentFile != "" {
		files[currentFile] = currentContent.String()
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
	// TODO: Implement Go backend generation
	result.Warnings = append(result.Warnings, "Go backend generation not yet implemented")
	return nil
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