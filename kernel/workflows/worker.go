package workflows

import (
	"log"

	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

const (
	// TaskQueue defines the task queue name for the factory workflow
	TaskQueue = "factory-task-queue"

	// WorkflowID prefix for factory workflows
	WorkflowIDPrefix = "factory-workflow"
)

// WorkerConfig contains configuration for the Temporal worker
type WorkerConfig struct {
	TemporalAddress string
	TaskQueue       string
	MaxConcurrent   int
}

// DefaultWorkerConfig returns a default worker configuration
func DefaultWorkerConfig() WorkerConfig {
	return WorkerConfig{
		TemporalAddress: "localhost:7233",
		TaskQueue:       TaskQueue,
		MaxConcurrent:   10,
	}
}

// StartWorker starts a Temporal worker that can execute workflows and activities
func StartWorker(config WorkerConfig) error {
	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: config.TemporalAddress,
	})
	if err != nil {
		return err
	}
	defer c.Close()

	// Create worker
	w := worker.New(c, config.TaskQueue, worker.Options{
		MaxConcurrentActivityExecutionSize: config.MaxConcurrent,
		MaxConcurrentWorkflowTaskExecutionSize: config.MaxConcurrent,
	})

	// Register workflow
	w.RegisterWorkflow(FactoryWorkflow)

	// Register activities
	w.RegisterActivity(ParseBriefActivity)
	w.RegisterActivity(ValidateIRActivity)
	w.RegisterActivity(GenerateCodeActivity)
	w.RegisterActivity(VerifyCodeActivity)
	w.RegisterActivity(PackageArtifactsActivity)

	log.Printf("Starting Temporal worker on task queue: %s", config.TaskQueue)

	// Start listening to the task queue
	err = w.Run(worker.InterruptCh())
	if err != nil {
		return err
	}

	return nil
}

// CreateTemporalClient creates a new Temporal client
func CreateTemporalClient(address string) (client.Client, error) {
	return client.Dial(client.Options{
		HostPort: address,
	})
}