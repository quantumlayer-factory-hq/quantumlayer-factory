package builder

import (
	"context"
	"fmt"
	"strings"
	"time"
)

// SecurityScanner implements the Scanner interface
type SecurityScanner struct {
	config *BuilderConfig
}

// NewSecurityScanner creates a new security scanner
func NewSecurityScanner() *SecurityScanner {
	return &SecurityScanner{
		config: DefaultBuilderConfig(),
	}
}

// ScanImage scans an image for vulnerabilities
func (s *SecurityScanner) ScanImage(ctx context.Context, imageName string) (*SecurityScanResult, error) {
	if imageName == "" {
		return nil, fmt.Errorf("image name cannot be empty")
	}

	// Check for malformed image names
	parts := strings.Split(imageName, "/")
	if len(parts) > 3 {
		return nil, fmt.Errorf("invalid image name format")
	}

	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
	}

	// This is a mock implementation - in production this would use Trivy
	result := &SecurityScanResult{
		Scanner:   s.config.Scanner,
		ScanTime:  time.Now(),
		ImageName: imageName,
		ImageTag:  "latest",
		Critical:  0,
		High:      0,
		Medium:    0,
		Low:       0,
		Threshold: s.config.DefaultThreshold,
		Passed:    true,
	}

	return result, nil
}

// ScanImageWithThreshold scans an image with a specific threshold
func (s *SecurityScanner) ScanImageWithThreshold(ctx context.Context, imageName string, threshold VulnerabilityThreshold) (*SecurityScanResult, error) {
	result, err := s.ScanImage(ctx, imageName)
	if err != nil {
		return nil, err
	}

	result.Threshold = threshold
	result.Passed = result.PassesThreshold(threshold)

	return result, nil
}