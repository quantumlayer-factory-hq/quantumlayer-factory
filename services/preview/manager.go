package preview

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/quantumlayer-factory-hq/quantumlayer-factory/services/builder"
	"github.com/quantumlayer-factory-hq/quantumlayer-factory/services/deploy"
)

// PreviewManagerImpl implements the PreviewManager interface
type PreviewManagerImpl struct {
	config        *PreviewConfig
	builder       builder.Builder
	deployer      deploy.Deployer
	urlManager    URLManager
	tlsProvider   TLSProvider
	healthMonitor HealthMonitor
	analytics     AnalyticsTracker
	storage       PreviewStorage
	mutex         sync.RWMutex
}

// NewPreviewManager creates a new preview manager
func NewPreviewManager(
	config *PreviewConfig,
	builder builder.Builder,
	deployer deploy.Deployer,
	urlManager URLManager,
	tlsProvider TLSProvider,
	healthMonitor HealthMonitor,
	analytics AnalyticsTracker,
	storage PreviewStorage,
) *PreviewManagerImpl {
	return &PreviewManagerImpl{
		config:        config,
		builder:       builder,
		deployer:      deployer,
		urlManager:    urlManager,
		tlsProvider:   tlsProvider,
		healthMonitor: healthMonitor,
		analytics:     analytics,
		storage:       storage,
	}
}

// Create creates a new preview environment
func (pm *PreviewManagerImpl) Create(ctx context.Context, req *PreviewRequest) (*PreviewResult, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	// Generate ID if not provided
	if req.ID == "" {
		var err error
		req.ID, err = pm.generatePreviewID()
		if err != nil {
			return nil, fmt.Errorf("failed to generate preview ID: %w", err)
		}
	}

	// Validate request
	if err := pm.validateRequest(req); err != nil {
		return nil, fmt.Errorf("invalid request: %w", err)
	}

	result := &PreviewResult{
		ID:        req.ID,
		AppName:   req.AppName,
		Status:    PreviewStatus{Phase: "Creating", Progress: 0, LastUpdated: time.Now()},
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
		ExpiresAt: time.Now().Add(req.TTL),
		TTL:       req.TTL,
		Warnings:  []string{},
		Errors:    []string{},
		Metadata:  make(map[string]interface{}),
	}

	// Store initial state
	if err := pm.storage.Store(ctx, result); err != nil {
		return result, fmt.Errorf("failed to store preview: %w", err)
	}

	// Generate subdomain
	var subdomain string
	var err error
	if req.Subdomain != "" {
		// Validate custom subdomain
		if err := pm.urlManager.ValidateSubdomain(req.Subdomain); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Invalid subdomain: %v", err))
			return result, err
		}
		subdomain = req.Subdomain
	} else {
		subdomain, err = pm.urlManager.GenerateSubdomain(req.AppName)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to generate subdomain: %v", err))
			return result, err
		}
	}

	// Reserve subdomain
	err = pm.urlManager.ReserveSubdomain(ctx, subdomain, req.ID, req.TTL)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to reserve subdomain: %v", err))
		return result, err
	}

	// Generate URLs
	result.URL = pm.urlManager.GetURL(subdomain, req.TLS)

	// Start async build and deploy process
	go pm.buildAndDeploy(context.Background(), req, result, subdomain)

	return result, nil
}

