package llm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// ProviderRouter routes requests to different LLM providers
type ProviderRouter struct {
	clients      map[Provider]Client
	config       *Config
	cache        Cache
	budgetTracker *BudgetTracker
	mu           sync.RWMutex
}

// NewProviderRouter creates a new provider router
func NewProviderRouter(config *Config) (*ProviderRouter, error) {
	router := &ProviderRouter{
		clients: make(map[Provider]Client),
		config:  config,
		mu:      sync.RWMutex{},
	}

	// Create cache if enabled
	if config.Cache.Enabled {
		cache, err := NewCache(config.Cache)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache: %w", err)
		}
		router.cache = cache
	}

	// Create budget tracker if enabled
	if config.Budget.TrackingEnabled {
		tracker, err := NewBudgetTracker(config.Budget)
		if err != nil {
			return nil, fmt.Errorf("failed to create budget tracker: %w", err)
		}
		router.budgetTracker = tracker
	}

	// Initialize clients
	factory := NewClientFactory(config)

	// Create Bedrock client if configured
	if config.Bedrock.Region != "" {
		client, err := factory.CreateClient(ProviderBedrock)
		if err == nil {
			router.clients[ProviderBedrock] = client
		}
	}

	// Create Azure client if configured
	if config.Azure.Endpoint != "" {
		client, err := factory.CreateClient(ProviderAzure)
		if err == nil {
			router.clients[ProviderAzure] = client
		}
	}

	if len(router.clients) == 0 {
		return nil, fmt.Errorf("no LLM providers configured")
	}

	return router, nil
}

// Generate routes a generation request to the appropriate provider
func (r *ProviderRouter) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	// Determine provider
	provider := r.selectProvider(req)
	if provider == "" {
		return nil, fmt.Errorf("no suitable provider available")
	}

	// Check cache first
	if r.cache != nil && req.CacheKey != "" {
		if cached, found := r.cache.Get(req.CacheKey); found {
			cached.Cached = true
			return cached, nil
		}
	}

	// Check budget
	if r.budgetTracker != nil {
		if err := r.budgetTracker.CheckBudget(provider, req); err != nil {
			return nil, err
		}
	}

	// Get client
	client, err := r.getClient(provider)
	if err != nil {
		return nil, err
	}

	// Make request
	response, err := client.Generate(ctx, req)
	if err != nil {
		// Try failover if error is retryable
		if llmErr, ok := err.(*Error); ok && llmErr.Retry {
			if fallbackProvider := r.selectFallbackProvider(provider); fallbackProvider != "" {
				if fallbackClient, fbErr := r.getClient(fallbackProvider); fbErr == nil {
					response, err = fallbackClient.Generate(ctx, req)
					if err == nil {
						response.Provider = fallbackProvider
					}
				}
			}
		}

		if err != nil {
			return nil, err
		}
	}

	// Track usage
	if r.budgetTracker != nil {
		r.budgetTracker.TrackUsage(response)
	}

	// Cache response
	if r.cache != nil && req.CacheKey != "" && response != nil {
		r.cache.Set(req.CacheKey, response)
	}

	return response, nil
}

// GenerateWithComparison generates using multiple providers for comparison
func (r *ProviderRouter) GenerateWithComparison(ctx context.Context, req *GenerateRequest, providers []Provider) (*ComparisonResult, error) {
	results := make(map[Provider]*GenerateResponse)
	errors := make(map[Provider]error)

	// Generate with each provider concurrently
	var wg sync.WaitGroup
	var mu sync.Mutex

	for _, provider := range providers {
		if _, exists := r.clients[provider]; !exists {
			continue
		}

		wg.Add(1)
		go func(p Provider) {
			defer wg.Done()

			client, err := r.getClient(p)
			if err != nil {
				mu.Lock()
				errors[p] = err
				mu.Unlock()
				return
			}

			// Create a copy of the request for this provider
			providerReq := *req
			response, err := client.Generate(ctx, &providerReq)

			mu.Lock()
			if err != nil {
				errors[p] = err
			} else {
				results[p] = response
				// Track usage
				if r.budgetTracker != nil {
					r.budgetTracker.TrackUsage(response)
				}
			}
			mu.Unlock()
		}(provider)
	}

	wg.Wait()

	return &ComparisonResult{
		Results: results,
		Errors:  errors,
	}, nil
}

