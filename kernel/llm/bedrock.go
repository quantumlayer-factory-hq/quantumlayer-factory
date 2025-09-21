package llm

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/bedrockruntime"
)

// BedrockClient implements the Client interface for AWS Bedrock
type BedrockClient struct {
	client *bedrockruntime.Client
	config BedrockConfig
}

// NewBedrockClient creates a new AWS Bedrock client
func NewBedrockClient(cfg BedrockConfig) (*BedrockClient, error) {
	// Set default region to London if not specified
	if cfg.Region == "" {
		cfg.Region = "eu-west-2"
	}

	// Create AWS config
	var awsCfg aws.Config
	var err error

	if cfg.AccessKeyID != "" && cfg.SecretAccessKey != "" {
		// Use provided credentials
		awsCfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(cfg.Region),
			config.WithCredentialsProvider(credentials.NewStaticCredentialsProvider(
				cfg.AccessKeyID,
				cfg.SecretAccessKey,
				"",
			)),
		)
	} else {
		// Use default AWS credential chain
		awsCfg, err = config.LoadDefaultConfig(context.TODO(),
			config.WithRegion(cfg.Region),
		)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to load AWS config: %w", err)
	}

	client := bedrockruntime.NewFromConfig(awsCfg)

	return &BedrockClient{
		client: client,
		config: cfg,
	}, nil
}

// Generate implements the Client interface
func (b *BedrockClient) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	startTime := time.Now()

	// Select model
	model := req.Model
	if model == "" {
		model = b.getDefaultModel()
	}

	// Validate model
	if !ValidateModel(ProviderBedrock, model) {
		return nil, &Error{
			Code:     ErrorCodeInvalidModel,
			Message:  fmt.Sprintf("invalid model: %s", model),
			Provider: ProviderBedrock,
			Retry:    false,
		}
	}

	// Build request body based on model
	body, err := b.buildRequestBody(req, model)
	if err != nil {
		return nil, fmt.Errorf("failed to build request body: %w", err)
	}

	// Make request to Bedrock
	input := &bedrockruntime.InvokeModelInput{
		ModelId:     aws.String(string(model)),
		Body:        body,
		ContentType: aws.String("application/json"),
		Accept:      aws.String("application/json"),
	}

	result, err := b.client.InvokeModel(ctx, input)
	if err != nil {
		return nil, b.handleError(err)
	}

	// Parse response
	response, err := b.parseResponse(result.Body, model)
	if err != nil {
		return nil, fmt.Errorf("failed to parse response: %w", err)
	}

	// Build final response
	return &GenerateResponse{
		Content:  response.Content,
		Model:    model,
		Provider: ProviderBedrock,
		Usage:    response.Usage,
		Metadata: map[string]interface{}{
			"model_id": string(model),
			"region":   b.config.Region,
		},
		Cached:   false,
		Duration: time.Since(startTime),
	}, nil
}

