# QuantumLayer Factory - Setup Guide

## System Requirements

### Minimum Requirements
- **OS**: Linux (Ubuntu 20.04+), macOS (12+), Windows 10+ with WSL2
- **RAM**: 8GB minimum, 16GB recommended
- **CPU**: 4 cores minimum, 8 cores recommended
- **Storage**: 10GB free space minimum
- **Network**: Internet connection for LLM providers and package downloads

### Software Dependencies
- **Docker**: 20.10+ with Docker Compose v2
- **Go**: 1.21+ for building from source
- **Make**: Build automation
- **Git**: Version control

## Installation Methods

### Method 1: Pre-built Binary (Recommended)

```bash
# Download latest release
curl -L https://releases.quantumlayer.dev/qlf-linux -o qlf
chmod +x qlf
sudo mv qlf /usr/local/bin/

# Verify installation
qlf --version
```

### Method 2: Build from Source

```bash
# Clone repository
git clone https://github.com/quantumlayer-factory-hq/quantumlayer-factory.git
cd quantumlayer-factory

# Install dependencies
make install-deps

# Build CLI
make build

# Add to PATH
export PATH=$PWD/bin:$PATH
echo 'export PATH=$PWD/bin:$PATH' >> ~/.bashrc
```

### Method 3: Docker Container

```bash
# Run via Docker
docker run -it --rm \
  -v $(pwd):/workspace \
  quantumlayer/qlf:latest \
  generate "user API" --dry-run

# Or use docker-compose for full environment
curl -L https://releases.quantumlayer.dev/docker-compose.yml -o docker-compose.yml
docker-compose up -d
```

## Initial Setup

### 1. Infrastructure Services

**Start Required Services**:
```bash
# Clone repository (if building from source)
git clone https://github.com/quantumlayer-factory-hq/quantumlayer-factory.git
cd quantumlayer-factory

# Start infrastructure
make dev

# Verify services are running
make status
```

**Service Verification**:
```bash
# Check Docker containers
docker ps | grep -E "(postgres|redis|temporal|qdrant|minio)"

# Test database connection
psql -h localhost -p 5432 -U factory -d factory -c "SELECT version();"

# Test Redis
redis-cli ping

# Test Temporal
curl -s http://localhost:8233/api/v1/namespaces

# Test MinIO
curl -s http://localhost:9000/minio/health/live
```

### 2. Security Tools Installation

**Install Trivy (Required for vulnerability scanning)**:
```bash
# Install Trivy
mkdir -p ~/bin
curl -sfL https://raw.githubusercontent.com/aquasecurity/trivy/main/contrib/install.sh | sh -s -- -b ~/bin

# Add to PATH
export PATH=$HOME/bin:$PATH
echo 'export PATH=$HOME/bin:$PATH' >> ~/.bashrc

# Verify installation
trivy --version
```

**Install Syft (Automatic via Go modules)**:
Syft is automatically installed as a Go dependency for SBOM generation.

### 3. CLI Configuration

**Create Configuration File**:
```bash
# Create user configuration
mkdir -p ~/.config/qlf
cat > ~/.config/qlf/config.yaml << 'EOF'
# QLF Configuration
llm:
  default_provider: "bedrock"
  cache_enabled: true
  budget_limit: 100.00

overlays:
  auto_detect: true
  confidence_threshold: 0.7

packaging:
  default_compression: "gzip"
  compression_level: 6
  output_dir: "./packages"
  sbom_enabled: true
  vuln_scan_enabled: true

security:
  sign_packages: false
  vuln_severity_threshold: "medium"
EOF
```

**Verify Configuration**:
```bash
# Test basic CLI functionality
qlf --help

# Test configuration loading
qlf generate "test" --dry-run
```

## LLM Provider Setup (Optional)

### AWS Bedrock Setup

**Prerequisites**:
- AWS Account with Bedrock access
- Claude model access enabled in AWS Bedrock console

**Configuration**:
```bash
# Install AWS CLI
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install

# Configure AWS credentials
aws configure
# Enter: Access Key ID, Secret Access Key, Region (eu-west-2)

# Test Bedrock access
aws bedrock list-foundation-models --region eu-west-2
```

**Environment Variables**:
```bash
# Add to ~/.bashrc
export AWS_PROFILE=your-profile
export AWS_REGION=eu-west-2
export QLF_LLM_PROVIDER=bedrock
```

### Azure OpenAI Setup

**Prerequisites**:
- Azure subscription with OpenAI service
- GPT-4 deployment in UK South region

**Configuration**:
```bash
# Set environment variables
export AZURE_OPENAI_API_KEY="your-api-key"
export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com/"
export AZURE_OPENAI_DEPLOYMENT="gpt-4-turbo"
export QLF_LLM_PROVIDER=azure

# Add to ~/.bashrc for persistence
echo 'export AZURE_OPENAI_API_KEY="your-api-key"' >> ~/.bashrc
echo 'export AZURE_OPENAI_ENDPOINT="https://your-resource.openai.azure.com/"' >> ~/.bashrc
echo 'export QLF_LLM_PROVIDER=azure' >> ~/.bashrc
```

