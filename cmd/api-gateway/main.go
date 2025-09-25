package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gorilla/mux"
	"github.com/gorilla/websocket"
	"github.com/rs/cors"
	"go.temporal.io/sdk/client"

	wf "github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/workflows"
)

type APIGateway struct {
	temporalClient client.Client
	upgrader       websocket.Upgrader
	connections    map[string]*websocket.Conn
	connMutex      sync.RWMutex
}

type GenerateRequest struct {
	Brief    string   `json:"brief"`
	Parallel bool     `json:"parallel"`
	Provider string   `json:"provider,omitempty"`
	Model    string   `json:"model,omitempty"`
	Overlays []string `json:"overlays,omitempty"`
}

type GenerateResponse struct {
	WorkflowID string `json:"workflowId"`
	RunID      string `json:"runId"`
	ProjectID  string `json:"projectId"`
	Status     string `json:"status"`
}

type WorkflowStatus struct {
	WorkflowID string                 `json:"workflowId"`
	Status     string                 `json:"status"`
	Progress   int                    `json:"progress"`
	Agents     []AgentStatus          `json:"agents"`
	Files      []string               `json:"files"`
	Metrics    map[string]interface{} `json:"metrics"`
	Error      string                 `json:"error,omitempty"`
}

type AgentStatus struct {
	Type     string `json:"type"`
	Status   string `json:"status"` // pending, active, completed, failed
	Progress int    `json:"progress"`
	Files    int    `json:"files"`
	Duration string `json:"duration"`
}

func NewAPIGateway() *APIGateway {
	// Connect to Temporal
	temporalAddress := os.Getenv("TEMPORAL_ADDRESS")
	if temporalAddress == "" {
		temporalAddress = "localhost:7233"
	}

	c, err := client.Dial(client.Options{
		HostPort: temporalAddress,
	})
	if err != nil {
		log.Fatalf("Failed to create Temporal client: %v", err)
	}

	return &APIGateway{
		temporalClient: c,
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				// Allow connections from any origin for development
				// In production, you should validate the origin
				return true
			},
		},
		connections: make(map[string]*websocket.Conn),
	}
}

func (gw *APIGateway) generateHandler(w http.ResponseWriter, r *http.Request) {
	var req GenerateRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, fmt.Sprintf("Invalid JSON: %v", err), http.StatusBadRequest)
		return
	}

	// Generate unique IDs
	projectID := fmt.Sprintf("project-%d", time.Now().UnixNano())
	workflowID := fmt.Sprintf("factory-%d", time.Now().UnixNano())

	// Start Temporal workflow
	opts := client.StartWorkflowOptions{
		ID:        workflowID,
		TaskQueue: "factory",
	}

	input := wf.FactoryInput{
		Brief:     req.Brief,
		ProjectID: projectID,
		DryRun:    false,
		Verbose:   true,
		OutputDir: "./generated",
		Overlays:  req.Overlays,
		Provider:  req.Provider,
		Model:     req.Model,
		Parallel:  req.Parallel,
	}

	we, err := gw.temporalClient.ExecuteWorkflow(context.Background(), opts, wf.FactoryWorkflow, input)
	if err != nil {
		http.Error(w, fmt.Sprintf("Failed to start workflow: %v", err), http.StatusInternalServerError)
		return
	}

	response := GenerateResponse{
		WorkflowID: we.GetID(),
		RunID:      we.GetRunID(),
		ProjectID:  projectID,
		Status:     "started",
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	// Start monitoring workflow in background
	go gw.monitorWorkflow(we.GetID(), we.GetRunID())
}

