package llm

import (
	"fmt"
	"sync"
	"time"
)

// BudgetTracker tracks LLM usage and costs
type BudgetTracker struct {
	config      BudgetConfig
	monthlyUsage map[string]*MonthlyUsage // key: YYYY-MM
	mu          sync.RWMutex
}

// MonthlyUsage tracks usage for a specific month
type MonthlyUsage struct {
	Month        string             `json:"month"`
	TotalCost    float64            `json:"total_cost"`
	TotalTokens  int                `json:"total_tokens"`
	RequestCount int                `json:"request_count"`
	Providers    map[Provider]*ProviderUsage `json:"providers"`
	CreatedAt    time.Time          `json:"created_at"`
	UpdatedAt    time.Time          `json:"updated_at"`
}

// ProviderUsage tracks usage for a specific provider
type ProviderUsage struct {
	Cost         float64          `json:"cost"`
	Tokens       int              `json:"tokens"`
	Requests     int              `json:"requests"`
	Models       map[Model]*ModelUsage `json:"models"`
}

// ModelUsage tracks usage for a specific model
type ModelUsage struct {
	Cost     float64 `json:"cost"`
	Tokens   int     `json:"tokens"`
	Requests int     `json:"requests"`
}

// NewBudgetTracker creates a new budget tracker
func NewBudgetTracker(config BudgetConfig) (*BudgetTracker, error) {
	return &BudgetTracker{
		config:       config,
		monthlyUsage: make(map[string]*MonthlyUsage),
		mu:           sync.RWMutex{},
	}, nil
}

// CheckBudget checks if a request would exceed the budget
func (b *BudgetTracker) CheckBudget(provider Provider, req *GenerateRequest) error {
	if !b.config.TrackingEnabled {
		return nil
	}

	b.mu.RLock()
	defer b.mu.RUnlock()

	month := time.Now().Format("2006-01")
	usage, exists := b.monthlyUsage[month]
	if !exists {
		return nil // No usage yet, allow request
	}

	// Check if we're already over budget
	if usage.TotalCost >= b.config.MonthlyLimit {
		return &Error{
			Code:     ErrorCodeQuotaExceeded,
			Message:  fmt.Sprintf("Monthly budget limit of $%.2f exceeded (current: $%.2f)", b.config.MonthlyLimit, usage.TotalCost),
			Provider: provider,
			Retry:    false,
		}
	}

	// Estimate cost for this request
	model := req.Model
	if model == "" {
		model = GetDefaultModel(provider)
	}

	// Rough estimate based on prompt length
	estimatedTokens := len(req.Prompt) / 4 // Rough tokens estimate
	if req.MaxTokens > 0 {
		estimatedTokens += req.MaxTokens
	} else {
		estimatedTokens += 1000 // Default assumption
	}

	estimatedCost := EstimateCost(provider, model, estimatedTokens/2, estimatedTokens/2)
	projectedTotal := usage.TotalCost + estimatedCost

	// Check if this request would exceed budget
	if projectedTotal > b.config.MonthlyLimit {
		return &Error{
			Code:     ErrorCodeQuotaExceeded,
			Message:  fmt.Sprintf("Request would exceed monthly budget (projected: $%.2f, limit: $%.2f)", projectedTotal, b.config.MonthlyLimit),
			Provider: provider,
			Retry:    false,
		}
	}

	return nil
}

