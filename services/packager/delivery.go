package packager

import (
	"context"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// DeliveryService manages package delivery channels
type DeliveryService struct {
	config   *DeliveryConfig
	channels map[string]DeliveryChannel
}

// DeliveryConfig contains configuration for delivery service
type DeliveryConfig struct {
	DefaultChannel   string            `json:"default_channel"`
	RetryAttempts    int               `json:"retry_attempts"`
	RetryDelay       time.Duration     `json:"retry_delay"`
	Timeout          time.Duration     `json:"timeout"`
	VerifyChecksums  bool              `json:"verify_checksums"`
	EnableCompression bool             `json:"enable_compression"`
	CustomHeaders    map[string]string `json:"custom_headers"`
}

// RegistryConfig contains configuration for container registry
type RegistryConfig struct {
	URL       string `json:"url"`
	Username  string `json:"username"`
	Password  string `json:"password"`
	Namespace string `json:"namespace"`
	TLS       bool   `json:"tls"`
}

// CDNConfig contains configuration for CDN delivery
type CDNConfig struct {
	URL         string `json:"url"`
	APIKey      string `json:"api_key"`
	BucketName  string `json:"bucket_name"`
	Region      string `json:"region"`
	PublicRead  bool   `json:"public_read"`
	CacheTTL    int    `json:"cache_ttl"`
}

// DirectConfig contains configuration for direct delivery
type DirectConfig struct {
	BaseURL     string `json:"base_url"`
	StoragePath string `json:"storage_path"`
	ServeHTTP   bool   `json:"serve_http"`
	HTTPPort    int    `json:"http_port"`
}

// PackageManagerConfig contains configuration for package managers
type PackageManagerConfig struct {
	Type       string `json:"type"`        // "npm", "pip", "maven", "cargo"
	Repository string `json:"repository"`
	APIKey     string `json:"api_key"`
	Namespace  string `json:"namespace"`
}

// DeliveryResult represents the result of package delivery
type DeliveryResult struct {
	Channel     string            `json:"channel"`
	URL         string            `json:"url"`
	Checksum    string            `json:"checksum"`
	Size        int64             `json:"size"`
	DeliveredAt time.Time         `json:"delivered_at"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

// NewDeliveryService creates a new delivery service
func NewDeliveryService(config *DeliveryConfig) *DeliveryService {
	if config == nil {
		config = DefaultDeliveryConfig()
	}

	return &DeliveryService{
		config:   config,
		channels: make(map[string]DeliveryChannel),
	}
}

// DefaultDeliveryConfig returns default delivery configuration
func DefaultDeliveryConfig() *DeliveryConfig {
	return &DeliveryConfig{
		DefaultChannel:    "direct",
		RetryAttempts:     3,
		RetryDelay:        time.Second * 5,
		Timeout:           time.Minute * 10,
		VerifyChecksums:   true,
		EnableCompression: true,
		CustomHeaders:     make(map[string]string),
	}
}

// RegisterChannel registers a delivery channel
func (ds *DeliveryService) RegisterChannel(name string, channel DeliveryChannel) {
	ds.channels[name] = channel
}

// GetChannel retrieves a delivery channel by name
func (ds *DeliveryService) GetChannel(name string) (DeliveryChannel, bool) {
	channel, exists := ds.channels[name]
	return channel, exists
}

// DeliverPackage delivers a package through the specified channel
func (ds *DeliveryService) DeliverPackage(ctx context.Context, packagePath string, channelName string, options map[string]interface{}) (*DeliveryResult, error) {
	channel, exists := ds.GetChannel(channelName)
	if !exists {
		return nil, fmt.Errorf("delivery channel not found: %s", channelName)
	}

	if !channel.Enabled {
		return nil, fmt.Errorf("delivery channel disabled: %s", channelName)
	}

	// Calculate checksum
	checksum, size, err := ds.calculateChecksum(packagePath)
	if err != nil {
		return nil, fmt.Errorf("failed to calculate checksum: %w", err)
	}

	var result *DeliveryResult
	var lastErr error

	// Retry delivery
	for attempt := 0; attempt <= ds.config.RetryAttempts; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return nil, ctx.Err()
			case <-time.After(ds.config.RetryDelay):
			}
		}

		result, lastErr = ds.deliverToChannel(ctx, packagePath, channel, options)
		if lastErr == nil {
			result.Checksum = checksum
			result.Size = size
			result.DeliveredAt = time.Now()
			result.Channel = channelName
			return result, nil
		}
	}

	return nil, fmt.Errorf("failed to deliver after %d attempts: %w", ds.config.RetryAttempts, lastErr)
}

// DeliverToMultipleChannels delivers a package to multiple channels
func (ds *DeliveryService) DeliverToMultipleChannels(ctx context.Context, packagePath string, channelNames []string, options map[string]interface{}) (map[string]*DeliveryResult, map[string]error) {
	results := make(map[string]*DeliveryResult)
	errors := make(map[string]error)

	for _, channelName := range channelNames {
		result, err := ds.DeliverPackage(ctx, packagePath, channelName, options)
		if err != nil {
			errors[channelName] = err
		} else {
			results[channelName] = result
		}
	}

	return results, errors
}

// Helper methods

func (ds *DeliveryService) deliverToChannel(ctx context.Context, packagePath string, channel DeliveryChannel, options map[string]interface{}) (*DeliveryResult, error) {
	switch channel.Type {
	case "registry":
		return ds.deliverToRegistry(ctx, packagePath, channel, options)
	case "cdn":
		return ds.deliverToCDN(ctx, packagePath, channel, options)
	case "direct":
		return ds.deliverDirect(ctx, packagePath, channel, options)
	case "package_manager":
		return ds.deliverToPackageManager(ctx, packagePath, channel, options)
	default:
		return nil, fmt.Errorf("unsupported delivery channel type: %s", channel.Type)
	}
}

func (ds *DeliveryService) deliverToRegistry(ctx context.Context, packagePath string, channel DeliveryChannel, options map[string]interface{}) (*DeliveryResult, error) {
	// Mock implementation for container registry delivery
	packageName := filepath.Base(packagePath)
	packageName = strings.TrimSuffix(packageName, ".qlcapsule")

	// In a real implementation, this would:
	// 1. Convert .qlcapsule to OCI image format
	// 2. Push to registry using Docker API
	// 3. Return the registry URL

	url := fmt.Sprintf("%s/%s:latest", channel.Endpoint, packageName)

	result := &DeliveryResult{
		URL: url,
		Metadata: map[string]string{
			"registry": channel.Endpoint,
			"tag":      "latest",
		},
	}

	return result, nil
}

func (ds *DeliveryService) deliverToCDN(ctx context.Context, packagePath string, channel DeliveryChannel, options map[string]interface{}) (*DeliveryResult, error) {
	// Mock implementation for CDN delivery
	packageName := filepath.Base(packagePath)

	// In a real implementation, this would:
	// 1. Upload to S3/GCS/Azure Storage
	// 2. Configure CDN distribution
	// 3. Return the CDN URL

	url := fmt.Sprintf("%s/%s", channel.Endpoint, packageName)

	result := &DeliveryResult{
		URL: url,
		Metadata: map[string]string{
			"cdn":    channel.Endpoint,
			"bucket": "packages",
		},
	}

	return result, nil
}

func (ds *DeliveryService) deliverDirect(ctx context.Context, packagePath string, channel DeliveryChannel, options map[string]interface{}) (*DeliveryResult, error) {
	// Direct delivery - copy to local storage and serve via HTTP
	packageName := filepath.Base(packagePath)

	// Get storage path from channel config
	storagePath := "/tmp/packages" // Default
	if storagePathValue, exists := channel.Config["storage_path"]; exists {
		if sp, ok := storagePathValue.(string); ok {
			storagePath = sp
		}
	}

	// Ensure storage directory exists
	if err := os.MkdirAll(storagePath, 0755); err != nil {
		return nil, fmt.Errorf("failed to create storage directory: %w", err)
	}

	// Copy package to storage
	destPath := filepath.Join(storagePath, packageName)
	if err := ds.copyFile(packagePath, destPath); err != nil {
		return nil, fmt.Errorf("failed to copy package: %w", err)
	}

	// Generate URL
	url := fmt.Sprintf("%s/%s", channel.Endpoint, packageName)

	result := &DeliveryResult{
		URL: url,
		Metadata: map[string]string{
			"storage_path": destPath,
			"method":       "direct",
		},
	}

	return result, nil
}

func (ds *DeliveryService) deliverToPackageManager(ctx context.Context, packagePath string, channel DeliveryChannel, options map[string]interface{}) (*DeliveryResult, error) {
	// Mock implementation for package manager delivery
	packageName := filepath.Base(packagePath)
	packageName = strings.TrimSuffix(packageName, ".qlcapsule")

	// Get package manager type from channel config
	pmType := "generic"
	if pmTypeValue, exists := channel.Config["type"]; exists {
		if pt, ok := pmTypeValue.(string); ok {
			pmType = pt
		}
	}

	// In a real implementation, this would:
	// 1. Convert .qlcapsule to package manager format (npm, pip, etc.)
	// 2. Upload to package repository
	// 3. Return the package URL

	var url string
	switch pmType {
	case "npm":
		url = fmt.Sprintf("https://npmjs.com/package/%s", packageName)
	case "pip":
		url = fmt.Sprintf("https://pypi.org/project/%s", packageName)
	case "maven":
		url = fmt.Sprintf("https://mvnrepository.com/artifact/%s", packageName)
	case "cargo":
		url = fmt.Sprintf("https://crates.io/crates/%s", packageName)
	default:
		url = fmt.Sprintf("%s/%s", channel.Endpoint, packageName)
	}

	result := &DeliveryResult{
		URL: url,
		Metadata: map[string]string{
			"package_manager": pmType,
			"repository":      channel.Endpoint,
		},
	}

	return result, nil
}

// Utility methods

func (ds *DeliveryService) calculateChecksum(filePath string) (string, int64, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", 0, err
	}
	defer file.Close()

	hasher := sha256.New()
	size, err := io.Copy(hasher, file)
	if err != nil {
		return "", 0, err
	}

	checksum := fmt.Sprintf("%x", hasher.Sum(nil))
	return checksum, size, nil
}

func (ds *DeliveryService) copyFile(src, dst string) error {
	srcFile, err := os.Open(src)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	dstFile, err := os.Create(dst)
	if err != nil {
		return err
	}
	defer dstFile.Close()

	_, err = io.Copy(dstFile, srcFile)
	return err
}

// HTTP Server for direct delivery

// StartHTTPServer starts an HTTP server for direct package delivery
func (ds *DeliveryService) StartHTTPServer(storagePath string, port int) error {
	http.HandleFunc("/packages/", func(w http.ResponseWriter, r *http.Request) {
		// Extract package name from URL
		packageName := strings.TrimPrefix(r.URL.Path, "/packages/")
		if packageName == "" {
			http.Error(w, "Package name required", http.StatusBadRequest)
			return
		}

		// Validate package name
		if !ds.isValidPackageName(packageName) {
			http.Error(w, "Invalid package name", http.StatusBadRequest)
			return
		}

		// Construct file path
		filePath := filepath.Join(storagePath, packageName)

		// Check if file exists
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			http.Error(w, "Package not found", http.StatusNotFound)
			return
		}

		// Serve file
		w.Header().Set("Content-Type", "application/octet-stream")
		w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", packageName))

		// Add custom headers
		for key, value := range ds.config.CustomHeaders {
			w.Header().Set(key, value)
		}

		http.ServeFile(w, r, filePath)
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"service": "QuantumLayer Package Delivery", "version": "1.0"}`))
	})

	addr := fmt.Sprintf(":%d", port)
	fmt.Printf("Starting package delivery server on %s\n", addr)
	return http.ListenAndServe(addr, nil)
}

