package builder

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSecurityScanner_ScanImage(t *testing.T) {
	scanner := NewSecurityScanner()

	// Note: This would normally use a mock Trivy client
	// For now, test the interface and configuration
	assert.NotNil(t, scanner)
}

func TestSecurityScanResult_PassesThreshold(t *testing.T) {
	threshold := VulnerabilityThreshold{
		Critical: 0,
		High:     5,
		Medium:   20,
		Low:      100,
	}

	tests := []struct {
		name     string
		result   *SecurityScanResult
		expected bool
	}{
		{
			name: "clean scan passes",
			result: &SecurityScanResult{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      5,
			},
			expected: true,
		},
		{
			name: "within threshold passes",
			result: &SecurityScanResult{
				Critical: 0,
				High:     3,
				Medium:   15,
				Low:      50,
			},
			expected: true,
		},
		{
			name: "critical vulnerability fails",
			result: &SecurityScanResult{
				Critical: 1,
				High:     0,
				Medium:   0,
				Low:      0,
			},
			expected: false,
		},
		{
			name: "high threshold exceeded fails",
			result: &SecurityScanResult{
				Critical: 0,
				High:     6,
				Medium:   0,
				Low:      0,
			},
			expected: false,
		},
		{
			name: "medium threshold exceeded fails",
			result: &SecurityScanResult{
				Critical: 0,
				High:     5,
				Medium:   21,
				Low:      0,
			},
			expected: false,
		},
		{
			name: "low vulnerabilities within threshold pass",
			result: &SecurityScanResult{
				Critical: 0,
				High:     0,
				Medium:   0,
				Low:      50,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			passed := tt.result.PassesThreshold(threshold)
			assert.Equal(t, tt.expected, passed)
		})
	}
}

func TestSecurityScanResult_TotalVulnerabilities(t *testing.T) {
	result := &SecurityScanResult{
		Critical: 2,
		High:     5,
		Medium:   10,
		Low:      20,
	}

	total := result.TotalVulnerabilities()
	assert.Equal(t, 37, total)
}

