package deploy

import (
	"context"
	"fmt"
	"io"
	"os"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// KubernetesDeployer implements Kubernetes deployment functionality
type KubernetesDeployer struct {
	client    *KubernetesClient
	nsManager NamespaceManager
	manifests ManifestGenerator
	config    *DeployerConfig
}

// NewKubernetesDeployer creates a new Kubernetes deployer
func NewKubernetesDeployer(config *DeployerConfig) (*KubernetesDeployer, error) {
	// Create Kubernetes client
	var kubeConfig *rest.Config
	var err error

	if config.InCluster {
		kubeConfig, err = rest.InClusterConfig()
	} else {
		kubeConfig, err = clientcmd.BuildConfigFromFlags("", config.KubeConfig)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(kubeConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create Kubernetes client: %w", err)
	}

	client := NewKubernetesClient(clientset, config)
	nsManager := NewNamespaceManager(client)
	manifests := NewManifestGenerator(config)

	return &KubernetesDeployer{
		client:    client,
		nsManager: nsManager,
		manifests: manifests,
		config:    config,
	}, nil
}

// Deploy deploys an application to Kubernetes
func (kd *KubernetesDeployer) Deploy(ctx context.Context, req *DeployRequest) (*DeployResult, error) {
	result := &DeployResult{
		AppName:    req.AppName,
		Namespace:  req.Namespace,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(req.TTL),
		Warnings:   []string{},
		Errors:     []string{},
		Events:     []string{},
		Metadata:   make(map[string]interface{}),
	}

	// Ensure namespace exists
	if req.Namespace != "default" {
		nsInfo, err := kd.nsManager.CreateNamespace(ctx, req.Namespace,
			map[string]string{
				"app.kubernetes.io/managed-by": "quantumlayer-factory",
				"quantumlayer.dev/app":         req.AppName,
			}, req.TTL)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create namespace: %v", err))
			return result, err
		}
		result.Events = append(result.Events, fmt.Sprintf("Namespace %s created", nsInfo.Name))
	}

	// Generate manifests
	manifests, err := kd.manifests.GenerateAll(req)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("Failed to generate manifests: %v", err))
		return result, err
	}

	// Validate manifests
	for name, manifest := range manifests {
		if err := kd.ValidateManifests(manifest); err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Manifest %s validation warning: %v", name, err))
		}
	}

	// Create deployment
	deployment, exists := manifests["deployment"]
	if exists {
		deploymentName, err := kd.createDeployment(ctx, req.Namespace, deployment)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create deployment: %v", err))
			return result, err
		}
		result.DeploymentName = deploymentName
		result.Events = append(result.Events, fmt.Sprintf("Deployment %s created", deploymentName))
	}

	// Create service
	service, exists := manifests["service"]
	if exists {
		serviceName, err := kd.createService(ctx, req.Namespace, service)
		if err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("Failed to create service: %v", err))
			return result, err
		}
		result.ServiceName = serviceName
		result.InternalURL = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", serviceName, req.Namespace, req.Port)
		result.Events = append(result.Events, fmt.Sprintf("Service %s created", serviceName))
	}

	// Create ingress if configured
	ingress, exists := manifests["ingress"]
	if exists && req.Ingress != nil && req.Ingress.Enabled {
		ingressName, err := kd.createIngress(ctx, req.Namespace, ingress)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to create ingress: %v", err))
		} else {
			result.IngressName = ingressName
			result.URL = fmt.Sprintf("https://%s", req.Ingress.Host)
			result.Events = append(result.Events, fmt.Sprintf("Ingress %s created", ingressName))
		}
	}

	// Create ConfigMap if needed
	configMap, exists := manifests["configmap"]
	if exists {
		_, err := kd.createConfigMap(ctx, req.Namespace, configMap)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to create configmap: %v", err))
		} else {
			result.Events = append(result.Events, "ConfigMap created")
		}
	}

	// Create PVCs if needed
	for i := range req.PersistentVolumes {
		pvcKey := fmt.Sprintf("pvc-%d", i)
		if pvc, exists := manifests[pvcKey]; exists {
			pvcName, err := kd.createPVC(ctx, req.Namespace, pvc)
			if err != nil {
				result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to create PVC %s: %v", pvcName, err))
			} else {
				result.Events = append(result.Events, fmt.Sprintf("PVC %s created", pvcName))
			}
		}
	}

	// Wait for deployment to be ready
	if result.DeploymentName != "" {
		err = kd.waitForDeployment(ctx, req.Namespace, result.DeploymentName, 5*time.Minute)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Deployment not ready: %v", err))
		}
	}

	// Get final status
	status, err := kd.GetStatus(ctx, req.Namespace, req.AppName)
	if err != nil {
		result.Warnings = append(result.Warnings, fmt.Sprintf("Failed to get status: %v", err))
	} else {
		result.Status = status.Status
		result.Pods = status.Pods
	}

	result.Success = len(result.Errors) == 0
	return result, nil
}

