# QuantumLayer Factory - API Reference

## Overview

This document provides comprehensive API reference for all QuantumLayer Factory components, including CLI commands, Go packages, and service interfaces.

## CLI API Reference

### `qlf generate`

Generate applications from natural language briefs.

#### Syntax
```bash
qlf generate <brief> [flags]
```

#### Parameters

| Parameter | Type | Required | Description | Default |
|-----------|------|----------|-------------|---------|
| `brief` | string | Yes | Natural language application description | - |

#### Flags

**Output Control**:
| Flag | Short | Type | Description | Default |
|------|-------|------|-------------|---------|
| `--output` | `-o` | string | Output directory path | `.` |
| `--dry-run` | | bool | Preview without generating files | `false` |
| `--verbose` | `-v` | bool | Detailed output | `false` |
| `--async` | | bool | Asynchronous generation | `false` |

**Overlay Management**:
| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--overlay` | []string | Specify overlays (comma-separated) | `[]` |
| `--suggest-overlays` | bool | Suggest overlays without generation | `false` |
| `--auto-detect` | bool | Enable automatic overlay detection | `true` |

**LLM Configuration**:
| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--provider` | string | LLM provider (`bedrock`, `azure`) | `bedrock` |
| `--model` | string | Specific model (`haiku`, `sonnet`, `gpt-4`) | `sonnet` |
| `--compare` | []string | Compare multiple providers | `[]` |
| `--temperature` | float | Generation creativity (0.0-1.0) | `0.7` |

#### Examples

**Basic Generation**:
```bash
# Simple API
qlf generate "user management API with PostgreSQL"

# With output directory
qlf generate "payment service" --output ./payment-app

# Preview generation
qlf generate "complex system" --dry-run --verbose
```

**Overlay Usage**:
```bash
# Automatic overlay detection
qlf generate "medical billing system with patient data encryption"
# Automatically applies: healthcare, hipaa overlays

# Explicit overlay specification
qlf generate "payment processor" --overlay fintech,pci

# Overlay suggestions
qlf generate "healthcare application" --suggest-overlays
```

**LLM Configuration**:
```bash
# Provider selection
qlf generate "API service" --provider bedrock --model sonnet
qlf generate "API service" --provider azure --model gpt-4

# Provider comparison
qlf generate "microservice architecture" --compare bedrock,azure --dry-run
```

#### Response Format

**Dry-Run Output** (SOC Format):
```json
{
  "type": "generation_preview",
  "timestamp": "2024-01-15T10:30:00Z",
  "brief": "user management API with PostgreSQL",
  "ir": {
    "app": {
      "name": "user-management-api",
      "type": "api",
      "language": "python",
      "framework": "fastapi"
    },
    "api": {
      "type": "rest",
      "endpoints": [...]
    },
    "data": {
      "primary_store": "postgresql",
      "entities": [...]
    }
  },
  "overlays": ["fintech"],
  "confidence": 0.92,
  "estimated_files": 12,
  "estimated_lines": 450
}
```

### `qlf package`

Package applications into secure .qlcapsule format.

#### Syntax
```bash
qlf package [name] --source <path> --language <lang> [flags]
```

#### Parameters

| Parameter | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | No* | Package name (auto-detected if not provided) |

*Required via `--source` path basename or first argument

#### Required Flags

| Flag | Short | Type | Description |
|------|-------|------|-------------|
| `--source` | `-s` | string | Source code path |
| `--language` | `-l` | string | Programming language |

#### Optional Flags

**Basic Package Information**:
| Flag | Short | Type | Description | Default |
|------|-------|------|-------------|---------|
| `--name` | `-n` | string | Package name | basename of source |
| `--version` | `-v` | string | Package version | `1.0.0` |
| `--description` | `-d` | string | Package description | `""` |
| `--author` | | string | Package author | `""` |
| `--license` | | string | Package license | `""` |

