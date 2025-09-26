#!/bin/bash
set -e

# Azure Deployment Script for QuantumLayer Factory
# This script deploys the complete QuantumLayer Factory platform to Azure

# Configuration
PROJECT_NAME="${PROJECT_NAME:-qlf}"
ENVIRONMENT="${ENVIRONMENT:-prod}"
LOCATION="${LOCATION:-eastus}"
RESOURCE_GROUP="${RESOURCE_GROUP:-${PROJECT_NAME}-${ENVIRONMENT}}"
SUBSCRIPTION_ID="${SUBSCRIPTION_ID:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check Azure CLI
    if ! command -v az &> /dev/null; then
        log_error "Azure CLI is not installed. Please install it first."
        exit 1
    fi

    # Check kubectl
    if ! command -v kubectl &> /dev/null; then
        log_error "kubectl is not installed. Please install it first."
        exit 1
    fi

    # Check Docker
    if ! command -v docker &> /dev/null; then
        log_error "Docker is not installed. Please install it first."
        exit 1
    fi

    # Check if logged in to Azure
    if ! az account show &> /dev/null; then
        log_error "Not logged in to Azure. Please run 'az login' first."
        exit 1
    fi

    log_success "Prerequisites check passed"
}

# Set Azure subscription
set_subscription() {
    if [ -n "$SUBSCRIPTION_ID" ]; then
        log_info "Setting Azure subscription to $SUBSCRIPTION_ID"
        az account set --subscription "$SUBSCRIPTION_ID"
    else
        log_info "Using default Azure subscription"
    fi
}

# Create resource group
create_resource_group() {
    log_info "Creating resource group: $RESOURCE_GROUP"

    if az group show --name "$RESOURCE_GROUP" &> /dev/null; then
        log_warning "Resource group $RESOURCE_GROUP already exists"
    else
        az group create \
            --name "$RESOURCE_GROUP" \
            --location "$LOCATION" \
            --tags "project=${PROJECT_NAME}" "environment=${ENVIRONMENT}"
        log_success "Resource group created successfully"
    fi
}

# Deploy Azure infrastructure
deploy_infrastructure() {
    log_info "Deploying Azure infrastructure..."

    # Get the directory of this script
    SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
    ARM_TEMPLATE="$SCRIPT_DIR/../arm/main.json"

    if [ ! -f "$ARM_TEMPLATE" ]; then
        log_error "ARM template not found at $ARM_TEMPLATE"
        exit 1
    fi

    # Prompt for PostgreSQL password if not set
    if [ -z "$POSTGRES_PASSWORD" ]; then
        log_info "PostgreSQL administrator password not set"
        read -s -p "Enter PostgreSQL administrator password: " POSTGRES_PASSWORD
        echo
        export POSTGRES_PASSWORD
    fi

    # Deploy ARM template
    az deployment group create \
        --resource-group "$RESOURCE_GROUP" \
        --template-file "$ARM_TEMPLATE" \
        --parameters \
            projectName="$PROJECT_NAME" \
            environment="$ENVIRONMENT" \
            location="$LOCATION" \
            administratorPassword="$POSTGRES_PASSWORD" \
        --output json > deployment_output.json

    if [ $? -eq 0 ]; then
        log_success "Infrastructure deployment completed"

        # Extract outputs
        ACR_NAME=$(jq -r '.properties.outputs.acrName.value' deployment_output.json)
        ACR_LOGIN_SERVER=$(jq -r '.properties.outputs.acrLoginServer.value' deployment_output.json)
        AKS_CLUSTER_NAME=$(jq -r '.properties.outputs.aksClusterName.value' deployment_output.json)

        export ACR_NAME ACR_LOGIN_SERVER AKS_CLUSTER_NAME

        log_info "ACR Name: $ACR_NAME"
        log_info "ACR Login Server: $ACR_LOGIN_SERVER"
        log_info "AKS Cluster Name: $AKS_CLUSTER_NAME"
    else
        log_error "Infrastructure deployment failed"
        exit 1
    fi
}

# Build and push Docker images
build_and_push_images() {
    log_info "Building and pushing Docker images..."

    # Login to ACR
    az acr login --name "$ACR_NAME"

    # Build images
    log_info "Building QLF server image..."
    docker build -t "$ACR_LOGIN_SERVER/qlf/server:latest" \
        -f Dockerfile.server .

    log_info "Building QLF worker image..."
    docker build -t "$ACR_LOGIN_SERVER/qlf/worker:latest" \
        -f Dockerfile.worker .

    # Build frontend image if it exists
    if [ -d "frontend" ]; then
        log_info "Building frontend image..."
        docker build -t "$ACR_LOGIN_SERVER/qlf/frontend:latest" \
            -f frontend/Dockerfile frontend/
    fi

    # Push images
    log_info "Pushing images to ACR..."
    docker push "$ACR_LOGIN_SERVER/qlf/server:latest"
    docker push "$ACR_LOGIN_SERVER/qlf/worker:latest"

    if [ -d "frontend" ]; then
        docker push "$ACR_LOGIN_SERVER/qlf/frontend:latest"
    fi

    log_success "Images built and pushed successfully"
}

