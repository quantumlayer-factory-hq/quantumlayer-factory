#!/bin/bash
set -e

echo "Starting QuantumLayer Factory in development mode..."

# Check if docker compose is available
if ! docker compose version &> /dev/null; then
    echo "Error: docker compose is required but not available"
    exit 1
fi

# Start dependencies without the gateway
echo "Starting infrastructure services..."
docker compose up -d temporal temporal-ui postgres redis qdrant minio

# Wait for services to be ready
echo "Waiting for services to be ready..."

# Wait for postgres
echo "Waiting for PostgreSQL..."
until docker compose exec -T postgres pg_isready -U factory -d factory; do
  echo "PostgreSQL is not ready yet..."
  sleep 2
done

# Wait for temporal
echo "Waiting for Temporal..."
until docker compose exec -T temporal tctl --address temporal:7233 cluster health; do
  echo "Temporal is not ready yet..."
  sleep 2
done

# Wait for redis
echo "Waiting for Redis..."
until docker compose exec -T redis redis-cli ping; do
  echo "Redis is not ready yet..."
  sleep 2
done

echo "All services are ready!"

# Run migrations
echo "Running database migrations..."
docker compose exec -T postgres psql -U factory -d factory -f /docker-entrypoint-initdb.d/../../../migrations/001_initial_schema.sql || true
docker compose exec -T postgres psql -U factory -d factory -f /docker-entrypoint-initdb.d/../../../migrations/002_seed_overlays.sql || true

echo "✅ Development environment is ready!"
echo ""
echo "Available services:"
echo "  🌐 Temporal UI:  http://localhost:8088"
echo "  🗄️  PostgreSQL:   localhost:5432 (user: factory, db: factory)"
echo "  🔴 Redis:        localhost:6379"
echo "  🔍 Qdrant:       http://localhost:6333"
echo "  📦 MinIO:        http://localhost:9001 (user: minioadmin)"
echo ""
echo "To run the factory:"
echo "  go run cmd/qlf/main.go serve --dev"
echo ""
echo "To stop services:"
echo "  docker compose down"