**Build Configuration**:
| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--framework` | `-f` | string | Framework or library | `""` |
| `--artifacts` | []string | Build artifacts to include | `[]` |
| `--manifests` | []string | Deployment manifests | `[]` |
| `--docs` | []string | Documentation files | `[]` |

**Security Options**:
| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--sbom` | bool | Generate SBOM | `true` |
| `--scan-vulns` | bool | Scan for vulnerabilities | `true` |
| `--sign` | bool | Sign the package | `false` |
| `--key` | string | Path to signing key | `""` |

**Output Options**:
| Flag | Short | Type | Description | Default |
|------|-------|------|-------------|---------|
| `--output-dir` | | string | Output directory | `./packages` |
| `--output` | `-o` | string | Output file path | `""` |
| `--compression` | | string | Compression type | `gzip` |
| `--compression-level` | | int | Compression level (1-9) | `6` |

**Publishing**:
| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--publish` | []string | Delivery channels | `[]` |
| `--tags` | []string | Package tags | `[]` |
| `--labels` | []string | Package labels (key=value) | `[]` |
| `--public` | bool | Make package public | `false` |

#### Examples

**Basic Packaging**:
```bash
# Minimal package
qlf package my-app --source ./src --language go

# With metadata
qlf package my-app \
  --source ./src \
  --language python \
  --framework fastapi \
  --version 1.2.0 \
  --author "Development Team" \
  --license MIT \
  --description "User management API"
```

**Security-Enhanced Packaging**:
```bash
# Full security features
qlf package secure-app \
  --source ./src \
  --language go \
  --framework gin \
  --scan-vulns \
  --sbom \
  --sign \
  --key ./private.pem
```

**Multi-Channel Publishing**:
```bash
# Publish to multiple channels
qlf package distributed-app \
  --source ./src \
  --language javascript \
  --framework express \
  --publish registry,cdn,direct \
  --tags "api,microservice" \
  --labels "team=backend,env=production" \
  --public
```

#### Response Format

**Package Creation Output**:
```
Creating package: my-app v1.0.0
Source: ./src
Language: python
Framework: fastapi

📦 Package: ./packages/my-app-v1.0.0.qlcapsule
📏 Size: 2.3 MB
🔍 Hash: sha256:a1b2c3d4e5f6...
⏱️  Build time: 1.234s
🗜️  Compression ratio: 67.89%

🛡️  Security scan results:
   Critical: 0
   High: 1
   Medium: 3
   Low: 5
   Fixable: 2

📚 Documentation generated: ./packages/docs/README.md
   Format: markdown
   Size: 45.2 KB
   Sections: overview, installation, configuration, deployment

🚀 Publishing package...
✅ Package published successfully!
   registry: https://registry.quantumlayer.dev/my-app/1.0.0
   cdn: https://cdn.quantumlayer.dev/packages/my-app-v1.0.0.qlcapsule

🎉 Package my-app ready!
```

### `qlf overlays`

Manage and explore domain overlays.

#### Subcommands

**`qlf overlays list`**:
```bash
# List all available overlays
qlf overlays list

# List specific type
qlf overlays list --type domain
qlf overlays list --type compliance
```

**`qlf overlays describe`**:
```bash
# Describe overlay details
qlf overlays describe fintech

# JSON output
qlf overlays describe healthcare --format json
```

**`qlf overlays suggest`**:
```bash
# Suggest overlays for brief
qlf overlays suggest "payment processing with fraud detection"
```

#### Response Formats

**List Output**:
```
Available Overlays:

Domain Overlays:
├── fintech (v1.0)          - Financial services patterns
├── healthcare (v1.0)       - Healthcare and medical systems
└── ecommerce (v1.0)        - E-commerce and retail systems

Compliance Overlays:
├── pci (v1.0)              - PCI-DSS compliance for payment data
├── hipaa (v1.0)            - HIPAA compliance for healthcare
└── gdpr (v1.0)             - GDPR compliance for data protection
```

**Describe Output**:
```
Overlay: fintech (v1.0)
Description: Financial services domain patterns and compliance requirements