// GenerateStream implements streaming generation
func (b *BedrockClient) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *StreamChunk, error) {
	chunks := make(chan *StreamChunk, 10)

	go func() {
		defer close(chunks)

		// For now, simulate streaming by chunking the full response
		// TODO: Implement actual Bedrock streaming when available
		response, err := b.Generate(ctx, req)
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
func (b *BedrockClient) GetProvider() Provider {
	return ProviderBedrock
}

// GetModels returns available models
func (b *BedrockClient) GetModels() []Model {
	return GetValidModels(ProviderBedrock)
}

// Close closes the client connection
func (b *BedrockClient) Close() error {
	// Bedrock client doesn't need explicit closing
	return nil
}

// Health checks if Bedrock is available
func (b *BedrockClient) Health(ctx context.Context) error {
	// Simple health check using a minimal request
	req := &GenerateRequest{
		Prompt:    "Hello",
		MaxTokens: 5,
	}

	_, err := b.Generate(ctx, req)
	return err
}

// getDefaultModel returns the default model for this client
func (b *BedrockClient) getDefaultModel() Model {
	if b.config.DefaultModel != "" {
		return b.config.DefaultModel
	}
	return ModelClaudeSonnet
}

// buildRequestBody builds the request body for different Claude models
func (b *BedrockClient) buildRequestBody(req *GenerateRequest, model Model) ([]byte, error) {
	// Claude 3+ family uses Messages API format
	if strings.Contains(string(model), "anthropic.claude-3") {
		body := ClaudeRequest{
			AnthropicVersion: "bedrock-2023-05-31",
			MaxTokens:        req.MaxTokens,
			Temperature:      req.Temperature,
			Messages: []ClaudeMessage{
				{
					Role:    "user",
					Content: req.Prompt,
				},
			},
		}

		// Add system prompt if provided
		if req.SystemPrompt != "" {
			body.System = req.SystemPrompt
		}

		// Set defaults
		if body.MaxTokens == 0 {
			body.MaxTokens = 4096
		}
		if body.Temperature == 0 {
			body.Temperature = 0.7
		}

		return json.Marshal(body)
	}

	return nil, fmt.Errorf("unsupported model: %s", model)
}

// parseResponse parses the response from Bedrock
func (b *BedrockClient) parseResponse(body []byte, model Model) (*BedrockResponse, error) {
	if strings.Contains(string(model), "anthropic.claude-3") {
		var claudeResp ClaudeResponse
		if err := json.Unmarshal(body, &claudeResp); err != nil {
			return nil, fmt.Errorf("failed to unmarshal Claude response: %w", err)
		}

		// Extract content from Claude response
		content := ""
		if len(claudeResp.Content) > 0 {
			content = claudeResp.Content[0].Text
		}

		// Calculate usage
		usage := Usage{
			PromptTokens:     claudeResp.Usage.InputTokens,
			CompletionTokens: claudeResp.Usage.OutputTokens,
			TotalTokens:      claudeResp.Usage.InputTokens + claudeResp.Usage.OutputTokens,
		}
		usage.Cost = EstimateCost(ProviderBedrock, model, usage.PromptTokens, usage.CompletionTokens)

		return &BedrockResponse{
			Content: content,
			Usage:   usage,
		}, nil
	}

	return nil, fmt.Errorf("unsupported model for response parsing: %s", model)
}

// handleError converts AWS errors to LLM errors
func (b *BedrockClient) handleError(err error) error {
	errStr := err.Error()

	// Map common AWS errors
	if strings.Contains(errStr, "ThrottlingException") {
		return &Error{
			Code:     ErrorCodeRateLimit,
			Message:  "Rate limit exceeded",
			Provider: ProviderBedrock,
			Retry:    true,
		}
	}

	if strings.Contains(errStr, "ValidationException") {
		return &Error{
			Code:     ErrorCodeBadRequest,
			Message:  "Invalid request",
			Provider: ProviderBedrock,
			Retry:    false,
		}
	}

	if strings.Contains(errStr, "UnauthorizedOperation") {
		return &Error{
			Code:     ErrorCodeUnauthorized,
			Message:  "Unauthorized access",
			Provider: ProviderBedrock,
			Retry:    false,
		}
	}

	// Default error
	return &Error{
		Code:     "UNKNOWN",
		Message:  errStr,
		Provider: ProviderBedrock,
		Retry:    true,
	}
}

// Request/Response structures for Claude models

// ClaudeRequest represents a request to Claude models
type ClaudeRequest struct {
	AnthropicVersion string          `json:"anthropic_version"`
	MaxTokens        int             `json:"max_tokens"`
	Temperature      float64         `json:"temperature,omitempty"`
	System           string          `json:"system,omitempty"`
	Messages         []ClaudeMessage `json:"messages"`
}

// ClaudeMessage represents a message in Claude conversation
type ClaudeMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

// ClaudeResponse represents a response from Claude models
type ClaudeResponse struct {
	ID      string `json:"id"`
	Type    string `json:"type"`
	Role    string `json:"role"`
	Content []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	} `json:"content"`
	Model        string `json:"model"`
	StopReason   string `json:"stop_reason"`
	StopSequence string `json:"stop_sequence"`
	Usage        struct {
		InputTokens  int `json:"input_tokens"`
		OutputTokens int `json:"output_tokens"`
	} `json:"usage"`
}

// BedrockResponse is the internal response structure
type BedrockResponse struct {
	Content string
	Usage   Usage
}