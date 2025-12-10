# QuietHire - Setup & Development Guide

## What's Been Completed

### Phase 1: Foundation (✅ Completed)
- ✅ Go workspace initialized with proper module structure
- ✅ All Go dependencies added (Fiber, PostgreSQL, Typesense, Redis client)
- ✅ Comprehensive `.env.example` file created
- ✅ Docker Compose updated with all services:
  - PostgreSQL, ClickHouse, Typesense, Dragonfly (Redis-compatible cache)
  - Temporal + Temporal UI
  - Grafana, Loki, Prometheus (Observability stack)
  - All application services (API, Parser, RealScore, Manager Extractor, Email Writer, Proxy Manager)
- ✅ Prometheus configuration for metrics collection

### Phase 2.1: Go API Service (✅ Completed)
- ✅ Go Fiber API server with middleware (CORS, logging, recovery)
- ✅ Health check endpoint
- ✅ Placeholder search, jobs, and stats endpoints
- ✅ Configuration management system
- ✅ Dockerfiles for all services

### Phase 2.2: Typesense Schema (✅ In Progress)
- ✅ Typesense schema initialization script
- Schema includes: job details, hiring manager info, real_score, location, remote status, salary, etc.

## Quick Start

### Automated Setup (Recommended)

```bash
# Run the setup script to initialize all dependencies
./setup.sh

# This will:
# - Create .env from .env.example if it doesn't exist
# - Download and tidy all Go dependencies
# - Sync all Python dependencies with uv
# - Setup py-common package
```

### Manual Setup

#### 1. Environment Setup

```bash
# Copy example env file
cp .env.example .env

# Edit .env and add your API keys:
# - TYPESENSE_API_KEY
# - GROQ_API_KEY (for parser)
# - LLAMA_API_KEY (for email writer)
# - DB_PASSWORD, CLICKHOUSE_PASSWORD
```

#### 2. Install Dependencies

**Go Services:**
```bash
# For each Go service (api, crawler-go, proxy-manager)
cd apps/api
go mod download
go mod tidy
```

**Python Services:**
```bash
# For each Python service (parser, realscore, manager-extractor, email-writer, crawler-python)
cd apps/parser
uv sync
```

### 2. Start Infrastructure Services

```bash
# Start only database and infrastructure services
docker compose up -d postgres clickhouse typesense dragonfly temporal

# Wait for services to be healthy
docker compose ps
```

### 3. Initialize Typesense Schema

```bash
cd apps/api
go run cmd/init-typesense/main.go
```

### 4. Start Application Services

```bash
# Start all application services
docker compose up -d

# Or start specific services
docker compose up -d api parser realscore

# View logs
docker compose logs -f api
```

### 5. Test the API

```bash
# Health check
curl http://localhost:3000/health

# Search (placeholder)
curl "http://localhost:3000/api/v1/search?q=software+engineer"

# Stats
curl http://localhost:3000/api/v1/stats
```

## Development

### Running Services Locally

#### Go API
```bash
cd apps/api
go run cmd/api/main.go
```

#### Python Services (using uv)
```bash
cd apps/parser  # or realscore, manager-extractor, email-writer, crawler-python

# Sync dependencies first (creates/updates .venv)
uv sync

# Run the service
uv run python main.py
# or for FastAPI services:
uv run uvicorn main:app --reload --host 0.0.0.0 --port 8000
```

### Adding Python Dependencies

```bash
cd apps/parser  # or any Python service

# Add a new dependency
uv add package-name

# Add a dev dependency
uv add --dev pytest

# Sync all dependencies
uv sync

# Update lock file
uv lock
```

### Scaling Services

```bash
# Scale crawlers for more throughput
docker compose up -d --scale crawler-python=8 --scale crawler-go=6
```

## Service Ports

| Service | Port | URL |
|---------|------|-----|
| API | 3000 | http://localhost:3000 |
| Temporal UI | 8080 | http://localhost:8080 |
| Grafana | 3001 | http://localhost:3001 |
| Prometheus | 9090 | http://localhost:9090 |
| Typesense | 8108 | http://localhost:8108 |
| PostgreSQL | 5432 | localhost:5432 |
| ClickHouse HTTP | 8123 | http://localhost:8123 |
| ClickHouse Native | 9000 | localhost:9000 |
| Dragonfly (Redis) | 6379 | localhost:6379 |
| Parser | 8001 | http://localhost:8001 |
| RealScore | 8002 | http://localhost:8002 |
| Manager Extractor | 8003 | http://localhost:8003 |
| Email Writer | 8004 | http://localhost:8004 |
| Proxy Manager | 8005 | http://localhost:8005 |

## Architecture

```
quiethire/
├── apps/
│   ├── api/              → Go Fiber API (search, jobs)
│   ├── crawler-go/       → Fast Go crawler (playwright-go)
│   ├── crawler-python/   → Stealth Python crawler
│   ├── parser/           → HTML → structured data (FastAPI + Groq)
│   ├── realscore/        → Authenticity scoring (0-100)
│   ├── manager-extractor/→ Extract hiring manager info
│   ├── email-writer/     → AI email generation (Llama-3.3)
│   └── proxy-manager/    → Proxy rotation service
├── pkg/
│   ├── go-common/        → Shared Go packages
│   └── py-common/        → Shared Python packages
├── config/
│   └── prometheus.yml    → Metrics configuration
└── docker-compose.yml    → Full stack orchestration
```

## Next Steps

### Immediate (Phase 2)
- [ ] Implement Typesense search in API
- [ ] Set up ClickHouse tables and schemas
- [ ] Create basic frontend (Next.js or HTMX)

### Short Term (Phase 3-4)
- [ ] Implement Temporal workflows for crawling
- [ ] Build Go crawler for public job boards
- [ ] Build Python stealth crawler for ATS platforms
- [ ] Implement proxy rotation logic

### Medium Term (Phase 5-8)
- [ ] Complete Parser service (Groq + Unstructured)
- [ ] Build RealScore authenticity engine
- [ ] Implement hiring manager extraction
- [ ] Build email generation service

## Troubleshooting

### Services won't start
```bash
# Check logs
docker compose logs [service-name]

# Restart specific service
docker compose restart [service-name]

# Clean rebuild
docker compose down -v
docker compose up --build
```

### Database connection errors
- Ensure PostgreSQL and ClickHouse are healthy: `docker compose ps`
- Check credentials in `.env` file
- Verify network connectivity: `docker network ls`

### Go module issues
```bash
cd apps/api  # or other Go service
go mod tidy
go mod download
```

## Contributing

This is a solo developer project. For questions or issues, refer to:
- `docs/plan.md` - Development roadmap
- `docs/architecture.md` - System architecture
- `docs/overview.md` - Project overview