// GetStatus gets the status of a deployment
func (kd *KubernetesDeployer) GetStatus(ctx context.Context, namespace, appName string) (*DeployResult, error) {
	clientset := kd.client.GetClientset()

	result := &DeployResult{
		AppName:   appName,
		Namespace: namespace,
		Pods:      []PodInfo{},
	}

	// Get deployment
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", appName),
	})
	if err != nil {
		return result, fmt.Errorf("failed to get deployments: %w", err)
	}

	if len(deployments.Items) == 0 {
		return result, fmt.Errorf("no deployment found for app %s", appName)
	}

	deployment := deployments.Items[0]
	result.DeploymentName = deployment.Name

	// Get deployment status
	result.Status = DeploymentStatus{
		Replicas:          *deployment.Spec.Replicas,
		ReadyReplicas:     deployment.Status.ReadyReplicas,
		AvailableReplicas: deployment.Status.AvailableReplicas,
		UpdatedReplicas:   deployment.Status.UpdatedReplicas,
		LastUpdateTime:    time.Now(),
	}

	// Determine phase
	if result.Status.ReadyReplicas == result.Status.Replicas {
		result.Status.Phase = "Running"
	} else if result.Status.ReadyReplicas > 0 {
		result.Status.Phase = "Pending"
	} else {
		result.Status.Phase = "Failed"
	}

	// Get pods
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", appName),
	})
	if err != nil {
		return result, fmt.Errorf("failed to get pods: %w", err)
	}

	for _, pod := range pods.Items {
		podInfo := PodInfo{
			Name:      pod.Name,
			Status:    string(pod.Status.Phase),
			Ready:     kd.isPodReady(&pod),
			Restarts:  kd.getPodRestarts(&pod),
			Age:       time.Since(pod.CreationTimestamp.Time),
			Node:      pod.Spec.NodeName,
			IP:        pod.Status.PodIP,
			Labels:    pod.Labels,
			CreatedAt: pod.CreationTimestamp.Time,
		}
		result.Pods = append(result.Pods, podInfo)
	}

	// Get service
	services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", appName),
	})
	if err == nil && len(services.Items) > 0 {
		service := services.Items[0]
		result.ServiceName = service.Name
		if len(service.Spec.Ports) > 0 {
			port := service.Spec.Ports[0].Port
			result.InternalURL = fmt.Sprintf("http://%s.%s.svc.cluster.local:%d", service.Name, namespace, port)
		}
	}

	// Get ingress
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", appName),
	})
	if err == nil && len(ingresses.Items) > 0 {
		ingress := ingresses.Items[0]
		result.IngressName = ingress.Name
		if len(ingress.Spec.Rules) > 0 {
			host := ingress.Spec.Rules[0].Host
			scheme := "http"
			if len(ingress.Spec.TLS) > 0 {
				scheme = "https"
			}
			result.URL = fmt.Sprintf("%s://%s", scheme, host)
		}
	}

	result.Success = true
	return result, nil
}

