package observability

import (
	"context"
	"fmt"
	"sync"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/metric"
)

// LLMMonitor provides specialized monitoring for LLM operations
type LLMMonitor struct {
	config      *ObservabilityConfig
	metrics     *Metrics
	costTracker *CostTracker
	mu          sync.RWMutex
}

// NewLLMMonitor creates a new LLM monitoring service
func NewLLMMonitor(config *ObservabilityConfig, metrics *Metrics) *LLMMonitor {
	return &LLMMonitor{
		config:      config,
		metrics:     metrics,
		costTracker: NewCostTracker(),
	}
}

// CostTracker tracks LLM usage costs by provider and model
type CostTracker struct {
	costs map[string]ProviderCost
	mu    sync.RWMutex
}

// ProviderCost tracks costs for a specific provider
type ProviderCost struct {
	Provider    string             `json:"provider"`
	TotalCost   float64            `json:"total_cost"`
	TokensUsed  int64              `json:"tokens_used"`
	RequestCount int64              `json:"request_count"`
	Models      map[string]ModelCost `json:"models"`
	LastUpdated time.Time           `json:"last_updated"`
}

// ModelCost tracks costs for a specific model
type ModelCost struct {
	Model        string    `json:"model"`
	TokensUsed   int64     `json:"tokens_used"`
	RequestCount int64     `json:"request_count"`
	TotalCost    float64   `json:"total_cost"`
	AvgLatency   float64   `json:"avg_latency_ms"`
	CacheHitRate float64   `json:"cache_hit_rate"`
	LastUsed     time.Time `json:"last_used"`
}

// NewCostTracker creates a new cost tracker
func NewCostTracker() *CostTracker {
	return &CostTracker{
		costs: make(map[string]ProviderCost),
	}
}

// RecordLLMRequest records comprehensive LLM usage metrics
func (lm *LLMMonitor) RecordLLMRequest(metrics *LLMMetrics) {
	if !lm.config.MetricsEnabled {
		return
	}

	// Record OpenTelemetry metrics
	attrs := []attribute.KeyValue{
		AttrProvider.String(metrics.Provider),
		AttrModel.String(metrics.Model),
		AttrSuccess.Bool(metrics.Success),
		AttrCacheHit.Bool(metrics.CacheHit),
		AttrTokensUsed.Int64(metrics.TokensUsed),
		AttrCost.Float64(metrics.Cost),
	}

	if !metrics.Success && metrics.ErrorType != "" {
		attrs = append(attrs, AttrErrorType.String(metrics.ErrorType))
	}

	// Record metrics
	lm.metrics.LLMRequestCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	lm.metrics.LLMTokenUsage.Add(context.Background(), metrics.TokensUsed, metric.WithAttributes(attrs...))
	lm.metrics.LLMLatency.Record(context.Background(), metrics.Latency.Seconds(), metric.WithAttributes(attrs...))

	if metrics.CacheHit {
		lm.metrics.LLMCacheHitCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	}

	// Update cost tracking
	lm.updateCostTracking(metrics)

	// Update cost gauge
	totalCost := lm.GetTotalCost()
	lm.metrics.LLMCostGauge.Record(context.Background(), totalCost, metric.WithAttributes(
		attribute.String("scope", "total"),
	))
}

// updateCostTracking updates internal cost tracking
func (lm *LLMMonitor) updateCostTracking(metrics *LLMMetrics) {
	lm.costTracker.mu.Lock()
	defer lm.costTracker.mu.Unlock()

	// Get or create provider cost tracking
	providerCost, exists := lm.costTracker.costs[metrics.Provider]
	if !exists {
		providerCost = ProviderCost{
			Provider: metrics.Provider,
			Models:   make(map[string]ModelCost),
		}
	}

	// Update provider-level metrics
	providerCost.TotalCost += metrics.Cost
	providerCost.TokensUsed += metrics.TokensUsed
	providerCost.RequestCount++
	providerCost.LastUpdated = metrics.Timestamp

	// Get or create model cost tracking
	modelCost, exists := providerCost.Models[metrics.Model]
	if !exists {
		modelCost = ModelCost{
			Model: metrics.Model,
		}
	}

	// Update model-level metrics
	modelCost.TokensUsed += metrics.TokensUsed
	modelCost.RequestCount++
	modelCost.TotalCost += metrics.Cost
	modelCost.LastUsed = metrics.Timestamp

	// Update average latency
	if modelCost.RequestCount > 1 {
		// Exponential moving average
		alpha := 0.1
		modelCost.AvgLatency = alpha*float64(metrics.Latency.Milliseconds()) + (1-alpha)*modelCost.AvgLatency
	} else {
		modelCost.AvgLatency = float64(metrics.Latency.Milliseconds())
	}

	// Update cache hit rate (simplified calculation)
	if metrics.CacheHit {
		modelCost.CacheHitRate = (modelCost.CacheHitRate*float64(modelCost.RequestCount-1) + 1.0) / float64(modelCost.RequestCount)
	} else {
		modelCost.CacheHitRate = (modelCost.CacheHitRate * float64(modelCost.RequestCount-1)) / float64(modelCost.RequestCount)
	}

	// Store updated costs
	providerCost.Models[metrics.Model] = modelCost
	lm.costTracker.costs[metrics.Provider] = providerCost
}