**Test Azure Connection**:
```bash
# Test Azure OpenAI access
curl -H "Authorization: Bearer $AZURE_OPENAI_API_KEY" \
     -H "Content-Type: application/json" \
     "$AZURE_OPENAI_ENDPOINT/openai/deployments?api-version=2023-12-01-preview"
```

## Verification and Testing

### 1. Installation Verification

**Basic Functionality Test**:
```bash
# Test CLI
qlf --help
qlf generate --help
qlf package --help

# Test dry-run generation
qlf generate "simple API" --dry-run

# Test overlay system
qlf overlays list
```

**Infrastructure Health Check**:
```bash
# Check all services
make status

# Test database
psql -h localhost -p 5432 -U factory -d factory -c "\dt"

# Test Temporal
curl http://localhost:8088
# Should show Temporal UI
```

### 2. End-to-End Test

**Complete Pipeline Test**:
```bash
# 1. Generate test application
qlf generate "user management API with PostgreSQL" --output /tmp/test-app

# 2. Verify generation
ls -la /tmp/test-app/
cat /tmp/test-app/README.md

# 3. Package application
qlf package test-app --source /tmp/test-app --language python --framework fastapi

# 4. Verify package
ls -la ./packages/test-app-v1.0.0.qlcapsule
tar -tzf ./packages/test-app-v1.0.0.qlcapsule

# 5. Extract and inspect
tar -xzf ./packages/test-app-v1.0.0.qlcapsule
cat manifest.json | jq .
```

**Expected Results**:
- Generated application code in `/tmp/test-app/`
- Package file `test-app-v1.0.0.qlcapsule` created
- Valid manifest.json with SBOM and metadata

### 3. LLM Integration Test (Optional)

**Test AWS Bedrock**:
```bash
# Generate with Bedrock
qlf generate "FastAPI user service" --provider bedrock --model haiku --output /tmp/bedrock-test

# Verify LLM-generated content
ls -la /tmp/bedrock-test/
# Should contain more sophisticated code than template-based generation
```

**Test Azure OpenAI**:
```bash
# Generate with Azure
qlf generate "Express.js API" --provider azure --model gpt-4 --output /tmp/azure-test

# Compare with Bedrock results
diff -r /tmp/bedrock-test/ /tmp/azure-test/
```

## Development Environment Setup

### For Contributors and Advanced Users

**Full Development Setup**:
```bash
# Clone repository
git clone https://github.com/quantumlayer-factory-hq/quantumlayer-factory.git
cd quantumlayer-factory

# Install Go dependencies
go mod download

# Install development tools
make install-dev-deps

# Start development environment
make dev

# Run all tests
make test

# Build everything
make build-all
```

**Development Tools**:
```bash
# Install additional tools
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install github.com/securecodewarrior/sast-scan@latest
go install github.com/aquasecurity/trivy@latest

# Code quality checks
make lint
make security-scan
make test-coverage
```

### IDE Configuration

**VS Code Setup**:
```json
// .vscode/settings.json
{
  "go.testFlags": ["-v"],
  "go.buildFlags": ["-v"],
  "go.lintTool": "golangci-lint",
  "go.formatTool": "goimports"
}
```

**GoLand/IntelliJ Setup**:
- Enable Go modules support
- Configure test runner for verbose output
- Set up golangci-lint integration

## Production Deployment

### Kubernetes Deployment

**Namespace Setup**:
```bash
# Create production namespace
kubectl create namespace quantumlayer-factory

# Apply resource quotas
kubectl apply -f k8s/resource-quota.yaml
```

**Service Deployment**:
```bash
# Deploy infrastructure
kubectl apply -f k8s/postgres.yaml
kubectl apply -f k8s/redis.yaml
kubectl apply -f k8s/temporal.yaml

# Deploy QLF services
kubectl apply -f k8s/qlf-api.yaml
kubectl apply -f k8s/qlf-worker.yaml
kubectl apply -f k8s/qlf-web.yaml

# Configure ingress
kubectl apply -f k8s/ingress.yaml
```

### Environment Variables for Production

**Required Environment Variables**:
```bash
# Database
DATABASE_URL=postgresql://user:pass@postgres:5432/factory

# Redis
REDIS_URL=redis://redis:6379

# Temporal
TEMPORAL_HOST=temporal:7233

# LLM Providers (optional)
AWS_REGION=eu-west-2
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com/

# Security
SIGNING_KEY_PATH=/etc/qlf/signing-key.pem
VULN_SCAN_ENABLED=true
SBOM_ENABLED=true
```