Features:
├── Payment processing patterns
├── Fraud detection algorithms
├── Financial data encryption
├── Audit logging for transactions
├── Risk assessment frameworks
└── Regulatory reporting

Dependencies:
└── compliance/pci (recommended)

Supported Languages:
├── Python (FastAPI, Django)
├── Go (Gin, Echo)
├── JavaScript (Express, NestJS)
└── Java (Spring Boot)

Example Usage:
qlf generate "payment API" --overlay fintech
```

### `qlf preview`

Deploy applications to ephemeral preview environments.

#### Subcommands

**`qlf preview deploy`**:
```bash
qlf preview deploy <app-path> [flags]
```

| Flag | Type | Description | Default |
|------|------|-------------|---------|
| `--namespace` | string | K8s namespace | auto-generated |
| `--ttl` | duration | Environment TTL | `2h` |
| `--domain` | string | Custom domain | auto-generated |
| `--replicas` | int | Number of replicas | `1` |

**`qlf preview list`**:
```bash
# List active previews
qlf preview list

# Filter by status
qlf preview list --status running
qlf preview list --status expired
```

**`qlf preview cleanup`**:
```bash
# Clean up specific preview
qlf preview cleanup <namespace>

# Clean up expired previews
qlf preview cleanup --expired

# Clean up all previews
qlf preview cleanup --all
```

### `qlf status`

Monitor workflow and system status.

#### Subcommands

**`qlf status workflow`**:
```bash
# Check specific workflow
qlf status workflow <workflow-id>

# List recent workflows
qlf status workflow --recent 10
```

**`qlf status health`**:
```bash
# System health check
qlf status health

# Service-specific health
qlf status health --service postgres
qlf status health --service temporal
```

## Go Package APIs

### Core Kernel APIs

#### SOC Parser (`kernel/soc`)

```go
package soc

// Parser interface for strict output contract parsing
type Parser interface {
    Parse(input string) (*ParseResult, error)
    Validate(result *ParseResult) error
    SetStrictMode(strict bool)
}

// Create new SOC parser
func NewParser(config *ParserConfig) *SOCParser

// Parse LLM output with grammar enforcement
func (p *SOCParser) Parse(llmOutput string) (*ParseResult, error)

// ParseResult contains structured parsing results
type ParseResult struct {
    Type        string                 `json:"type"`
    Content     map[string]interface{} `json:"content"`
    Confidence  float64               `json:"confidence"`
    Warnings    []string              `json:"warnings"`
    Errors      []string              `json:"errors"`
}
```

#### IR Compiler (`kernel/ir`)

```go
package ir

// Compiler interface for natural language processing
type Compiler interface {
    CompileBrief(brief string) (*IR, error)
    DetectOverlays(brief string) ([]OverlayMatch, error)
    SetOverlays(overlays []string) error
}

// Create new IR compiler
func NewCompiler(config *CompilerConfig) *IRCompiler

// Compile natural language brief to structured IR
func (c *IRCompiler) CompileBrief(brief string) (*IR, error)

// IR represents complete application specification
type IR struct {
    App      AppSpec      `json:"app"`
    API      APISpec      `json:"api"`
    Data     DataSpec     `json:"data"`
    UI       UISpec       `json:"ui"`
    Ops      OpsSpec      `json:"ops"`
    Security SecuritySpec `json:"security"`
}

// Overlay detection with confidence scoring
type OverlayMatch struct {
    Name       string  `json:"name"`
    Type       string  `json:"type"`  // "domain" or "compliance"
    Confidence float64 `json:"confidence"`
    Reason     string  `json:"reason"`
}
```

#### Agent Factory (`kernel/agents`)

```go
package agents