// Delete removes a deployment
func (kd *KubernetesDeployer) Delete(ctx context.Context, namespace, appName string) error {
	clientset := kd.client.GetClientset()
	labelSelector := fmt.Sprintf("app=%s", appName)
	deletePolicy := metav1.DeletePropagationForeground

	// Delete ingresses
	ingresses, err := clientset.NetworkingV1().Ingresses(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err == nil {
		for _, ingress := range ingresses.Items {
			err = clientset.NetworkingV1().Ingresses(namespace).Delete(ctx, ingress.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
			if err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete ingress %s: %w", ingress.Name, err)
			}
		}
	}

	// Delete services
	services, err := clientset.CoreV1().Services(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err == nil {
		for _, service := range services.Items {
			err = clientset.CoreV1().Services(namespace).Delete(ctx, service.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
			if err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete service %s: %w", service.Name, err)
			}
		}
	}

	// Delete deployments
	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err == nil {
		for _, deployment := range deployments.Items {
			err = clientset.AppsV1().Deployments(namespace).Delete(ctx, deployment.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
			if err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete deployment %s: %w", deployment.Name, err)
			}
		}
	}

	// Delete ConfigMaps
	configMaps, err := clientset.CoreV1().ConfigMaps(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err == nil {
		for _, configMap := range configMaps.Items {
			err = clientset.CoreV1().ConfigMaps(namespace).Delete(ctx, configMap.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
			if err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete configmap %s: %w", configMap.Name, err)
			}
		}
	}

	// Delete PVCs
	pvcs, err := clientset.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err == nil {
		for _, pvc := range pvcs.Items {
			err = clientset.CoreV1().PersistentVolumeClaims(namespace).Delete(ctx, pvc.Name, metav1.DeleteOptions{PropagationPolicy: &deletePolicy})
			if err != nil && !errors.IsNotFound(err) {
				return fmt.Errorf("failed to delete PVC %s: %w", pvc.Name, err)
			}
		}
	}

	return nil
}

// List lists deployments in a namespace
func (kd *KubernetesDeployer) List(ctx context.Context, namespace string) ([]*DeployResult, error) {
	clientset := kd.client.GetClientset()

	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app.kubernetes.io/managed-by=quantumlayer-factory",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list deployments: %w", err)
	}

	var results []*DeployResult

	for _, deployment := range deployments.Items {
		appName := deployment.Labels["app"]
		if appName == "" {
			continue
		}

		status, err := kd.GetStatus(ctx, namespace, appName)
		if err != nil {
			continue
		}

		results = append(results, status)
	}

	return results, nil
}

// Scale scales a deployment
func (kd *KubernetesDeployer) Scale(ctx context.Context, namespace, appName string, replicas int32) error {
	clientset := kd.client.GetClientset()

	deployments, err := clientset.AppsV1().Deployments(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", appName),
	})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	if len(deployments.Items) == 0 {
		return fmt.Errorf("no deployment found for app %s", appName)
	}

	deployment := &deployments.Items[0]
	deployment.Spec.Replicas = &replicas

	_, err = clientset.AppsV1().Deployments(namespace).Update(ctx, deployment, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("failed to scale deployment: %w", err)
	}

	return nil
}

// GetLogs gets logs from pods
func (kd *KubernetesDeployer) GetLogs(ctx context.Context, namespace, appName string, lines int64) ([]string, error) {
	clientset := kd.client.GetClientset()

	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: fmt.Sprintf("app=%s", appName),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to get pods: %w", err)
	}

	var logs []string

	for _, pod := range pods.Items {
		podLogs, err := clientset.CoreV1().Pods(namespace).GetLogs(pod.Name, &corev1.PodLogOptions{
			TailLines: &lines,
		}).Stream(ctx)
		if err != nil {
			logs = append(logs, fmt.Sprintf("Error getting logs from pod %s: %v", pod.Name, err))
			continue
		}
		defer podLogs.Close()

		buf := make([]byte, 2048)
		for {
			n, err := podLogs.Read(buf)
			if n > 0 {
				logs = append(logs, fmt.Sprintf("[%s] %s", pod.Name, string(buf[:n])))
			}
			if err != nil {
				break
			}
		}
	}

	return logs, nil
}

// ValidateManifests validates Kubernetes manifests
func (kd *KubernetesDeployer) ValidateManifests(manifest map[string]interface{}) error {
	// Basic validation - check required fields
	kind, exists := manifest["kind"]
	if !exists {
		return fmt.Errorf("manifest missing 'kind' field")
	}

	metadata, exists := manifest["metadata"]
	if !exists {
		return fmt.Errorf("manifest missing 'metadata' field")
	}

	if metadataMap, ok := metadata.(map[string]interface{}); ok {
		if _, exists := metadataMap["name"]; !exists {
			return fmt.Errorf("manifest missing 'metadata.name' field")
		}
	}

	// Kind-specific validation
	switch kind {
	case "Deployment":
		return kd.validateDeployment(manifest)
	case "Service":
		return kd.validateService(manifest)
	case "Ingress":
		return kd.validateIngress(manifest)
	}

	return nil
}