**Kubernetes Secret Configuration**:
```yaml
# k8s/secrets.yaml
apiVersion: v1
kind: Secret
metadata:
  name: qlf-secrets
type: Opaque
data:
  aws-access-key-id: <base64-encoded>
  aws-secret-access-key: <base64-encoded>
  azure-openai-api-key: <base64-encoded>
  signing-key: <base64-encoded-private-key>
```

## Maintenance and Updates

### Regular Maintenance

**Update Dependencies**:
```bash
# Update Go dependencies
go get -u ./...
go mod tidy

# Update Docker images
docker-compose pull

# Update Trivy database
trivy image --download-db-only
```

**Health Monitoring**:
```bash
# Check system health
make health-check

# Monitor logs
docker-compose logs -f

# Check disk usage
df -h
docker system df
```

### Backup and Recovery

**Database Backup**:
```bash
# Backup PostgreSQL
docker exec factory-postgres-1 pg_dump -U factory factory > backup.sql

# Restore PostgreSQL
docker exec -i factory-postgres-1 psql -U factory factory < backup.sql
```

**Configuration Backup**:
```bash
# Backup configuration
cp ~/.config/qlf/config.yaml config-backup.yaml

# Backup overlays (if customized)
tar -czf overlays-backup.tar.gz overlays/
```

## Troubleshooting Common Setup Issues

### Docker Issues

**Permission Denied**:
```bash
# Add user to docker group
sudo usermod -aG docker $USER
newgrp docker

# Or run with sudo
sudo make dev
```

**Port Conflicts**:
```bash
# Check port usage
netstat -tulpn | grep -E "(5432|6379|7233|8088)"

# Modify ports in docker-compose.yml if needed
```

**Container Startup Failures**:
```bash
# Check logs
docker-compose logs postgres
docker-compose logs temporal

# Restart services
docker-compose restart
```

### Go Build Issues

**Module Download Failures**:
```bash
# Clear module cache
go clean -modcache

# Use Go proxy
export GOPROXY=https://proxy.golang.org,direct

# Retry build
go mod download
make build
```

**Missing Dependencies**:
```bash
# Install system dependencies (Ubuntu)
sudo apt-get update
sudo apt-get install -y build-essential git curl

# Install system dependencies (macOS)
brew install git make
```

### Network Issues

**LLM Provider Connectivity**:
```bash
# Test AWS Bedrock connectivity
aws bedrock list-foundation-models --region eu-west-2

# Test Azure OpenAI connectivity
curl -H "Authorization: Bearer $AZURE_OPENAI_API_KEY" \
     "$AZURE_OPENAI_ENDPOINT/openai/deployments?api-version=2023-12-01-preview"
```

**Docker Network Issues**:
```bash
# Reset Docker networks
docker network prune

# Recreate development environment
make dev-clean
make dev
```

## Platform-Specific Setup

### Ubuntu/Debian Setup

**System Dependencies**:
```bash
# Update system
sudo apt-get update && sudo apt-get upgrade -y

# Install required packages
sudo apt-get install -y \
  curl \
  git \
  make \
  build-essential \
  docker.io \
  docker-compose-v2

# Install Go
wget https://go.dev/dl/go1.21.5.linux-amd64.tar.gz
sudo tar -C /usr/local -xzf go1.21.5.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> ~/.bashrc
source ~/.bashrc

# Configure Docker
sudo systemctl enable docker
sudo systemctl start docker
sudo usermod -aG docker $USER
```

### macOS Setup

**Using Homebrew**:
```bash
# Install Homebrew (if not installed)
/bin/bash -c "$(curl -fsSL https://raw.githubusercontent.com/Homebrew/install/HEAD/install.sh)"

# Install dependencies
brew install go make git docker

# Start Docker Desktop
open /Applications/Docker.app

# Install QLF
curl -L https://releases.quantumlayer.dev/qlf-macos -o qlf
chmod +x qlf
sudo mv qlf /usr/local/bin/
```

### Windows (WSL2) Setup

**Enable WSL2**:
```powershell
# Run in PowerShell as Administrator
wsl --install
wsl --set-default-version 2
```

**Ubuntu in WSL2**:
```bash
# Install Ubuntu 22.04 LTS
wsl --install -d Ubuntu-22.04

# Follow Ubuntu setup instructions above
# Install Docker Desktop for Windows with WSL2 backend
```

## Configuration

### Basic Configuration

