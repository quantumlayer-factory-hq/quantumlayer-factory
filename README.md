# QuantumLayer Factory

## Overview
A production-grade system that transforms natural language specifications into complete, deployable applications with full DevOps tooling.

## Core Architecture
- **Kernel**: Universal engine that doesn't know about specific domains
- **Overlays**: Composable domain/capability/regulatory modifications
- **SOC (Strict Output Contract)**: Grammar-enforced responses, no prose allowed
- **Verification Mesh**: Multi-layer quality gates before deployment

## Technology Stack
- Language: Go 1.21+
- Orchestration: Temporal
- Database: PostgreSQL 15
- Cache: Redis 7
- Vector Store: Qdrant
- Container: Docker
- Deploy: Kubernetes/ECS
- IaC: Terraform

## Development

### Quick Start
```bash
# Start development environment
make dev

# Run tests
make test

# Build and deploy
make build
make deploy
```

### First Test
```bash
echo "Create a function to add two numbers" | qlf generate --dry-run
```

## Project Structure
```
quantumlayer-factory/
├── kernel/           # Core system components
│   ├── soc/         # CRITICAL: Parser for strict output format
│   ├── ir/          # Intermediate Representation compiler
│   ├── agents/      # Agent factory and specifications
│   ├── verifier/    # Gate runners (static, unit, mutation, etc)
│   ├── planner/     # DAG builder for Temporal
│   └── gateway/     # API entry point
├── overlays/        # Composable modifications
├── templates/       # Base application stacks
├── workflows/       # Temporal workflow definitions
├── cmd/            # CLI tools
└── tests/          # Test suites
```

## Key Components

### SOC Parser
The critical component that enforces strict output contracts and prevents AI-generated prose/refusals.

### Agent Factory
Creates specialized agents for different tasks (backend, frontend, QA, repair).

### Verification Mesh
Multi-layer quality gates ensuring code quality before deployment.