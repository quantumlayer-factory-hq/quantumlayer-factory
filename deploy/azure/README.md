# Azure Deployment for QuantumLayer Factory

This directory contains all the necessary files and configurations to deploy QuantumLayer Factory to Microsoft Azure.

## 🚀 Quick Start

### Prerequisites

1. **Azure CLI** installed and configured
2. **Docker** for building container images
3. **kubectl** for Kubernetes management
4. **Azure subscription** with appropriate permissions

### One-Command Deployment

```bash
cd deploy/azure/scripts
chmod +x deploy.sh
./deploy.sh
```

This script will:
- ✅ Create all Azure resources (AKS, ACR, PostgreSQL, Redis, KeyVault)
- ✅ Build and push Docker images
- ✅ Deploy to Kubernetes with monitoring and security
- ✅ Configure ingress and SSL certificates

## 📁 Directory Structure

```
deploy/azure/
├── README.md                 # This file
├── azure-deployment-guide.md # Detailed deployment guide
├── arm/                      # Azure Resource Manager templates
│   └── main.json            # Infrastructure as Code
├── k8s/                     # Kubernetes manifests
│   ├── namespace.yaml       # Namespace and RBAC
│   ├── secrets.yaml         # Application secrets
│   ├── qlf-server.yaml      # Main server deployment
│   ├── qlf-worker.yaml      # Worker deployment with HPA
│   ├── temporal.yaml        # Temporal workflow engine
│   └── ingress.yaml         # Load balancer and SSL
├── scripts/                 # Deployment automation
│   └── deploy.sh           # Main deployment script
├── pipelines/               # CI/CD pipelines
│   └── azure-pipelines.yml # Azure DevOps pipeline
└── docker/                 # Docker configurations
    ├── Dockerfile.server    # Server container
    ├── Dockerfile.worker    # Worker container
    └── Dockerfile.frontend  # Frontend container
```

## 🏗️ Architecture

### Azure Services Used

| Service | Purpose | Cost (Est/month) |
|---------|---------|------------------|
| **Azure Kubernetes Service** | Container orchestration | $300-800 |
| **Azure Container Registry** | Container image storage | $25-100 |
| **Azure Database for PostgreSQL** | Primary database | $100-500 |
| **Azure Redis Cache** | Session & task caching | $50-200 |
| **Azure KeyVault** | Secret management | $5-15 |
| **Azure Monitor** | Observability & alerts | $50-150 |
| **Application Gateway** | Load balancing & SSL | $50-200 |

**Total Estimated Cost**: $580-1,965/month (depending on scale)

### Deployment Targets

#### 1. Azure Kubernetes Service (Recommended)
- **Best for**: Production workloads, enterprise features
- **Features**: Auto-scaling, monitoring, security, high availability
- **Management**: Requires Kubernetes knowledge

#### 2. Azure Container Apps
- **Best for**: Serverless containers, simpler management
- **Features**: Automatic scaling, pay-per-use
- **Management**: Minimal operational overhead

#### 3. Azure App Service
- **Best for**: Quick deployment, integrated CI/CD
- **Features**: Managed platform, easy scaling
- **Management**: Limited customization

## 🔧 Configuration

### Environment Variables

Update `k8s/secrets.yaml` with your actual values:

```yaml
# Azure OpenAI (Primary LLM Provider)
AZURE_OPENAI_ENDPOINT: "https://your-instance.openai.azure.com/"
AZURE_OPENAI_API_KEY: "your-api-key"

# AWS Bedrock (Secondary LLM Provider)
AWS_ACCESS_KEY_ID: "your-aws-key"
AWS_SECRET_ACCESS_KEY: "your-aws-secret"

# Database Configuration
POSTGRES_PASSWORD: "secure-password"
REDIS_PASSWORD: "redis-password"

# Security
JWT_SECRET_KEY: "your-jwt-secret"
```

### Custom Domains

Update `k8s/ingress.yaml` to use your domains:

```yaml
spec:
  tls:
  - hosts:
    - api.yourdomain.com      # API endpoint
    - app.yourdomain.com      # Frontend application
    - temporal.yourdomain.com # Temporal UI
```

## 🚢 Deployment Options

### Option 1: Automated Script (Recommended)

```bash
# Set environment variables
export PROJECT_NAME="qlf"
export ENVIRONMENT="prod"
export LOCATION="eastus"
export POSTGRES_PASSWORD="your-secure-password"

# Run deployment
./scripts/deploy.sh
```

