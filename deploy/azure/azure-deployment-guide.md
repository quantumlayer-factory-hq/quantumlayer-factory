# Azure Deployment Guide for QuantumLayer Factory

## Architecture Overview

QuantumLayer Factory can be deployed to Azure using multiple approaches:

### Option 1: Azure Kubernetes Service (AKS) - Recommended
- **Best for**: Production workloads, scalability, enterprise features
- **Components**: AKS cluster, Azure Container Registry, Azure Database, Redis Cache
- **Cost**: Medium-High, but most scalable

### Option 2: Azure Container Apps
- **Best for**: Serverless containers, automatic scaling, simpler management
- **Components**: Container Apps Environment, Azure Container Registry, managed services
- **Cost**: Low-Medium, pay-per-use

### Option 3: Azure App Service
- **Best for**: Quick deployment, managed infrastructure, integrated CI/CD
- **Components**: App Service Plan, Web Apps, Azure Database, Redis Cache
- **Cost**: Medium, predictable pricing

## Recommended Architecture (AKS)

```
┌─────────────────────────────────────────────────────────────┐
│                      Azure Subscription                     │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────┐  ┌─────────────────┐  ┌──────────────┐ │
│  │   Azure DNS     │  │  Application    │  │   Azure      │ │
│  │      Zone       │  │    Gateway      │  │  Front Door  │ │
│  └─────────────────┘  └─────────────────┘  └──────────────┘ │
│              │                 │                    │       │
│  ┌─────────────────────────────────────────────────────────┐ │
│  │               Azure Kubernetes Service (AKS)            │ │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐│ │
│  │  │    QLF      │ │   Worker    │ │      Frontend       ││ │
│  │  │   Server    │ │   Nodes     │ │     (Next.js)       ││ │
│  │  └─────────────┘ └─────────────┘ └─────────────────────┘│ │
│  │  ┌─────────────┐ ┌─────────────┐ ┌─────────────────────┐│ │
│  │  │  Temporal   │ │   Vector    │ │      MinIO          ││ │
│  │  │  Workflows  │ │   Search    │ │     Storage         ││ │
│  │  └─────────────┘ └─────────────┘ └─────────────────────┘│ │
│  └─────────────────────────────────────────────────────────┘ │
│              │                 │                    │       │
│  ┌─────────────────┐ ┌─────────────────┐ ┌──────────────────┐│
│  │   Azure Database│ │   Azure Redis   │ │  Azure Container ││
│  │   for PostgreSQL│ │     Cache       │ │     Registry     ││
│  └─────────────────┘ └─────────────────┘ └──────────────────┘│
│              │                 │                    │       │
│  ┌─────────────────┐ ┌─────────────────┐ ┌──────────────────┐│
│  │  Azure KeyVault │ │   Azure Monitor │ │  Azure Log       ││
│  │   (Secrets)     │ │   (Metrics)     │ │   Analytics      ││
│  └─────────────────┘ └─────────────────┘ └──────────────────┘│
└─────────────────────────────────────────────────────────────┘
```

## Prerequisites

1. **Azure CLI** installed and configured
2. **kubectl** for Kubernetes management
3. **Docker** for container building
4. **Terraform** (optional, for infrastructure as code)
5. **Azure Subscription** with sufficient permissions

## Deployment Steps

### Step 1: Set up Azure Resources

```bash
# Login to Azure
az login

# Create Resource Group
az group create --name qlf-prod --location eastus

# Create Azure Container Registry
az acr create --resource-group qlf-prod --name qlfregistry --sku Standard

# Create AKS Cluster
az aks create \
    --resource-group qlf-prod \
    --name qlf-cluster \
    --node-count 3 \
    --node-vm-size Standard_B4ms \
    --attach-acr qlfregistry \
    --enable-addons monitoring

# Create Azure Database for PostgreSQL
az postgres flexible-server create \
    --resource-group qlf-prod \
    --name qlf-postgres \
    --admin-user qlfadmin \
    --admin-password "SecurePassword123!" \
    --sku-name Standard_B2s \
    --tier Burstable \
    --storage-size 32

# Create Azure Redis Cache
az redis create \
    --resource-group qlf-prod \
    --name qlf-redis \
    --location eastus \
    --sku Standard \
    --vm-size c1
```

