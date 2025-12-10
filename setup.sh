#!/bin/bash

# QuietHire Setup Script
# This script initializes all dependencies for the project

set -e

echo "🚀 QuietHire Setup Script"
echo "========================="
echo ""

# Check if .env exists
if [ ! -f .env ]; then
    echo "⚠️  No .env file found. Creating from .env.example..."
    cp .env.example .env
    echo "✅ Created .env file. Please update it with your API keys."
    echo ""
else
    echo "✅ .env file exists"
    echo ""
fi

# Setup Go services
echo "📦 Setting up Go services..."
echo ""

GO_SERVICES=("api" "crawler-go" "proxy-manager")

for service in "${GO_SERVICES[@]}"; do
    echo "  → $service"
    cd "apps/$service"
    go mod download
    go mod tidy
    cd ../..
done

echo "✅ Go services ready"
echo ""

# Setup Python services
echo "🐍 Setting up Python services..."
echo ""

PYTHON_SERVICES=("parser" "realscore" "manager-extractor" "email-writer" "crawler-python")

for service in "${PYTHON_SERVICES[@]}"; do
    echo "  → $service"
    cd "apps/$service"
    uv sync
    cd ../..
done

echo "✅ Python services ready"
echo ""

# Setup py-common
echo "📦 Setting up py-common..."
cd pkg/py-common
uv sync
cd ../..
echo "✅ py-common ready"
echo ""

echo "🎉 Setup complete!"
echo ""
echo "Next steps:"
echo "  1. Update .env with your API keys"
echo "  2. Start infrastructure: docker compose up -d postgres clickhouse typesense dragonfly temporal"
echo "  3. Initialize Typesense: cd apps/api && go run cmd/init-typesense/main.go"
echo "  4. Start all services: docker compose up -d"
echo "  5. Test API: curl http://localhost:3000/health"
echo ""