func TestSecurityScanResult_HasCriticalVulnerabilities(t *testing.T) {
	tests := []struct {
		name     string
		result   *SecurityScanResult
		expected bool
	}{
		{
			name: "has critical",
			result: &SecurityScanResult{
				Critical: 1,
				High:     0,
				Medium:   0,
				Low:      0,
			},
			expected: true,
		},
		{
			name: "no critical",
			result: &SecurityScanResult{
				Critical: 0,
				High:     5,
				Medium:   10,
				Low:      20,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.result.HasCriticalVulnerabilities()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSecurityScanResult_Summary(t *testing.T) {
	result := &SecurityScanResult{
		Critical: 1,
		High:     2,
		Medium:   5,
		Low:      10,
		Scanner:  "trivy",
	}

	summary := result.Summary()
	assert.Contains(t, summary, "Critical: 1")
	assert.Contains(t, summary, "High: 2")
	assert.Contains(t, summary, "Medium: 5")
	assert.Contains(t, summary, "Low: 10")
	assert.Contains(t, summary, "Scanner: trivy")
}

func TestVulnerabilityThreshold_IsValid(t *testing.T) {
	tests := []struct {
		name      string
		threshold VulnerabilityThreshold
		expected  bool
	}{
		{
			name: "valid threshold",
			threshold: VulnerabilityThreshold{
				Critical: 0,
				High:     5,
				Medium:   20,
				Low:      100,
			},
			expected: true,
		},
		{
			name: "negative critical invalid",
			threshold: VulnerabilityThreshold{
				Critical: -1,
				High:     5,
				Medium:   20,
				Low:      100,
			},
			expected: false,
		},
		{
			name: "negative high invalid",
			threshold: VulnerabilityThreshold{
				Critical: 0,
				High:     -1,
				Medium:   20,
				Low:      100,
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.threshold.IsValid()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestSecurityScanner_ConfigValidation(t *testing.T) {
	config := DefaultBuilderConfig()

	// Test default security configuration
	assert.True(t, config.EnableScanning)
	assert.Equal(t, "trivy", config.Scanner)
	assert.True(t, config.DefaultThreshold.IsValid())

	// Test security thresholds
	threshold := config.DefaultThreshold
	assert.Equal(t, 0, threshold.Critical)
	assert.Equal(t, 5, threshold.High)
	assert.Equal(t, 20, threshold.Medium)
}

func TestSecurityScanResult_ToJSON(t *testing.T) {
	result := &SecurityScanResult{
		Critical:      1,
		High:          2,
		Medium:        5,
		Low:           10,
		Scanner:       "trivy",
		ScanTime:      time.Now(),
		ImageID:       "sha256:abc123",
		ImageName:     "test-app",
		ImageTag:      "latest",
		Vulnerabilities: []Vulnerability{
			{
				ID:          "CVE-2021-1234",
				Severity:    "HIGH",
				Title:       "Test vulnerability",
				Description: "Test description",
				Package:     "test-package",
				Version:     "1.0.0",
				FixedVersion: "1.0.1",
			},
		},
	}

	// Test that result can be marshaled to JSON
	data, err := result.ToJSON()
	require.NoError(t, err)
	assert.NotEmpty(t, data)
	assert.Contains(t, string(data), "CVE-2021-1234")
}

func TestVulnerability_IsFixable(t *testing.T) {
	tests := []struct {
		name    string
		vuln    Vulnerability
		fixable bool
	}{
		{
			name: "fixable vulnerability",
			vuln: Vulnerability{
				ID:           "CVE-2021-1234",
				FixedVersion: "1.0.1",
			},
			fixable: true,
		},
		{
			name: "unfixable vulnerability",
			vuln: Vulnerability{
				ID:           "CVE-2021-5678",
				FixedVersion: "",
			},
			fixable: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.vuln.IsFixable()
			assert.Equal(t, tt.fixable, result)
		})
	}
}

func TestSecurityScanResult_GetVulnerabilitiesBySeverity(t *testing.T) {
	result := &SecurityScanResult{
		Vulnerabilities: []Vulnerability{
			{ID: "CVE-1", Severity: "CRITICAL"},
			{ID: "CVE-2", Severity: "HIGH"},
			{ID: "CVE-3", Severity: "HIGH"},
			{ID: "CVE-4", Severity: "MEDIUM"},
			{ID: "CVE-5", Severity: "LOW"},
		},
	}

	critical := result.GetVulnerabilitiesBySeverity("CRITICAL")
	assert.Len(t, critical, 1)
	assert.Equal(t, "CVE-1", critical[0].ID)

	high := result.GetVulnerabilitiesBySeverity("HIGH")
	assert.Len(t, high, 2)

	medium := result.GetVulnerabilitiesBySeverity("MEDIUM")
	assert.Len(t, medium, 1)

	low := result.GetVulnerabilitiesBySeverity("LOW")
	assert.Len(t, low, 1)
}

func TestSecurityScanResult_GetFixableVulnerabilities(t *testing.T) {
	result := &SecurityScanResult{
		Vulnerabilities: []Vulnerability{
			{ID: "CVE-1", FixedVersion: "1.0.1"},
			{ID: "CVE-2", FixedVersion: ""},
			{ID: "CVE-3", FixedVersion: "2.0.0"},
		},
	}

	fixable := result.GetFixableVulnerabilities()
	assert.Len(t, fixable, 2)
	assert.Equal(t, "CVE-1", fixable[0].ID)
	assert.Equal(t, "CVE-3", fixable[1].ID)
}

func TestSecurityScanResult_FilterBySeverity(t *testing.T) {
	result := &SecurityScanResult{
		Critical: 1,
		High:     2,
		Medium:   3,
		Low:      4,
		Vulnerabilities: []Vulnerability{
			{ID: "CVE-1", Severity: "CRITICAL"},
			{ID: "CVE-2", Severity: "HIGH"},
			{ID: "CVE-3", Severity: "HIGH"},
			{ID: "CVE-4", Severity: "MEDIUM"},
			{ID: "CVE-5", Severity: "MEDIUM"},
			{ID: "CVE-6", Severity: "MEDIUM"},
			{ID: "CVE-7", Severity: "LOW"},
		},
	}

	// Filter to only show critical and high
	filtered := result.FilterBySeverity([]string{"CRITICAL", "HIGH"})

	assert.Equal(t, 1, filtered.Critical)
	assert.Equal(t, 2, filtered.High)
	assert.Equal(t, 0, filtered.Medium)
	assert.Equal(t, 0, filtered.Low)
	assert.Len(t, filtered.Vulnerabilities, 3)
}

func TestSecurityScanner_ScanTimeout(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Millisecond)
	defer cancel()

	scanner := NewSecurityScanner()

	// Wait for context to timeout
	time.Sleep(2 * time.Millisecond)

	// This should timeout immediately
	_, err := scanner.ScanImage(ctx, "test-image:latest")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context deadline exceeded")
}

func TestSecurityScanner_InvalidImage(t *testing.T) {
	ctx := context.Background()
	scanner := NewSecurityScanner()

	// Test with invalid image name
	_, err := scanner.ScanImage(ctx, "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "image name cannot be empty")

	// Test with malformed image name
	_, err = scanner.ScanImage(ctx, "invalid/image/name/with/too/many/slashes")
	assert.Error(t, err)
}

// Benchmark tests

func BenchmarkSecurityScanResult_PassesThreshold(b *testing.B) {
	threshold := VulnerabilityThreshold{
		Critical: 0,
		High:     5,
		Medium:   20,
		Low:      100,
	}

	result := &SecurityScanResult{
		Critical: 0,
		High:     3,
		Medium:   15,
		Low:      50,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.PassesThreshold(threshold)
	}
}

func BenchmarkSecurityScanResult_GetVulnerabilitiesBySeverity(b *testing.B) {
	// Create a result with many vulnerabilities
	vulnerabilities := make([]Vulnerability, 1000)
	for i := 0; i < 1000; i++ {
		severity := "LOW"
		if i%4 == 0 {
			severity = "CRITICAL"
		} else if i%3 == 0 {
			severity = "HIGH"
		} else if i%2 == 0 {
			severity = "MEDIUM"
		}

		vulnerabilities[i] = Vulnerability{
			ID:       fmt.Sprintf("CVE-%d", i),
			Severity: severity,
		}
	}

	result := &SecurityScanResult{
		Vulnerabilities: vulnerabilities,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = result.GetVulnerabilitiesBySeverity("HIGH")
	}
}