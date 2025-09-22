package workflows

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.temporal.io/sdk/activity"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/agents"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
)

// ParallelGenerateCodeActivity generates code using multiple agents in parallel
func ParallelGenerateCodeActivity(ctx context.Context, irSpec *ir.IRSpec, overlays []string, config map[string]interface{}, provider, model string) (CodeGenerationResult, error) {

	result := CodeGenerationResult{
		Success:       true,
		GeneratedCode: make(map[string]string),
		Artifacts:     []string{},
		Errors:        []string{},
		Warnings:      []string{},
	}

	// Create agent factory (LLM-enabled if provider specified)
	factory, err := createLLMEnabledFactory(provider, model)
	if err != nil {
		result.Success = false
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to create agent factory: %v", err))
		return result, nil
	}

	// Plan which agents need to be executed based on the IR spec
	agentTasks := planAgentExecution(irSpec)
	if len(agentTasks) == 0 {
		result.Warnings = append(result.Warnings, "No agents required for this specification")
		return result, nil
	}

	// Execute agents in parallel
	activity.RecordHeartbeat(ctx, fmt.Sprintf("Starting parallel execution of %d agents...", len(agentTasks)))

	var wg sync.WaitGroup
	var mu sync.Mutex
	agentResults := make([]AgentExecutionResult, len(agentTasks))

	for i, task := range agentTasks {
		wg.Add(1)
		go func(index int, task AgentTask) {
			defer wg.Done()

			// Execute the agent
			agentResult := executeAgentTask(ctx, factory, task, irSpec, overlays)

			// Store result safely
			mu.Lock()
			agentResults[index] = agentResult
			mu.Unlock()

			// Report progress
			activity.RecordHeartbeat(ctx, fmt.Sprintf("Completed %s agent execution", task.AgentType))
		}(i, task)
	}

	// Wait for all agents to complete
	wg.Wait()

	// Aggregate results
	activity.RecordHeartbeat(ctx, "Aggregating results from parallel agent execution...")
	aggregateAgentResults(&result, agentResults)

	return result, nil
}

// AgentTask represents a task for a specific agent
type AgentTask struct {
	AgentType agents.AgentType
	Target    agents.GenerationTarget
	Priority  int // Lower numbers = higher priority
	Required  bool // If true, failure stops the workflow
}

// AgentExecutionResult represents the result of executing a single agent
type AgentExecutionResult struct {
	AgentType agents.AgentType
	Success   bool
	Files     []agents.GeneratedFile
	Errors    []string
	Warnings  []string
	Duration  time.Duration
}

// planAgentExecution determines which agents need to run based on the IR spec
func planAgentExecution(irSpec *ir.IRSpec) []AgentTask {
	var tasks []AgentTask

	// Backend agent - highest priority, required
	if irSpec.App.Stack.Backend.Language != "" {
		tasks = append(tasks, AgentTask{
			AgentType: agents.AgentTypeBackend,
			Target: agents.GenerationTarget{
				Type:      "backend",
				Language:  irSpec.App.Stack.Backend.Language,
				Framework: irSpec.App.Stack.Backend.Framework,
			},
			Priority: 1,
			Required: true,
		})
	}

	// Frontend agent - can run in parallel with backend
	if irSpec.App.Stack.Frontend.Framework != "" {
		tasks = append(tasks, AgentTask{
			AgentType: agents.AgentTypeFrontend,
			Target: agents.GenerationTarget{
				Type:      "frontend",
				Language:  "javascript", // Default, could be determined from spec
				Framework: irSpec.App.Stack.Frontend.Framework,
			},
			Priority: 2,
			Required: false, // Frontend is optional
		})
	}

	// Database agent - can run in parallel with backend and frontend
	if irSpec.App.Stack.Database.Type != "" {
		tasks = append(tasks, AgentTask{
			AgentType: agents.AgentTypeDatabase,
			Target: agents.GenerationTarget{
				Type:     "database",
				Language: "sql",
				Framework: irSpec.App.Stack.Database.Type,
			},
			Priority: 2,
			Required: false,
		})
	}

	// API agent - can run in parallel if separate from backend
	if irSpec.App.Type == "api" && len(irSpec.API.Endpoints) > 0 {
		tasks = append(tasks, AgentTask{
			AgentType: agents.AgentTypeAPI,
			Target: agents.GenerationTarget{
				Type:      "api",
				Language:  irSpec.App.Stack.Backend.Language,
				Framework: irSpec.App.Stack.Backend.Framework,
			},
			Priority: 3,
			Required: false,
		})
	}

	// Test agent - lowest priority, runs after main components
	tasks = append(tasks, AgentTask{
		AgentType: agents.AgentTypeTest,
		Target: agents.GenerationTarget{
			Type:      "test",
			Language:  irSpec.App.Stack.Backend.Language,
			Framework: "testing",
		},
		Priority: 4,
		Required: false,
	})

	return tasks
}