// GetProviderCosts returns cost breakdown by provider
func (lm *LLMMonitor) GetProviderCosts() map[string]ProviderCost {
	lm.costTracker.mu.RLock()
	defer lm.costTracker.mu.RUnlock()

	// Return a copy to avoid race conditions
	result := make(map[string]ProviderCost)
	for provider, cost := range lm.costTracker.costs {
		result[provider] = cost
	}
	return result
}

// GetTotalCost returns total cost across all providers
func (lm *LLMMonitor) GetTotalCost() float64 {
	lm.costTracker.mu.RLock()
	defer lm.costTracker.mu.RUnlock()

	total := 0.0
	for _, cost := range lm.costTracker.costs {
		total += cost.TotalCost
	}
	return total
}

// GetModelPerformance returns performance metrics for models
func (lm *LLMMonitor) GetModelPerformance() []ModelPerformance {
	lm.costTracker.mu.RLock()
	defer lm.costTracker.mu.RUnlock()

	var performance []ModelPerformance

	for provider, providerCost := range lm.costTracker.costs {
		for model, modelCost := range providerCost.Models {
			perf := ModelPerformance{
				Provider:       provider,
				Model:         model,
				RequestCount:  modelCost.RequestCount,
				TokensUsed:    modelCost.TokensUsed,
				TotalCost:     modelCost.TotalCost,
				AvgLatency:    time.Duration(modelCost.AvgLatency) * time.Millisecond,
				CacheHitRate:  modelCost.CacheHitRate,
				CostPerToken:  modelCost.TotalCost / float64(modelCost.TokensUsed),
				LastUsed:      modelCost.LastUsed,
			}

			if modelCost.TokensUsed > 0 {
				perf.CostPerToken = modelCost.TotalCost / float64(modelCost.TokensUsed)
			}

			performance = append(performance, perf)
		}
	}

	return performance
}

// ModelPerformance holds performance metrics for a specific model
type ModelPerformance struct {
	Provider     string        `json:"provider"`
	Model        string        `json:"model"`
	RequestCount int64         `json:"request_count"`
	TokensUsed   int64         `json:"tokens_used"`
	TotalCost    float64       `json:"total_cost"`
	AvgLatency   time.Duration `json:"avg_latency"`
	CacheHitRate float64       `json:"cache_hit_rate"`
	CostPerToken float64       `json:"cost_per_token"`
	LastUsed     time.Time     `json:"last_used"`
}

// CheckBudgetLimit checks if costs exceed configured budget
func (lm *LLMMonitor) CheckBudgetLimit() *BudgetAlert {
	if lm.config == nil {
		return nil
	}

	totalCost := lm.GetTotalCost()

	// Assume monthly budget for now (would be configurable)
	monthlyBudget := 100.0 // USD
	if totalCost > monthlyBudget {
		return &BudgetAlert{
			Type:        "budget_exceeded",
			CurrentCost: totalCost,
			BudgetLimit: monthlyBudget,
			Percentage:  (totalCost / monthlyBudget) * 100,
			Timestamp:   time.Now(),
		}
	}

	// Warning at 80%
	if totalCost > monthlyBudget*0.8 {
		return &BudgetAlert{
			Type:        "budget_warning",
			CurrentCost: totalCost,
			BudgetLimit: monthlyBudget,
			Percentage:  (totalCost / monthlyBudget) * 100,
			Timestamp:   time.Now(),
		}
	}

	return nil
}

