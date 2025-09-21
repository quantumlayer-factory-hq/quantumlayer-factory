package deploy

import (
	"context"
	"fmt"
	"strconv"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// KubernetesNamespaceManager implements namespace management for Kubernetes
type KubernetesNamespaceManager struct {
	client *KubernetesClient
}

// NewNamespaceManager creates a new namespace manager
func NewNamespaceManager(client *KubernetesClient) *KubernetesNamespaceManager {
	return &KubernetesNamespaceManager{
		client: client,
	}
}

// CreateNamespace creates a new namespace with TTL
func (nm *KubernetesNamespaceManager) CreateNamespace(ctx context.Context, name string, labels map[string]string, ttl time.Duration) (*NamespaceInfo, error) {
	clientset := nm.client.GetClientset()

	// Check if namespace already exists
	existing, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err == nil {
		// Namespace exists, update TTL if needed
		return nm.updateNamespaceTTL(ctx, existing, ttl)
	}

	if !errors.IsNotFound(err) {
		return nil, fmt.Errorf("failed to check namespace existence: %w", err)
	}

	// Prepare namespace labels and annotations
	if labels == nil {
		labels = make(map[string]string)
	}

	annotations := make(map[string]string)
	expiresAt := time.Now().Add(ttl)

	// Add default labels
	labels["app.kubernetes.io/managed-by"] = "quantumlayer-factory"
	labels["quantumlayer.dev/type"] = "preview"

	// Add TTL annotations
	annotations["quantumlayer.dev/created-at"] = time.Now().Format(time.RFC3339)
	annotations["quantumlayer.dev/expires-at"] = expiresAt.Format(time.RFC3339)
	annotations["quantumlayer.dev/ttl"] = ttl.String()

	// Create namespace
	namespace := &corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name:        name,
			Labels:      labels,
			Annotations: annotations,
		},
	}

	created, err := clientset.CoreV1().Namespaces().Create(ctx, namespace, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create namespace: %w", err)
	}

	return &NamespaceInfo{
		Name:        created.Name,
		Status:      string(created.Status.Phase),
		Labels:      created.Labels,
		Annotations: created.Annotations,
		CreatedAt:   created.CreationTimestamp.Time,
		ExpiresAt:   expiresAt,
		TTL:         ttl,
	}, nil
}

// DeleteNamespace deletes a namespace
func (nm *KubernetesNamespaceManager) DeleteNamespace(ctx context.Context, name string) error {
	clientset := nm.client.GetClientset()

	// Check if namespace exists and is managed by us
	namespace, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		if errors.IsNotFound(err) {
			return nil // Already deleted
		}
		return fmt.Errorf("failed to get namespace: %w", err)
	}

	// Only delete namespaces managed by QuantumLayer Factory
	managedBy := namespace.Labels["app.kubernetes.io/managed-by"]
	if managedBy != "quantumlayer-factory" {
		return fmt.Errorf("namespace %s is not managed by quantumlayer-factory", name)
	}

	// Delete namespace
	deletePolicy := metav1.DeletePropagationForeground
	err = clientset.CoreV1().Namespaces().Delete(ctx, name, metav1.DeleteOptions{
		PropagationPolicy: &deletePolicy,
	})
	if err != nil && !errors.IsNotFound(err) {
		return fmt.Errorf("failed to delete namespace: %w", err)
	}

	return nil
}

// GetNamespace gets namespace information
func (nm *KubernetesNamespaceManager) GetNamespace(ctx context.Context, name string) (*NamespaceInfo, error) {
	clientset := nm.client.GetClientset()

	namespace, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get namespace: %w", err)
	}

	info := &NamespaceInfo{
		Name:        namespace.Name,
		Status:      string(namespace.Status.Phase),
		Labels:      namespace.Labels,
		Annotations: namespace.Annotations,
		CreatedAt:   namespace.CreationTimestamp.Time,
	}

	// Parse TTL and expires-at from annotations
	if expiresAtStr, exists := namespace.Annotations["quantumlayer.dev/expires-at"]; exists {
		if expiresAt, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
			info.ExpiresAt = expiresAt
		}
	}

	if ttlStr, exists := namespace.Annotations["quantumlayer.dev/ttl"]; exists {
		if ttl, err := time.ParseDuration(ttlStr); err == nil {
			info.TTL = ttl
		}
	}

	return info, nil
}