**User Configuration** (`~/.config/qlf/config.yaml`):
```yaml
# Core settings
api:
  timeout: 30s
  retry_attempts: 3

# LLM providers (optional)
llm:
  default_provider: "bedrock"  # or "azure"
  cache_enabled: true
  budget_limit: 100.00         # USD per month

# Overlay preferences
overlays:
  auto_detect: true
  confidence_threshold: 0.7
  preferred: ["fintech", "healthcare"]

# Packaging defaults
packaging:
  default_compression: "gzip"
  compression_level: 6
  output_dir: "./packages"
  sbom_enabled: true
  vuln_scan_enabled: true

# Security settings
security:
  sign_packages: false
  signing_key_path: ""
  vuln_severity_threshold: "medium"

# Development settings
development:
  verbose_logging: false
  debug_mode: false
  template_fallback: true
```

### Advanced Configuration

**Project Configuration** (`.qlf.yaml` in project root):
```yaml
# Project-specific settings
project:
  name: "my-project"
  version: "1.0.0"
  author: "Your Team"
  license: "MIT"

# Generation preferences
generation:
  default_language: "python"
  default_framework: "fastapi"
  default_overlays: ["fintech", "pci"]
  include_tests: true
  include_docs: true
  include_docker: true

# Custom overlay paths
overlays:
  custom_paths:
    - "./overlays"
    - "~/.qlf/overlays"

# Build configuration
build:
  dockerfile_template: "custom"
  security_scan: true
  multi_stage: true
```

### LLM Provider Configuration

**AWS Bedrock Configuration**:
```yaml
# ~/.config/qlf/config.yaml
llm:
  providers:
    bedrock:
      region: "eu-west-2"
      models:
        haiku: "anthropic.claude-3-haiku-20240307-v1:0"
        sonnet: "anthropic.claude-3-sonnet-20240229-v1:0"
        sonnet-3-5: "anthropic.claude-3-7-sonnet-20250219-v1:0"
      timeout: 30s
      max_tokens: 4096
```

**Azure OpenAI Configuration**:
```yaml
# ~/.config/qlf/config.yaml
llm:
  providers:
    azure:
      endpoint: "https://your-resource.openai.azure.com/"
      api_version: "2023-12-01-preview"
      models:
        gpt-4: "gpt-4-turbo"
        gpt-35: "gpt-35-turbo"
      timeout: 30s
      max_tokens: 4096
```

## Validation

### Installation Validation

**Quick Validation**:
```bash
# 1. Check CLI
qlf --version

# 2. Check services
make status

# 3. Test generation
qlf generate "hello world API" --dry-run

# 4. Test packaging
mkdir /tmp/test-validation
echo 'package main; import "fmt"; func main() { fmt.Println("Hello") }' > /tmp/test-validation/main.go
qlf package test --source /tmp/test-validation --language go

# 5. Verify package
ls -la ./packages/test-v1.0.0.qlcapsule
```

**Expected Output**:
```
✅ QLF CLI: Working
✅ Services: All healthy
✅ Generation: IR compilation successful
✅ Packaging: Package created successfully
✅ Package: Valid .qlcapsule format
```

### Comprehensive Validation

**Run Full Test Suite**:
```bash
# Unit tests
go test ./... -v

# Integration tests
make test-integration

# CLI tests
make test-cli

# End-to-end test
make test-e2e
```

**Performance Validation**:
```bash
# Benchmark tests
make benchmark

# Load test
make load-test

# Memory usage
make profile-memory
```

## Getting Help

### Documentation Resources
- **[USER_MANUAL.md](USER_MANUAL.md)**: Complete user guide
- **[ARCHITECTURE.md](ARCHITECTURE.md)**: System architecture
- **[USER_TESTING_GUIDE.md](USER_TESTING_GUIDE.md)**: Testing procedures
- **[DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md)**: Development guide

### Support Channels
- **GitHub Issues**: Bug reports and feature requests
- **Documentation**: Comprehensive guides and examples
- **Community**: User forums and discussions

### Common Setup Commands

**Reset Environment**:
```bash
# Clean Docker environment
make dev-clean

# Reset configuration
rm -rf ~/.config/qlf/

# Restart from scratch
make dev
make build
```

**Update Installation**:
```bash
# Update from source
git pull
make build

# Update Docker images
docker-compose pull
make dev-restart
```

## Next Steps

After successful setup:

1. **Read the User Manual**: [USER_MANUAL.md](USER_MANUAL.md) for detailed usage
2. **Try Examples**: Start with simple APIs, then move to complex applications
3. **Explore Overlays**: Test domain-specific generation with overlays
4. **Configure LLM**: Set up AWS Bedrock or Azure OpenAI for AI-powered generation
5. **Production Deployment**: Follow production deployment guidelines

**First Application Tutorial**:
```bash
# Generate your first application
qlf generate "REST API for task management with user authentication and PostgreSQL storage" \
  --output ./my-first-app \
  --verbose

# Explore the generated code
cd ./my-first-app
find . -name "*.py" -o -name "*.go" -o -name "*.js" | head -10

# Package for distribution
qlf package my-first-app --source . --language python --framework fastapi
```

Your QuantumLayer Factory installation is now complete and ready for application generation!