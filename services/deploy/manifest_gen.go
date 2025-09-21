package deploy

import (
	"fmt"
)

// ManifestGeneratorImpl implements ManifestGenerator interface
type ManifestGeneratorImpl struct {
	config *DeployerConfig
}

// NewManifestGenerator creates a new manifest generator
func NewManifestGenerator(config *DeployerConfig) ManifestGenerator {
	return &ManifestGeneratorImpl{
		config: config,
	}
}

// GenerateDeployment generates a Deployment manifest
func (mg *ManifestGeneratorImpl) GenerateDeployment(req *DeployRequest) (map[string]interface{}, error) {
	labels := map[string]interface{}{
		"app":                          req.AppName,
		"app.kubernetes.io/name":       req.AppName,
		"app.kubernetes.io/instance":   req.AppName,
		"app.kubernetes.io/component":  "app",
		"app.kubernetes.io/managed-by": "quantumlayer-factory",
	}

	// Add custom labels
	for k, v := range req.Labels {
		labels[k] = v
	}

	// Build container spec
	container := map[string]interface{}{
		"name":  req.AppName,
		"image": fmt.Sprintf("%s:%s", req.ImageName, req.ImageTag),
		"ports": []map[string]interface{}{
			{
				"containerPort": req.Port,
				"protocol":      "TCP",
			},
		},
	}

	// Add environment variables
	if len(req.Environment) > 0 {
		var envVars []map[string]interface{}
		for key, value := range req.Environment {
			envVars = append(envVars, map[string]interface{}{
				"name":  key,
				"value": value,
			})
		}
		container["env"] = envVars
	}

	// Add resource limits
	if req.Resources != nil {
		resources := map[string]interface{}{}

		if req.Resources.CPU.Requests != "" || req.Resources.Memory.Requests != "" {
			requests := map[string]interface{}{}
			if req.Resources.CPU.Requests != "" {
				requests["cpu"] = req.Resources.CPU.Requests
			}
			if req.Resources.Memory.Requests != "" {
				requests["memory"] = req.Resources.Memory.Requests
			}
			resources["requests"] = requests
		}

		if req.Resources.CPU.Limits != "" || req.Resources.Memory.Limits != "" {
			limits := map[string]interface{}{}
			if req.Resources.CPU.Limits != "" {
				limits["cpu"] = req.Resources.CPU.Limits
			}
			if req.Resources.Memory.Limits != "" {
				limits["memory"] = req.Resources.Memory.Limits
			}
			resources["limits"] = limits
		}

		if len(resources) > 0 {
			container["resources"] = resources
		}
	}

	// Add health checks
	if req.HealthCheck != nil {
		if req.HealthCheck.LivenessProbe != nil {
			container["livenessProbe"] = mg.buildProbe(req.HealthCheck.LivenessProbe)
		}
		if req.HealthCheck.ReadinessProbe != nil {
			container["readinessProbe"] = mg.buildProbe(req.HealthCheck.ReadinessProbe)
		}
		if req.HealthCheck.StartupProbe != nil {
			container["startupProbe"] = mg.buildProbe(req.HealthCheck.StartupProbe)
		}
	}

	// Add volume mounts
	if len(req.PersistentVolumes) > 0 {
		var volumeMounts []map[string]interface{}
		for _, pv := range req.PersistentVolumes {
			volumeMounts = append(volumeMounts, map[string]interface{}{
				"name":      pv.Name,
				"mountPath": pv.MountPath,
			})
		}
		container["volumeMounts"] = volumeMounts
	}

	podSpec := map[string]interface{}{
		"containers": []map[string]interface{}{container},
	}

	// Add volumes
	if len(req.PersistentVolumes) > 0 {
		var volumes []map[string]interface{}
		for _, pv := range req.PersistentVolumes {
			volumes = append(volumes, map[string]interface{}{
				"name": pv.Name,
				"persistentVolumeClaim": map[string]interface{}{
					"claimName": pv.Name,
				},
			})
		}
		podSpec["volumes"] = volumes
	}

	// Build deployment manifest
	deployment := map[string]interface{}{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]interface{}{
			"name":      req.AppName,
			"namespace": req.Namespace,
			"labels":    labels,
		},
		"spec": map[string]interface{}{
			"replicas": req.Replicas,
			"selector": map[string]interface{}{
				"matchLabels": map[string]interface{}{
					"app": req.AppName,
				},
			},
			"template": map[string]interface{}{
				"metadata": map[string]interface{}{
					"labels": labels,
				},
				"spec": podSpec,
			},
		},
	}

	return deployment, nil
}