// Agent interface for code generation
type Agent interface {
    CanHandle(request *GenerationRequest) float64
    Generate(ctx context.Context, request *GenerationRequest) (*GenerationResult, error)
    GetMetadata() AgentMetadata
}

// Factory for managing agents
type AgentFactory struct {
    agents    map[AgentType]Agent
    llmClient LLMClient
}

// Create new agent factory
func NewAgentFactory(config *FactoryConfig) *AgentFactory

// Register agent with factory
func (af *AgentFactory) RegisterAgent(agentType AgentType, agent Agent) error

// Generate code using best matching agent
func (af *AgentFactory) Generate(ctx context.Context, request *GenerationRequest) (*GenerationResult, error)

// Generation request structure
type GenerationRequest struct {
    IR          *ir.IR         `json:"ir"`
    AgentType   AgentType      `json:"agent_type"`
    Language    string         `json:"language"`
    Framework   string         `json:"framework"`
    Overlays    []string       `json:"overlays"`
    LLMProvider string         `json:"llm_provider"`
    LLMModel    string         `json:"llm_model"`
}

// Generation result structure
type GenerationResult struct {
    Files       map[string]string `json:"files"`
    Metadata    GenerationMeta    `json:"metadata"`
    Warnings    []string          `json:"warnings"`
    Errors      []string          `json:"errors"`
    LLMUsage    *LLMUsageInfo     `json:"llm_usage,omitempty"`
}
```

#### LLM Integration (`kernel/llm`)

```go
package llm

// LLM client interface
type LLMClient interface {
    GenerateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error)
    GetProvider() string
    GetModels() []string
    GetUsage() *UsageStats
}

// Provider router for multi-provider support
type ProviderRouter struct {
    providers map[string]LLMClient
    cache     *ResponseCache
    budget    *BudgetTracker
}

// Create new provider router
func NewProviderRouter(config *RouterConfig) *ProviderRouter

// Completion request
type CompletionRequest struct {
    Prompt      string            `json:"prompt"`
    Model       string            `json:"model"`
    MaxTokens   int               `json:"max_tokens"`
    Temperature float64           `json:"temperature"`
    Context     map[string]string `json:"context"`
}

// Completion response
type CompletionResponse struct {
    Content    string     `json:"content"`
    Model      string     `json:"model"`
    Provider   string     `json:"provider"`
    Usage      UsageInfo  `json:"usage"`
    Cached     bool       `json:"cached"`
    Duration   time.Duration `json:"duration"`
}
```

### Production Service APIs

#### Builder Service (`services/builder`)

```go
package builder

// Builder service interface
type BuilderService interface {
    BuildContainer(ctx context.Context, req *BuildRequest) (*BuildResult, error)
    ScanSecurity(ctx context.Context, imageName string) (*ScanResult, error)
    GenerateDockerfile(req *DockerfileRequest) (string, error)
}

// Create new builder service
func NewBuilderService(config *BuilderConfig) *BuilderService

// Build request structure
type BuildRequest struct {
    SourcePath    string            `json:"source_path"`
    Language      string            `json:"language"`
    Framework     string            `json:"framework"`
    BuildArgs     map[string]string `json:"build_args"`
    Tags          []string          `json:"tags"`
    SecurityScan  bool              `json:"security_scan"`
    MultiStage    bool              `json:"multi_stage"`
}

// Build result structure
type BuildResult struct {
    ImageID       string        `json:"image_id"`
    ImageSize     int64         `json:"image_size"`
    BuildTime     time.Duration `json:"build_time"`
    ScanResult    *ScanResult   `json:"scan_result,omitempty"`
    Dockerfile    string        `json:"dockerfile"`
    BuildLogs     []string      `json:"build_logs"`
}
```

#### Deploy Service (`services/deploy`)

```go
package deploy

