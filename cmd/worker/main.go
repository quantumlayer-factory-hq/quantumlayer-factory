package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/workflows"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/pkg/observability"
)

func main() {
	ctx := context.Background()

	// Initialize observability
	obsConfig := &observability.ObservabilityConfig{
		ServiceName:        "qlf-worker",
		ServiceVersion:     "1.0.0",
		Environment:        "development",
		TracingEnabled:     true,
		JaegerEndpoint:     "http://localhost:14268/api/traces",
		TraceRatio:         1.0,
		MetricsEnabled:     true,
		PrometheusEndpoint: "http://localhost:9090",
	}

	// Start tracing
	tracingService, err := observability.NewTracingService(obsConfig)
	if err != nil {
		log.Printf("Failed to initialize tracing: %v", err)
	} else {
		defer tracingService.Shutdown(ctx)
		log.Println("✅ Observability: Tracing initialized")
	}

	// Start metrics
	metricsService, err := observability.NewMetricsService(obsConfig)
	if err != nil {
		log.Printf("Failed to initialize metrics: %v", err)
	} else {
		defer metricsService.Shutdown(ctx)
		log.Println("✅ Observability: Metrics initialized")
	}

	// Start health checks
	healthService := observability.NewHealthService(obsConfig, metricsService)
	if err := healthService.Start(ctx); err != nil {
		log.Printf("Failed to start health service: %v", err)
	} else {
		defer healthService.Stop()
		log.Println("✅ Observability: Health checks initialized")
	}

	// Start health check HTTP server
	http.HandleFunc("/health", healthService.HTTPHandler())
	http.HandleFunc("/health/readiness", healthService.HTTPHandler())
	http.HandleFunc("/health/liveness", healthService.HTTPHandler())

	go func() {
		log.Println("🏥 Health check server starting on :8091")
		if err := http.ListenAndServe(":8091", nil); err != nil {
			log.Printf("Health server error: %v", err)
		}
	}()

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
		time.Sleep(100 * time.Millisecond) // Give observability time to flush
		os.Exit(0)
	}()

	// Start worker (blocks until interrupted)
	if err := workflows.StartWorker(config); err != nil {
		log.Fatalf("Failed to start worker: %v", err)
	}
}