// buildAndDeploy handles the async build and deploy process
func (pm *PreviewManagerImpl) buildAndDeploy(ctx context.Context, req *PreviewRequest, result *PreviewResult, subdomain string) {
	// Update status: Building
	pm.updateStatus(ctx, result.ID, PreviewStatus{
		Phase:       "Building",
		Progress:    10,
		Message:     "Building container image",
		LastUpdated: time.Now(),
	})

	// Build container
	buildReq := &builder.BuildRequest{
		ProjectPath:    req.ProjectPath,
		Language:       req.Language,
		Framework:      req.Framework,
		ImageName:      fmt.Sprintf("preview-%s", req.ID),
		ImageTag:       "latest",
		SecurityScan:   true,
		PushToRegistry: false,
	}

	buildResult, err := pm.builder.Build(ctx, buildReq)
	if err != nil {
		pm.updateStatusWithError(ctx, result.ID, "Building", fmt.Sprintf("Build failed: %v", err))
		return
	}

	// Update build result
	result.BuildResult = &BuildInfo{
		ImageID:     buildResult.ImageID,
		ImageName:   buildResult.ImageName,
		ImageTag:    buildResult.ImageTag,
		BuildTime:   buildResult.BuildTime,
		ImageSize:   buildResult.ImageSize,
	}

	if buildResult.SecurityScan != nil {
		result.BuildResult.SecurityScan = &SecurityInfo{
			Scanner:    buildResult.SecurityScan.Scanner,
			TotalVulns: buildResult.SecurityScan.TotalVulns,
			Critical:   buildResult.SecurityScan.Critical,
			High:       buildResult.SecurityScan.High,
			Medium:     buildResult.SecurityScan.Medium,
			Low:        buildResult.SecurityScan.Low,
			Passed:     buildResult.SecurityScan.Passed,
		}
	}

	pm.storage.Store(ctx, result)

	// Update status: Deploying
	pm.updateStatus(ctx, result.ID, PreviewStatus{
		Phase:       "Deploying",
		Progress:    50,
		Message:     "Deploying to Kubernetes",
		LastUpdated: time.Now(),
	})

	// Deploy to Kubernetes
	namespace := fmt.Sprintf("preview-%s", req.ID)
	deployReq := &deploy.DeployRequest{
		AppName:   req.AppName,
		Namespace: namespace,
		ImageName: buildResult.ImageName,
		ImageTag:  buildResult.ImageTag,
		Port:      req.Port,
		Replicas:  1,
		Environment: req.Environment,
		TTL:       req.TTL,
		Ingress: &deploy.IngressConfig{
			Enabled: true,
			Host:    fmt.Sprintf("%s.%s", subdomain, pm.config.BaseDomain),
			Path:    "/",
			TLS:     req.TLS,
			Annotations: pm.config.LoadBalancer.Annotations,
		},
		Labels: map[string]string{
			"app.kubernetes.io/managed-by": "quantumlayer-factory",
			"quantumlayer.dev/preview-id":  req.ID,
			"quantumlayer.dev/app":         req.AppName,
		},
	}

	deployResult, err := pm.deployer.Deploy(ctx, deployReq)
	if err != nil {
		pm.updateStatusWithError(ctx, result.ID, "Deploying", fmt.Sprintf("Deploy failed: %v", err))
		return
	}

	// Update deploy result
	result.DeployResult = &DeployInfo{
		Namespace:      deployResult.Namespace,
		DeploymentName: deployResult.DeploymentName,
		ServiceName:    deployResult.ServiceName,
		IngressName:    deployResult.IngressName,
		Replicas:       deployResult.Status.Replicas,
		ReadyReplicas:  deployResult.Status.ReadyReplicas,
	}

	result.InternalURL = deployResult.InternalURL
	pm.storage.Store(ctx, result)

	// Provision TLS certificate if needed
	if req.TLS && pm.tlsProvider != nil {
		domain := fmt.Sprintf("%s.%s", subdomain, pm.config.BaseDomain)
		_, err := pm.tlsProvider.ProvisionCertificate(ctx, domain)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("TLS certificate provisioning failed: %v", err))
		}
	}

	// Update status: Running
	pm.updateStatus(ctx, result.ID, PreviewStatus{
		Phase:       "Running",
		Progress:    90,
		Message:     "Preview is running",
		LastUpdated: time.Now(),
	})

	// Start health monitoring
	if pm.healthMonitor != nil {
		err = pm.healthMonitor.StartMonitoring(ctx, req.ID, result.URL)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Health monitoring failed to start: %v", err))
		}
	}

	// Final status update
	pm.updateStatus(ctx, result.ID, PreviewStatus{
		Phase:       "Running",
		Progress:    100,
		Message:     "Preview is ready",
		LastUpdated: time.Now(),
	})

	pm.storage.Store(ctx, result)
}