// Deploy service interface
type DeployService interface {
    DeployApplication(ctx context.Context, req *DeployRequest) (*DeployResult, error)
    CreateNamespace(ctx context.Context, name string, ttl time.Duration) error
    GetDeploymentStatus(ctx context.Context, namespace string) (*DeploymentStatus, error)
    CleanupExpired(ctx context.Context) error
}

// Create new deploy service
func NewDeployService(config *DeployConfig) *DeployService

// Deploy request structure
type DeployRequest struct {
    ApplicationPath string            `json:"application_path"`
    Namespace      string            `json:"namespace"`
    ImageName      string            `json:"image_name"`
    Replicas       int               `json:"replicas"`
    Resources      ResourceLimits    `json:"resources"`
    Environment    map[string]string `json:"environment"`
    TTL            time.Duration     `json:"ttl"`
}

// Deploy result structure
type DeployResult struct {
    Namespace    string            `json:"namespace"`
    URL          string            `json:"url"`
    HealthURL    string            `json:"health_url"`
    Manifests    map[string]string `json:"manifests"`
    Status       DeploymentStatus  `json:"status"`
    ExpiresAt    time.Time         `json:"expires_at"`
}
```

#### Packager Service (`services/packager`)

```go
package packager

// Packager service interface
type PackagerService interface {
    CreatePackage(ctx context.Context, req *PackageRequest) (*PackageResult, error)
    ValidatePackage(capsulePath string) (*ValidationResult, error)
    ExtractPackage(capsulePath, outputPath string) error
}

// Create new packager service
func NewPackagerService(config *PackagerConfig) *PackagerService

// Package request structure
type PackageRequest struct {
    Name           string            `json:"name"`
    Version        string            `json:"version"`
    Description    string            `json:"description"`
    Author         string            `json:"author"`
    License        string            `json:"license"`
    SourcePath     string            `json:"source_path"`
    Language       string            `json:"language"`
    Framework      string            `json:"framework"`
    BuildArtifacts []string          `json:"build_artifacts"`
    Manifests      []string          `json:"manifests"`
    Documentation  []string          `json:"documentation"`
    Runtime        RuntimeSpec       `json:"runtime"`
    GenerateSBOM   bool              `json:"generate_sbom"`
    ScanVulns      bool              `json:"scan_vulns"`
    SigningKey     string            `json:"signing_key"`
    Tags           []string          `json:"tags"`
    Labels         map[string]string `json:"labels"`
}

// Package result structure
type PackageResult struct {
    CapsulePath       string               `json:"capsule_path"`
    Size              int64                `json:"size"`
    Hash              string               `json:"hash"`
    Manifest          CapsuleManifest      `json:"manifest"`
    BuildTime         time.Duration        `json:"build_time"`
    Compressed        bool                 `json:"compressed"`
    CompressionRatio  float64              `json:"compression_ratio"`
    VulnScanResult    *VulnScanResult      `json:"vuln_scan_result,omitempty"`
    DocumentationPath string               `json:"documentation_path,omitempty"`
}
```

## Service Interfaces

### Packager Service REST API

#### Create Package
```http
POST /api/v1/packages
Content-Type: application/json

{
  "name": "my-app",
  "version": "1.0.0",
  "source_path": "./src",
  "language": "python",
  "framework": "fastapi",
  "generate_sbom": true,
  "scan_vulns": true
}
```

**Response**:
```json
{
  "capsule_path": "./packages/my-app-v1.0.0.qlcapsule",
  "size": 2457600,
  "hash": "sha256:a1b2c3d4e5f6...",
  "build_time": "1.234s",
  "compressed": true,
  "compression_ratio": 0.6789,
  "vuln_scan_result": {
    "critical": 0,
    "high": 1,
    "medium": 3,
    "low": 5,
    "fixable": 2
  }
}
```

#### Get Package Info
```http
GET /api/v1/packages/{name}/{version}
```

**Response**:
```json
{
  "name": "my-app",
  "version": "1.0.0",
  "manifest": {
    "version": "1.0.0",
    "name": "my-app",
    "sbom": {...},
    "attestation": {...},
    "signatures": [...]
  },
  "download_url": "https://packages.quantumlayer.dev/my-app-v1.0.0.qlcapsule"
}
```

### Builder Service REST API

#### Build Container
```http
POST /api/v1/builds
Content-Type: application/json

