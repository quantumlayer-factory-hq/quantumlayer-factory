# QuantumLayer Factory - User Testing Guide

## Overview
This guide provides step-by-step instructions to test and validate all implemented features of the QuantumLayer Factory system (Weeks 1-7 complete).

## Prerequisites

### 1. Environment Setup
```bash
# Start infrastructure services
make dev

# Verify services are running
docker ps | grep -E "(postgres|redis|temporal|qdrant|minio)"

# Build CLI
make build

# Verify CLI binary
ls -la ./bin/qlf
```

### 2. Verify Infrastructure Health
```bash
# Check database connection
psql -h localhost -p 5432 -U factory -d factory -c "SELECT version();"

# Check Redis
redis-cli ping

# Check Temporal
curl -s http://localhost:8233/api/v1/namespaces | jq .

# Check MinIO
curl -s http://localhost:9000/minio/health/live
```

## Testing Guide by Week

### Week 1: Foundation Testing ✅

#### Static Analysis Gate
```bash
# Test Go vet integration
cd /tmp
mkdir test-go-project
cd test-go-project
cat > main.go << 'EOF'
package main

import "fmt"

func main() {
    var x int
    fmt.Println("Hello")
}
EOF

# Test verifier (should catch unused variable)
go test -v github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/verifier
```

#### Temporal Workflow
```bash
# Start worker
./bin/worker &

# Test workflow execution
./bin/qlf generate "simple API" --dry-run
# Expected: SOC-formatted output with IR compilation
```

#### CLI Basic Commands
```bash
# Test help
./bin/qlf --help

# Test config
./bin/qlf generate --help

# Test dry-run
./bin/qlf generate "user management API" --dry-run --verbose
```

### Week 2: Overlay System Testing ✅

#### Domain Overlays
```bash
# List overlays
./bin/qlf overlays list
# Expected: fintech, healthcare, ecommerce, pci, hipaa, gdpr

# Describe overlay
./bin/qlf overlays describe fintech
# Expected: Financial services overlay details

# Test overlay detection
./bin/qlf generate "payment processing with fraud detection" --suggest-overlays
# Expected: Suggests fintech overlay with confidence score
```

#### Overlay Integration
```bash
# Generate with overlay
./bin/qlf generate "HIPAA-compliant patient data API" --overlay healthcare,hipaa --dry-run
# Expected: Healthcare-specific patterns in generated IR
```

### Week 3: LLM Integration Testing ✅

#### Provider Configuration
```bash
# Test provider flags
./bin/qlf generate "simple API" --provider bedrock --model haiku --dry-run
./bin/qlf generate "simple API" --provider azure --model gpt4 --dry-run
# Expected: Different provider configurations in workflow
```

#### LLM-Enabled Generation
```bash
# Test LLM generation (requires API keys)
export AWS_PROFILE=your-profile
export AZURE_OPENAI_API_KEY=your-key

./bin/qlf generate "FastAPI user service" --provider bedrock --model sonnet
# Expected: LLM-generated FastAPI code instead of templates
```

### Week 4: Multi-Agent Pipeline Testing ✅

#### Agent Types
```bash
# Test all agent types
./bin/qlf generate "full-stack e-commerce platform" --overlay ecommerce --dry-run
# Expected: Backend, Frontend, Database, API, Test agents all triggered

# Verify agent factory
go test -v github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/agents -run TestFactory
```

#### Agent Integration
```bash
# Test agent selection
./bin/qlf generate "React frontend with PostgreSQL backend" --dry-run
# Expected: Frontend agent (React), Backend agent (API), Database agent (PostgreSQL)
```

### Week 5: Verification Mesh Testing ✅

#### Unit Test Gate
```bash
# Create test project
mkdir /tmp/test-verification
cd /tmp/test-verification

# Create Go project with test
cat > main.go << 'EOF'
package main
func Add(a, b int) int { return a + b }
func main() {}
EOF

cat > main_test.go << 'EOF'
package main
import "testing"
func TestAdd(t *testing.T) {
    if Add(2, 3) != 5 {
        t.Error("Add failed")
    }
}
EOF

# Test verification
go test -v github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/verifier -run TestUnitGate
```

#### Contract Test Gate
```bash
# Test contract validation
go test -v github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/verifier -run TestContractGate
# Expected: OpenAPI specification validation
```

#### LLM Repair Loop
```bash
# Test repair functionality
go test -v github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/verifier -run TestRepairLoop
# Expected: LLM-powered issue analysis and repair
```

### Week 6: Preview Deploy Testing ✅

