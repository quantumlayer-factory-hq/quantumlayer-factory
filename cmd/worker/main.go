package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/workflows"
)

func main() {
	config := workflows.DefaultWorkerConfig()

	// Override from environment if provided
	if addr := os.Getenv("TEMPORAL_ADDRESS"); addr != "" {
		config.TemporalAddress = addr
	}
	if queue := os.Getenv("TASK_QUEUE"); queue != "" {
		config.TaskQueue = queue
	}

	log.Printf("Starting QuantumLayer Factory worker...")
	log.Printf("Temporal Address: %s", config.TemporalAddress)
	log.Printf("Task Queue: %s", config.TaskQueue)

	// Handle shutdown signals
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)

	go func() {
		<-c
		log.Println("Shutting down worker...")
		os.Exit(0)
	}()

	// Start worker (blocks until interrupted)
	if err := workflows.StartWorker(config); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}
}