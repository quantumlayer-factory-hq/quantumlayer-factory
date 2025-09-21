.PHONY: help dev test build deploy clean

help:
	@echo "Available targets:"
	@echo "  dev      - Start local development environment"
	@echo "  test     - Run tests"
	@echo "  build    - Build containers"
	@echo "  deploy   - Deploy to k8s cluster"
	@echo "  clean    - Clean up docker containers and volumes"

dev:
	docker compose up -d
	@echo "Waiting for services..."
	@sleep 10
	@echo "Dev environment ready:"
	@echo "  Temporal: http://localhost:8088"
	@echo "  Postgres: localhost:5432"
	@echo "  Redis: localhost:6379"
	@echo "  Qdrant: http://localhost:6333"

test:
	go test -v ./...

build:
	docker build -t quantumlayer/factory:latest .

deploy:
	kubectl create namespace quantumlayer-factory --dry-run=client -o yaml | kubectl apply -f -
	kubectl apply -k deployments/k8s/

clean:
	docker compose down -v
	docker system prune -f

soc-test:
	go test -v ./kernel/soc/...

ir-test:
	go test -v ./kernel/ir/...

agents-test:
	go test -v ./kernel/agents/...

e2e-test:
	go test -v ./tests/e2e/...

.PHONY: cli cli-test
cli:
	go build -o bin/qlf ./cmd/qlf

cli-test:
	go test ./cmd/qlf/...