// Get retrieves a preview by ID
func (pm *PreviewManagerImpl) Get(ctx context.Context, id string) (*PreviewResult, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	result, err := pm.storage.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get preview: %w", err)
	}

	// Update health status if monitoring is enabled
	if pm.healthMonitor != nil && result.Status.Phase == "Running" {
		healthStatus, err := pm.healthMonitor.GetHealthStatus(ctx, id)
		if err == nil {
			result.HealthCheck = healthStatus
		}
	}

	// Update analytics if enabled
	if pm.analytics != nil {
		analyticsInfo, err := pm.analytics.GetAnalytics(ctx, id)
		if err == nil {
			result.Analytics = analyticsInfo
		}
	}

	return result, nil
}

// List lists all previews
func (pm *PreviewManagerImpl) List(ctx context.Context) ([]*PreviewResult, error) {
	pm.mutex.RLock()
	defer pm.mutex.RUnlock()

	return pm.storage.List(ctx)
}

// Update updates a preview
func (pm *PreviewManagerImpl) Update(ctx context.Context, id string, updates map[string]interface{}) (*PreviewResult, error) {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	result, err := pm.storage.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get preview: %w", err)
	}

	// Apply updates
	for key, value := range updates {
		switch key {
		case "ttl":
			if ttl, ok := value.(time.Duration); ok {
				if ttl > pm.config.MaxTTL {
					return nil, fmt.Errorf("TTL exceeds maximum allowed: %v", pm.config.MaxTTL)
				}
				result.TTL = ttl
				result.ExpiresAt = time.Now().Add(ttl)
			}
		case "metadata":
			if metadata, ok := value.(map[string]interface{}); ok {
				for k, v := range metadata {
					result.Metadata[k] = v
				}
			}
		default:
			result.Metadata[key] = value
		}
	}

	result.UpdatedAt = time.Now()

	return result, pm.storage.Store(ctx, result)
}

// Delete removes a preview
func (pm *PreviewManagerImpl) Delete(ctx context.Context, id string) error {
	pm.mutex.Lock()
	defer pm.mutex.Unlock()

	result, err := pm.storage.Get(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to get preview: %w", err)
	}

	// Stop health monitoring
	if pm.healthMonitor != nil {
		pm.healthMonitor.StopMonitoring(ctx, id)
	}

	// Delete from Kubernetes
	if result.DeployResult != nil {
		err = pm.deployer.Delete(ctx, result.DeployResult.Namespace, result.AppName)
		if err != nil {
			return fmt.Errorf("failed to delete deployment: %w", err)
		}
	}

	// Release subdomain
	if result.URL != "" {
		parts := strings.Split(result.URL, ".")
		if len(parts) > 0 {
			subdomain := strings.TrimPrefix(parts[0], "https://")
			subdomain = strings.TrimPrefix(subdomain, "http://")
			pm.urlManager.ReleaseSubdomain(ctx, subdomain)
		}
	}

	// Revoke TLS certificate
	if pm.tlsProvider != nil && result.URL != "" {
		domain := strings.TrimPrefix(result.URL, "https://")
		domain = strings.TrimPrefix(domain, "http://")
		pm.tlsProvider.RevokeCertificate(ctx, domain)
	}

	// Remove from storage
	return pm.storage.Delete(ctx, id)
}

// Extend extends preview TTL
func (pm *PreviewManagerImpl) Extend(ctx context.Context, id string, ttl time.Duration) error {
	if ttl > pm.config.MaxTTL {
		return fmt.Errorf("TTL exceeds maximum allowed: %v", pm.config.MaxTTL)
	}

	updates := map[string]interface{}{
		"ttl": ttl,
	}

	_, err := pm.Update(ctx, id, updates)
	return err
}