### Option 2: Manual Step-by-Step

```bash
# 1. Create resource group
az group create --name qlf-prod --location eastus

# 2. Deploy infrastructure
az deployment group create \
  --resource-group qlf-prod \
  --template-file arm/main.json \
  --parameters administratorPassword="secure-password"

# 3. Build and push images
az acr login --name qlfregistry
docker build -t qlfregistry.azurecr.io/qlf/server:latest .
docker push qlfregistry.azurecr.io/qlf/server:latest

# 4. Deploy to Kubernetes
az aks get-credentials --resource-group qlf-prod --name qlf-prod-cluster
kubectl apply -f k8s/
```

### Option 3: CI/CD Pipeline

Use the provided Azure DevOps pipeline:

1. Import `pipelines/azure-pipelines.yml`
2. Configure service connections
3. Set up variable groups
4. Enable automatic deployments

## 📊 Monitoring & Observability

### Built-in Monitoring

- **Azure Monitor**: Infrastructure metrics and logs
- **Application Insights**: Application performance monitoring
- **Prometheus**: Custom application metrics (already integrated)
- **Grafana**: Visualization dashboards

### Health Checks

All services include comprehensive health checks:
- **Liveness probes**: Detect if containers need restart
- **Readiness probes**: Detect if services are ready for traffic
- **Startup probes**: Handle slow-starting applications

### Alerting

Pre-configured alerts for:
- High CPU/memory usage
- Pod restart loops
- Database connection issues
- LLM API failures
- Queue backlog buildup

## 🔒 Security Features

### Network Security
- **Network policies**: Restrict inter-pod communication
- **Private networking**: Database and Redis not exposed publicly
- **SSL/TLS**: All traffic encrypted in transit

### Secret Management
- **Azure KeyVault**: Centralized secret storage
- **Kubernetes secrets**: Runtime secret injection
- **Managed identities**: No stored credentials for Azure services

### Access Control
- **RBAC**: Role-based access control in Kubernetes
- **Azure AD**: Identity and access management
- **Service accounts**: Least-privilege principle

## 🔄 Updates & Maintenance

### Rolling Updates
```bash
# Update server image
kubectl set image deployment/qlf-server qlf-server=qlfregistry.azurecr.io/qlf/server:v2.0.0

# Monitor rollout
kubectl rollout status deployment/qlf-server

# Rollback if needed
kubectl rollout undo deployment/qlf-server
```

### Database Migrations
```bash
# Run migrations as a Kubernetes job
kubectl create job migrate-db --from=cronjob/database-migrate
```

### Scaling
```bash
# Manual scaling
kubectl scale deployment qlf-worker --replicas=10

# Automatic scaling is configured via HPA
```

## 🆘 Troubleshooting

### Common Issues

1. **Pods stuck in Pending**
   ```bash
   kubectl describe pod <pod-name> -n quantumlayer-factory
   # Check resource constraints and node capacity
   ```

2. **LLM API failures**
   ```bash
   kubectl logs deployment/qlf-worker -n quantumlayer-factory
   # Check API keys and rate limits
   ```

3. **Database connection issues**
   ```bash
   kubectl exec -it deployment/qlf-server -- sh
   # Test database connectivity
   ```

### Support Commands

```bash
# Get all resources
kubectl get all -n quantumlayer-factory

# Check logs
kubectl logs -f deployment/qlf-server -n quantumlayer-factory

# Port forward for debugging
kubectl port-forward service/qlf-server 8080:8080 -n quantumlayer-factory

# Check resource usage
kubectl top pods -n quantumlayer-factory
```

## 📈 Performance Tuning

### Worker Scaling
- Default: 5 workers
- Auto-scales: 3-20 based on CPU/memory
- Tune via `k8s/qlf-worker.yaml`

### Database Performance
- Connection pooling: Configured
- Read replicas: Available for high-read workloads
- Backup: 7-day retention

### Caching Strategy
- Redis: Session and LLM response caching
- Application-level: Template and overlay caching
- CDN: For static assets (frontend)

## 📞 Support

For deployment issues:

1. **Check logs**: `kubectl logs` commands above
2. **Review documentation**: See `azure-deployment-guide.md`
3. **Monitor metrics**: Azure Monitor and Application Insights
4. **Contact team**: File issues in the project repository

---

**🎉 Congratulations!** You now have QuantumLayer Factory running on enterprise-grade Azure infrastructure with automatic scaling, monitoring, and security features.