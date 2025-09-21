package llm

import (
	"context"
	"fmt"
	"time"
)

// Provider represents LLM provider types
type Provider string

const (
	ProviderBedrock Provider = "bedrock"
	ProviderAzure   Provider = "azure"
)

// Model represents specific model names
type Model string

const (
	// AWS Bedrock Claude models
	ModelClaudeHaiku   Model = "anthropic.claude-3-haiku-20240307-v1:0"
	ModelClaudeSonnet  Model = "anthropic.claude-3-sonnet-20240229-v1:0"
	ModelClaude35      Model = "anthropic.claude-3-5-sonnet-20240620-v1:0"
	ModelClaude37      Model = "anthropic.claude-3-7-sonnet-20250219-v1:0"

	// Azure OpenAI models
	ModelGPT35     Model = "gpt-35-turbo"
	ModelGPT4      Model = "gpt-4"
	ModelGPT4Turbo Model = "gpt-4-turbo"
	ModelGPT41     Model = "gpt-4.1"
	ModelGPT41Mini Model = "gpt-4.1-mini"
	ModelGPT41Nano Model = "gpt-4.1-nano"
	ModelGPT5      Model = "gpt-5"
	ModelO4Mini    Model = "o4-mini"
)

// Client interface for LLM providers
type Client interface {
	// Generate text using the LLM
	Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error)

	// GenerateStream generates text with streaming response
	GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *StreamChunk, error)

	// GetProvider returns the provider type
	GetProvider() Provider

	// GetModels returns available models for this provider
	GetModels() []Model

	// Close closes the client connection
	Close() error

	// Health checks if the provider is available
	Health(ctx context.Context) error
}

// GenerateRequest represents a request to generate text
type GenerateRequest struct {
	// Prompt to send to the LLM
	Prompt string `json:"prompt"`

	// Model to use (optional, uses default if empty)
	Model Model `json:"model,omitempty"`

	// MaxTokens limits the response length
	MaxTokens int `json:"max_tokens,omitempty"`

	// Temperature controls randomness (0.0-1.0)
	Temperature float64 `json:"temperature,omitempty"`

	// SystemPrompt provides system context
	SystemPrompt string `json:"system_prompt,omitempty"`

	// Metadata for tracking and debugging
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// CacheKey for response caching (optional)
	CacheKey string `json:"cache_key,omitempty"`
}

// GenerateResponse represents the response from an LLM
type GenerateResponse struct {
	// Content is the generated text
	Content string `json:"content"`

	// Model used for generation
	Model Model `json:"model"`

	// Provider used for generation
	Provider Provider `json:"provider"`

	// Usage statistics
	Usage Usage `json:"usage"`

	// Metadata from the provider
	Metadata map[string]interface{} `json:"metadata,omitempty"`

	// Cached indicates if response came from cache
	Cached bool `json:"cached"`

	// Duration of the request
	Duration time.Duration `json:"duration"`
}

// StreamChunk represents a chunk in a streaming response
type StreamChunk struct {
	// Content is the incremental text
	Content string `json:"content"`

	// Done indicates if this is the final chunk
	Done bool `json:"done"`

	// Error if something went wrong
	Error error `json:"error,omitempty"`
}

// Usage represents token usage statistics
type Usage struct {
	// PromptTokens is the number of tokens in the input
	PromptTokens int `json:"prompt_tokens"`

	// CompletionTokens is the number of tokens in the output
	CompletionTokens int `json:"completion_tokens"`

	// TotalTokens is the sum of prompt and completion tokens
	TotalTokens int `json:"total_tokens"`

	// Cost is the estimated cost in USD
	Cost float64 `json:"cost"`
}

// Config represents LLM configuration
type Config struct {
	// DefaultProvider to use when none specified
	DefaultProvider Provider `yaml:"defaultProvider"`

	// Bedrock configuration
	Bedrock BedrockConfig `yaml:"bedrock"`

	// Azure configuration
	Azure AzureConfig `yaml:"azure"`

	// Cache configuration
	Cache CacheConfig `yaml:"cache"`

	// RateLimit configuration
	RateLimit RateLimitConfig `yaml:"rateLimit"`

	// Budget configuration
	Budget BudgetConfig `yaml:"budget"`
}

// BedrockConfig represents AWS Bedrock configuration
type BedrockConfig struct {
	// Region (e.g., "eu-west-2" for London)
	Region string `yaml:"region"`

	// AccessKeyID for AWS credentials
	AccessKeyID string `yaml:"accessKeyId"`

	// SecretAccessKey for AWS credentials
	SecretAccessKey string `yaml:"secretAccessKey"`

	// DefaultModel to use
	DefaultModel Model `yaml:"defaultModel"`

	// Models configuration
	Models map[string]Model `yaml:"models"`
}

// AzureConfig represents Azure OpenAI configuration
type AzureConfig struct {
	// Endpoint URL
	Endpoint string `yaml:"endpoint"`

	// APIKey for authentication
	APIKey string `yaml:"apiKey"`

	// APIVersion to use
	APIVersion string `yaml:"apiVersion"`

	// Location (e.g., "uksouth")
	Location string `yaml:"location"`

	// Deployments maps model names to deployment names
	Deployments map[string]string `yaml:"deployments"`
}

// CacheConfig represents caching configuration
type CacheConfig struct {
	// Enabled indicates if caching is enabled
	Enabled bool `yaml:"enabled"`

	// TTL is the time-to-live for cached responses
	TTL time.Duration `yaml:"ttl"`

	// RedisURL for cache storage
	RedisURL string `yaml:"redisUrl"`
}

// RateLimitConfig represents rate limiting configuration
type RateLimitConfig struct {
	// RequestsPerMinute limit
	RequestsPerMinute int `yaml:"requestsPerMinute"`

	// TokensPerMinute limit
	TokensPerMinute int `yaml:"tokensPerMinute"`
}

// BudgetConfig represents budget tracking configuration
type BudgetConfig struct {
	// MonthlyLimit in USD
	MonthlyLimit float64 `yaml:"monthlyLimit"`

	// AlertThreshold when to alert (0.0-1.0)
	AlertThreshold float64 `yaml:"alertThreshold"`

	// TrackingEnabled indicates if cost tracking is enabled
	TrackingEnabled bool `yaml:"trackingEnabled"`
}

// Error types for LLM operations
type Error struct {
	Code     string
	Message  string
	Provider Provider
	Retry    bool
}

func (e *Error) Error() string {
	return fmt.Sprintf("[%s] %s: %s", e.Provider, e.Code, e.Message)
}

// Common error codes
const (
	ErrorCodeRateLimit     = "RATE_LIMIT"
	ErrorCodeQuotaExceeded = "QUOTA_EXCEEDED"
	ErrorCodeInvalidModel  = "INVALID_MODEL"
	ErrorCodeBadRequest    = "BAD_REQUEST"
	ErrorCodeUnauthorized  = "UNAUTHORIZED"
	ErrorCodeProviderDown  = "PROVIDER_DOWN"
	ErrorCodeTimeout       = "TIMEOUT"
)