// ListNamespaces lists namespaces managed by QuantumLayer Factory
func (nm *KubernetesNamespaceManager) ListNamespaces(ctx context.Context) ([]*NamespaceInfo, error) {
	clientset := nm.client.GetClientset()

	namespaces, err := clientset.CoreV1().Namespaces().List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/managed-by=quantumlayer-factory",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list namespaces: %w", err)
	}

	var result []*NamespaceInfo

	for _, namespace := range namespaces.Items {
		info := &NamespaceInfo{
			Name:        namespace.Name,
			Status:      string(namespace.Status.Phase),
			Labels:      namespace.Labels,
			Annotations: namespace.Annotations,
			CreatedAt:   namespace.CreationTimestamp.Time,
		}

		// Parse TTL and expires-at from annotations
		if expiresAtStr, exists := namespace.Annotations["quantumlayer.dev/expires-at"]; exists {
			if expiresAt, err := time.Parse(time.RFC3339, expiresAtStr); err == nil {
				info.ExpiresAt = expiresAt
			}
		}

		if ttlStr, exists := namespace.Annotations["quantumlayer.dev/ttl"]; exists {
			if ttl, err := time.ParseDuration(ttlStr); err == nil {
				info.TTL = ttl
			}
		}

		result = append(result, info)
	}

	return result, nil
}

// CleanupExpiredNamespaces removes expired namespaces
func (nm *KubernetesNamespaceManager) CleanupExpiredNamespaces(ctx context.Context) error {
	namespaces, err := nm.ListNamespaces(ctx)
	if err != nil {
		return fmt.Errorf("failed to list namespaces for cleanup: %w", err)
	}

	now := time.Now()
	var errors []string

	for _, ns := range namespaces {
		// Skip if no expiration time set
		if ns.ExpiresAt.IsZero() {
			continue
		}

		// Check if namespace has expired
		if now.After(ns.ExpiresAt) {
			err := nm.DeleteNamespace(ctx, ns.Name)
			if err != nil {
				errors = append(errors, fmt.Sprintf("failed to delete expired namespace %s: %v", ns.Name, err))
			}
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("cleanup errors: %v", errors)
	}

	return nil
}

// ExtendTTL extends namespace TTL
func (nm *KubernetesNamespaceManager) ExtendTTL(ctx context.Context, name string, ttl time.Duration) error {
	clientset := nm.client.GetClientset()

	namespace, err := clientset.CoreV1().Namespaces().Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get namespace: %w", err)
	}

	// Check if namespace is managed by us
	managedBy := namespace.Labels["app.kubernetes.io/managed-by"]
	if managedBy != "quantumlayer-factory" {
		return fmt.Errorf("namespace %s is not managed by quantumlayer-factory", name)
	}

	// Check TTL limits
	config := nm.client.GetConfig()
	if ttl > config.MaxTTL {
		return fmt.Errorf("TTL %v exceeds maximum allowed TTL %v", ttl, config.MaxTTL)
	}

	// Update annotations
	newExpiresAt := time.Now().Add(ttl)
	if namespace.Annotations == nil {
		namespace.Annotations = make(map[string]string)
	}

	namespace.Annotations["quantumlayer.dev/expires-at"] = newExpiresAt.Format(time.RFC3339)
	namespace.Annotations["quantumlayer.dev/ttl"] = ttl.String()
	namespace.Annotations["quantumlayer.dev/extended-at"] = time.Now().Format(time.RFC3339)

	_, err = clientset.CoreV1().Namespaces().Update(ctx, namespace, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to update namespace TTL: %w", err)
	}

	return nil
}

// updateNamespaceTTL updates TTL for existing namespace
func (nm *KubernetesNamespaceManager) updateNamespaceTTL(ctx context.Context, namespace *corev1.Namespace, ttl time.Duration) (*NamespaceInfo, error) {
	// Check if it's our namespace
	managedBy := namespace.Labels["app.kubernetes.io/managed-by"]
	if managedBy != "quantumlayer-factory" {
		return nil, fmt.Errorf("namespace %s is not managed by quantumlayer-factory", namespace.Name)
	}

	// Update TTL if the new one is longer
	var currentExpiresAt time.Time
	if expiresAtStr, exists := namespace.Annotations["quantumlayer.dev/expires-at"]; exists {
		currentExpiresAt, _ = time.Parse(time.RFC3339, expiresAtStr)
	}

	newExpiresAt := time.Now().Add(ttl)
	if newExpiresAt.After(currentExpiresAt) {
		err := nm.ExtendTTL(ctx, namespace.Name, ttl)
		if err != nil {
			return nil, err
		}
	}

	// Return updated namespace info
	return nm.GetNamespace(ctx, namespace.Name)
}