{
  "source_path": "./src",
  "language": "go",
  "framework": "gin",
  "security_scan": true,
  "tags": ["my-app:latest"]
}
```

**Response**:
```json
{
  "image_id": "sha256:1234567890ab...",
  "image_size": 157286400,
  "build_time": "45.678s",
  "scan_result": {
    "vulnerabilities": [...],
    "summary": {
      "critical": 0,
      "high": 0,
      "medium": 2,
      "low": 5
    }
  }
}
```

### Deploy Service REST API

#### Deploy Application
```http
POST /api/v1/deployments
Content-Type: application/json

{
  "application_path": "./app",
  "namespace": "my-preview",
  "image_name": "my-app:latest",
  "replicas": 2,
  "ttl": "4h"
}
```

**Response**:
```json
{
  "namespace": "my-preview",
  "url": "https://my-preview.quantumlayer.dev",
  "health_url": "https://my-preview.quantumlayer.dev/health",
  "status": {
    "phase": "Running",
    "ready_replicas": 2,
    "available_replicas": 2
  },
  "expires_at": "2024-01-15T18:30:00Z"
}
```

## Data Formats

### .qlcapsule Package Format

**Structure**:
```
package.qlcapsule (TAR+GZIP)
├── manifest.json         # Package metadata and security info
├── source/              # Application source code
│   ├── backend/
│   ├── frontend/
│   └── database/
├── artifacts/           # Build artifacts (optional)
│   ├── binaries/
│   └── assets/
├── manifests/           # Deployment manifests (optional)
│   ├── deployment.yaml
│   ├── service.yaml
│   └── ingress.yaml
└── docs/               # Generated documentation (optional)
    ├── README.md
    ├── API.md
    └── DEPLOYMENT.md
