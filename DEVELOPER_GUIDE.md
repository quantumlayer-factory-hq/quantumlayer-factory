# QuantumLayer Factory - Developer Guide

## Table of Contents
1. [Development Setup](#development-setup)
2. [Code Organization](#code-organization)
3. [Contributing Guidelines](#contributing-guidelines)
4. [Testing Framework](#testing-framework)
5. [Adding New Features](#adding-new-features)
6. [Debugging and Troubleshooting](#debugging-and-troubleshooting)

## Development Setup

### Prerequisites
- Go 1.21+
- Docker and Docker Compose v2
- Make
- Git

### Initial Setup
```bash
# Clone repository
git clone https://github.com/quantumlayer-factory-hq/quantumlayer-factory.git
cd quantumlayer-factory

# Install dependencies
go mod download

# Install development tools
make install-dev-tools

# Start infrastructure
make dev

# Build CLI
make build

# Run all tests
make test
```

### Development Tools

**Required Tools**:
```bash
# Linting and formatting
go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
go install golang.org/x/tools/cmd/goimports@latest

# Security scanning
go install github.com/securecodewarrior/sast-scan@latest

# Testing tools
go install github.com/onsi/ginkgo/v2/ginkgo@latest
go install gotest.tools/gotestsum@latest

# Documentation
go install github.com/swaggo/swag/cmd/swag@latest
```

**Optional Tools**:
```bash
# Debugging
go install github.com/go-delve/delve/cmd/dlv@latest

# Profiling
go install github.com/google/pprof@latest

# Benchmarking
go install golang.org/x/perf/cmd/benchstat@latest
```

### IDE Configuration

**VS Code** (`.vscode/settings.json`):
```json
{
  "go.testFlags": ["-v", "-race"],
  "go.buildFlags": ["-v"],
  "go.lintTool": "golangci-lint",
  "go.formatTool": "goimports",
  "go.useCodeSnippetsOnFunctionSuggest": true,
  "go.testTimeout": "300s",
  "editor.formatOnSave": true,
  "editor.codeActionsOnSave": {
    "source.organizeImports": true
  }
}
```

**GoLand/IntelliJ**:
- Enable Go modules support
- Configure golangci-lint integration
- Set up test configurations for verbose output
- Enable code formatting on save

## Code Organization

### Package Structure

```
quantumlayer-factory/
├── kernel/                    # Core domain-agnostic engine
│   ├── soc/                  # SOC parser (grammar enforcement)
│   │   ├── parser.go         # Main parsing logic
│   │   ├── validators.go     # Content validation
│   │   ├── types.go          # Data structures
│   │   └── parser_test.go    # Test suite (11 tests)
│   ├── ir/                   # Intermediate Representation
│   │   ├── compiler.go       # Natural language → IR compilation
│   │   ├── schema.go         # IR data structures
│   │   ├── overlay_detector.go # Automatic overlay detection
│   │   └── *_test.go         # Test suite (13+ tests)
│   ├── agents/               # Multi-agent generation system
│   │   ├── factory.go        # Agent management
│   │   ├── backend.go        # Backend code generation
│   │   ├── frontend.go       # Frontend code generation
│   │   ├── database.go       # Database schema generation
│   │   ├── api.go            # API specification generation
│   │   ├── test.go           # Test generation
│   │   └── *_test.go         # Test suite (15+ tests)
│   ├── verifier/             # Verification mesh (quality gates)
│   │   ├── pipeline.go       # Gate orchestration
│   │   ├── static_gate.go    # Static analysis
│   │   ├── unit_gate.go      # Unit test execution
│   │   ├── contract_gate.go  # API contract validation
│   │   ├── repair_loop.go    # LLM-powered repair
│   │   └── *_test.go         # Test suite (20+ tests)
│   ├── workflows/            # Temporal orchestration
│   │   ├── factory_workflow.go # Main workflow
│   │   ├── activities.go     # Workflow activities
│   │   ├── worker.go         # Worker implementation
│   │   └── *_test.go         # Test suite (8 tests)
│   ├── prompts/              # Meta-prompt composition
│   │   ├── composer.go       # Prompt composition
│   │   ├── template_manager.go # Template management
│   │   ├── templates/        # Agent prompt templates
│   │   └── *_test.go         # Test suite (30 tests)
│   └── llm/                  # Multi-provider LLM integration
│       ├── client.go         # Generic LLM interface
│       ├── bedrock.go        # AWS Bedrock client
│       ├── azure_openai.go   # Azure OpenAI client
│       ├── provider_router.go # Provider selection
│       ├── cache.go          # Response caching
│       ├── budget.go         # Cost tracking
│       └── *_test.go         # Test suite (10+ tests)
├── services/                 # Production services
│   ├── builder/              # Containerization service
│   │   ├── dockerfile_gen.go # Dockerfile generation
│   │   ├── container_build.go # Docker build orchestration
│   │   ├── security_scan.go  # Trivy integration
│   │   └── *_test.go         # Test suite (6 tests)
│   ├── deploy/               # Kubernetes deployment service
│   │   ├── k8s_deployer.go   # K8s deployment
│   │   ├── namespace_manager.go # Namespace management
│   │   ├── ingress_config.go # Ingress configuration
│   │   └── *_test.go         # Test suite (5 tests)
│   └── packager/             # Packaging service
│       ├── packager.go       # Core packaging logic
│       ├── types.go          # Package format types
│       ├── docs_generator.go # Documentation generation
│       ├── delivery.go       # Multi-channel delivery
│       └── *_test.go         # Test suite (18 tests)
├── overlays/                 # Domain and compliance expertise
│   ├── types.go              # Overlay data structures
│   ├── resolver.go           # Overlay composition
│   ├── domains/              # Domain overlays
│   │   ├── fintech.yaml      # Financial services
│   │   ├── healthcare.yaml   # Healthcare systems
│   │   └── ecommerce.yaml    # E-commerce platforms
│   ├── compliance/           # Compliance overlays
│   │   ├── pci.yaml          # PCI-DSS compliance
│   │   ├── hipaa.yaml        # HIPAA compliance
│   │   └── gdpr.yaml         # GDPR compliance
│   └── *_test.go             # Test suite (15 tests)
├── cmd/                      # CLI applications
│   ├── qlf/                  # Main CLI
│   │   ├── main.go           # CLI entry point
│   │   └── commands/         # Command implementations
│   │       ├── generate.go   # Generate command
│   │       ├── package.go    # Package command
│   │       ├── overlays.go   # Overlay commands
│   │       ├── preview.go    # Preview commands
│   │       ├── status.go     # Status commands
│   │       ├── root.go       # Root command
│   │       └── *_test.go     # Test suite (5+ tests)
│   └── worker/               # Temporal worker
│       └── main.go           # Worker entry point
└── docs/                     # Documentation
    ├── README.md             # Project overview
    ├── ARCHITECTURE.md       # System architecture
    ├── USER_MANUAL.md        # User guide
    ├── SETUP_GUIDE.md        # Installation guide
    ├── API_REFERENCE.md      # API documentation
    └── DEVELOPER_GUIDE.md    # This file
```

### Coding Standards

#### Go Style Guidelines
- Follow [Effective Go](https://golang.org/doc/effective_go.html)
- Use `gofmt` and `goimports` for formatting
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use meaningful variable and function names
- Write comprehensive comments for exported functions

#### File Naming Conventions
- **Implementation**: `service.go`, `client.go`
- **Tests**: `service_test.go`, `client_test.go`
- **Types**: `types.go` (when shared across files)
- **Interfaces**: Include in relevant implementation files

#### Package Organization
- **One concept per package**: Clear separation of concerns
- **Minimize dependencies**: Avoid circular dependencies
- **Interface segregation**: Small, focused interfaces
- **Dependency injection**: Use interfaces for external dependencies

## Contributing Guidelines

### Git Workflow

**Branch Naming**:
- `feature/week-X-component-name`: New features
- `fix/issue-description`: Bug fixes
- `refactor/component-name`: Code refactoring
- `docs/update-documentation`: Documentation updates

**Commit Message Format**:
```
type(scope): short description

Longer description if needed.

Fixes #123
```

**Types**: `feat`, `fix`, `refactor`, `test`, `docs`, `style`, `perf`

### Code Review Process

**Pre-Commit Checklist**:
- [ ] All tests pass (`make test`)
- [ ] No linting errors (`make lint`)
- [ ] Security scan clean (`make security-scan`)
- [ ] Documentation updated
- [ ] Performance impact assessed

**Review Criteria**:
- Code correctness and logic
- Test coverage and quality
- Performance implications
- Security considerations
- Documentation completeness
- API design consistency

### Pull Request Template

```markdown
## Description
Brief description of changes made.

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Unit tests added/updated
- [ ] Integration tests pass
- [ ] Manual testing completed

## Checklist
- [ ] Code follows style guidelines
- [ ] Self-review completed
- [ ] Comments added to complex code
- [ ] Documentation updated
- [ ] No security issues introduced
```

## Testing Framework

### Test Organization

**Test Types**:
1. **Unit Tests**: Individual function/method testing
2. **Integration Tests**: Component interaction testing
3. **End-to-End Tests**: Complete workflow testing
4. **Performance Tests**: Benchmark and load testing
5. **Security Tests**: Vulnerability and penetration testing

### Running Tests

**All Tests**:
```bash
# Run complete test suite
make test

# Run with coverage
make test-coverage

# Run with race detection
make test-race

# Verbose output
go test ./... -v
```

**Specific Package Tests**:
```bash
# SOC parser tests
go test ./kernel/soc/... -v

# Agent tests
go test ./kernel/agents/... -v

# Packager tests
go test ./services/packager/... -v
```

**Test Categories**:
```bash
# Unit tests only
go test ./... -short

# Integration tests
go test ./... -tags=integration

# End-to-end tests
go test ./... -tags=e2e
```

### Writing Tests

**Unit Test Example**:
```go
package soc

import (
    "testing"
    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
)

func TestParser_Parse(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected *ParseResult
        wantErr  bool
    }{
        {
            name:  "valid_input",
            input: `{"type": "generation", "content": {...}}`,
            expected: &ParseResult{
                Type: "generation",
                Content: map[string]interface{}{...},
            },
            wantErr: false,
        },
        {
            name:    "invalid_input",
            input:   "invalid json",
            wantErr: true,
        },
    }

    parser := NewParser(DefaultConfig())

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := parser.Parse(tt.input)

            if tt.wantErr {
                assert.Error(t, err)
                return
            }

            require.NoError(t, err)
            assert.Equal(t, tt.expected.Type, result.Type)
        })
    }
}
```

**Integration Test Example**:
```go
//go:build integration

package agents

import (
    "context"
    "testing"
    "github.com/stretchr/testify/suite"
)

type AgentIntegrationSuite struct {
    suite.Suite
    factory *AgentFactory
    ctx     context.Context
}

func (s *AgentIntegrationSuite) SetupSuite() {
    s.ctx = context.Background()
    s.factory = NewAgentFactory(DefaultConfig())
}

func (s *AgentIntegrationSuite) TestEndToEndGeneration() {
    request := &GenerationRequest{
        IR: &ir.IR{
            App: ir.AppSpec{
                Type:      "api",
                Language:  "python",
                Framework: "fastapi",
            },
        },
        Language:  "python",
        Framework: "fastapi",
    }

    result, err := s.factory.Generate(s.ctx, request)
    s.Require().NoError(err)
    s.Assert().NotEmpty(result.Files)
    s.Assert().Contains(result.Files, "main.py")
}

func TestAgentIntegrationSuite(t *testing.T) {
    suite.Run(t, new(AgentIntegrationSuite))
}
```

### Test Utilities

**Test Helpers** (`internal/testutil/`):
```go
package testutil

// Create temporary test directory
func CreateTempDir(t *testing.T) string {
    dir, err := os.MkdirTemp("", "qlf-test-*")
    require.NoError(t, err)
    t.Cleanup(func() { os.RemoveAll(dir) })
    return dir
}

// Create mock LLM client
func NewMockLLMClient() *MockLLMClient {
    return &MockLLMClient{
        responses: make(map[string]string),
    }
}

// Assert file exists
func AssertFileExists(t *testing.T, path string) {
    _, err := os.Stat(path)
    assert.NoError(t, err, "file should exist: %s", path)
}
```

## Adding New Features

### Adding a New Agent

**1. Define Agent Interface**:
```go
// kernel/agents/custom.go
package agents

type CustomAgent struct {
    metadata   AgentMetadata
    llmClient  LLMClient
    templates  map[string]string
}

func NewCustomAgent(config *CustomAgentConfig) *CustomAgent {
    return &CustomAgent{
        metadata: AgentMetadata{
            Type:        AgentTypeCustom,
            Name:        "Custom Agent",
            Description: "Custom code generation agent",
            Languages:   []string{"go", "python"},
            Frameworks:  []string{"gin", "fastapi"},
        },
        llmClient: config.LLMClient,
        templates: loadTemplates("custom"),
    }
}

func (ca *CustomAgent) CanHandle(request *GenerationRequest) float64 {
    score := 0.0

    // Language support
    if contains(ca.metadata.Languages, request.Language) {
        score += 0.3
    }

    // Framework support
    if contains(ca.metadata.Frameworks, request.Framework) {
        score += 0.4
    }

    // IR compatibility
    if ca.isCompatibleWithIR(request.IR) {
        score += 0.3
    }

    return score
}

func (ca *CustomAgent) Generate(ctx context.Context, request *GenerationRequest) (*GenerationResult, error) {
    // Implementation logic
    if ca.llmClient != nil {
        return ca.generateWithLLM(ctx, request)
    }
    return ca.generateWithTemplate(request)
}
```

**2. Write Tests**:
```go
// kernel/agents/custom_test.go
func TestCustomAgent_CanHandle(t *testing.T) {
    agent := NewCustomAgent(DefaultCustomConfig())

    tests := []struct {
        name     string
        request  *GenerationRequest
        expected float64
    }{
        {
            name: "perfect_match",
            request: &GenerationRequest{
                Language:  "go",
                Framework: "gin",
                IR:        createCompatibleIR(),
            },
            expected: 1.0,
        },
    }

    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            score := agent.CanHandle(tt.request)
            assert.Equal(t, tt.expected, score)
        })
    }
}
```

**3. Register Agent**:
```go
// kernel/agents/factory.go
func init() {
    RegisterAgentType(AgentTypeCustom, "custom")
}

func (af *AgentFactory) registerDefaultAgents() {
    // ... existing agents
    af.RegisterAgent(AgentTypeCustom, NewCustomAgent(af.config.Custom))
}
```

### Adding a New Verification Gate

**1. Implement Gate Interface**:
```go
// kernel/verifier/custom_gate.go
package verifier

type CustomGate struct {
    name   string
    config *CustomGateConfig
}

func NewCustomGate(config *CustomGateConfig) *CustomGate {
    return &CustomGate{
        name:   "custom",
        config: config,
    }
}

func (cg *CustomGate) Name() string {
    return cg.name
}

func (cg *CustomGate) Execute(ctx context.Context, request *VerificationRequest) (*VerificationResult, error) {
    // Custom verification logic
    issues := []Issue{}

    // Analyze code
    for filePath, content := range request.Files {
        if violations := cg.analyzeFile(filePath, content); len(violations) > 0 {
            issues = append(issues, violations...)
        }
    }

    return &VerificationResult{
        Gate:     cg.name,
        Passed:   len(issues) == 0,
        Issues:   issues,
        Duration: time.Since(start),
    }, nil
}

func (cg *CustomGate) CanRepair() bool {
    return true // if repair is supported
}
```

**2. Register Gate**:
```go
// kernel/verifier/pipeline.go
func (vp *VerificationPipeline) registerDefaultGates() {
    // ... existing gates
    vp.AddGate(NewCustomGate(vp.config.Custom))
}
```

### Adding a New Overlay

**1. Create Overlay Definition**:
```yaml
# overlays/domains/custom.yaml
name: custom
description: Custom domain patterns
version: "1.0"
type: domain

# Prompt enhancements for different agents
prompt_enhancements:
  backend:
    before:
      - "Implement custom domain-specific patterns"
      - "Add specialized business logic"
    after:
      - "Ensure custom validation rules"

  database:
    before:
      - "Create custom entity relationships"
      - "Add domain-specific indexes"

# Pattern matching for automatic detection
patterns:
  - pattern: "custom business logic"
    confidence: 0.9
    enhancement: "Apply custom domain patterns"
  - pattern: "specialized workflow"
    confidence: 0.8
    enhancement: "Add workflow optimization"

# Security requirements
security_requirements:
  - "Custom security pattern implementation"
  - "Domain-specific encryption requirements"

# Code examples and snippets
code_examples:
  python:
    validation: |
      def validate_custom_entity(entity):
          # Custom validation logic
          return True

  go:
    middleware: |
      func CustomMiddleware() gin.HandlerFunc {
          return gin.LoggerWithFormatter(customFormat)
      }

# Dependencies on other overlays
dependencies: []

# Compatibility matrix
compatibility:
  languages: ["go", "python", "javascript"]
  frameworks: ["gin", "fastapi", "express"]
```

**2. Update Overlay Tests**:
```go
// overlays/resolver_test.go
func TestResolver_LoadCustomOverlay(t *testing.T) {
    resolver := NewResolver(DefaultConfig())

    overlay, err := resolver.LoadOverlay("custom")
    require.NoError(t, err)
    assert.Equal(t, "custom", overlay.Name)
    assert.Equal(t, "domain", overlay.Type)
}
```

### Adding a New Service

**1. Define Service Interface**:
```go
// services/custom/types.go
package custom

type CustomService interface {
    ProcessRequest(ctx context.Context, req *CustomRequest) (*CustomResult, error)
    GetStatus() ServiceStatus
    Shutdown(ctx context.Context) error
}

type CustomRequest struct {
    Input  string            `json:"input"`
    Config map[string]string `json:"config"`
}

type CustomResult struct {
    Output    string        `json:"output"`
    Metadata  CustomMeta    `json:"metadata"`
    Duration  time.Duration `json:"duration"`
}
```

**2. Implement Service**:
```go
// services/custom/service.go
package custom

type customService struct {
    config *CustomConfig
    logger *slog.Logger
}

func NewCustomService(config *CustomConfig) CustomService {
    return &customService{
        config: config,
        logger: slog.New(slog.NewJSONHandler(os.Stdout, nil)),
    }
}

func (cs *customService) ProcessRequest(ctx context.Context, req *CustomRequest) (*CustomResult, error) {
    start := time.Now()

    // Implementation logic
    output, err := cs.processInput(req.Input)
    if err != nil {
        return nil, fmt.Errorf("processing failed: %w", err)
    }

    return &CustomResult{
        Output:   output,
        Duration: time.Since(start),
    }, nil
}
```

**3. Add Service Tests**:
```go
// services/custom/service_test.go
package custom

func TestCustomService_ProcessRequest(t *testing.T) {
    service := NewCustomService(DefaultConfig())

    request := &CustomRequest{
        Input: "test input",
    }

    result, err := service.ProcessRequest(context.Background(), request)
    require.NoError(t, err)
    assert.NotEmpty(t, result.Output)
}
```

## Testing Framework

### Test Categories

#### Unit Tests
- **Purpose**: Test individual functions and methods
- **Scope**: Single package, isolated functionality
- **Tools**: Go testing package, testify/assert

```go
func TestFunction(t *testing.T) {
    // Arrange
    input := "test input"
    expected := "expected output"

    // Act
    result := MyFunction(input)

    // Assert
    assert.Equal(t, expected, result)
}
```

#### Integration Tests
- **Purpose**: Test component interactions
- **Scope**: Multiple packages, service integration
- **Tools**: testify/suite, Docker test containers

```go
//go:build integration

func TestServiceIntegration(t *testing.T) {
    // Setup test environment
    db := setupTestDatabase(t)
    service := NewService(db)

    // Test interaction
    result, err := service.ProcessData("test")
    require.NoError(t, err)
    assert.NotNil(t, result)
}
```

#### End-to-End Tests
- **Purpose**: Test complete user workflows
- **Scope**: Full system, CLI to output
- **Tools**: CLI execution, file system validation

```go
//go:build e2e

func TestCompleteWorkflow(t *testing.T) {
    // Execute CLI command
    cmd := exec.Command("./bin/qlf", "generate", "test API", "--dry-run")
    output, err := cmd.CombinedOutput()
    require.NoError(t, err)

    // Validate output
    assert.Contains(t, string(output), "generation_preview")
}
```

### Test Infrastructure

**Docker Test Containers**:
```go
// internal/testutil/containers.go
func StartPostgresContainer(t *testing.T) *testcontainers.Container {
    req := testcontainers.ContainerRequest{
        Image:        "postgres:15",
        ExposedPorts: []string{"5432/tcp"},
        Env: map[string]string{
            "POSTGRES_DB":       "test",
            "POSTGRES_USER":     "test",
            "POSTGRES_PASSWORD": "test",
        },
        WaitingFor: wait.ForListeningPort("5432/tcp"),
    }

    container, err := testcontainers.GenericContainer(context.Background(), testcontainers.GenericContainerRequest{
        ContainerRequest: req,
        Started:          true,
    })
    require.NoError(t, err)

    t.Cleanup(func() {
        container.Terminate(context.Background())
    })

    return container
}
```

**Mock Implementations**:
```go
// internal/mocks/llm_client.go
type MockLLMClient struct {
    responses map[string]string
    usage     *UsageStats
}

func (m *MockLLMClient) GenerateCompletion(ctx context.Context, req *CompletionRequest) (*CompletionResponse, error) {
    response, exists := m.responses[req.Prompt]
    if !exists {
        return nil, errors.New("no mock response configured")
    }

    return &CompletionResponse{
        Content:  response,
        Provider: "mock",
        Usage:    UsageInfo{Tokens: 100},
    }, nil
}
```

### Performance Testing

**Benchmark Tests**:
```go
func BenchmarkIRCompiler_CompileBrief(b *testing.B) {
    compiler := NewCompiler(DefaultConfig())
    brief := "user management API with PostgreSQL"

    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        _, err := compiler.CompileBrief(brief)
        if err != nil {
            b.Fatal(err)
        }
    }
}
```

**Load Tests**:
```bash
# Load testing with custom tool
make load-test

# Or manual load test
for i in {1..100}; do
    ./bin/qlf generate "test API $i" --dry-run &
done
wait
```

## Debugging and Troubleshooting

### Logging Configuration

**Development Logging**:
```go
// Use structured logging throughout
import "log/slog"

logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
    Level: slog.LevelDebug,
}))

logger.Info("processing request",
    "request_id", requestID,
    "user_id", userID,
    "action", "generate",
)
```

**Debug Mode**:
```bash
# Enable debug logging
export QLF_DEBUG=true
export QLF_LOG_LEVEL=debug

# Run with verbose output
./bin/qlf generate "test" --verbose --dry-run
```

### Debugging Techniques

**Delve Debugging**:
```bash
# Debug CLI command
dlv exec ./bin/qlf -- generate "test API" --dry-run

# Debug tests
dlv test ./kernel/agents/

# Debug with breakpoints
dlv exec ./bin/qlf
(dlv) break main.main
(dlv) continue
```

**Profiling**:
```bash
# CPU profiling
go test -cpuprofile=cpu.prof -bench=.

# Memory profiling
go test -memprofile=mem.prof -bench=.

# Analyze profiles
go tool pprof cpu.prof
go tool pprof mem.prof
```

### Common Issues

#### Import Cycle Detection
```bash
# Check for import cycles
go list -deps ./... | sort | uniq -c | sort -nr
```

#### Memory Leaks
```bash
# Memory leak detection
go test -race ./...

# Memory profiling
go test -memprofile=mem.prof ./...
go tool pprof mem.prof
```

#### Performance Issues
```bash
# Benchmark comparison
go test -bench=. -benchmem ./...

# Profile specific functions
go test -bench=BenchmarkSpecific -cpuprofile=cpu.prof
```

## Code Quality

### Linting Configuration

**`.golangci.yml`**:
```yaml
run:
  timeout: 5m
  modules-download-mode: readonly

linters:
  enable:
    - gofmt
    - goimports
    - govet
    - staticcheck
    - gosec
    - errcheck
    - ineffassign
    - misspell
    - revive

linters-settings:
  gosec:
    severity: "medium"
    confidence: "medium"

  revive:
    rules:
      - name: exported
        severity: warning
```

### Security Guidelines

**Secure Coding Practices**:
- Never log sensitive data (API keys, passwords)
- Validate all user inputs
- Use parameterized queries for database access
- Implement proper error handling
- Use secure random number generation

**Security Scanning**:
```bash
# Run security scans
make security-scan

# Manual security audit
gosec ./...

# Dependency vulnerability check
go list -json -deps ./... | nancy sleuth
```

### Documentation Standards

**Function Documentation**:
```go
// CompileBrief transforms a natural language brief into structured IR.
// It performs text analysis, pattern matching, and overlay detection
// to create a comprehensive application specification.
//
// Parameters:
//   - brief: Natural language description of the application
//
// Returns:
//   - *IR: Structured intermediate representation
//   - error: Compilation error if parsing fails
//
// Example:
//   ir, err := compiler.CompileBrief("user management API with PostgreSQL")
//   if err != nil {
//       return fmt.Errorf("compilation failed: %w", err)
//   }
func (c *IRCompiler) CompileBrief(brief string) (*IR, error) {
    // Implementation
}
```

**Package Documentation**:
```go
// Package agents provides a multi-agent code generation system
// with specialized agents for different application components.
//
// The agent factory manages agent registration, selection, and
// execution. Each agent specializes in generating specific types
// of code (backend, frontend, database, etc.) and can operate
// in template mode or LLM-powered mode.
//
// Example usage:
//   factory := agents.NewAgentFactory(config)
//   result, err := factory.Generate(ctx, request)
package agents
```

## Build and Release

### Build Configuration

**Makefile Targets**:
```makefile
# Development
.PHONY: dev
dev:
	docker-compose -f docker-compose.dev.yml up -d

.PHONY: build
build:
	go build -o bin/qlf ./cmd/qlf
	go build -o bin/worker ./cmd/worker

.PHONY: test
test:
	go test ./... -race -timeout=300s

.PHONY: lint
lint:
	golangci-lint run

.PHONY: security-scan
security-scan:
	gosec ./...

# Production
.PHONY: build-prod
build-prod:
	CGO_ENABLED=0 GOOS=linux go build -a -installsuffix cgo -o bin/qlf ./cmd/qlf

.PHONY: docker-build
docker-build:
	docker build -t quantumlayer/qlf:latest .

.PHONY: release
release:
	goreleaser release --clean
```

### Release Process

**Version Management**:
```bash
# Tag new version
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Build release artifacts
make build-prod

# Create release
goreleaser release --clean
```

**Release Checklist**:
- [ ] All tests pass
- [ ] Security scan clean
- [ ] Documentation updated
- [ ] Changelog updated
- [ ] Version bumped
- [ ] Tag created
- [ ] Release artifacts built

## Contribution Workflow

### Setting Up Development Environment

```bash
# 1. Fork repository
# 2. Clone your fork
git clone https://github.com/yourusername/quantumlayer-factory.git
cd quantumlayer-factory

# 3. Add upstream remote
git remote add upstream https://github.com/quantumlayer-factory-hq/quantumlayer-factory.git

# 4. Create feature branch
git checkout -b feature/my-new-feature

# 5. Set up development environment
make dev
make build
make test
```

### Development Cycle

```bash
# 1. Sync with upstream
git fetch upstream
git rebase upstream/main

# 2. Make changes
# Edit code, add tests, update docs

# 3. Run quality checks
make lint
make test
make security-scan

# 4. Commit changes
git add .
git commit -m "feat(agents): add custom agent implementation"

# 5. Push and create PR
git push origin feature/my-new-feature
# Create pull request on GitHub
```

### Code Review Guidelines

**For Authors**:
- Keep changes focused and atomic
- Write comprehensive tests
- Update documentation
- Test locally before submitting
- Respond to feedback promptly

**For Reviewers**:
- Check code correctness and style
- Verify test coverage
- Assess performance impact
- Review security implications
- Validate documentation

## Performance Optimization

### Profiling

**CPU Profiling**:
```bash
# Profile CLI command
go run -cpuprofile=cpu.prof ./cmd/qlf generate "test" --dry-run
go tool pprof cpu.prof
```

**Memory Profiling**:
```bash
# Profile memory usage
go run -memprofile=mem.prof ./cmd/qlf generate "large system"
go tool pprof mem.prof
```

**Trace Analysis**:
```bash
# Generate execution trace
go run -trace=trace.out ./cmd/qlf generate "test"
go tool trace trace.out
```

### Optimization Guidelines

**General Performance**:
- Use buffered I/O for file operations
- Implement connection pooling
- Cache expensive computations
- Use goroutines for parallel processing
- Optimize memory allocations

**LLM Integration**:
- Cache LLM responses aggressively
- Use appropriate model for task complexity
- Implement request batching
- Monitor token usage and costs
- Implement circuit breakers

This developer guide provides comprehensive information for contributing to and extending the QuantumLayer Factory system.