// Helper functions

func (kd *KubernetesDeployer) createDeployment(ctx context.Context, namespace string, manifest map[string]interface{}) (string, error) {
	clientset := kd.client.GetClientset()

	// Convert manifest to Deployment object
	deployment := &appsv1.Deployment{}
	err := kd.convertManifest(manifest, deployment)
	if err != nil {
		return "", fmt.Errorf("failed to convert deployment manifest: %w", err)
	}

	result, err := clientset.AppsV1().Deployments(namespace).Create(ctx, deployment, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return result.Name, nil
}

func (kd *KubernetesDeployer) createService(ctx context.Context, namespace string, manifest map[string]interface{}) (string, error) {
	clientset := kd.client.GetClientset()

	service := &corev1.Service{}
	err := kd.convertManifest(manifest, service)
	if err != nil {
		return "", fmt.Errorf("failed to convert service manifest: %w", err)
	}

	result, err := clientset.CoreV1().Services(namespace).Create(ctx, service, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return result.Name, nil
}

func (kd *KubernetesDeployer) createIngress(ctx context.Context, namespace string, manifest map[string]interface{}) (string, error) {
	clientset := kd.client.GetClientset()

	ingress := &networkingv1.Ingress{}
	err := kd.convertManifest(manifest, ingress)
	if err != nil {
		return "", fmt.Errorf("failed to convert ingress manifest: %w", err)
	}

	result, err := clientset.NetworkingV1().Ingresses(namespace).Create(ctx, ingress, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return result.Name, nil
}

func (kd *KubernetesDeployer) createConfigMap(ctx context.Context, namespace string, manifest map[string]interface{}) (string, error) {
	clientset := kd.client.GetClientset()

	configMap := &corev1.ConfigMap{}
	err := kd.convertManifest(manifest, configMap)
	if err != nil {
		return "", fmt.Errorf("failed to convert configmap manifest: %w", err)
	}

	result, err := clientset.CoreV1().ConfigMaps(namespace).Create(ctx, configMap, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return result.Name, nil
}

func (kd *KubernetesDeployer) createPVC(ctx context.Context, namespace string, manifest map[string]interface{}) (string, error) {
	clientset := kd.client.GetClientset()

	pvc := &corev1.PersistentVolumeClaim{}
	err := kd.convertManifest(manifest, pvc)
	if err != nil {
		return "", fmt.Errorf("failed to convert PVC manifest: %w", err)
	}

	result, err := clientset.CoreV1().PersistentVolumeClaims(namespace).Create(ctx, pvc, metav1.CreateOptions{})
	if err != nil {
		return "", err
	}

	return result.Name, nil
}

func (kd *KubernetesDeployer) waitForDeployment(ctx context.Context, namespace, name string, timeout time.Duration) error {
	clientset := kd.client.GetClientset()

	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for deployment %s to be ready", name)
		default:
			deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, name, metav1.GetOptions{})
			if err != nil {
				return fmt.Errorf("failed to get deployment: %w", err)
			}

			if deployment.Status.ReadyReplicas >= *deployment.Spec.Replicas {
				return nil
			}

			time.Sleep(5 * time.Second)
		}
	}
}

func (kd *KubernetesDeployer) isPodReady(pod *corev1.Pod) bool {
	for _, condition := range pod.Status.Conditions {
		if condition.Type == corev1.PodReady {
			return condition.Status == corev1.ConditionTrue
		}
	}
	return false
}

func (kd *KubernetesDeployer) getPodRestarts(pod *corev1.Pod) int32 {
	var restarts int32
	for _, containerStatus := range pod.Status.ContainerStatuses {
		restarts += containerStatus.RestartCount
	}
	return restarts
}

func (kd *KubernetesDeployer) convertManifest(manifest map[string]interface{}, target interface{}) error {
	// This is a simplified conversion - in production, use proper JSON marshaling/unmarshaling
	// or a library like runtime.Convert from k8s.io/apimachinery/pkg/runtime
	return fmt.Errorf("manifest conversion not implemented")
}

func (kd *KubernetesDeployer) validateDeployment(manifest map[string]interface{}) error {
	spec, exists := manifest["spec"]
	if !exists {
		return fmt.Errorf("deployment missing 'spec' field")
	}

	if specMap, ok := spec.(map[string]interface{}); ok {
		if _, exists := specMap["replicas"]; !exists {
			return fmt.Errorf("deployment spec missing 'replicas' field")
		}
		if _, exists := specMap["selector"]; !exists {
			return fmt.Errorf("deployment spec missing 'selector' field")
		}
		if _, exists := specMap["template"]; !exists {
			return fmt.Errorf("deployment spec missing 'template' field")
		}
	}

	return nil
}

func (kd *KubernetesDeployer) validateService(manifest map[string]interface{}) error {
	spec, exists := manifest["spec"]
	if !exists {
		return fmt.Errorf("service missing 'spec' field")
	}

	if specMap, ok := spec.(map[string]interface{}); ok {
		if _, exists := specMap["ports"]; !exists {
			return fmt.Errorf("service spec missing 'ports' field")
		}
		if _, exists := specMap["selector"]; !exists {
			return fmt.Errorf("service spec missing 'selector' field")
		}
	}

	return nil
}

func (kd *KubernetesDeployer) validateIngress(manifest map[string]interface{}) error {
	spec, exists := manifest["spec"]
	if !exists {
		return fmt.Errorf("ingress missing 'spec' field")
	}

	if specMap, ok := spec.(map[string]interface{}); ok {
		if _, exists := specMap["rules"]; !exists {
			return fmt.Errorf("ingress spec missing 'rules' field")
		}
	}

	return nil
}

// WaitForReady waits for a deployment to become ready
func (kd *KubernetesDeployer) WaitForReady(ctx context.Context, namespace, deploymentName string, timeout time.Duration) error {
	clientset := kd.client.GetClientset()

	// Create a timeout context
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	for {
		select {
		case <-ctx.Done():
			return fmt.Errorf("timeout waiting for deployment %s to be ready", deploymentName)
		default:
			// Check deployment status
			deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
			if err != nil {
				if errors.IsNotFound(err) {
					return fmt.Errorf("deployment %s not found", deploymentName)
				}
				return fmt.Errorf("failed to get deployment status: %w", err)
			}

			// Check if deployment is ready
			if deployment.Status.ReadyReplicas == deployment.Status.Replicas && deployment.Status.Replicas > 0 {
				return nil
			}

			// Wait before checking again
			time.Sleep(2 * time.Second)
		}
	}
}

// FollowLogs follows logs from a deployment
func (kd *KubernetesDeployer) FollowLogs(ctx context.Context, namespace, deploymentName string) error {
	clientset := kd.client.GetClientset()

	// Get deployment to find pods
	deployment, err := clientset.AppsV1().Deployments(namespace).Get(ctx, deploymentName, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("failed to get deployment: %w", err)
	}

	// Get pods for the deployment
	labelSelector := fmt.Sprintf("app=%s", deployment.Spec.Selector.MatchLabels["app"])
	pods, err := clientset.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{LabelSelector: labelSelector})
	if err != nil {
		return fmt.Errorf("failed to list pods: %w", err)
	}

	if len(pods.Items) == 0 {
		return fmt.Errorf("no pods found for deployment %s", deploymentName)
	}

	// Follow logs from the first pod
	podName := pods.Items[0].Name
	req := clientset.CoreV1().Pods(namespace).GetLogs(podName, &corev1.PodLogOptions{
		Follow: true,
	})

	logStream, err := req.Stream(ctx)
	if err != nil {
		return fmt.Errorf("failed to stream logs: %w", err)
	}
	defer logStream.Close()

	// Copy logs to stdout
	fmt.Printf("Following logs from pod %s in namespace %s:\n", podName, namespace)
	fmt.Println("Press Ctrl+C to stop following logs")
	fmt.Println("---")

	_, err = io.Copy(os.Stdout, logStream)
	if err != nil {
		return fmt.Errorf("failed to copy logs: %w", err)
	}

	return nil
}