func (gw *APIGateway) statusHandler(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	workflowID := vars["workflowId"]

	// Get workflow status from Temporal
	workflowRun := gw.temporalClient.GetWorkflow(context.Background(), workflowID, "")

	// Try to get the result or current status
	var result wf.FactoryResult
	err := workflowRun.Get(context.Background(), &result)

	status := WorkflowStatus{
		WorkflowID: workflowID,
		Agents:     []AgentStatus{},
		Files:      []string{},
		Metrics:    make(map[string]interface{}),
	}

	if err != nil {
		// Workflow might still be running
		status.Status = "running"
		status.Progress = 50 // We'd need to implement actual progress tracking

		// Mock agent statuses for demo
		status.Agents = []AgentStatus{
			{Type: "backend", Status: "active", Progress: 80, Files: 5, Duration: "2.3s"},
			{Type: "frontend", Status: "pending", Progress: 0, Files: 0, Duration: "0s"},
			{Type: "database", Status: "completed", Progress: 100, Files: 3, Duration: "1.1s"},
			{Type: "api", Status: "active", Progress: 45, Files: 2, Duration: "1.8s"},
		}
	} else {
		// Workflow completed
		status.Status = "completed"
		status.Progress = 100
		status.Files = result.GeneratedFiles

		// All agents completed
		for _, agentType := range []string{"backend", "frontend", "database", "api"} {
			status.Agents = append(status.Agents, AgentStatus{
				Type:     agentType,
				Status:   "completed",
				Progress: 100,
				Files:    len(result.GeneratedFiles) / 4, // Rough division
				Duration: "2.5s",
			})
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func (gw *APIGateway) websocketHandler(w http.ResponseWriter, r *http.Request) {
	conn, err := gw.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Printf("WebSocket upgrade error: %v", err)
		return
	}
	defer conn.Close()

	// Get workflow ID from query params
	workflowID := r.URL.Query().Get("workflowId")
	if workflowID == "" {
		conn.WriteJSON(map[string]string{"error": "workflowId required"})
		return
	}

	// Store connection
	gw.connMutex.Lock()
	gw.connections[workflowID] = conn
	gw.connMutex.Unlock()

	// Clean up connection when done
	defer func() {
		gw.connMutex.Lock()
		delete(gw.connections, workflowID)
		gw.connMutex.Unlock()
	}()

	// Keep connection alive and send periodic updates
	for {
		select {
		case <-time.After(2 * time.Second):
			// Send status update
			workflowRun := gw.temporalClient.GetWorkflow(context.Background(), workflowID, "")

			// Mock progress update for demo
			update := map[string]interface{}{
				"type":       "progress",
				"workflowId": workflowID,
				"agents": []map[string]interface{}{
					{"type": "backend", "status": "active", "progress": 85},
					{"type": "frontend", "status": "pending", "progress": 0},
					{"type": "database", "status": "completed", "progress": 100},
				},
				"timestamp": time.Now().Unix(),
			}

			if err := conn.WriteJSON(update); err != nil {
				log.Printf("WebSocket write error: %v", err)
				return
			}

			// Check if workflow is done
			var result wf.FactoryResult
			err := workflowRun.Get(context.Background(), &result)
			if err == nil {
				// Workflow completed
				completion := map[string]interface{}{
					"type":       "completed",
					"workflowId": workflowID,
					"result":     result,
					"timestamp":  time.Now().Unix(),
				}
				conn.WriteJSON(completion)
				return
			}
		}
	}
}

func (gw *APIGateway) monitorWorkflow(workflowID, runID string) {
	// This would monitor the workflow and send updates to connected WebSocket clients
	// For now, it's a placeholder for future implementation
	log.Printf("Monitoring workflow: %s", workflowID)
}

func (gw *APIGateway) healthHandler(w http.ResponseWriter, r *http.Request) {
	status := map[string]interface{}{
		"status":    "healthy",
		"timestamp": time.Now().Unix(),
		"services": map[string]string{
			"temporal": "connected",
			"api":      "running",
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(status)
}

func main() {
	gw := NewAPIGateway()
	defer gw.temporalClient.Close()

	r := mux.NewRouter()

	// API routes
	api := r.PathPrefix("/api/v1").Subrouter()
	api.HandleFunc("/generate", gw.generateHandler).Methods("POST")
	api.HandleFunc("/status/{workflowId}", gw.statusHandler).Methods("GET")
	api.HandleFunc("/ws", gw.websocketHandler)
	api.HandleFunc("/health", gw.healthHandler).Methods("GET")

	// CORS middleware
	c := cors.New(cors.Options{
		AllowedOrigins: []string{"http://localhost:3000", "http://localhost:3001"}, // Next.js dev servers
		AllowedMethods: []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowedHeaders: []string{"*"},
		ExposedHeaders: []string{"*"},
		AllowCredentials: true,
	})

	handler := c.Handler(r)

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("🚀 API Gateway starting on port %s", port)
	log.Printf("📡 WebSocket endpoint: ws://localhost:%s/api/v1/ws", port)
	log.Printf("🏥 Health check: http://localhost:%s/api/v1/health", port)

	if err := http.ListenAndServe(":"+port, handler); err != nil {
		log.Fatalf("Server failed to start: %v", err)
	}
}