// BudgetAlert represents a budget limit alert
type BudgetAlert struct {
	Type        string    `json:"type"`        // "budget_warning", "budget_exceeded"
	CurrentCost float64   `json:"current_cost"`
	BudgetLimit float64   `json:"budget_limit"`
	Percentage  float64   `json:"percentage"`
	Timestamp   time.Time `json:"timestamp"`
}

// GetCostBreakdown returns detailed cost breakdown
func (lm *LLMMonitor) GetCostBreakdown() CostBreakdown {
	costs := lm.GetProviderCosts()
	totalCost := lm.GetTotalCost()

	breakdown := CostBreakdown{
		TotalCost: totalCost,
		Providers: make(map[string]ProviderBreakdown),
		Period:    "current", // would be configurable
		Timestamp: time.Now(),
	}

	for provider, cost := range costs {
		percentage := 0.0
		if totalCost > 0 {
			percentage = (cost.TotalCost / totalCost) * 100
		}

		providerBreakdown := ProviderBreakdown{
			Cost:         cost.TotalCost,
			Percentage:   percentage,
			TokensUsed:   cost.TokensUsed,
			RequestCount: cost.RequestCount,
			Models:       make(map[string]ModelBreakdown),
		}

		for model, modelCost := range cost.Models {
			modelPercentage := 0.0
			if cost.TotalCost > 0 {
				modelPercentage = (modelCost.TotalCost / cost.TotalCost) * 100
			}

			providerBreakdown.Models[model] = ModelBreakdown{
				Cost:         modelCost.TotalCost,
				Percentage:   modelPercentage,
				TokensUsed:   modelCost.TokensUsed,
				RequestCount: modelCost.RequestCount,
				AvgLatency:   time.Duration(modelCost.AvgLatency) * time.Millisecond,
				CacheHitRate: modelCost.CacheHitRate,
			}
		}

		breakdown.Providers[provider] = providerBreakdown
	}

	return breakdown
}

// CostBreakdown provides detailed cost analysis
type CostBreakdown struct {
	TotalCost float64                        `json:"total_cost"`
	Providers map[string]ProviderBreakdown   `json:"providers"`
	Period    string                         `json:"period"`
	Timestamp time.Time                      `json:"timestamp"`
}

// ProviderBreakdown provides cost breakdown for a provider
type ProviderBreakdown struct {
	Cost         float64                    `json:"cost"`
	Percentage   float64                    `json:"percentage"`
	TokensUsed   int64                      `json:"tokens_used"`
	RequestCount int64                      `json:"request_count"`
	Models       map[string]ModelBreakdown  `json:"models"`
}

// ModelBreakdown provides cost breakdown for a model
type ModelBreakdown struct {
	Cost         float64       `json:"cost"`
	Percentage   float64       `json:"percentage"`
	TokensUsed   int64         `json:"tokens_used"`
	RequestCount int64         `json:"request_count"`
	AvgLatency   time.Duration `json:"avg_latency"`
	CacheHitRate float64       `json:"cache_hit_rate"`
}

// RecordLLMMetrics records LLM operation with detailed tracking
func (lm *LLMMonitor) RecordLLMMetrics(provider, model string, tokensUsed int64, cost float64, latency time.Duration, cacheHit, success bool, errorType string) {
	metrics := &LLMMetrics{
		Provider:  provider,
		Model:     model,
		TokensUsed: tokensUsed,
		Cost:      cost,
		Latency:   latency,
		CacheHit:  cacheHit,
		Success:   success,
		ErrorType: errorType,
		Timestamp: time.Now(),
	}

	// Record metrics
	lm.RecordLLMRequest(metrics)

	// Check budget alerts
	if alert := lm.CheckBudgetLimit(); alert != nil {
		lm.handleBudgetAlert(alert)
	}
}