#### Container Builder
```bash
# Test Dockerfile generation
mkdir /tmp/test-container
cd /tmp/test-container
cat > main.go << 'EOF'
package main
import "fmt"
func main() { fmt.Println("Hello Container") }
EOF

# Test builder service
go test -v github.com/quantumlayer-factory-hq/quantumlayer-factory/services/builder
# Expected: Dockerfile generation and security scanning tests pass
```

#### K8s Deploy Service
```bash
# Test K8s manifests
go test -v github.com/quantumlayer-factory-hq/quantumlayer-factory/services/deploy
# Expected: Deployment, Service, Ingress manifest generation tests pass
```

#### Trivy Security Scanning
```bash
# Verify Trivy installation
~/bin/trivy --version
# Expected: Trivy version output

# Test vulnerability scanning
~/bin/trivy fs /tmp/test-container
# Expected: Vulnerability scan results
```

### Week 7: Capsule Packager Testing ✅

#### .qlcapsule Format Testing
```bash
# Create test package
mkdir -p /tmp/test-package-validation
cd /tmp/test-package-validation
cat > main.go << 'EOF'
package main
import "fmt"
func main() { fmt.Println("Hello, QuantumLayer!") }
EOF

# Test packaging
./bin/qlf package test-validation --source /tmp/test-package-validation --language go --framework gin
# Expected: Creates test-validation-v1.0.0.qlcapsule
```

#### Package Content Validation
```bash
# Extract and inspect package
cd ./packages
tar -tzf test-validation-v1.0.0.qlcapsule
# Expected: manifest.json, source files

tar -xzf test-validation-v1.0.0.qlcapsule
cat manifest.json | jq .
# Expected: Valid JSON with version, name, SBOM, attestation
```

#### SBOM Generation Testing
```bash
# Verify SBOM in package
cat manifest.json | jq '.sbom'
# Expected: SBOM with packages, vulnerabilities, signature

# Test SBOM formats
./bin/qlf package test-sbom --source /tmp/test-package-validation --language go --sbom
# Expected: SPDX/CycloneDX format SBOM
```

#### Multi-Channel Delivery Testing
```bash
# Test delivery channels
./bin/qlf package test-delivery --source /tmp/test-package-validation --language go --publish registry,cdn,direct
# Expected: Simulated delivery to multiple channels (registry, CDN, direct)

# Verify delivery service
go test -v github.com/quantumlayer-factory-hq/quantumlayer-factory/services/packager -run TestDelivery
# Expected: All delivery tests pass
```

#### Documentation Generation Testing
```bash
# Test documentation generation
./bin/qlf package test-docs --source /tmp/test-package-validation --language go --generate-docs --docs-format markdown
# Expected: Generated documentation in ./packages/docs/

ls -la ./packages/docs/
cat ./packages/docs/README.md
# Expected: Auto-generated documentation
```

## Comprehensive Integration Tests

### End-to-End Workflow Test
```bash
# Complete pipeline test
./bin/qlf generate "PCI-compliant payment API with fraud detection" \
  --overlay fintech,pci \
  --provider bedrock \
  --model sonnet \
  --output /tmp/payment-api

# Expected output structure:
/tmp/payment-api/
├── backend/           # FastAPI application
├── database/          # PostgreSQL schema
├── api/              # OpenAPI specification
├── tests/            # Unit and integration tests
└── docs/             # Generated documentation
```

### Package and Deploy Pipeline
```bash
# Package the generated application
cd /tmp/payment-api
../quantumlayerplatform-dev-ai-hq/bin/qlf package payment-api \
  --source . \
  --language python \
  --framework fastapi \
  --scan-vulns \
  --generate-docs \
  --publish registry

# Expected: payment-api-v1.0.0.qlcapsule with SBOM and security scan
```

## Verification Checklist

### ✅ Core System Validation

1. **SOC Parser**
   ```bash
   go test ./kernel/soc/... -v
   # ✅ Expected: 11/11 tests pass
   ```

2. **IR Compiler with Overlays**
   ```bash
   go test ./kernel/ir/... -v
   # ✅ Expected: 13+ tests pass, overlay detection working
   ```

3. **Agent Factory**
   ```bash
   go test ./kernel/agents/... -v
   # ✅ Expected: 15+ tests pass, all 5 agents functional
   ```

4. **Temporal Workflows**
   ```bash
   go test ./kernel/workflows/... -v
   # ✅ Expected: 8/8 tests pass
   ```