```

**Manifest Schema**:
```json
{
  "$schema": "https://schemas.quantumlayer.dev/qlcapsule/v1.0.0",
  "version": "1.0.0",
  "name": "my-app",
  "description": "My application description",
  "author": "Development Team",
  "license": "MIT",
  "created_at": "2024-01-15T10:30:00Z",
  "language": "python",
  "framework": "fastapi",
  "sbom": {
    "format": "spdx",
    "version": "2.3",
    "packages": [...],
    "vulnerabilities": [...],
    "signature": "sha256:..."
  },
  "attestation": {
    "build_platform": "linux/amd64",
    "build_timestamp": "2024-01-15T10:30:00Z",
    "builder_version": "qlf-1.0.0",
    "source_hash": "sha256:...",
    "reproducible": true
  },
  "signatures": [
    {
      "algorithm": "RS256",
      "signature": "base64-encoded-signature",
      "public_key_fingerprint": "sha256:...",
      "signed_at": "2024-01-15T10:30:00Z"
    }
  ],
  "delivery": {
    "channels": ["registry", "cdn"],
    "registry_url": "https://registry.quantumlayer.dev/my-app/1.0.0",
    "cdn_url": "https://cdn.quantumlayer.dev/packages/my-app-v1.0.0.qlcapsule"
  }
}
```

### SBOM Format (SPDX)

```json
{
  "spdxVersion": "SPDX-2.3",
  "dataLicense": "CC0-1.0",
  "SPDXID": "SPDXRef-DOCUMENT",
  "name": "my-app-1.0.0",
  "documentNamespace": "https://packages.quantumlayer.dev/my-app/1.0.0/sbom",
  "creators": ["Tool: QLF", "Tool: Syft"],
  "created": "2024-01-15T10:30:00Z",
  "packages": [
    {
      "SPDXID": "SPDXRef-Package",
      "name": "my-app",
      "downloadLocation": "NOASSERTION",
      "filesAnalyzed": true,
      "packageVerificationCode": {
        "packageVerificationCodeValue": "sha256:..."
      },
      "copyrightText": "NOASSERTION"
    }
  ],
  "relationships": [
    {
      "spdxElementId": "SPDXRef-DOCUMENT",
      "relationshipType": "DESCRIBES",
      "relatedSpdxElement": "SPDXRef-Package"
    }
  ]
}
```

## Error Codes and Handling

### CLI Error Codes

| Code | Description | Resolution |
|------|-------------|------------|
| 1 | General CLI error | Check command syntax and flags |
| 2 | Configuration error | Verify configuration file syntax |
| 3 | Service unavailable | Check infrastructure services |
| 4 | Validation failed | Review input parameters |
| 5 | Generation failed | Check LLM provider configuration |
| 6 | Package creation failed | Verify source path and dependencies |
| 7 | Deployment failed | Check K8s cluster connectivity |

### API Error Responses

**Standard Error Format**:
```json
{
  "error": {
    "code": "VALIDATION_FAILED",
    "message": "Package validation failed",
    "details": {
      "field": "source_path",
      "reason": "path does not exist"
    },
    "timestamp": "2024-01-15T10:30:00Z",
    "request_id": "req_1234567890"
  }
}
```

**Common Error Codes**:
- `INVALID_INPUT`: Invalid request parameters
- `SERVICE_UNAVAILABLE`: Backend service not accessible
- `LLM_PROVIDER_ERROR`: LLM provider API error
- `VALIDATION_FAILED`: Input validation failure
- `PACKAGING_FAILED`: Package creation error
- `DEPLOYMENT_FAILED`: K8s deployment error
- `SECURITY_SCAN_FAILED`: Vulnerability scan failure

## Configuration Reference

### CLI Configuration (`~/.config/qlf/config.yaml`)

```yaml
# Complete configuration schema
api:
  base_url: "http://localhost:8080"
  timeout: "30s"
  retry_attempts: 3
  retry_delay: "1s"

llm:
  default_provider: "bedrock"        # bedrock | azure
  cache_enabled: true
  cache_ttl: "24h"
  budget_limit: 100.00              # USD per month
  rate_limit: 60                    # requests per minute

  providers:
    bedrock:
      region: "eu-west-2"
      models:
        haiku: "anthropic.claude-3-haiku-20240307-v1:0"
        sonnet: "anthropic.claude-3-sonnet-20240229-v1:0"
        sonnet-3-5: "anthropic.claude-3-7-sonnet-20250219-v1:0"
      timeout: "30s"
      max_tokens: 4096

    azure:
      endpoint: "https://your-resource.openai.azure.com/"
      api_version: "2023-12-01-preview"
      models:
        gpt-4: "gpt-4-turbo"
        gpt-35: "gpt-35-turbo"
      timeout: "30s"
      max_tokens: 4096

overlays:
  auto_detect: true
  confidence_threshold: 0.7
  preferred: ["fintech", "healthcare"]
  custom_paths:
    - "~/.qlf/overlays"
    - "./overlays"

generation:
  default_language: "python"
  default_framework: "fastapi"
  include_tests: true
  include_docs: true
  include_docker: true
  template_fallback: true

verification:
  gates: ["static", "unit", "contract"]
  auto_repair: true
  repair_confidence: 0.8
  max_repair_attempts: 3

packaging:
  default_compression: "gzip"       # gzip | lz4 | zstd
  compression_level: 6              # 1-9
  output_dir: "./packages"
  sbom_enabled: true
  sbom_format: "spdx"               # spdx | cyclonedx
  vuln_scan_enabled: true
  generate_docs: true
  docs_format: "markdown"           # markdown | html

security:
  sign_packages: false
  signing_key_path: ""
  signing_algorithm: "RS256"        # RS256 | ES256
  vuln_severity_threshold: "medium" # low | medium | high | critical