// selectProvider selects the best provider for a request
func (r *ProviderRouter) selectProvider(req *GenerateRequest) Provider {
	// If provider is explicitly specified, use it
	if req.Metadata != nil {
		if providerStr, ok := req.Metadata["provider"].(string); ok {
			provider := Provider(providerStr)
			if _, exists := r.clients[provider]; exists {
				return provider
			}
		}
	}

	// Auto-select based on task complexity and availability
	complexity := AnalyzeTaskComplexity(req.Prompt)

	// Prefer providers based on complexity and cost
	switch complexity {
	case TaskComplexitySimple:
		// Use cheaper/faster options first
		if _, exists := r.clients[ProviderBedrock]; exists {
			return ProviderBedrock // Claude Haiku is fast and cheap
		}
		if _, exists := r.clients[ProviderAzure]; exists {
			return ProviderAzure // GPT-3.5 is also fast
		}

	case TaskComplexityComplex:
		// Use more capable models
		if _, exists := r.clients[ProviderAzure]; exists {
			return ProviderAzure // GPT-4 for complex tasks
		}
		if _, exists := r.clients[ProviderBedrock]; exists {
			return ProviderBedrock // Claude 3.5 Sonnet
		}

	default:
		// Medium complexity - use default provider
		if _, exists := r.clients[r.config.DefaultProvider]; exists {
			return r.config.DefaultProvider
		}
	}

	// Fallback to any available provider
	for provider := range r.clients {
		return provider
	}

	return ""
}

// selectFallbackProvider selects a fallback provider
func (r *ProviderRouter) selectFallbackProvider(primary Provider) Provider {
	for provider := range r.clients {
		if provider != primary {
			return provider
		}
	}
	return ""
}

// getClient gets a client for a provider with health check
func (r *ProviderRouter) getClient(provider Provider) (Client, error) {
	r.mu.RLock()
	client, exists := r.clients[provider]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("provider %s not available", provider)
	}

	return client, nil
}

// AddProvider adds a new provider client
func (r *ProviderRouter) AddProvider(provider Provider, client Client) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.clients[provider] = client
}

// RemoveProvider removes a provider client
func (r *ProviderRouter) RemoveProvider(provider Provider) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if client, exists := r.clients[provider]; exists {
		client.Close()
		delete(r.clients, provider)
	}
}

// GetAvailableProviders returns list of available providers
func (r *ProviderRouter) GetAvailableProviders() []Provider {
	r.mu.RLock()
	defer r.mu.RUnlock()

	providers := make([]Provider, 0, len(r.clients))
	for provider := range r.clients {
		providers = append(providers, provider)
	}
	return providers
}

// HealthCheck checks health of all providers
func (r *ProviderRouter) HealthCheck(ctx context.Context) map[Provider]error {
	r.mu.RLock()
	clients := make(map[Provider]Client)
	for p, c := range r.clients {
		clients[p] = c
	}
	r.mu.RUnlock()

	results := make(map[Provider]error)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for provider, client := range clients {
		wg.Add(1)
		go func(p Provider, c Client) {
			defer wg.Done()
			err := c.Health(ctx)
			mu.Lock()
			results[p] = err
			mu.Unlock()
		}(provider, client)
	}

	wg.Wait()
	return results
}

// Close closes all provider clients
func (r *ProviderRouter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	var lastErr error
	for provider, client := range r.clients {
		if err := client.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close %s client: %w", provider, err)
		}
	}

	if r.cache != nil {
		if err := r.cache.Close(); err != nil {
			lastErr = fmt.Errorf("failed to close cache: %w", err)
		}
	}

	return lastErr
}

// ComparisonResult represents the result of comparing multiple providers
type ComparisonResult struct {
	Results map[Provider]*GenerateResponse
	Errors  map[Provider]error
}

// GetBestResult returns the best result from the comparison
func (cr *ComparisonResult) GetBestResult() *GenerateResponse {
	// Simple heuristic: return the longest response
	var best *GenerateResponse
	maxLength := 0

	for _, result := range cr.Results {
		if len(result.Content) > maxLength {
			maxLength = len(result.Content)
			best = result
		}
	}

	return best
}

// GetFastestResult returns the result with the shortest duration
func (cr *ComparisonResult) GetFastestResult() *GenerateResponse {
	var fastest *GenerateResponse
	minDuration := time.Duration(0)

	for _, result := range cr.Results {
		if minDuration == 0 || result.Duration < minDuration {
			minDuration = result.Duration
			fastest = result
		}
	}

	return fastest
}

// GetCheapestResult returns the result with the lowest cost
func (cr *ComparisonResult) GetCheapestResult() *GenerateResponse {
	var cheapest *GenerateResponse
	minCost := float64(0)

	for _, result := range cr.Results {
		if minCost == 0 || result.Usage.Cost < minCost {
			minCost = result.Usage.Cost
			cheapest = result
		}
	}

	return cheapest
}