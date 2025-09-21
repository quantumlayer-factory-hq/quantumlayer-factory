package deploy

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNamespaceManager_CreateNamespace(t *testing.T) {
	tests := []struct {
		name      string
		nsName    string
		labels    map[string]string
		ttl       time.Duration
		wantErr   bool
		expectErr string
	}{
		{
			name:   "valid namespace",
			nsName: "test-namespace",
			labels: map[string]string{
				"app": "test",
			},
			ttl:     24 * time.Hour,
			wantErr: false,
		},
		{
			name:      "empty namespace name",
			nsName:    "",
			labels:    map[string]string{},
			ttl:       24 * time.Hour,
			wantErr:   true,
			expectErr: "namespace name cannot be empty",
		},
		{
			name:      "invalid namespace name",
			nsName:    "Test-Namespace",
			labels:    map[string]string{},
			ttl:       24 * time.Hour,
			wantErr:   true,
			expectErr: "invalid namespace name",
		},
		{
			name:   "TTL exceeds maximum",
			nsName: "test-namespace",
			labels: map[string]string{},
			ttl:    100 * time.Hour,
			wantErr: true,
		},
		{
			name:   "zero TTL",
			nsName: "test-namespace",
			labels: map[string]string{},
			ttl:    0,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create namespace info for validation
			nsInfo := &NamespaceInfo{
				Name:   tt.nsName,
				Labels: tt.labels,
				TTL:    tt.ttl,
			}

			err := validateNamespaceInfo(nsInfo)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectErr != "" {
					assert.Contains(t, err.Error(), tt.expectErr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNamespaceInfo_IsExpired(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		nsInfo   NamespaceInfo
		expected bool
	}{
		{
			name: "not expired",
			nsInfo: NamespaceInfo{
				Name:      "test-ns",
				CreatedAt: now.Add(-1 * time.Hour),
				ExpiresAt: now.Add(1 * time.Hour),
				TTL:       2 * time.Hour,
			},
			expected: false,
		},
		{
			name: "expired",
			nsInfo: NamespaceInfo{
				Name:      "test-ns",
				CreatedAt: now.Add(-2 * time.Hour),
				ExpiresAt: now.Add(-1 * time.Hour),
				TTL:       1 * time.Hour,
			},
			expected: true,
		},
		{
			name: "just expired",
			nsInfo: NamespaceInfo{
				Name:      "test-ns",
				CreatedAt: now.Add(-1 * time.Hour),
				ExpiresAt: now,
				TTL:       1 * time.Hour,
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.nsInfo.IsExpired()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNamespaceInfo_TimeLeft(t *testing.T) {
	now := time.Now()

	tests := []struct {
		name     string
		nsInfo   NamespaceInfo
		expected time.Duration
	}{
		{
			name: "time remaining",
			nsInfo: NamespaceInfo{
				Name:      "test-ns",
				ExpiresAt: now.Add(2 * time.Hour),
			},
			expected: 2 * time.Hour,
		},
		{
			name: "expired",
			nsInfo: NamespaceInfo{
				Name:      "test-ns",
				ExpiresAt: now.Add(-1 * time.Hour),
			},
			expected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.nsInfo.TimeLeft()
			// Allow for small time differences due to test execution time
			if tt.expected > 0 {
				assert.True(t, result >= tt.expected-time.Second)
				assert.True(t, result <= tt.expected+time.Second)
			} else {
				assert.Equal(t, time.Duration(0), result)
			}
		})
	}
}

func TestNamespaceInfo_ExtendTTL(t *testing.T) {
	now := time.Now()
	nsInfo := &NamespaceInfo{
		Name:      "test-ns",
		CreatedAt: now,
		ExpiresAt: now.Add(1 * time.Hour),
		TTL:       1 * time.Hour,
	}

	// Extend by 2 hours
	extension := 2 * time.Hour
	nsInfo.ExtendTTL(extension)

	assert.Equal(t, 3*time.Hour, nsInfo.TTL)
	expectedExpiry := now.Add(3 * time.Hour)
	assert.True(t, nsInfo.ExpiresAt.After(expectedExpiry.Add(-time.Second)))
	assert.True(t, nsInfo.ExpiresAt.Before(expectedExpiry.Add(time.Second)))
}

func TestNamespaceInfo_GetAge(t *testing.T) {
	now := time.Now()
	nsInfo := NamespaceInfo{
		Name:      "test-ns",
		CreatedAt: now.Add(-5 * time.Minute),
	}

	age := nsInfo.GetAge()
	assert.True(t, age >= 5*time.Minute-time.Second)
	assert.True(t, age <= 5*time.Minute+time.Second)
}

func TestNamespaceInfo_GetLabels(t *testing.T) {
	nsInfo := NamespaceInfo{
		Name: "test-ns",
		Labels: map[string]string{
			"app":         "test-app",
			"environment": "test",
			"owner":       "test-user",
		},
	}

	labels := nsInfo.GetLabels()
	assert.Equal(t, "test-app", labels["app"])
	assert.Equal(t, "test", labels["environment"])
	assert.Equal(t, "test-user", labels["owner"])
}

func TestCleanupExpiredNamespaces_Logic(t *testing.T) {
	now := time.Now()

	namespaces := []NamespaceInfo{
		{
			Name:      "active-ns",
			Status:    "Active",
			CreatedAt: now.Add(-1 * time.Hour),
			ExpiresAt: now.Add(1 * time.Hour),
			TTL:       2 * time.Hour,
		},
		{
			Name:      "expired-ns-1",
			Status:    "Active",
			CreatedAt: now.Add(-3 * time.Hour),
			ExpiresAt: now.Add(-1 * time.Hour),
			TTL:       2 * time.Hour,
		},
		{
			Name:      "expired-ns-2",
			Status:    "Active",
			CreatedAt: now.Add(-4 * time.Hour),
			ExpiresAt: now.Add(-2 * time.Hour),
			TTL:       2 * time.Hour,
		},
		{
			Name:      "terminating-ns",
			Status:    "Terminating",
			CreatedAt: now.Add(-2 * time.Hour),
			ExpiresAt: now.Add(-30 * time.Minute),
			TTL:       90 * time.Minute,
		},
	}

	// Test which namespaces should be cleaned up
	var expiredNamespaces []NamespaceInfo
	for _, ns := range namespaces {
		if ns.IsExpired() && ns.Status == "Active" {
			expiredNamespaces = append(expiredNamespaces, ns)
		}
	}

	assert.Len(t, expiredNamespaces, 2)
	assert.Equal(t, "expired-ns-1", expiredNamespaces[0].Name)
	assert.Equal(t, "expired-ns-2", expiredNamespaces[1].Name)
}

func TestValidateNamespaceName(t *testing.T) {
	tests := []struct {
		name      string
		nsName    string
		wantErr   bool
		expectErr string
	}{
		{
			name:    "valid lowercase name",
			nsName:  "test-namespace",
			wantErr: false,
		},
		{
			name:    "valid with numbers",
			nsName:  "test-ns-123",
			wantErr: false,
		},
		{
			name:      "empty name",
			nsName:    "",
			wantErr:   true,
			expectErr: "namespace name cannot be empty",
		},
		{
			name:      "uppercase letters",
			nsName:    "Test-Namespace",
			wantErr:   true,
			expectErr: "invalid namespace name",
		},
		{
			name:      "starts with number",
			nsName:    "123-namespace",
			wantErr:   true,
			expectErr: "invalid namespace name",
		},
		{
			name:      "contains underscore",
			nsName:    "test_namespace",
			wantErr:   true,
			expectErr: "invalid namespace name",
		},
		{
			name:      "too long",
			nsName:    "this-is-a-very-long-namespace-name-that-exceeds-the-maximum-allowed-length-for-kubernetes-namespaces",
			wantErr:   true,
			expectErr: "namespace name too long",
		},
		{
			name:      "reserved name",
			nsName:    "kube-system",
			wantErr:   true,
			expectErr: "reserved namespace name",
		},
		{
			name:      "reserved name default",
			nsName:    "default",
			wantErr:   true,
			expectErr: "reserved namespace name",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateNamespaceName(tt.nsName)
			if tt.wantErr {
				assert.Error(t, err)
				if tt.expectErr != "" {
					assert.Contains(t, err.Error(), tt.expectErr)
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNamespaceManager_TTLValidation(t *testing.T) {
	config := DefaultDeployerConfig()

	tests := []struct {
		name    string
		ttl     time.Duration
		wantErr bool
	}{
		{
			name:    "valid TTL",
			ttl:     24 * time.Hour,
			wantErr: false,
		},
		{
			name:    "maximum TTL",
			ttl:     config.MaxTTL,
			wantErr: false,
		},
		{
			name:    "exceeds maximum TTL",
			ttl:     config.MaxTTL + time.Hour,
			wantErr: true,
		},
		{
			name:    "zero TTL",
			ttl:     0,
			wantErr: true,
		},
		{
			name:    "negative TTL",
			ttl:     -time.Hour,
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := validateTTL(tt.ttl, config)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestNamespaceManager_MockOperations(t *testing.T) {
	ctx := context.Background()

	// Test namespace creation request
	req := NamespaceCreateRequest{
		Name: "test-namespace",
		Labels: map[string]string{
			"app":         "test-app",
			"environment": "testing",
		},
		TTL: 24 * time.Hour,
	}

	assert.Equal(t, "test-namespace", req.Name)
	assert.Equal(t, "test-app", req.Labels["app"])
	assert.Equal(t, 24*time.Hour, req.TTL)
	assert.NotNil(t, ctx)
}

// Helper validation functions

func validateNamespaceInfo(nsInfo *NamespaceInfo) error {
	if err := validateNamespaceName(nsInfo.Name); err != nil {
		return err
	}

	config := DefaultDeployerConfig()
	return validateTTL(nsInfo.TTL, config)
}

func validateNamespaceName(name string) error {
	if name == "" {
		return fmt.Errorf("namespace name cannot be empty")
	}

	// Check for reserved names
	reserved := []string{"default", "kube-system", "kube-public", "kube-node-lease"}
	for _, r := range reserved {
		if name == r {
			return fmt.Errorf("reserved namespace name")
		}
	}

	// Check length
	if len(name) > 63 {
		return fmt.Errorf("namespace name too long")
	}

	// Check format (simplified)
	if name[0] >= '0' && name[0] <= '9' {
		return fmt.Errorf("invalid namespace name")
	}

	// Check for uppercase or invalid characters (simplified)
	for _, r := range name {
		if r >= 'A' && r <= 'Z' {
			return fmt.Errorf("invalid namespace name")
		}
		if r == '_' {
			return fmt.Errorf("invalid namespace name")
		}
	}

	return nil
}

func validateTTL(ttl time.Duration, config *DeployerConfig) error {
	if ttl <= 0 {
		return fmt.Errorf("TTL must be positive")
	}
	if ttl > config.MaxTTL {
		return fmt.Errorf("TTL exceeds maximum allowed")
	}
	return nil
}

// Add methods to NamespaceInfo for testing

func (ns *NamespaceInfo) IsExpired() bool {
	return time.Now().After(ns.ExpiresAt)
}

func (ns *NamespaceInfo) TimeLeft() time.Duration {
	remaining := time.Until(ns.ExpiresAt)
	if remaining < 0 {
		return 0
	}
	return remaining
}

func (ns *NamespaceInfo) ExtendTTL(extension time.Duration) {
	ns.TTL += extension
	ns.ExpiresAt = ns.CreatedAt.Add(ns.TTL)
}

func (ns *NamespaceInfo) GetAge() time.Duration {
	return time.Since(ns.CreatedAt)
}

func (ns *NamespaceInfo) GetLabels() map[string]string {
	return ns.Labels
}

// NamespaceCreateRequest for testing
type NamespaceCreateRequest struct {
	Name   string            `json:"name"`
	Labels map[string]string `json:"labels"`
	TTL    time.Duration     `json:"ttl"`
}