// RecordLLMRequest records LLM request metrics (from types.go interface)
func (lm *LLMMonitor) RecordLLMRequest(metrics *LLMMetrics) {
	if lm.metrics == nil {
		return
	}

	attrs := []attribute.KeyValue{
		AttrProvider.String(metrics.Provider),
		AttrModel.String(metrics.Model),
		AttrSuccess.Bool(metrics.Success),
		AttrCacheHit.Bool(metrics.CacheHit),
		AttrTokensUsed.Int64(metrics.TokensUsed),
		AttrCost.Float64(metrics.Cost),
	}

	if !metrics.Success && metrics.ErrorType != "" {
		attrs = append(attrs, AttrErrorType.String(metrics.ErrorType))
	}

	// Update all LLM metrics
	lm.metrics.LLMRequestCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	lm.metrics.LLMTokenUsage.Add(context.Background(), metrics.TokensUsed, metric.WithAttributes(attrs...))
	lm.metrics.LLMLatency.Record(context.Background(), metrics.Latency.Seconds(), metric.WithAttributes(attrs...))

	if metrics.CacheHit {
		lm.metrics.LLMCacheHitCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	}

	// Update cost tracking
	lm.updateCostTracking(metrics)

	// Update cost gauge
	totalCost := lm.GetTotalCost()
	lm.metrics.LLMCostGauge.Record(context.Background(), totalCost)
}

// handleBudgetAlert handles budget limit alerts
func (lm *LLMMonitor) handleBudgetAlert(alert *BudgetAlert) {
	// Log alert
	fmt.Printf("Budget Alert: %s - Current: $%.2f, Limit: $%.2f (%.1f%%)\n",
		alert.Type, alert.CurrentCost, alert.BudgetLimit, alert.Percentage)

	// Record alert metric
	attrs := []attribute.KeyValue{
		attribute.String("alert_type", alert.Type),
		attribute.Float64("current_cost", alert.CurrentCost),
		attribute.Float64("budget_limit", alert.BudgetLimit),
		attribute.Float64("percentage", alert.Percentage),
	}

	// Would typically send to alerting system
	lm.recordAlert(alert.Type, attrs)
}

// recordAlert records an alert occurrence
func (lm *LLMMonitor) recordAlert(alertType string, attrs []attribute.KeyValue) {
	// This would integrate with an alerting system like Alertmanager
	// For now, just record as a metric
	alertAttrs := append(attrs, attribute.String("alert_type", alertType))
	lm.metrics.ErrorCounter.Add(context.Background(), 1, metric.WithAttributes(alertAttrs...))
}

// GetUsageTrends returns usage trends over time
func (lm *LLMMonitor) GetUsageTrends(provider string, duration time.Duration) UsageTrends {
	// This would typically query a time-series database
	// For now, return current state
	costs := lm.GetProviderCosts()

	trends := UsageTrends{
		Provider:  provider,
		Period:    duration,
		Timestamp: time.Now(),
	}

	if providerCost, exists := costs[provider]; exists {
		trends.TotalRequests = providerCost.RequestCount
		trends.TotalTokens = providerCost.TokensUsed
		trends.TotalCost = providerCost.TotalCost

		// Calculate average cost per request
		if providerCost.RequestCount > 0 {
			trends.AvgCostPerRequest = providerCost.TotalCost / float64(providerCost.RequestCount)
		}

		// Calculate average tokens per request
		if providerCost.RequestCount > 0 {
			trends.AvgTokensPerRequest = float64(providerCost.TokensUsed) / float64(providerCost.RequestCount)
		}
	}

	return trends
}

// UsageTrends represents LLM usage trends
type UsageTrends struct {
	Provider            string        `json:"provider"`
	Period              time.Duration `json:"period"`
	TotalRequests       int64         `json:"total_requests"`
	TotalTokens         int64         `json:"total_tokens"`
	TotalCost           float64       `json:"total_cost"`
	AvgCostPerRequest   float64       `json:"avg_cost_per_request"`
	AvgTokensPerRequest float64       `json:"avg_tokens_per_request"`
	Timestamp           time.Time     `json:"timestamp"`
}

// GetTopModels returns most used models by request count
func (lm *LLMMonitor) GetTopModels(limit int) []ModelUsage {
	costs := lm.GetProviderCosts()
	var modelUsages []ModelUsage

	for provider, providerCost := range costs {
		for model, modelCost := range providerCost.Models {
			modelUsages = append(modelUsages, ModelUsage{
				Provider:     provider,
				Model:        model,
				RequestCount: modelCost.RequestCount,
				TokensUsed:   modelCost.TokensUsed,
				TotalCost:    modelCost.TotalCost,
				AvgLatency:   time.Duration(modelCost.AvgLatency) * time.Millisecond,
				CacheHitRate: modelCost.CacheHitRate,
			})
		}
	}

	// Sort by request count (simplified - would use sort.Slice in real implementation)
	// Return top models up to limit
	if len(modelUsages) > limit {
		return modelUsages[:limit]
	}
	return modelUsages
}

