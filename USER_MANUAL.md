# QuantumLayer Factory - User Manual

## Table of Contents
1. [Getting Started](#getting-started)
2. [Core Concepts](#core-concepts)
3. [Command Reference](#command-reference)
4. [Real-World Examples](#real-world-examples)
5. [Best Practices](#best-practices)
6. [Troubleshooting](#troubleshooting)

## Getting Started

### What You Can Build
QuantumLayer Factory generates complete, production-ready applications from natural language descriptions:

- **APIs and Services**: REST APIs, GraphQL services, microservices
- **Web Applications**: Full-stack applications with frontend and backend
- **Data Processing**: ETL pipelines, data analytics applications
- **Compliance-Ready Systems**: PCI-DSS, HIPAA, GDPR compliant applications

### Quick Examples
```bash
# Simple API
qlf generate "user management API with JWT authentication"

# Domain-specific application
qlf generate "PCI-compliant payment processor with fraud detection" --overlay fintech,pci

# Full-stack application
qlf generate "React dashboard with PostgreSQL backend for inventory management" --overlay ecommerce
```

## Core Concepts

### 1. Natural Language Briefs
Write your application requirements in plain English. The system understands:

**Technical Requirements**:
- "REST API with JWT authentication"
- "React frontend with TypeScript"
- "PostgreSQL database with migrations"

**Business Logic**:
- "User registration and login"
- "Payment processing with Stripe"
- "Inventory management with low-stock alerts"

**Compliance Requirements**:
- "PCI-DSS compliant card processing"
- "HIPAA-compliant patient data handling"
- "GDPR-compliant user consent management"

### 2. Overlays (Domain Expertise)
Overlays inject specialized knowledge into your applications:

#### Domain Overlays
- **fintech**: Payment processing, fraud detection, financial regulations
- **healthcare**: Patient data, medical workflows, healthcare standards
- **ecommerce**: Inventory, orders, customer management, pricing

#### Compliance Overlays
- **pci**: PCI-DSS compliance for payment card data
- **hipaa**: HIPAA compliance for healthcare data
- **gdpr**: GDPR compliance for data protection

### 3. Multi-Agent Generation
Different agents specialize in different components:
- **Backend Agent**: API services, business logic
- **Frontend Agent**: User interfaces, web applications
- **Database Agent**: Schemas, migrations, queries
- **API Agent**: OpenAPI specs, documentation
- **Test Agent**: Unit tests, integration tests

### 4. Verification and Quality
Every generated application goes through quality gates:
- **Static Analysis**: Code quality and security checks
- **Unit Tests**: Automated test execution
- **Contract Tests**: API specification validation
- **Security Scanning**: Vulnerability assessment

## Command Reference

### Core Commands

#### `qlf generate`
Generate applications from natural language briefs.

**Basic Usage**:
```bash
qlf generate "brief description" [flags]
```

**Key Flags**:
- `--output, -o`: Output directory (default: current directory)
- `--dry-run`: Preview without generating files
- `--verbose`: Detailed output
- `--overlay`: Specify overlays (e.g., `--overlay fintech,pci`)
- `--provider`: LLM provider (`bedrock`, `azure`)
- `--model`: Specific model (`haiku`, `sonnet`, `gpt-4`)

**Examples**:
```bash
# Basic generation
qlf generate "user authentication API"

# With domain expertise
qlf generate "payment processing service" --overlay fintech

# With compliance
qlf generate "patient records API" --overlay healthcare,hipaa

# LLM-powered generation
qlf generate "complex microservice architecture" --provider bedrock --model sonnet

# Preview before generating
qlf generate "e-commerce platform" --overlay ecommerce --dry-run
```

#### `qlf package`
Package applications into secure .qlcapsule format.

**Basic Usage**:
```bash
qlf package [name] --source <path> --language <lang> [flags]
```

**Key Flags**:
- `--source, -s`: Source code path (required)
- `--language, -l`: Programming language (required)
- `--framework, -f`: Framework (optional)
- `--version, -v`: Package version (default: 1.0.0)
- `--scan-vulns`: Enable vulnerability scanning (default: true)
- `--sbom`: Generate SBOM (default: true)
- `--sign`: Sign package with digital signature
- `--key`: Path to signing key
- `--publish`: Delivery channels (registry, cdn, direct)

**Examples**:
```bash
# Basic packaging
qlf package my-app --source ./src --language go

# With security features
qlf package secure-app --source ./src --language python --framework fastapi --scan-vulns --sign --key ./private.pem

# Multi-channel publishing
qlf package distributed-app --source ./src --language javascript --framework express --publish registry,cdn,direct
```

#### `qlf overlays`
Manage and explore domain overlays.

**Subcommands**:
```bash
# List available overlays
qlf overlays list

# Describe overlay details
qlf overlays describe fintech

# Suggest overlays for a brief
qlf overlays suggest "payment processing with fraud detection"
```

### Advanced Commands

#### `qlf preview`
Deploy applications to ephemeral preview environments.

```bash
# Deploy generated application
qlf preview deploy ./generated-app

# Deploy with custom configuration
qlf preview deploy ./app --namespace my-preview --ttl 2h

# List active previews
qlf preview list

# Clean up preview
qlf preview cleanup my-preview
```

#### `qlf status`
Monitor workflow and system status.

```bash
# Check workflow status
qlf status workflow <workflow-id>

# System health check
qlf status health

# Service status
qlf status services
```

## Real-World Examples

### Example 1: PCI-Compliant Payment API

**Requirement**: Build a payment processing API that complies with PCI-DSS requirements.

```bash
# Generate the application
qlf generate "PCI-compliant payment processing API with fraud detection and audit logging" \
  --overlay fintech,pci \
  --output ./payment-api

# Review generated structure
ls -la ./payment-api/
# Expected:
# ├── backend/           # FastAPI application with PCI compliance
# ├── database/          # PostgreSQL schema with audit tables
# ├── api/              # OpenAPI spec with security definitions
# ├── tests/            # Unit and integration tests
# └── docs/             # Generated documentation

# Package for distribution
qlf package payment-api \
  --source ./payment-api \
  --language python \
  --framework fastapi \
  --scan-vulns \
  --sign \
  --key ./signing-key.pem \
  --publish registry

# Deploy for testing
qlf preview deploy ./payment-api --namespace payment-preview
```

**Generated Features**:
- PCI-DSS compliant card data handling
- Encrypted data storage
- Audit logging for all transactions
- Fraud detection algorithms
- Secure API endpoints with rate limiting
- Comprehensive test suite

### Example 2: HIPAA-Compliant Healthcare App

**Requirement**: Patient data management system with HIPAA compliance.

```bash
# Generate healthcare application
qlf generate "HIPAA-compliant patient management system with React frontend and encrypted data storage" \
  --overlay healthcare,hipaa \
  --provider bedrock \
  --model sonnet \
  --output ./healthcare-app

# Package with documentation
qlf package healthcare-app \
  --source ./healthcare-app \
  --language python \
  --framework fastapi \
  --generate-docs \
  --docs-format html \
  --scan-vulns
```

**Generated Features**:
- HIPAA-compliant PHI handling
- Encrypted data at rest and in transit
- Audit trails for all data access
- Patient consent management
- Role-based access control
- Business Associate Agreement (BAA) compliance

### Example 3: E-commerce Platform

**Requirement**: Full-stack e-commerce platform with inventory management.

```bash
# Generate e-commerce platform
qlf generate "full-stack e-commerce platform with React frontend, inventory management, order processing, and payment integration" \
  --overlay ecommerce \
  --output ./ecommerce-platform

# Package and deploy
qlf package ecommerce-platform \
  --source ./ecommerce-platform \
  --language javascript \
  --framework express \
  --publish cdn,direct

qlf preview deploy ./ecommerce-platform --ttl 24h
```

**Generated Features**:
- React frontend with shopping cart
- Express.js backend with REST API
- PostgreSQL database with product/order schemas
- Inventory management with low-stock alerts
- Payment processing integration
- Order fulfillment workflow

## Best Practices

### 1. Writing Effective Briefs

**Be Specific About Requirements**:
```bash
# Good: Specific and detailed
qlf generate "REST API for user management with JWT authentication, password reset via email, role-based permissions, and PostgreSQL storage"

# Avoid: Too vague
qlf generate "user system"
```

**Include Compliance Requirements**:
```bash
# Specify compliance needs upfront
qlf generate "payment processing API that must be PCI-DSS compliant with tokenization and audit logging"
```

**Mention Technology Preferences**:
```bash
# Specify tech stack when needed
qlf generate "microservice with Go and Gin framework, Redis caching, and PostgreSQL persistence"
```

### 2. Using Overlays Effectively

**Automatic Detection**:
```bash
# Let the system detect overlays
qlf generate "credit card payment processing with fraud prevention"
# System will automatically suggest: fintech, pci overlays
```

**Explicit Overlay Selection**:
```bash
# Be explicit for critical compliance
qlf generate "patient data API" --overlay healthcare,hipaa
```

**Overlay Combination**:
```bash
# Combine domain and compliance overlays
qlf generate "medical billing system" --overlay healthcare,fintech,hipaa,pci
```

### 3. LLM Provider Selection

**Model Selection by Complexity**:
```bash
# Simple tasks: Use fast, cost-effective models
qlf generate "CRUD API" --provider bedrock --model haiku

# Complex tasks: Use advanced reasoning models
qlf generate "distributed microservice architecture" --provider bedrock --model sonnet

# Creative tasks: Use latest models
qlf generate "innovative user experience design" --provider azure --model gpt-4
```

**Provider Comparison**:
```bash
# Compare approaches
qlf generate "recommendation engine" --compare bedrock,azure --dry-run
```

### 4. Quality Assurance

**Always Use Dry-Run First**:
```bash
# Preview before generating
qlf generate "complex system" --dry-run --verbose
```

**Enable All Security Features**:
```bash
# Comprehensive security
qlf package my-app \
  --source ./src \
  --language python \
  --scan-vulns \
  --sbom \
  --sign \
  --key ./private.pem
```

**Test Generated Applications**:
```bash
# Follow the testing guide
# See USER_TESTING_GUIDE.md for comprehensive validation
```

### 5. Packaging and Distribution

**Production Packaging**:
```bash
# Include all security features
qlf package production-app \
  --source ./app \
  --language go \
  --framework gin \
  --version 1.2.0 \
  --author "Your Team" \
  --license MIT \
  --scan-vulns \
  --sbom \
  --sign \
  --key ./production.pem \
  --generate-docs \
  --publish registry,cdn
```

**Development Packaging**:
```bash
# Lightweight for development
qlf package dev-app \
  --source ./app \
  --language python \
  --version 0.1.0-dev \
  --compression lz4
```

## Configuration

### Environment Setup

**Required Environment Variables**:
```bash
# For LLM integration (optional)
export AWS_PROFILE=your-aws-profile
export AZURE_OPENAI_API_KEY=your-azure-key
export AZURE_OPENAI_ENDPOINT=your-azure-endpoint

# For package signing (optional)
export QLF_SIGNING_KEY_PATH=/path/to/private.pem
```

**Configuration File** (`~/.qlf.yaml`):
```yaml
# LLM Configuration
llm:
  default_provider: bedrock
  budget_limit: 100.00  # USD per month
  cache_enabled: true

# Overlay Configuration
overlays:
  auto_detect: true
  confidence_threshold: 0.7
  preferred: ["fintech", "healthcare"]

# Packaging Configuration
packaging:
  default_compression: gzip
  compression_level: 6
  output_dir: "./packages"
  sbom_enabled: true
  vuln_scan_enabled: true

# Security Configuration
security:
  sign_packages: false
  signing_key_path: ""
  vuln_severity_threshold: "medium"
```

### Workspace Configuration

**Project-Level Configuration** (`.qlf.yaml`):
```yaml
# Project-specific settings
project:
  name: "my-project"
  default_language: "python"
  default_framework: "fastapi"
  default_overlays: ["fintech", "pci"]

# Generation preferences
generation:
  output_format: "structured"
  include_tests: true
  include_docs: true
  include_docker: true
```

## Advanced Usage

### Multi-Language Projects

**Generate Multiple Components**:
```bash
# Backend service
qlf generate "user management API with PostgreSQL" \
  --overlay fintech \
  --output ./backend \
  --language python \
  --framework fastapi

# Frontend application
qlf generate "user management dashboard with React and TypeScript" \
  --overlay fintech \
  --output ./frontend \
  --language javascript \
  --framework react

# Package both components
qlf package full-stack-app \
  --source . \
  --language python \
  --artifacts "./backend,./frontend" \
  --manifests "./k8s/*.yaml"
```

### CI/CD Integration

**GitHub Actions Example**:
```yaml
# .github/workflows/qlf-build.yml
name: QLF Build and Package
on: [push]
jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Setup QLF
        run: |
          curl -L https://releases.quantumlayer.dev/qlf-linux -o qlf
          chmod +x qlf
      - name: Generate Application
        run: |
          ./qlf generate "${{ github.event.head_commit.message }}" \
            --overlay ${{ vars.QLF_OVERLAYS }} \
            --output ./generated
      - name: Package Application
        run: |
          ./qlf package ${{ github.repository }} \
            --source ./generated \
            --language ${{ vars.LANGUAGE }} \
            --version ${{ github.ref_name }} \
            --scan-vulns \
            --sbom \
            --publish registry
```

### Custom Overlays

**Creating Custom Domain Overlay**:
```yaml
# overlays/domains/custom.yaml
name: custom
description: Custom domain patterns
version: "1.0"

prompt_enhancements:
  backend:
    before:
      - "Implement custom business logic patterns"
      - "Add domain-specific validation rules"

  database:
    before:
      - "Create domain-specific entities"
      - "Add custom indexes and constraints"

patterns:
  - pattern: "custom pattern keyword"
    confidence: 0.8
    enhancement: "Apply custom domain logic"

security_requirements:
  - "Custom security pattern"
  - "Domain-specific encryption"

dependencies: []
```

## Real-World Examples

### Example 1: Fintech Trading Platform

**Business Requirement**: Build a stock trading platform with real-time quotes, portfolio management, and regulatory compliance.

```bash
# Generate trading platform
qlf generate "stock trading platform with real-time quotes, portfolio management, order execution, risk assessment, and regulatory reporting for MiFID II compliance" \
  --overlay fintech \
  --provider bedrock \
  --model sonnet \
  --output ./trading-platform

# Generated structure:
./trading-platform/
├── backend/
│   ├── main.py              # FastAPI application
│   ├── models/
│   │   ├── portfolio.py     # Portfolio management
│   │   ├── orders.py        # Order execution
│   │   └── quotes.py        # Real-time quotes
│   ├── services/
│   │   ├── trading_service.py
│   │   ├── risk_service.py
│   │   └── compliance_service.py
│   └── requirements.txt
├── database/
│   ├── schema.sql           # Trading-specific tables
│   └── migrations/
├── api/
│   └── openapi.yaml         # Trading API specification
└── tests/
    ├── test_trading.py
    ├── test_portfolio.py
    └── test_compliance.py

# Package for production
qlf package trading-platform \
  --source ./trading-platform \
  --language python \
  --framework fastapi \
  --version 1.0.0 \
  --author "Trading Team" \
  --license "Proprietary" \
  --scan-vulns \
  --sbom \
  --sign \
  --key ./production.pem \
  --generate-docs \
  --publish registry,cdn
```

### Example 2: Healthcare Patient Portal

**Business Requirement**: Patient portal with appointment scheduling, medical records, and HIPAA compliance.

```bash
# Generate patient portal
qlf generate "HIPAA-compliant patient portal with React frontend, appointment scheduling, medical records management, provider communication, and encrypted PHI storage" \
  --overlay healthcare,hipaa \
  --provider azure \
  --model gpt-4 \
  --output ./patient-portal

# Generated structure:
./patient-portal/
├── frontend/
│   ├── src/
│   │   ├── components/
│   │   │   ├── AppointmentScheduler.tsx
│   │   │   ├── MedicalRecords.tsx
│   │   │   └── SecureMessaging.tsx
│   │   ├── hooks/
│   │   │   └── useEncryptedData.ts
│   │   └── utils/
│   │       └── hipaaCompliance.ts
│   └── package.json
├── backend/
│   ├── models/
│   │   ├── patient.py       # HIPAA-compliant patient model
│   │   ├── appointment.py   # Appointment management
│   │   └── medical_record.py
│   ├── services/
│   │   ├── encryption_service.py
│   │   ├── audit_service.py
│   │   └── consent_service.py
│   └── main.py
└── docs/
    ├── HIPAA_Compliance.md
    ├── Privacy_Policy.md
    └── API_Documentation.md

# Deploy for testing
qlf preview deploy ./patient-portal --namespace healthcare-preview --ttl 4h
```

### Example 3: E-commerce Microservices

**Business Requirement**: Microservices-based e-commerce platform with inventory, orders, payments, and notifications.

```bash
# Generate microservices platform
qlf generate "microservices e-commerce platform with inventory service, order service, payment service, notification service, API gateway, and React admin dashboard" \
  --overlay ecommerce \
  --output ./ecommerce-microservices

# Generated structure:
./ecommerce-microservices/
├── services/
│   ├── inventory-service/   # Go service with Gin
│   ├── order-service/       # Python service with FastAPI
│   ├── payment-service/     # Go service with Stripe integration
│   └── notification-service/ # Node.js service with email/SMS
├── gateway/
│   └── api-gateway/         # Kong/Nginx configuration
├── frontend/
│   └── admin-dashboard/     # React TypeScript application
├── database/
│   ├── inventory.sql
│   ├── orders.sql
│   └── payments.sql
└── docker-compose.yaml

# Package each service individually
for service in inventory order payment notification; do
  qlf package ${service}-service \
    --source ./ecommerce-microservices/services/${service}-service \
    --language $(detect_language ${service}) \
    --scan-vulns \
    --publish registry
done

# Package complete platform
qlf package ecommerce-platform \
  --source ./ecommerce-microservices \
  --language docker-compose \
  --manifests "./k8s/*.yaml" \
  --artifacts "./services/*/dist/*" \
  --generate-docs \
  --publish registry,cdn
```

## Best Practices

### 1. Brief Writing Guidelines

**Use Clear, Specific Language**:
```bash
# Good: Specific requirements
"REST API for user authentication with JWT tokens, password reset via email, and role-based authorization"

# Avoid: Vague requirements
"user stuff with security"
```

**Include Non-Functional Requirements**:
```bash
# Include performance, security, compliance
"high-performance user API that handles 10k requests/second with Redis caching, supports OAuth2, and includes audit logging for compliance"
```

**Specify Technology When Important**:
```bash
# When you have specific tech requirements
"Go microservice with Gin framework, PostgreSQL database, Redis caching, and gRPC communication"
```

### 2. Overlay Strategy

**Start with Domain, Add Compliance**:
```bash
# First apply domain expertise
qlf generate "payment system" --overlay fintech

# Then add compliance requirements
qlf generate "payment system" --overlay fintech,pci,gdpr
```

**Use Automatic Detection for Exploration**:
```bash
# Let system suggest overlays
qlf generate "medical billing system" --suggest-overlays
# Review suggestions, then generate with appropriate overlays
```

### 3. Quality Assurance Workflow

**Development Cycle**:
```bash
# 1. Preview and validate
qlf generate "my application" --overlay domain --dry-run --verbose

# 2. Generate application
qlf generate "my application" --overlay domain --output ./app

# 3. Review generated code
ls -la ./app/
cat ./app/README.md

# 4. Test generated application
cd ./app
# Run language-specific tests (go test, pytest, npm test)

# 5. Package with security
qlf package my-app --source ./app --language go --scan-vulns --sbom

# 6. Deploy for testing
qlf preview deploy ./app --namespace test-env
```

### 4. Security Best Practices

**Always Enable Security Features**:
```bash
# Production packaging should include:
qlf package prod-app \
  --scan-vulns \              # Vulnerability scanning
  --sbom \                    # Software Bill of Materials
  --sign \                    # Digital signature
  --key ./production.pem      # Signing key
```

**Use Appropriate Compliance Overlays**:
```bash
# Financial applications
qlf generate "banking app" --overlay fintech,pci

# Healthcare applications
qlf generate "medical app" --overlay healthcare,hipaa

# Data processing applications
qlf generate "data platform" --overlay gdpr
```

**Review Security Scan Results**:
```bash
# Check vulnerability scan results
tar -xzf my-app-v1.0.0.qlcapsule
cat manifest.json | jq '.sbom.vulnerabilities'
```

### 5. Performance Optimization

**Choose Appropriate Models**:
- **Haiku**: Simple CRUD, basic APIs (fast, cost-effective)
- **Sonnet**: Complex business logic, multi-component systems
- **GPT-4**: Advanced reasoning, architectural decisions

**Use Caching**:
- LLM responses are automatically cached
- Repeated similar briefs will generate faster
- Template fallback for offline development

## Troubleshooting

### Common Issues

#### CLI Not Found
```bash
# Build CLI if missing
make build
export PATH=$PWD/bin:$PATH
```

#### Services Not Running
```bash
# Start infrastructure
make dev

# Check service health
docker ps | grep -E "(postgres|redis|temporal)"
```

#### Generation Fails
```bash
# Check Temporal worker
./bin/worker &

# Check overlay syntax
qlf overlays describe <overlay-name>

# Try simpler brief first
qlf generate "simple API" --dry-run
```

#### Package Creation Fails
```bash
# Check Trivy installation
which trivy
# Should show: /home/satish/bin/trivy

# Verify source path exists
ls -la <source-path>

# Check language is supported
qlf package --help | grep -A 10 "language"
```

#### LLM Integration Issues
```bash
# Check provider configuration
echo $AWS_PROFILE
echo $AZURE_OPENAI_API_KEY

# Test with template fallback
qlf generate "test app" --dry-run
# Should work without LLM
```

### Debug Mode

**Enable Verbose Output**:
```bash
# Detailed generation information
qlf generate "my app" --verbose --dry-run

# Temporal workflow debugging
qlf status workflow <workflow-id>
```

**Log Inspection**:
```bash
# Check Docker logs
docker logs factory-temporal-1
docker logs factory-postgres-1

# Check worker logs
./bin/worker --log-level debug
```

### Getting Help

**Command Help**:
```bash
# General help
qlf --help

# Command-specific help
qlf generate --help
qlf package --help
qlf overlays --help
```

**Documentation References**:
- **Architecture**: See [ARCHITECTURE.md](ARCHITECTURE.md)
- **Setup**: See [SETUP_GUIDE.md](SETUP_GUIDE.md)
- **Testing**: See [USER_TESTING_GUIDE.md](USER_TESTING_GUIDE.md)
- **Development**: See [DEVELOPER_GUIDE.md](DEVELOPER_GUIDE.md)

## Performance Expectations

### Generation Times
- **Simple API**: 5-30 seconds
- **Full-stack Application**: 1-3 minutes
- **Complex Microservices**: 3-5 minutes

### Package Creation
- **Small Applications**: 1-3 seconds
- **Medium Applications**: 3-10 seconds
- **Large Applications**: 10-30 seconds

### Deployment
- **Preview Environment**: 30-120 seconds
- **Container Build**: 10-60 seconds
- **K8s Deployment**: 30-90 seconds

## Summary

QuantumLayer Factory provides a complete platform for AI-powered application generation with:
- **Natural Language Input**: Describe applications in plain English
- **Domain Expertise**: Built-in knowledge for fintech, healthcare, e-commerce
- **Security by Design**: SBOM, vulnerability scanning, digital signatures
- **Production Ready**: Containerization, K8s deployment, documentation
- **Quality Assurance**: Multi-stage verification with auto-repair

Follow this manual to effectively use all features and generate production-ready applications efficiently.