5. **CLI Commands**
   ```bash
   go test ./cmd/qlf/... -v
   # ✅ Expected: 5+ tests pass
   ```

### ✅ Advanced Features Validation

6. **LLM Integration**
   ```bash
   go test ./kernel/llm/... -v
   # ✅ Expected: 10+ tests pass, provider support
   ```

7. **Prompt System**
   ```bash
   go test ./kernel/prompts/... -v
   # ✅ Expected: 30/30 tests pass
   ```

8. **Verifier Mesh**
   ```bash
   go test ./kernel/verifier/... -v
   # ✅ Expected: 20+ tests pass, all gates working
   ```

9. **Builder Service**
   ```bash
   go test ./services/builder/... -v
   # ✅ Expected: 6/6 tests pass, Trivy integration
   ```

10. **Deploy Service**
    ```bash
    go test ./services/deploy/... -v
    # ✅ Expected: 5/5 tests pass, K8s integration
    ```

11. **Packager Service**
    ```bash
    go test ./services/packager/... -v
    # ✅ Expected: 18/18 tests pass, .qlcapsule format
    ```

### ✅ Feature Demonstrations

#### Natural Language Processing
```bash
# Test complex brief processing
./bin/qlf generate "Build a GDPR-compliant user registration system with OAuth2, email verification, and audit logging" --overlay ecommerce,gdpr --dry-run

# Expected validation:
# ✅ Detects: ecommerce and GDPR overlays automatically
# ✅ Generates: Comprehensive IR with security requirements
# ✅ Includes: OAuth2, email verification, audit logging in IR
# ✅ Applies: GDPR compliance patterns and validation rules
```

#### Multi-Language Support
```bash
# Test different language/framework combinations
./bin/qlf generate "user API" --language python --framework fastapi --dry-run
./bin/qlf generate "user API" --language go --framework gin --dry-run
./bin/qlf generate "user API" --language javascript --framework express --dry-run

# Expected: Language-specific code generation patterns
```

#### Security and Compliance
```bash
# Test PCI compliance
./bin/qlf generate "payment processor" --overlay fintech,pci --dry-run
# Expected: PCI-DSS compliance patterns in generated IR

# Test HIPAA compliance
./bin/qlf generate "patient records API" --overlay healthcare,hipaa --dry-run
# Expected: HIPAA compliance patterns and PHI protection
```

#### Package Security Features
```bash
# Test vulnerability scanning
./bin/qlf package secure-test --source /tmp/test-package-validation --language go --scan-vulns
# Expected: Vulnerability scan results in package

# Test package signing
openssl genrsa -out test.key 2048
./bin/qlf package signed-test --source /tmp/test-package-validation --language go --sign --key test.key
# Expected: Digital signature in package manifest
```

## Performance Benchmarks

### Expected Performance Targets

1. **IR Compilation**: < 100ms for typical briefs
2. **Agent Generation**: 1-5 seconds (template mode)
3. **LLM Generation**: 5-30 seconds (depending on provider/model)
4. **Package Creation**: 1-3 seconds for small applications
5. **Container Build**: 10-60 seconds (depending on language)

### Benchmark Tests
```bash
# Test IR compilation speed
time ./bin/qlf generate "simple CRUD API" --dry-run
# Expected: < 1 second total

# Test package creation speed
time ./bin/qlf package speed-test --source /tmp/test-package-validation --language go
# Expected: 1-3 seconds

# Test full pipeline
time ./bin/qlf generate "user management service" --overlay fintech --output /tmp/benchmark
# Expected: < 30 seconds for complete generation
```

## Validation Checklist

### ✅ Core Functionality
- [ ] CLI commands execute without errors
- [ ] All unit tests pass (100+ tests)
- [ ] Integration tests pass
- [ ] IR compilation produces valid JSON
- [ ] Agent factory selects appropriate agents
- [ ] Overlay detection works automatically
- [ ] Temporal workflows complete successfully

### ✅ Week 6 Features (Preview Deploy)
- [ ] Dockerfile generation for multiple languages
- [ ] Trivy security scanning works
- [ ] K8s manifests generate correctly
- [ ] Preview URL allocation works
- [ ] Health checks configured properly

### ✅ Week 7 Features (Capsule Packager)
- [ ] .qlcapsule files create successfully
- [ ] TAR+GZIP format is valid
- [ ] Manifest.json contains all required fields
- [ ] SBOM generation works with Syft
- [ ] Vulnerability scanning integrates with Trivy
- [ ] Digital signatures work with RSA/ECDSA
- [ ] Documentation generation produces output
- [ ] Multi-channel delivery simulates correctly