delivery:
  channels:
    registry:
      url: "https://registry.quantumlayer.dev"
      username: ""
      password: ""
      namespace: "quantumlayer"

    cdn:
      url: "https://cdn.quantumlayer.dev"
      api_key: ""
      bucket: "packages"
      public_read: false

    direct:
      base_url: "https://packages.quantumlayer.dev"
      storage_path: "/var/packages"
      serve_http: true

logging:
  level: "info"                     # debug | info | warn | error
  format: "json"                    # json | text
  output: "stdout"                  # stdout | file
  file_path: "/var/log/qlf.log"

development:
  verbose_logging: false
  debug_mode: false
  profiling_enabled: false
  hot_reload: false
```

### Environment Variables

**Core Configuration**:
```bash
# Service URLs
QLF_API_URL=http://localhost:8080
QLF_TEMPORAL_URL=localhost:7233
QLF_DATABASE_URL=postgresql://factory:factory@localhost:5432/factory
QLF_REDIS_URL=redis://localhost:6379

# LLM Providers
AWS_PROFILE=your-profile
AWS_REGION=eu-west-2
AZURE_OPENAI_API_KEY=your-key
AZURE_OPENAI_ENDPOINT=https://your-resource.openai.azure.com/

# Security
QLF_SIGNING_KEY_PATH=/path/to/private.pem
QLF_TRIVY_PATH=/home/satish/bin/trivy

# Packaging
QLF_OUTPUT_DIR=./packages
QLF_COMPRESSION=gzip
QLF_COMPRESSION_LEVEL=6

# Development
QLF_DEBUG=false
QLF_VERBOSE=false
QLF_LOG_LEVEL=info
```

## Integration Examples

### Programmatic Usage

**Using Go Packages**:
```go
package main

import (
    "context"
    "fmt"

    "github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/ir"
    "github.com/quantumlayer-factory-hq/quantumlayer-factory/kernel/agents"
    "github.com/quantumlayer-factory-hq/quantumlayer-factory/services/packager"
)

func main() {
    ctx := context.Background()

    // 1. Compile brief to IR
    compiler := ir.NewCompiler(ir.DefaultConfig())
    irResult, err := compiler.CompileBrief("user management API with PostgreSQL")
    if err != nil {
        panic(err)
    }

    // 2. Generate code with agents
    factory := agents.NewAgentFactory(agents.DefaultConfig())
    request := &agents.GenerationRequest{
        IR:        irResult,
        Language:  "python",
        Framework: "fastapi",
    }

    result, err := factory.Generate(ctx, request)
    if err != nil {
        panic(err)
    }

    // 3. Package application
    packagerSvc := packager.NewPackagerService(packager.DefaultConfig())
    packageReq := &packager.PackageRequest{
        Name:       "my-app",
        Version:    "1.0.0",
        SourcePath: "./generated",
        Language:   "python",
        Framework:  "fastapi",
    }

    packageResult, err := packagerSvc.CreatePackage(ctx, packageReq)
    if err != nil {
        panic(err)
    }

    fmt.Printf("Package created: %s\n", packageResult.CapsulePath)
}
```

### REST API Usage

**Using curl**:
```bash
# Generate application via API
curl -X POST http://localhost:8080/api/v1/generate \
  -H "Content-Type: application/json" \
  -d '{
    "brief": "user management API with PostgreSQL",
    "language": "python",
    "framework": "fastapi",
    "overlays": ["fintech"]
  }'

# Package application via API
curl -X POST http://localhost:8080/api/v1/packages \
  -H "Content-Type: application/json" \
  -d '{
    "name": "my-app",
    "source_path": "./src",
    "language": "python",
    "generate_sbom": true,
    "scan_vulns": true
  }'
```

This API reference provides complete documentation for all QuantumLayer Factory interfaces, enabling both CLI usage and programmatic integration.