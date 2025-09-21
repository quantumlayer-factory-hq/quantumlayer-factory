package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// AzureOpenAIClient implements the Client interface for Azure OpenAI
type AzureOpenAIClient struct {
	config     AzureConfig
	httpClient *http.Client
	baseURL    string
}

// NewAzureOpenAIClient creates a new Azure OpenAI client
func NewAzureOpenAIClient(cfg AzureConfig) (*AzureOpenAIClient, error) {
	if cfg.Endpoint == "" {
		return nil, fmt.Errorf("Azure OpenAI endpoint is required")
	}

	if cfg.APIKey == "" {
		return nil, fmt.Errorf("Azure OpenAI API key is required")
	}

	// Set defaults
	if cfg.APIVersion == "" {
		cfg.APIVersion = "2024-02-01"
	}

	// Clean endpoint URL
	baseURL := strings.TrimSuffix(cfg.Endpoint, "/")

	return &AzureOpenAIClient{
		config:     cfg,
		httpClient: &http.Client{Timeout: 30 * time.Second},
		baseURL:    baseURL,
	}, nil
}

// Generate implements the Client interface
func (a *AzureOpenAIClient) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	startTime := time.Now()

	// Select model/deployment
	deployment := a.getDeploymentName(req.Model)
	if deployment == "" {
		return nil, &Error{
			Code:     ErrorCodeInvalidModel,
			Message:  fmt.Sprintf("no deployment found for model: %s", req.Model),
			Provider: ProviderAzure,
			Retry:    false,
		}
	}

	// Build request
	azureReq := a.buildRequest(req)

	// Make HTTP request
	response, err := a.makeRequest(ctx, deployment, azureReq)
	if err != nil {
		return nil, err
	}

	// Build final response
	return &GenerateResponse{
		Content:  response.Content,
		Model:    req.Model,
		Provider: ProviderAzure,
		Usage:    response.Usage,
		Metadata: map[string]interface{}{
			"deployment": deployment,
			"location":   a.config.Location,
		},
		Cached:   false,
		Duration: time.Since(startTime),
	}, nil
}

// GenerateStream implements streaming generation
func (a *AzureOpenAIClient) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *StreamChunk, error) {
	chunks := make(chan *StreamChunk, 10)

	go func() {
		defer close(chunks)

		// For now, simulate streaming by chunking the full response
		response, err := a.Generate(ctx, req)
		if err != nil {
			chunks <- &StreamChunk{Error: err}
			return
		}

		// Split content into chunks
		content := response.Content
		chunkSize := 50 // Characters per chunk
		for i := 0; i < len(content); i += chunkSize {
			end := i + chunkSize
			if end > len(content) {
				end = len(content)
			}

			chunk := content[i:end]
			chunks <- &StreamChunk{
				Content: chunk,
				Done:    end >= len(content),
			}

			// Small delay to simulate streaming
			time.Sleep(10 * time.Millisecond)
		}
	}()

	return chunks, nil
}

// GetProvider returns the provider type
func (a *AzureOpenAIClient) GetProvider() Provider {
	return ProviderAzure
}

// GetModels returns available models
func (a *AzureOpenAIClient) GetModels() []Model {
	return GetValidModels(ProviderAzure)
}

// Close closes the client connection
func (a *AzureOpenAIClient) Close() error {
	// HTTP client doesn't need explicit closing
	return nil
}

// Health checks if Azure OpenAI is available
func (a *AzureOpenAIClient) Health(ctx context.Context) error {
	// Simple health check using a minimal request
	req := &GenerateRequest{
		Prompt:    "Hello",
		Model:     ModelGPT35, // Use cheaper model for health check
		MaxTokens: 5,
	}

	_, err := a.Generate(ctx, req)
	return err
}

// getDeploymentName gets the deployment name for a model
func (a *AzureOpenAIClient) getDeploymentName(model Model) string {
	if model == "" {
		model = ModelGPT5 // Default to the most capable model
	}

	// Check configured deployments
	if deployment, ok := a.config.Deployments[string(model)]; ok {
		return deployment
	}

	// Fallback mapping
	switch model {
	case ModelGPT35:
		return "gpt-35-turbo"
	case ModelGPT4:
		return "gpt-4"
	case ModelGPT4Turbo:
		return "gpt-4-turbo"
	case ModelGPT41:
		return "gpt-4.1"
	case ModelGPT41Mini:
		return "gpt-4.1-mini"
	case ModelGPT41Nano:
		return "gpt-4.1-nano"
	case ModelGPT5:
		return "gpt-5"
	case ModelO4Mini:
		return "o4-mini"
	default:
		return string(model)
	}
}

// buildRequest builds the Azure OpenAI request
func (a *AzureOpenAIClient) buildRequest(req *GenerateRequest) *AzureOpenAIRequest {
	azureReq := &AzureOpenAIRequest{
		Messages: []AzureMessage{},
		Stream:   false,
	}

	// Add system message if provided
	if req.SystemPrompt != "" {
		azureReq.Messages = append(azureReq.Messages, AzureMessage{
			Role:    "system",
			Content: req.SystemPrompt,
		})
	}

	// Add user message
	azureReq.Messages = append(azureReq.Messages, AzureMessage{
		Role:    "user",
		Content: req.Prompt,
	})

	// Set parameters
	if req.MaxTokens > 0 {
		azureReq.MaxTokens = req.MaxTokens
	} else {
		azureReq.MaxTokens = 4096
	}

	if req.Temperature > 0 {
		azureReq.Temperature = req.Temperature
	} else {
		azureReq.Temperature = 0.7
	}

	return azureReq
}