func (ds *DeliveryService) isValidPackageName(name string) bool {
	// Basic validation for package names
	if name == "" || strings.Contains(name, "..") || strings.Contains(name, "/") {
		return false
	}

	// Must end with .qlcapsule
	return strings.HasSuffix(name, ".qlcapsule")
}

// Channel management methods

// CreateRegistryChannel creates a registry delivery channel
func CreateRegistryChannel(name string, config RegistryConfig) DeliveryChannel {
	return DeliveryChannel{
		Name:     name,
		Type:     "registry",
		Endpoint: config.URL,
		Credentials: map[string]string{
			"username": config.Username,
			"password": config.Password,
		},
		Config: map[string]interface{}{
			"namespace": config.Namespace,
			"tls":       config.TLS,
		},
		Enabled: true,
	}
}

// CreateCDNChannel creates a CDN delivery channel
func CreateCDNChannel(name string, config CDNConfig) DeliveryChannel {
	return DeliveryChannel{
		Name:     name,
		Type:     "cdn",
		Endpoint: config.URL,
		Credentials: map[string]string{
			"api_key": config.APIKey,
		},
		Config: map[string]interface{}{
			"bucket_name": config.BucketName,
			"region":      config.Region,
			"public_read": config.PublicRead,
			"cache_ttl":   config.CacheTTL,
		},
		Enabled: true,
	}
}

// CreateDirectChannel creates a direct delivery channel
func CreateDirectChannel(name string, config DirectConfig) DeliveryChannel {
	return DeliveryChannel{
		Name:     name,
		Type:     "direct",
		Endpoint: config.BaseURL,
		Config: map[string]interface{}{
			"storage_path": config.StoragePath,
			"serve_http":   config.ServeHTTP,
			"http_port":    config.HTTPPort,
		},
		Enabled: true,
	}
}

// CreatePackageManagerChannel creates a package manager delivery channel
func CreatePackageManagerChannel(name string, config PackageManagerConfig) DeliveryChannel {
	return DeliveryChannel{
		Name:     name,
		Type:     "package_manager",
		Endpoint: config.Repository,
		Credentials: map[string]string{
			"api_key": config.APIKey,
		},
		Config: map[string]interface{}{
			"type":      config.Type,
			"namespace": config.Namespace,
		},
		Enabled: true,
	}
}