# Create Dockerfiles if they don't exist
create_dockerfiles() {
    log_info "Creating Dockerfiles..."

    # Create server Dockerfile
    if [ ! -f "Dockerfile.server" ]; then
        cat > Dockerfile.server << 'EOF'
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o qlf ./cmd/qlf

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl
WORKDIR /root/
COPY --from=builder /app/qlf .
EXPOSE 8080 8091
CMD ["./qlf", "server", "--bind", "0.0.0.0:8080"]
EOF
        log_info "Created Dockerfile.server"
    fi

    # Create worker Dockerfile
    if [ ! -f "Dockerfile.worker" ]; then
        cat > Dockerfile.worker << 'EOF'
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o worker ./cmd/worker

FROM alpine:latest
RUN apk --no-cache add ca-certificates curl
WORKDIR /root/
COPY --from=builder /app/worker .
EXPOSE 8091
CMD ["./worker"]
EOF
        log_info "Created Dockerfile.worker"
    fi
}

# Deploy to Kubernetes
deploy_to_kubernetes() {
    log_info "Deploying to Kubernetes..."

    # Get AKS credentials
    az aks get-credentials \
        --resource-group "$RESOURCE_GROUP" \
        --name "$AKS_CLUSTER_NAME" \
        --overwrite-existing

    # Update image references in YAML files
    K8S_DIR="$(dirname "${BASH_SOURCE[0]}")/../k8s"

    # Create temporary directory for modified YAML files
    TEMP_K8S_DIR="/tmp/qlf-k8s-${RANDOM}"
    mkdir -p "$TEMP_K8S_DIR"
    cp -r "$K8S_DIR"/* "$TEMP_K8S_DIR/"

    # Replace image references
    find "$TEMP_K8S_DIR" -name "*.yaml" -exec sed -i "s|qlfregistry.azurecr.io|${ACR_LOGIN_SERVER}|g" {} \;

    # Apply Kubernetes manifests
    kubectl apply -f "$TEMP_K8S_DIR/namespace.yaml"

    # Update secrets with actual values
    log_info "Please update the secrets in $TEMP_K8S_DIR/secrets.yaml with actual values before continuing"
    log_info "Press Enter when ready to continue..."
    read

    kubectl apply -f "$TEMP_K8S_DIR/secrets.yaml"
    kubectl apply -f "$TEMP_K8S_DIR/temporal.yaml"
    kubectl apply -f "$TEMP_K8S_DIR/qlf-server.yaml"
    kubectl apply -f "$TEMP_K8S_DIR/qlf-worker.yaml"

    # Install ingress controller if not exists
    if ! kubectl get namespace ingress-nginx &> /dev/null; then
        log_info "Installing NGINX ingress controller..."
        kubectl apply -f https://raw.githubusercontent.com/kubernetes/ingress-nginx/controller-v1.8.1/deploy/static/provider/cloud/deploy.yaml

        # Wait for ingress controller to be ready
        kubectl wait --namespace ingress-nginx \
            --for=condition=ready pod \
            --selector=app.kubernetes.io/component=controller \
            --timeout=300s
    fi

    kubectl apply -f "$TEMP_K8S_DIR/ingress.yaml"

    # Clean up temporary directory
    rm -rf "$TEMP_K8S_DIR"

    log_success "Kubernetes deployment completed"
}

# Wait for deployment to be ready
wait_for_deployment() {
    log_info "Waiting for deployment to be ready..."

    # Wait for deployments to be ready
    kubectl wait --namespace quantumlayer-factory \
        --for=condition=available deployment/qlf-server \
        --timeout=300s

    kubectl wait --namespace quantumlayer-factory \
        --for=condition=available deployment/qlf-worker \
        --timeout=300s

    kubectl wait --namespace quantumlayer-factory \
        --for=condition=available deployment/temporal-server \
        --timeout=300s

    log_success "All deployments are ready"
}

# Show deployment status
show_status() {
    log_info "Deployment Status:"

    echo -e "\n${BLUE}Kubernetes Resources:${NC}"
    kubectl get all -n quantumlayer-factory

    echo -e "\n${BLUE}Ingress Status:${NC}"
    kubectl get ingress -n quantumlayer-factory

    echo -e "\n${BLUE}External IP:${NC}"
    kubectl get service -n ingress-nginx ingress-nginx-controller

    log_success "QuantumLayer Factory deployed successfully to Azure!"
    log_info "Access the application at the ingress external IP or configured domain"
}

# Main deployment flow
main() {
    log_info "Starting QuantumLayer Factory deployment to Azure..."

    check_prerequisites
    set_subscription
    create_resource_group
    deploy_infrastructure
    create_dockerfiles
    build_and_push_images
    deploy_to_kubernetes
    wait_for_deployment
    show_status

    log_success "Deployment completed successfully!"
}

# Run main function
main "$@"