### ✅ Security Features
- [ ] No secrets in generated code
- [ ] Proper input validation
- [ ] Secure defaults applied
- [ ] Vulnerability scanning catches issues
- [ ] Digital signatures verify correctly
- [ ] SBOM includes dependency information

### ✅ LLM Integration (if configured)
- [ ] AWS Bedrock integration works
- [ ] Azure OpenAI integration works
- [ ] Provider failover functions
- [ ] Cost tracking operates
- [ ] Response caching works
- [ ] Budget limits enforced

## Common Issues and Troubleshooting

### Infrastructure Issues
```bash
# If Docker services not running
make dev-restart

# If Temporal not accessible
docker logs factory-temporal-1

# If database connection fails
docker exec -it factory-postgres-1 psql -U factory -d factory
```

### CLI Issues
```bash
# If CLI not found
make build
export PATH=$PWD/bin:$PATH

# If worker not responding
pkill worker
./bin/worker &
```

### Package Creation Issues
```bash
# If Trivy not found
which trivy
# Should show: /home/satish/bin/trivy

# If SBOM generation fails
which syft
go list -m github.com/anchore/syft
```

### Test Failures
```bash
# Run specific package tests
go test ./services/packager/... -v -run TestPackagerService

# Run with race detection
go test ./... -race

# Check for lint issues
golangci-lint run
```

## Success Criteria Validation

### Week 1-5 Success Metrics ✅
All previously validated and working.

### Week 6 Success Metrics ✅
```bash
# Containerization pipeline
./bin/qlf generate "simple service" --output /tmp/container-test
# Should generate: Dockerfile, K8s manifests, health checks

# Security scanning
trivy fs /tmp/container-test
# Should show: Vulnerability scan results
```

### Week 7 Success Metrics ✅
```bash
# Package creation
./bin/qlf package validation-test --source /tmp/test-package-validation --language go
# Should create: validation-test-v1.0.0.qlcapsule (~ 838 bytes)

# Package inspection
tar -tzf ./packages/validation-test-v1.0.0.qlcapsule
# Should list: manifest.json + source files

# Package manifest validation
tar -xzf ./packages/validation-test-v1.0.0.qlcapsule
cat manifest.json | jq '.version, .name, .sbom, .attestation'
# Should show: Valid manifest with all fields
```

## Complete System Test

### Full Pipeline Validation
```bash
# 1. Generate application
./bin/qlf generate "PCI-compliant payment processor with PostgreSQL" \
  --overlay fintech,pci \
  --output /tmp/full-test

# 2. Verify generation
ls -la /tmp/full-test/
# Expected: backend/, database/, api/, tests/, docs/

# 3. Package application
./bin/qlf package payment-processor \
  --source /tmp/full-test \
  --language python \
  --framework fastapi \
  --scan-vulns \
  --generate-docs \
  --version 1.0.0

# 4. Validate package
ls -la ./packages/payment-processor-v1.0.0.qlcapsule
tar -tzf ./packages/payment-processor-v1.0.0.qlcapsule
tar -xzf ./packages/payment-processor-v1.0.0.qlcapsule
cat manifest.json | jq .

# 5. Check documentation
ls -la ./packages/docs/
cat ./packages/docs/README.md
```

### Expected Final Validation
```bash
# All tests should pass
go test ./... -v
# Expected: 100+ tests passing across all packages

# CLI should be fully functional
./bin/qlf --help
./bin/qlf generate --help
./bin/qlf package --help
./bin/qlf overlays --help

# Services should be healthy
curl http://localhost:8080/health
# Expected: {"status": "healthy"}
```

## Reporting Issues

If any test fails or functionality doesn't work as expected:

1. **Check Prerequisites**: Ensure Docker services running, Trivy installed
2. **Run Specific Tests**: Isolate the failing component
3. **Check Logs**: Review Docker logs and CLI output
4. **Verify Configuration**: Ensure environment variables set correctly
5. **Test Step-by-Step**: Follow this guide systematically

## Summary

This testing guide validates:
- ✅ **7 weeks of development** (Foundation → Packaging)
- ✅ **100+ unit tests** across all packages
- ✅ **End-to-end workflows** from brief to packaged application
- ✅ **Security features** (SBOM, vulnerability scanning, digital signatures)
- ✅ **Multi-language support** (Go, Python, JavaScript, Java, Rust)
- ✅ **Production readiness** (containerization, K8s, monitoring)

**Result**: Complete validation that the QuantumLayer Factory system delivers on all claimed functionality.