// GetLogs retrieves preview logs
func (pm *PreviewManagerImpl) GetLogs(ctx context.Context, id string, lines int) ([]string, error) {
	result, err := pm.storage.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get preview: %w", err)
	}

	if result.DeployResult == nil {
		return []string{}, nil
	}

	return pm.deployer.GetLogs(ctx, result.DeployResult.Namespace, result.AppName, int64(lines))
}

// GetStatus gets detailed status
func (pm *PreviewManagerImpl) GetStatus(ctx context.Context, id string) (*PreviewStatus, error) {
	result, err := pm.storage.Get(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("failed to get preview: %w", err)
	}

	// Get deployment status if available
	if result.DeployResult != nil {
		deployStatus, err := pm.deployer.GetStatus(ctx, result.DeployResult.Namespace, result.AppName)
		if err == nil {
			result.Status.Message = deployStatus.Status.Message
			if deployStatus.Status.ReadyReplicas > 0 {
				result.Status.Phase = "Running"
				result.Status.Progress = 100
			}
		}
	}

	return &result.Status, nil
}

// CleanupExpired removes expired previews
func (pm *PreviewManagerImpl) CleanupExpired(ctx context.Context) error {
	previews, err := pm.List(ctx)
	if err != nil {
		return fmt.Errorf("failed to list previews: %w", err)
	}

	now := time.Now()
	var errors []string

	for _, preview := range previews {
		if now.After(preview.ExpiresAt) {
			err := pm.Delete(ctx, preview.ID)
			if err != nil {
				errors = append(errors, fmt.Sprintf("failed to delete expired preview %s: %v", preview.ID, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// Helper functions

func (pm *PreviewManagerImpl) generatePreviewID() (string, error) {
	bytes := make([]byte, 8)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}
	return hex.EncodeToString(bytes), nil
}

func (pm *PreviewManagerImpl) validateRequest(req *PreviewRequest) error {
	if req.AppName == "" {
		return fmt.Errorf("app name is required")
	}

	if req.ProjectPath == "" {
		return fmt.Errorf("project path is required")
	}

	if req.Language == "" {
		return fmt.Errorf("language is required")
	}

	if req.Port <= 0 {
		return fmt.Errorf("port must be positive")
	}

	if req.TTL <= 0 {
		req.TTL = pm.config.DefaultTTL
	}

	if req.TTL > pm.config.MaxTTL {
		return fmt.Errorf("TTL exceeds maximum allowed: %v", pm.config.MaxTTL)
	}

	return nil
}

func (pm *PreviewManagerImpl) updateStatus(ctx context.Context, id string, status PreviewStatus) {
	result, err := pm.storage.Get(ctx, id)
	if err != nil {
		return
	}

	result.Status = status
	result.UpdatedAt = time.Now()
	pm.storage.Store(ctx, result)
}

func (pm *PreviewManagerImpl) updateStatusWithError(ctx context.Context, id, phase, errorMsg string) {
	result, err := pm.storage.Get(ctx, id)
	if err != nil {
		return
	}

	result.Status = PreviewStatus{
		Phase:       "Failed",
		Progress:    0,
		Message:     errorMsg,
		LastUpdated: time.Now(),
	}
	result.Errors = append(result.Errors, errorMsg)
	result.UpdatedAt = time.Now()
	pm.storage.Store(ctx, result)
}

// StartCleanupScheduler starts a goroutine that periodically cleans up expired previews
func (pm *PreviewManagerImpl) StartCleanupScheduler(ctx context.Context) {
	ticker := time.NewTicker(pm.config.CleanupInterval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := pm.CleanupExpired(ctx); err != nil {
					// Log error (in production, use proper logging)
					fmt.Printf("Preview cleanup error: %v\n", err)
				}
			}
		}
	}()
}

// PreviewStorage interface defines storage operations for previews
type PreviewStorage interface {
	Store(ctx context.Context, result *PreviewResult) error
	Get(ctx context.Context, id string) (*PreviewResult, error)
	List(ctx context.Context) ([]*PreviewResult, error)
	Delete(ctx context.Context, id string) error
}