### Step 2: Configure Container Registry

```bash
# Login to ACR
az acr login --name qlfregistry

# Tag and push images
docker tag quantumlayer-factory:latest qlfregistry.azurecr.io/qlf/server:latest
docker tag qlf-worker:latest qlfregistry.azurecr.io/qlf/worker:latest
docker tag qlf-frontend:latest qlfregistry.azurecr.io/qlf/frontend:latest

docker push qlfregistry.azurecr.io/qlf/server:latest
docker push qlfregistry.azurecr.io/qlf/worker:latest
docker push qlfregistry.azurecr.io/qlf/frontend:latest
```

### Step 3: Deploy to AKS

```bash
# Get AKS credentials
az aks get-credentials --resource-group qlf-prod --name qlf-cluster

# Apply Kubernetes manifests (see azure/k8s/ directory)
kubectl apply -f azure/k8s/namespace.yaml
kubectl apply -f azure/k8s/secrets.yaml
kubectl apply -f azure/k8s/configmap.yaml
kubectl apply -f azure/k8s/postgres.yaml
kubectl apply -f azure/k8s/redis.yaml
kubectl apply -f azure/k8s/temporal.yaml
kubectl apply -f azure/k8s/qlf-server.yaml
kubectl apply -f azure/k8s/qlf-worker.yaml
kubectl apply -f azure/k8s/qlf-frontend.yaml
kubectl apply -f azure/k8s/ingress.yaml
```

## Cost Optimization

### Development Environment
- **AKS**: 2-node cluster (Standard_B2s) - ~$150/month
- **PostgreSQL**: Burstable B1ms - ~$25/month
- **Redis**: Basic C0 - ~$20/month
- **Total**: ~$200/month

### Production Environment
- **AKS**: 3-node cluster (Standard_D4s_v3) - ~$500/month
- **PostgreSQL**: General Purpose GP_Gen5_4 - ~$200/month
- **Redis**: Standard C2 - ~$180/month
- **Application Gateway**: ~$50/month
- **Total**: ~$930/month

## Monitoring and Observability

The deployment includes:
- **Azure Monitor**: Metrics and alerts
- **Azure Log Analytics**: Centralized logging
- **Application Insights**: Application performance monitoring
- **Prometheus**: Custom metrics (already integrated in QLF)
- **Grafana**: Visualization dashboards

## Security Features

- **Azure KeyVault**: Secret management
- **Azure AD**: Identity and access management
- **Network Security Groups**: Network-level security
- **Azure Policy**: Compliance and governance
- **Container scanning**: Built into Azure Container Registry
- **RBAC**: Role-based access control in AKS

## CI/CD Pipeline

Use Azure DevOps or GitHub Actions for automated deployment:

1. **Build**: Compile Go binaries, build Docker images
2. **Test**: Run unit tests, security scans
3. **Push**: Push images to Azure Container Registry
4. **Deploy**: Deploy to AKS using kubectl or Helm
5. **Monitor**: Verify deployment health and metrics

## Backup and Disaster Recovery

- **Database**: Automated backups with point-in-time recovery
- **Storage**: Geo-redundant storage for generated artifacts
- **Configuration**: Infrastructure as Code with Terraform
- **Multi-region**: Deploy across multiple Azure regions for HA

## Support and Maintenance

- **Auto-scaling**: Configure HPA for worker nodes
- **Updates**: Use Azure AKS managed updates
- **Patching**: Automated security patching for nodes
- **Monitoring**: 24/7 monitoring with Azure Monitor alerts