// GenerateService generates a Service manifest
func (mg *ManifestGeneratorImpl) GenerateService(req *DeployRequest) (map[string]interface{}, error) {
	labels := map[string]interface{}{
		"app":                          req.AppName,
		"app.kubernetes.io/name":       req.AppName,
		"app.kubernetes.io/instance":   req.AppName,
		"app.kubernetes.io/component":  "service",
		"app.kubernetes.io/managed-by": "quantumlayer-factory",
	}

	service := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata": map[string]interface{}{
			"name":      req.AppName,
			"namespace": req.Namespace,
			"labels":    labels,
		},
		"spec": map[string]interface{}{
			"selector": map[string]interface{}{
				"app": req.AppName,
			},
			"ports": []map[string]interface{}{
				{
					"port":       req.Port,
					"targetPort": req.Port,
					"protocol":   "TCP",
				},
			},
			"type": "ClusterIP",
		},
	}

	return service, nil
}

// GenerateIngress generates an Ingress manifest
func (mg *ManifestGeneratorImpl) GenerateIngress(req *DeployRequest) (map[string]interface{}, error) {
	if req.Ingress == nil || !req.Ingress.Enabled {
		return nil, fmt.Errorf("ingress not enabled")
	}

	labels := map[string]interface{}{
		"app":                          req.AppName,
		"app.kubernetes.io/name":       req.AppName,
		"app.kubernetes.io/instance":   req.AppName,
		"app.kubernetes.io/component":  "ingress",
		"app.kubernetes.io/managed-by": "quantumlayer-factory",
	}

	ingress := map[string]interface{}{
		"apiVersion": "networking.k8s.io/v1",
		"kind":       "Ingress",
		"metadata": map[string]interface{}{
			"name":      req.AppName,
			"namespace": req.Namespace,
			"labels":    labels,
		},
		"spec": map[string]interface{}{
			"rules": []map[string]interface{}{
				{
					"host": req.Ingress.Host,
					"http": map[string]interface{}{
						"paths": []map[string]interface{}{
							{
								"path":     req.Ingress.Path,
								"pathType": "Prefix",
								"backend": map[string]interface{}{
									"service": map[string]interface{}{
										"name": req.AppName,
										"port": map[string]interface{}{
											"number": req.Port,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}

	// Add annotations
	if len(req.Ingress.Annotations) > 0 {
		annotations := make(map[string]interface{})
		for k, v := range req.Ingress.Annotations {
			annotations[k] = v
		}
		metadata := ingress["metadata"].(map[string]interface{})
		metadata["annotations"] = annotations
	}

	// Add ingress class
	if req.Ingress.ClassName != "" {
		spec := ingress["spec"].(map[string]interface{})
		spec["ingressClassName"] = req.Ingress.ClassName
	}

	// Add TLS
	if req.Ingress.TLS {
		spec := ingress["spec"].(map[string]interface{})
		spec["tls"] = []map[string]interface{}{
			{
				"hosts": []string{req.Ingress.Host},
				"secretName": fmt.Sprintf("%s-tls", req.AppName),
			},
		}
	}

	return ingress, nil
}

// GenerateConfigMap generates a ConfigMap manifest
func (mg *ManifestGeneratorImpl) GenerateConfigMap(req *DeployRequest) (map[string]interface{}, error) {
	if len(req.Environment) == 0 {
		return nil, fmt.Errorf("no environment variables to create configmap")
	}

	labels := map[string]interface{}{
		"app":                          req.AppName,
		"app.kubernetes.io/name":       req.AppName,
		"app.kubernetes.io/instance":   req.AppName,
		"app.kubernetes.io/component":  "config",
		"app.kubernetes.io/managed-by": "quantumlayer-factory",
	}

	configMap := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "ConfigMap",
		"metadata": map[string]interface{}{
			"name":      fmt.Sprintf("%s-config", req.AppName),
			"namespace": req.Namespace,
			"labels":    labels,
		},
		"data": req.Environment,
	}

	return configMap, nil
}

// GenerateSecret generates a Secret manifest
func (mg *ManifestGeneratorImpl) GenerateSecret(req *DeployRequest) (map[string]interface{}, error) {
	// This would be used for sensitive environment variables
	// For now, return empty as we're not handling secrets
	return nil, fmt.Errorf("secret generation not implemented")
}

// GeneratePVC generates a PersistentVolumeClaim manifest
func (mg *ManifestGeneratorImpl) GeneratePVC(req *DeployRequest, pv PVConfig) (map[string]interface{}, error) {
	labels := map[string]interface{}{
		"app":                          req.AppName,
		"app.kubernetes.io/name":       req.AppName,
		"app.kubernetes.io/instance":   req.AppName,
		"app.kubernetes.io/component":  "storage",
		"app.kubernetes.io/managed-by": "quantumlayer-factory",
	}

	pvc := map[string]interface{}{
		"apiVersion": "v1",
		"kind":       "PersistentVolumeClaim",
		"metadata": map[string]interface{}{
			"name":      pv.Name,
			"namespace": req.Namespace,
			"labels":    labels,
		},
		"spec": map[string]interface{}{
			"accessModes": pv.AccessModes,
			"resources": map[string]interface{}{
				"requests": map[string]interface{}{
					"storage": pv.Size,
				},
			},
		},
	}

	// Add storage class if specified
	if pv.StorageClass != "" {
		spec := pvc["spec"].(map[string]interface{})
		spec["storageClassName"] = pv.StorageClass
	}

	return pvc, nil
}

// GenerateAll generates all required manifests
func (mg *ManifestGeneratorImpl) GenerateAll(req *DeployRequest) (map[string]map[string]interface{}, error) {
	manifests := make(map[string]map[string]interface{})

	// Generate deployment
	deployment, err := mg.GenerateDeployment(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate deployment: %w", err)
	}
	manifests["deployment"] = deployment

	// Generate service
	service, err := mg.GenerateService(req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate service: %w", err)
	}
	manifests["service"] = service

	// Generate ingress if enabled
	if req.Ingress != nil && req.Ingress.Enabled {
		ingress, err := mg.GenerateIngress(req)
		if err == nil {
			manifests["ingress"] = ingress
		}
	}

	// Generate ConfigMap if environment variables exist
	if len(req.Environment) > 0 {
		configMap, err := mg.GenerateConfigMap(req)
		if err == nil {
			manifests["configmap"] = configMap
		}
	}

	// Generate PVCs
	for i, pv := range req.PersistentVolumes {
		pvc, err := mg.GeneratePVC(req, pv)
		if err == nil {
			manifests[fmt.Sprintf("pvc-%d", i)] = pvc
		}
	}

	return manifests, nil
}

// buildProbe builds a probe configuration
func (mg *ManifestGeneratorImpl) buildProbe(probe *ProbeConfig) map[string]interface{} {
	probeConfig := map[string]interface{}{
		"httpGet": map[string]interface{}{
			"path": probe.Path,
			"port": probe.Port,
		},
		"initialDelaySeconds": probe.InitialDelaySeconds,
		"periodSeconds":       probe.PeriodSeconds,
		"timeoutSeconds":      probe.TimeoutSeconds,
		"successThreshold":    probe.SuccessThreshold,
		"failureThreshold":    probe.FailureThreshold,
	}

	return probeConfig
}