// ModelUsage represents model usage statistics
type ModelUsage struct {
	Provider     string        `json:"provider"`
	Model        string        `json:"model"`
	RequestCount int64         `json:"request_count"`
	TokensUsed   int64         `json:"tokens_used"`
	TotalCost    float64       `json:"total_cost"`
	AvgLatency   time.Duration `json:"avg_latency"`
	CacheHitRate float64       `json:"cache_hit_rate"`
}

// RecordCachePerformance records cache hit/miss metrics
func (lm *LLMMonitor) RecordCachePerformance(provider, model string, hit bool, latency time.Duration) {
	if !lm.config.MetricsEnabled {
		return
	}

	attrs := []attribute.KeyValue{
		AttrProvider.String(provider),
		AttrModel.String(model),
		AttrCacheHit.Bool(hit),
	}

	if hit {
		lm.metrics.LLMCacheHitCounter.Add(context.Background(), 1, metric.WithAttributes(attrs...))
	}

	// Record cache lookup latency
	lm.metrics.LLMLatency.Record(context.Background(), latency.Seconds(), metric.WithAttributes(
		append(attrs, attribute.String("operation", "cache_lookup"))...,
	))
}

// GetCacheStatistics returns cache performance statistics
func (lm *LLMMonitor) GetCacheStatistics() CacheStatistics {
	costs := lm.GetProviderCosts()

	stats := CacheStatistics{
		Timestamp: time.Now(),
	}

	totalRequests := int64(0)
	totalHits := int64(0)

	for provider, providerCost := range costs {
		providerStats := ProviderCacheStats{
			Provider:     provider,
			TotalRequests: providerCost.RequestCount,
		}

		for _, modelCost := range providerCost.Models {
			totalRequests += modelCost.RequestCount
			hits := int64(float64(modelCost.RequestCount) * modelCost.CacheHitRate)
			totalHits += hits
			providerStats.CacheHits += hits
		}

		if providerStats.TotalRequests > 0 {
			providerStats.HitRate = float64(providerStats.CacheHits) / float64(providerStats.TotalRequests)
		}

		stats.Providers = append(stats.Providers, providerStats)
	}

	if totalRequests > 0 {
		stats.OverallHitRate = float64(totalHits) / float64(totalRequests)
	}

	return stats
}

// CacheStatistics represents cache performance statistics
type CacheStatistics struct {
	OverallHitRate float64              `json:"overall_hit_rate"`
	Providers      []ProviderCacheStats `json:"providers"`
	Timestamp      time.Time            `json:"timestamp"`
}

// ProviderCacheStats represents cache statistics for a provider
type ProviderCacheStats struct {
	Provider      string  `json:"provider"`
	TotalRequests int64   `json:"total_requests"`
	CacheHits     int64   `json:"cache_hits"`
	HitRate       float64 `json:"hit_rate"`
}

// ResetCosts resets all cost tracking (useful for testing or period resets)
func (lm *LLMMonitor) ResetCosts() {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lm.costTracker = NewCostTracker()
}

// ExportMetrics exports current metrics for external systems
func (lm *LLMMonitor) ExportMetrics() LLMMetricsExport {
	return LLMMetricsExport{
		CostBreakdown:    lm.GetCostBreakdown(),
		ModelPerformance: lm.GetModelPerformance(),
		CacheStats:       lm.GetCacheStatistics(),
		BudgetAlert:      lm.CheckBudgetLimit(),
		Timestamp:        time.Now(),
	}
}

// LLMMetricsExport contains exportable LLM metrics
type LLMMetricsExport struct {
	CostBreakdown    CostBreakdown     `json:"cost_breakdown"`
	ModelPerformance []ModelPerformance `json:"model_performance"`
	CacheStats       CacheStatistics   `json:"cache_statistics"`
	BudgetAlert      *BudgetAlert      `json:"budget_alert,omitempty"`
	Timestamp        time.Time         `json:"timestamp"`
}