// TrackUsage records usage from a response
func (b *BudgetTracker) TrackUsage(response *GenerateResponse) {
	if !b.config.TrackingEnabled {
		return
	}

	b.mu.Lock()
	defer b.mu.Unlock()

	month := time.Now().Format("2006-01")

	// Get or create monthly usage
	usage, exists := b.monthlyUsage[month]
	if !exists {
		usage = &MonthlyUsage{
			Month:        month,
			TotalCost:    0,
			TotalTokens:  0,
			RequestCount: 0,
			Providers:    make(map[Provider]*ProviderUsage),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
		b.monthlyUsage[month] = usage
	}

	// Update monthly totals
	usage.TotalCost += response.Usage.Cost
	usage.TotalTokens += response.Usage.TotalTokens
	usage.RequestCount++
	usage.UpdatedAt = time.Now()

	// Get or create provider usage
	providerUsage, exists := usage.Providers[response.Provider]
	if !exists {
		providerUsage = &ProviderUsage{
			Cost:     0,
			Tokens:   0,
			Requests: 0,
			Models:   make(map[Model]*ModelUsage),
		}
		usage.Providers[response.Provider] = providerUsage
	}

	// Update provider totals
	providerUsage.Cost += response.Usage.Cost
	providerUsage.Tokens += response.Usage.TotalTokens
	providerUsage.Requests++

	// Get or create model usage
	modelUsage, exists := providerUsage.Models[response.Model]
	if !exists {
		modelUsage = &ModelUsage{
			Cost:     0,
			Tokens:   0,
			Requests: 0,
		}
		providerUsage.Models[response.Model] = modelUsage
	}

	// Update model totals
	modelUsage.Cost += response.Usage.Cost
	modelUsage.Tokens += response.Usage.TotalTokens
	modelUsage.Requests++
}

// GetCurrentUsage returns the current month's usage
func (b *BudgetTracker) GetCurrentUsage() *MonthlyUsage {
	b.mu.RLock()
	defer b.mu.RUnlock()

	month := time.Now().Format("2006-01")
	usage, exists := b.monthlyUsage[month]
	if !exists {
		return &MonthlyUsage{
			Month:        month,
			TotalCost:    0,
			TotalTokens:  0,
			RequestCount: 0,
			Providers:    make(map[Provider]*ProviderUsage),
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}
	}

	// Return a copy to avoid race conditions
	return b.copyUsage(usage)
}

// GetUsageHistory returns usage for multiple months
func (b *BudgetTracker) GetUsageHistory(months int) map[string]*MonthlyUsage {
	b.mu.RLock()
	defer b.mu.RUnlock()

	result := make(map[string]*MonthlyUsage)
	now := time.Now()

	for i := 0; i < months; i++ {
		month := now.AddDate(0, -i, 0).Format("2006-01")
		if usage, exists := b.monthlyUsage[month]; exists {
			result[month] = b.copyUsage(usage)
		}
	}

	return result
}

// GetBudgetStatus returns the current budget status
func (b *BudgetTracker) GetBudgetStatus() *BudgetStatus {
	usage := b.GetCurrentUsage()

	remainingBudget := b.config.MonthlyLimit - usage.TotalCost
	utilizationRate := usage.TotalCost / b.config.MonthlyLimit

	status := &BudgetStatus{
		MonthlyLimit:     b.config.MonthlyLimit,
		CurrentSpend:     usage.TotalCost,
		RemainingBudget:  remainingBudget,
		UtilizationRate:  utilizationRate,
		AlertThreshold:   b.config.AlertThreshold,
		ShouldAlert:      utilizationRate >= b.config.AlertThreshold,
		Month:            usage.Month,
	}

	return status
}

// IsOverBudget checks if we're over budget
func (b *BudgetTracker) IsOverBudget() bool {
	usage := b.GetCurrentUsage()
	return usage.TotalCost >= b.config.MonthlyLimit
}

// ShouldAlert checks if we should alert based on threshold
func (b *BudgetTracker) ShouldAlert() bool {
	usage := b.GetCurrentUsage()
	utilizationRate := usage.TotalCost / b.config.MonthlyLimit
	return utilizationRate >= b.config.AlertThreshold
}

// copyUsage creates a deep copy of MonthlyUsage
func (b *BudgetTracker) copyUsage(original *MonthlyUsage) *MonthlyUsage {
	copy := &MonthlyUsage{
		Month:        original.Month,
		TotalCost:    original.TotalCost,
		TotalTokens:  original.TotalTokens,
		RequestCount: original.RequestCount,
		Providers:    make(map[Provider]*ProviderUsage),
		CreatedAt:    original.CreatedAt,
		UpdatedAt:    original.UpdatedAt,
	}

	for provider, providerUsage := range original.Providers {
		copyProvider := &ProviderUsage{
			Cost:     providerUsage.Cost,
			Tokens:   providerUsage.Tokens,
			Requests: providerUsage.Requests,
			Models:   make(map[Model]*ModelUsage),
		}

		for model, modelUsage := range providerUsage.Models {
			copyProvider.Models[model] = &ModelUsage{
				Cost:     modelUsage.Cost,
				Tokens:   modelUsage.Tokens,
				Requests: modelUsage.Requests,
			}
		}

		copy.Providers[provider] = copyProvider
	}

	return copy
}

// BudgetStatus represents the current budget status
type BudgetStatus struct {
	MonthlyLimit     float64 `json:"monthly_limit"`
	CurrentSpend     float64 `json:"current_spend"`
	RemainingBudget  float64 `json:"remaining_budget"`
	UtilizationRate  float64 `json:"utilization_rate"`
	AlertThreshold   float64 `json:"alert_threshold"`
	ShouldAlert      bool    `json:"should_alert"`
	Month            string  `json:"month"`
}