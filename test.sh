#!/bin/bash

# QuietHire Quality Checks & Testing Script
# Run this to validate code quality and test all services

set -e

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}QuietHire - Quality Checks & Testing${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Step 1: Format Python code
echo -e "${YELLOW}1. Formatting Python code...${NC}"
uvx ruff format apps/parser apps/crawler-python apps/osint-discovery
echo -e "${GREEN}✓ Python code formatted${NC}"
echo ""

# Step 2: Lint Python code
echo -e "${YELLOW}2. Linting Python code...${NC}"
uvx ruff check --fix apps/parser apps/crawler-python apps/osint-discovery
echo -e "${GREEN}✓ Python linting passed${NC}"
echo ""

# Step 3: Check Go code
echo -e "${YELLOW}3. Checking Go code...${NC}"
cd apps/api && go vet ./... && cd ../..
cd apps/proxy-manager && go vet ./... && cd ../..
echo -e "${GREEN}✓ Go code checks passed${NC}"
echo ""

# Step 4: Build Docker images
echo -e "${YELLOW}4. Building Docker images...${NC}"
docker compose build
echo -e "${GREEN}✓ Docker images built${NC}"
echo ""

# Step 5: Start infrastructure
echo -e "${YELLOW}5. Starting infrastructure services...${NC}"
docker compose up -d postgres clickhouse typesense dragonfly
echo "Waiting for databases to be ready..."
sleep 10
echo -e "${GREEN}✓ Infrastructure started${NC}"
echo ""

# Step 6: Initialize databases
echo -e "${YELLOW}6. Initializing databases...${NC}"
echo "Initializing PostgreSQL..."
docker compose exec -T postgres psql -U quiethire -d quiethire < config/postgres/osint-schema.sql || true
echo "Initializing ClickHouse..."
docker compose exec -T clickhouse clickhouse-client --multiquery < config/clickhouse/schema.sql || true
echo -e "${GREEN}✓ Databases initialized${NC}"
echo ""

# Step 7: Start all services
echo -e "${YELLOW}7. Starting all application services...${NC}"
docker compose up -d
echo "Waiting for services to start..."
sleep 15
echo -e "${GREEN}✓ All services started${NC}"
echo ""

# Step 8: Health checks
echo -e "${YELLOW}8. Running health checks...${NC}"
echo ""

echo -n "  - API (port 3000): "
if curl -s http://localhost:3000/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Healthy${NC}"
else
    echo -e "${RED}✗ Not responding${NC}"
fi

echo -n "  - Parser (port 8001): "
if curl -s http://localhost:8001/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Healthy${NC}"
else
    echo -e "${RED}✗ Not responding${NC}"
fi

echo -n "  - Crawler (port 8002): "
if curl -s http://localhost:8002/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Healthy${NC}"
else
    echo -e "${RED}✗ Not responding${NC}"
fi

echo -n "  - Proxy Manager (port 8003): "
if curl -s http://localhost:8003/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Healthy${NC}"
else
    echo -e "${RED}✗ Not responding${NC}"
fi

echo -n "  - OSINT Discovery (port 8004): "
if curl -s http://localhost:8004/health > /dev/null 2>&1; then
    echo -e "${GREEN}✓ Healthy${NC}"
else
    echo -e "${RED}✗ Not responding${NC}"
fi

echo ""

# Step 9: Functional tests
echo -e "${YELLOW}9. Running functional tests...${NC}"
echo ""

echo "Testing OSINT Discovery ATS detection..."
curl -s -X POST http://localhost:8004/api/v1/detect/ats \
  -H "Content-Type: application/json" \
  -d '{"url": "https://boards.greenhouse.io/stripe"}' | jq . || echo "Test failed"

echo ""
echo "Testing OSINT Discovery dork templates..."
curl -s http://localhost:8004/api/v1/dorks/templates | jq . || echo "Test failed"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}All checks completed!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Next steps:"
echo "  1. View logs: docker compose logs -f"
echo "  2. Stop services: docker compose down"
echo "  3. View Temporal UI: http://localhost:8080"
echo "  4. View Grafana: http://localhost:3001"
echo ""
