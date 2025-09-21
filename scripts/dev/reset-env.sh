#!/bin/bash
set -e

echo "Resetting QuantumLayer Factory development environment..."

# Stop all services
echo "Stopping all services..."
docker compose down -v

# Remove volumes to reset data
echo "Removing volumes..."
docker volume prune -f

# Remove any factory-related containers
echo "Cleaning up containers..."
docker container prune -f

# Remove unused networks
echo "Cleaning up networks..."
docker network prune -f

echo "✅ Environment reset complete!"
echo ""
echo "To start fresh:"
echo "  ./scripts/dev/run-local.sh"