// makeRequest makes the HTTP request to Azure OpenAI
func (a *AzureOpenAIClient) makeRequest(ctx context.Context, deployment string, req *AzureOpenAIRequest) (*AzureResponse, error) {
	// Build URL
	url := fmt.Sprintf("%s/openai/deployments/%s/chat/completions?api-version=%s",
		a.baseURL, deployment, a.config.APIVersion)

	// Marshal request body
	body, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	// Create HTTP request
	httpReq, err := http.NewRequestWithContext(ctx, "POST", url, strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	// Set headers
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("api-key", a.config.APIKey)

	// Make request
	resp, err := a.httpClient.Do(httpReq)
	if err != nil {
		return nil, a.handleError(err)
	}
	defer resp.Body.Close()

	// Read response body
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response: %w", err)
	}

	// Check for HTTP errors
	if resp.StatusCode != http.StatusOK {
		return nil, a.handleHTTPError(resp.StatusCode, respBody)
	}

	// Parse response
	var azureResp AzureOpenAIResponse
	if err := json.Unmarshal(respBody, &azureResp); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}

	// Extract content
	content := ""
	if len(azureResp.Choices) > 0 && azureResp.Choices[0].Message.Content != "" {
		content = azureResp.Choices[0].Message.Content
	}

	// Calculate usage
	usage := Usage{
		PromptTokens:     azureResp.Usage.PromptTokens,
		CompletionTokens: azureResp.Usage.CompletionTokens,
		TotalTokens:      azureResp.Usage.TotalTokens,
	}

	// Estimate cost (use the model from the request)
	model := ModelGPT4Turbo // Default assumption
	if req.Messages != nil && len(req.Messages) > 0 {
		// We don't have model info in the request, use default
	}
	usage.Cost = EstimateCost(ProviderAzure, model, usage.PromptTokens, usage.CompletionTokens)

	return &AzureResponse{
		Content: content,
		Usage:   usage,
	}, nil
}

// handleError converts errors to LLM errors
func (a *AzureOpenAIClient) handleError(err error) error {
	errStr := err.Error()

	if strings.Contains(errStr, "timeout") {
		return &Error{
			Code:     ErrorCodeTimeout,
			Message:  "Request timeout",
			Provider: ProviderAzure,
			Retry:    true,
		}
	}

	return &Error{
		Code:     "NETWORK_ERROR",
		Message:  errStr,
		Provider: ProviderAzure,
		Retry:    true,
	}
}

// handleHTTPError converts HTTP errors to LLM errors
func (a *AzureOpenAIClient) handleHTTPError(statusCode int, body []byte) error {
	switch statusCode {
	case http.StatusUnauthorized:
		return &Error{
			Code:     ErrorCodeUnauthorized,
			Message:  "Invalid API key",
			Provider: ProviderAzure,
			Retry:    false,
		}
	case http.StatusTooManyRequests:
		return &Error{
			Code:     ErrorCodeRateLimit,
			Message:  "Rate limit exceeded",
			Provider: ProviderAzure,
			Retry:    true,
		}
	case http.StatusBadRequest:
		return &Error{
			Code:     ErrorCodeBadRequest,
			Message:  "Invalid request",
			Provider: ProviderAzure,
			Retry:    false,
		}
	default:
		return &Error{
			Code:     "HTTP_ERROR",
			Message:  fmt.Sprintf("HTTP %d: %s", statusCode, string(body)),
			Provider: ProviderAzure,
			Retry:    statusCode >= 500,
		}
	}
}

// Request/Response structures for Azure OpenAI

// AzureOpenAIRequest represents a request to Azure OpenAI
type AzureOpenAIRequest struct {
	Messages    []AzureMessage `json:"messages"`
	MaxTokens   int            `json:"max_tokens,omitempty"`
	Temperature float64        `json:"temperature,omitempty"`
	Stream      bool           `json:"stream"`
}

// AzureMessage represents a message in Azure OpenAI conversation
type AzureMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// AzureOpenAIResponse represents a response from Azure OpenAI
type AzureOpenAIResponse struct {
	ID      string `json:"id"`
	Object  string `json:"object"`
	Created int64  `json:"created"`
	Model   string `json:"model"`
	Choices []struct {
		Index   int `json:"index"`
		Message struct {
			Role    string `json:"role"`
			Content string `json:"content"`
		} `json:"message"`
		FinishReason string `json:"finish_reason"`
	} `json:"choices"`
	Usage struct {
		PromptTokens     int `json:"prompt_tokens"`
		CompletionTokens int `json:"completion_tokens"`
		TotalTokens      int `json:"total_tokens"`
	} `json:"usage"`
}

// AzureResponse is the internal response structure
type AzureResponse struct {
	Content string
	Usage   Usage
}