// generateNamespaceName generates a unique namespace name
func (nm *KubernetesNamespaceManager) GenerateNamespaceName(prefix string) string {
	timestamp := time.Now().Unix()
	return fmt.Sprintf("%s-%d", prefix, timestamp)
}

// StartCleanupScheduler starts a goroutine that periodically cleans up expired namespaces
func (nm *KubernetesNamespaceManager) StartCleanupScheduler(ctx context.Context, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if err := nm.CleanupExpiredNamespaces(ctx); err != nil {
					// Log error (in production, use proper logging)
					fmt.Printf("Cleanup error: %v\n", err)
				}
			}
		}
	}()
}

// GetNamespaceUsage returns resource usage statistics for a namespace
func (nm *KubernetesNamespaceManager) GetNamespaceUsage(ctx context.Context, name string) (map[string]interface{}, error) {
	clientset := nm.client.GetClientset()

	usage := make(map[string]interface{})

	// Get pods
	pods, err := clientset.CoreV1().Pods(name).List(ctx, metav1.ListOptions{})
	if err != nil {
		return usage, fmt.Errorf("failed to get pods: %w", err)
	}

	usage["pods"] = len(pods.Items)
	usage["running_pods"] = 0
	usage["pending_pods"] = 0
	usage["failed_pods"] = 0

	for _, pod := range pods.Items {
		switch pod.Status.Phase {
		case corev1.PodRunning:
			usage["running_pods"] = usage["running_pods"].(int) + 1
		case corev1.PodPending:
			usage["pending_pods"] = usage["pending_pods"].(int) + 1
		case corev1.PodFailed:
			usage["failed_pods"] = usage["failed_pods"].(int) + 1
		}
	}

	// Get services
	services, err := clientset.CoreV1().Services(name).List(ctx, metav1.ListOptions{})
	if err == nil {
		usage["services"] = len(services.Items)
	}

	// Get deployments
	deployments, err := clientset.AppsV1().Deployments(name).List(ctx, metav1.ListOptions{})
	if err == nil {
		usage["deployments"] = len(deployments.Items)
	}

	// Get ingresses
	ingresses, err := clientset.NetworkingV1().Ingresses(name).List(ctx, metav1.ListOptions{})
	if err == nil {
		usage["ingresses"] = len(ingresses.Items)
	}

	// Get PVCs
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(name).List(ctx, metav1.ListOptions{})
	if err == nil {
		usage["pvcs"] = len(pvcs.Items)
	}

	return usage, nil
}

// SetNamespaceQuota sets resource quotas for a namespace
func (nm *KubernetesNamespaceManager) SetNamespaceQuota(ctx context.Context, name string, quotas map[string]string) error {
	clientset := nm.client.GetClientset()

	// Create ResourceQuota
	resourceQuota := &corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-quota", name),
			Namespace: name,
			Labels: map[string]string{
				"app.kubernetes.io/managed-by": "quantumlayer-factory",
			},
		},
		Spec: corev1.ResourceQuotaSpec{
			Hard: make(corev1.ResourceList),
		},
	}

	// Convert quota strings to resource quantities
	for resource, limit := range quotas {
		if quantity, err := parseResourceQuantity(limit); err == nil {
			resourceQuota.Spec.Hard[corev1.ResourceName(resource)] = quantity
		}
	}

	_, err := clientset.CoreV1().ResourceQuotas(name).Create(ctx, resourceQuota, metav1.CreateOptions{})
	if err != nil && !errors.IsAlreadyExists(err) {
		return fmt.Errorf("failed to create resource quota: %w", err)
	}

	return nil
}

// parseResourceQuantity parses a resource quantity string
func parseResourceQuantity(quantity string) (resource.Quantity, error) {
	// Try parsing as a regular quantity first
	if q, err := resource.ParseQuantity(quantity); err == nil {
		return q, nil
	}

	// Try parsing as a number (for counts)
	if count, err := strconv.ParseInt(quantity, 10, 64); err == nil {
		return *resource.NewQuantity(count, resource.DecimalSI), nil
	}

	return resource.Quantity{}, fmt.Errorf("invalid resource quantity: %s", quantity)
}