// executeAgentTask executes a single agent task
func executeAgentTask(ctx context.Context, factory *agents.AgentFactory, task AgentTask, irSpec *ir.IRSpec, overlays []string) AgentExecutionResult {
	startTime := time.Now()

	result := AgentExecutionResult{
		AgentType: task.AgentType,
		Success:   false,
	}

	// Create the agent
	agent, err := factory.CreateAgent(task.AgentType)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to create %s agent: %v", task.AgentType, err))
		result.Duration = time.Since(startTime)
		return result
	}

	// Create generation request
	request := &agents.GenerationRequest{
		Spec:   irSpec,
		Target: task.Target,
		Options: agents.GenerationOptions{
			CreateDirectories: true,
			FormatCode:       true,
			ValidateOutput:   true,
		},
		Context: map[string]interface{}{
			"workflow": "factory",
			"overlays": overlays,
			"parallel": true,
		},
	}

	// Execute the agent
	output, err := agent.Generate(ctx, request)
	result.Duration = time.Since(startTime)

	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("%s generation failed: %v", task.AgentType, err))
		return result
	}

	if !output.Success {
		result.Errors = append(result.Errors, output.Errors...)
		result.Warnings = append(result.Warnings, output.Warnings...)
		return result
	}

	// Success case
	result.Success = true
	result.Files = output.Files
	result.Warnings = append(result.Warnings, output.Warnings...)

	return result
}

// aggregateAgentResults combines results from all agents
func aggregateAgentResults(finalResult *CodeGenerationResult, agentResults []AgentExecutionResult) {
	var totalErrors []string
	var totalWarnings []string
	hasRequiredFailures := false

	for _, agentResult := range agentResults {
		if !agentResult.Success {
			// Check if this was a required agent (backend is typically required)
			if agentResult.AgentType == agents.AgentTypeBackend {
				hasRequiredFailures = true
			}
			totalErrors = append(totalErrors, agentResult.Errors...)
		} else {
			// Add generated files to final result
			for _, file := range agentResult.Files {
				finalResult.GeneratedCode[file.Path] = file.Content
				finalResult.Artifacts = append(finalResult.Artifacts, file.Path)
			}
		}
		totalWarnings = append(totalWarnings, agentResult.Warnings...)
	}

	// Set final success status
	if hasRequiredFailures {
		finalResult.Success = false
		finalResult.Errors = append(finalResult.Errors, "One or more required agents failed")
	}

	finalResult.Errors = append(finalResult.Errors, totalErrors...)
	finalResult.Warnings = append(finalResult.Warnings, totalWarnings...)
}

// ParallelGenerationConfig allows configuring parallel execution
type ParallelGenerationConfig struct {
	MaxConcurrentAgents int           `json:"max_concurrent_agents"`
	AgentTimeout        time.Duration `json:"agent_timeout"`
	RequiredAgents      []agents.AgentType `json:"required_agents"`
	EnableParallel      bool          `json:"enable_parallel"`
}

// GetDefaultParallelConfig returns sensible defaults for parallel execution
func GetDefaultParallelConfig() ParallelGenerationConfig {
	return ParallelGenerationConfig{
		MaxConcurrentAgents: 4, // Backend, Frontend, Database, API
		AgentTimeout:        5 * time.Minute,
		RequiredAgents:      []agents.AgentType{agents.AgentTypeBackend},
		EnableParallel:      true,
	}
}