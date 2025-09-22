package llm

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BatchRequest represents a single request in a batch
type BatchRequest struct {
	ID      string
	Request *GenerateRequest
	Result  chan *BatchResult
}

// BatchResult contains the response and any error for a batched request
type BatchResult struct {
	Response *GenerateResponse
	Error    error
}

// BatchConfig configures batch processing behavior
type BatchConfig struct {
	MaxBatchSize   int           `json:"max_batch_size"`
	MaxWaitTime    time.Duration `json:"max_wait_time"`
	FlushInterval  time.Duration `json:"flush_interval"`
	EnableBatching bool          `json:"enable_batching"`
}

// DefaultBatchConfig returns sensible defaults for batch processing
func DefaultBatchConfig() BatchConfig {
	return BatchConfig{
		MaxBatchSize:   5,  // Process up to 5 requests together
		MaxWaitTime:    2 * time.Second,
		FlushInterval:  500 * time.Millisecond,
		EnableBatching: true,
	}
}

// BatchClient wraps an LLM client with request batching capabilities
type BatchClient struct {
	client      Client
	config      BatchConfig
	requestChan chan *BatchRequest
	stopChan    chan struct{}
	wg          sync.WaitGroup
	mu          sync.RWMutex
	running     bool
}

// NewBatchClient creates a new batching client wrapper
func NewBatchClient(client Client, config BatchConfig) *BatchClient {
	bc := &BatchClient{
		client:      client,
		config:      config,
		requestChan: make(chan *BatchRequest, config.MaxBatchSize*2),
		stopChan:    make(chan struct{}),
	}

	if config.EnableBatching {
		bc.start()
	}

	return bc
}

// start begins the batch processing goroutine
func (bc *BatchClient) start() {
	bc.mu.Lock()
	if bc.running {
		bc.mu.Unlock()
		return
	}
	bc.running = true
	bc.mu.Unlock()

	bc.wg.Add(1)
	go bc.batchProcessor()
}

// stop shuts down the batch processor
func (bc *BatchClient) stop() {
	bc.mu.Lock()
	if !bc.running {
		bc.mu.Unlock()
		return
	}
	bc.running = false
	bc.mu.Unlock()

	close(bc.stopChan)
	bc.wg.Wait()
}

// Generate implements the Client interface with batching
func (bc *BatchClient) Generate(ctx context.Context, req *GenerateRequest) (*GenerateResponse, error) {
	// Skip batching if disabled or for streaming requests
	if !bc.config.EnableBatching {
		return bc.client.Generate(ctx, req)
	}

	// Create batch request
	batchReq := &BatchRequest{
		ID:      fmt.Sprintf("batch-%d", time.Now().UnixNano()),
		Request: req,
		Result:  make(chan *BatchResult, 1),
	}

	// Submit to batch processor
	select {
	case bc.requestChan <- batchReq:
		// Wait for result
		select {
		case result := <-batchReq.Result:
			return result.Response, result.Error
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		// Channel full, process immediately
		return bc.client.Generate(ctx, req)
	}
}

// batchProcessor runs the main batching loop
func (bc *BatchClient) batchProcessor() {
	defer bc.wg.Done()

	var batch []*BatchRequest
	flushTimer := time.NewTimer(bc.config.FlushInterval)
	defer flushTimer.Stop()

	for {
		select {
		case req := <-bc.requestChan:
			batch = append(batch, req)

			// Process batch if it's full
			if len(batch) >= bc.config.MaxBatchSize {
				bc.processBatch(batch)
				batch = nil
				flushTimer.Reset(bc.config.FlushInterval)
			}

		case <-flushTimer.C:
			// Process any pending requests
			if len(batch) > 0 {
				bc.processBatch(batch)
				batch = nil
			}
			flushTimer.Reset(bc.config.FlushInterval)

		case <-bc.stopChan:
			// Process any remaining requests before shutting down
			if len(batch) > 0 {
				bc.processBatch(batch)
			}
			return
		}
	}
}

// processBatch handles a batch of requests
func (bc *BatchClient) processBatch(batch []*BatchRequest) {
	if len(batch) == 0 {
		return
	}

	// For now, process requests in parallel within the batch
	// In the future, this could be optimized for specific providers that support
	// true batch APIs (like OpenAI's batch endpoint)

	var wg sync.WaitGroup
	for _, req := range batch {
		wg.Add(1)
		go func(batchReq *BatchRequest) {
			defer wg.Done()
			bc.processSingleRequest(batchReq)
		}(req)
	}
	wg.Wait()
}

// processSingleRequest processes a single request within a batch
func (bc *BatchClient) processSingleRequest(batchReq *BatchRequest) {
	ctx, cancel := context.WithTimeout(context.Background(), bc.config.MaxWaitTime)
	defer cancel()

	response, err := bc.client.Generate(ctx, batchReq.Request)

	result := &BatchResult{
		Response: response,
		Error:    err,
	}

	// Send result back to waiting goroutine
	select {
	case batchReq.Result <- result:
	default:
		// Channel might be closed if context was cancelled
	}
}

// GenerateStream implements the Client interface
// Note: Streaming is not batched as it's inherently real-time
func (bc *BatchClient) GenerateStream(ctx context.Context, req *GenerateRequest) (<-chan *StreamChunk, error) {
	return bc.client.GenerateStream(ctx, req)
}

// GetProvider returns the provider type from the wrapped client
func (bc *BatchClient) GetProvider() Provider {
	return bc.client.GetProvider()
}

// GetModels returns available models from the wrapped client
func (bc *BatchClient) GetModels() []Model {
	return bc.client.GetModels()
}

// Close closes the batch processor and the wrapped client
func (bc *BatchClient) Close() error {
	bc.stop()
	return bc.client.Close()
}

// Health checks the health of the wrapped client
func (bc *BatchClient) Health(ctx context.Context) error {
	return bc.client.Health(ctx)
}

// GetBatchStats returns statistics about batch processing
func (bc *BatchClient) GetBatchStats() BatchStats {
	// TODO: Implement batch statistics tracking
	return BatchStats{
		TotalBatches:    0,
		TotalRequests:   0,
		AverageBatchSize: 0,
		ProcessingTime:  0,
	}
}

// BatchStats tracks batch processing metrics
type BatchStats struct {
	TotalBatches     int64         `json:"total_batches"`
	TotalRequests    int64         `json:"total_requests"`
	AverageBatchSize float64       `json:"average_batch_size"`
	ProcessingTime   time.Duration `json:"processing_time"`
}