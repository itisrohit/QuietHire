# QuietHire - Job Aggregation Platform with AI-Powered Discovery

QuietHire is a comprehensive job aggregation platform that uses AI and OSINT techniques to discover and crawl job postings from multiple sources. Built with microservices architecture, Temporal workflows, and distributed crawling capabilities.

## Status: MVP Complete ✅

### Completed Features

#### Core Infrastructure (✅)
- ✅ Microservices architecture with 11 services
- ✅ Multi-database setup: PostgreSQL, ClickHouse, Typesense
- ✅ Temporal workflow orchestration
- ✅ Docker Compose orchestration with health checks
- ✅ All services containerized and production-ready

#### API Service (✅)
- ✅ Go Fiber REST API with comprehensive endpoints
- ✅ Health monitoring and statistics
- ✅ Job search via Typesense integration
- ✅ Job listing with filters (limit, offset, location, remote)
- ✅ Individual job retrieval
- ✅ ClickHouse integration for analytics

#### Crawling & Discovery (✅)
- ✅ Python-based stealth crawler with Playwright
- ✅ Batch URL crawling endpoint
- ✅ HTML parsing with JSON-LD JobPosting support
- ✅ OSINT discovery service for career pages
- ✅ ATS detection (Lever, Greenhouse, Workday, etc.)
- ✅ Google Dork-based company discovery
- ✅ Subdomain enumeration
- ✅ Proxy management service

#### Temporal Workflows (✅)
- ✅ 5 registered workflows:
  - CrawlCoordinatorWorkflow
  - ScheduledCrawlWorkflow
  - CompanyDiscoveryWorkflow
  - ContinuousDiscoveryWorkflow
  - GoogleDorkDiscoveryWorkflow
- ✅ 13 registered activities for crawling and discovery
- ✅ Worker service connected to PostgreSQL and ClickHouse

#### Database Schemas (✅)
- ✅ PostgreSQL: 7 tables (companies, discovered_urls, subdomains, etc.)
- ✅ ClickHouse: 6 tables (jobs, crawl_history, analytics tables)
- ✅ Typesense: jobs collection with 18 searchable fields

#### Code Quality (✅)
- ✅ All critical linting errors resolved
- ✅ Pre-commit hooks configured (yamllint, golangci-lint)
- ✅ Comprehensive error handling
- ✅ Proper resource cleanup (defer statements)

## Quick Start

### Prerequisites

- Docker & Docker Compose
- Go 1.21+ (for local development)
- Python 3.12+ with uv (for local Python development)
- Make (optional, for convenience commands)

### 1. Clone and Setup

```bash
# Clone the repository
git clone <repository-url>
cd QuietHire

# Run automated setup
./setup.sh

# This will:
# - Create .env from .env.example
# - Download Go dependencies
# - Sync Python dependencies with uv
# - Setup shared packages
```

### 2. Configure Environment

```bash
# Edit .env and set required API keys
cp .env.example .env

# Required keys:
# - TYPESENSE_API_KEY (generate a secure key)
# - GROQ_API_KEY (for job parsing - get from https://groq.com)
# - SERPAPI_KEY (optional, for Google Dork searches)
```

### 3. Start All Services

```bash
# Start all services with Docker Compose
docker-compose up -d

# Check service health
docker-compose ps

# View logs
docker-compose logs -f api worker
```

### 4. Initialize Databases

```bash
# Initialize ClickHouse schema
docker exec quiethire-clickhouse clickhouse-client --database=quiethire < config/clickhouse/schema.sql

# Initialize PostgreSQL schema
docker exec quiethire-postgres psql -U quiethire -d quiethire < config/postgres/osint-schema.sql

# Initialize Typesense (automatically initialized by API service)
```

### 5. Verify Installation

```bash
# Test API health
curl http://localhost:3000/health

# Check statistics
curl http://localhost:3000/api/v1/stats

# Test microservices
curl http://localhost:8001/health  # Parser
curl http://localhost:8002/health  # Crawler
curl http://localhost:8004/health  # OSINT Discovery
curl http://localhost:8003/health  # Proxy Manager
```

## API Endpoints

### Core Endpoints

```bash
# Health check
GET /health

# Job statistics
GET /api/v1/stats
# Returns: TotalJobs, ActiveJobs, Companies, AvgRealScore, LastCrawledAt

# Search jobs (Typesense)
GET /api/v1/search?q=engineer&limit=20
# Query params: q (query), limit (default: 20)

# List jobs with filters
GET /api/v1/jobs?limit=10&offset=0&location=Remote&remote=true
# Query params: limit, offset, location, remote (true/false)

# Get single job
GET /api/v1/jobs/:id
# Returns complete job details
```

### Microservice Endpoints

#### Parser Service (Port 8001)
```bash
# Health check
GET /health

# Parse job HTML
POST /api/v1/parse
Content-Type: application/json
{
  "html": "<html>...</html>",
  "url": "https://example.com/job/123"
}
# Returns: title, description, company, location, salary, job_type, etc.
```

#### Crawler Service (Port 8002)
```bash
# Health check
GET /health

# Batch crawl URLs
POST /crawl-batch
Content-Type: application/json
["https://example.com/jobs", "https://another.com/careers"]
# Returns: array of {url, html, status, success, error}
```

#### OSINT Discovery Service (Port 8004)
```bash
# Health check
GET /health

# Detect ATS platform
POST /api/v1/detect/ats
Content-Type: application/json
{"url": "https://jobs.lever.co/company"}
# Returns: {is_ats, platform, confidence, job_listing_urls}

# Discover career pages
POST /api/v1/discover/career-pages
Content-Type: application/json
{"domain": "example.com"}
# Returns: {career_pages, domain, total_found}

# Enumerate subdomains
POST /api/v1/enumerate/subdomains
Content-Type: application/json
{"domain": "example.com"}

# Search with Google Dorks
POST /api/v1/search/dork
Content-Type: application/json
{"query": "site:example.com 'careers'", "max_results": 10}
```

#### Proxy Manager (Port 8003)
```bash
# Health check
GET /health

# Get proxy
GET /proxy
# Returns: proxy URL for rotation
```

## Service Ports

| Service | Port | URL | Description |
|---------|------|-----|-------------|
| **API** | 3000 | http://localhost:3000 | Main REST API |
| **Parser** | 8001 | http://localhost:8001 | Job HTML parser |
| **Crawler** | 8002 | http://localhost:8002 | Web crawler |
| **Proxy Manager** | 8003 | http://localhost:8003 | Proxy rotation |
| **OSINT Discovery** | 8004 | http://localhost:8004 | Company discovery |
| **Temporal** | 7233 | localhost:7233 | Workflow orchestration |
| **Temporal UI** | 8080 | http://localhost:8080 | Temporal dashboard |
| **PostgreSQL** | 5432 | localhost:5432 | Primary database |
| **ClickHouse HTTP** | 8123 | http://localhost:8123 | Analytics queries |
| **ClickHouse Native** | 9000 | localhost:9000 | Native protocol |
| **Typesense** | 8108 | http://localhost:8108 | Search engine |
| **Dragonfly (Redis)** | 6379 | localhost:6379 | Cache/queue |

## Architecture

QuietHire uses a microservices architecture with distributed crawling and workflow orchestration:

```
┌─────────────────────────────────────────────────────────────┐
│                        API Gateway                           │
│                   (Go Fiber - Port 3000)                     │
│  Endpoints: /search, /jobs, /stats, /health                 │
└─────────────────────────────────────────────────────────────┘
                            │
          ┌─────────────────┼─────────────────┐
          │                 │                 │
┌─────────▼────────┐ ┌─────▼──────┐ ┌───────▼────────┐
│   Typesense      │ │ ClickHouse │ │  PostgreSQL    │
│  (Search Index)  │ │(Analytics) │ │ (Primary DB)   │
└──────────────────┘ └────────────┘ └────────────────┘

┌─────────────────────────────────────────────────────────────┐
│            Temporal Workflows (Orchestration)                │
│  ├── CrawlCoordinatorWorkflow                               │
│  ├── CompanyDiscoveryWorkflow                               │
│  └── GoogleDorkDiscoveryWorkflow                            │
└─────────────────────────────────────────────────────────────┘
                            │
          ┌─────────────────┼─────────────────┐
          │                 │                 │
┌─────────▼────────┐ ┌─────▼──────┐ ┌───────▼────────┐
│     Crawler      │ │   Parser   │ │ OSINT Discovery│
│  (Playwright)    │ │  (Groq AI) │ │ (ATS Detection)│
│   Port 8002      │ │ Port 8001  │ │   Port 8004    │
└──────────────────┘ └────────────┘ └────────────────┘
```

### Directory Structure

```
quiethire/
├── apps/
│   ├── api/                    # Go Fiber REST API
│   │   ├── cmd/
│   │   │   ├── api/           # Main API server
│   │   │   ├── worker/        # Temporal worker
│   │   │   ├── init-clickhouse/
│   │   │   └── init-typesense/
│   │   └── internal/
│   │       ├── activities/    # Temporal activities
│   │       ├── workflows/     # Temporal workflows
│   │       └── config/        # Configuration
│   ├── crawler-python/        # Python web crawler
│   ├── parser/                # Job HTML parser
│   ├── osint-discovery/       # Company discovery service
│   └── proxy-manager/         # Proxy rotation
├── pkg/
│   ├── go-common/             # Shared Go packages
│   └── py-common/             # Shared Python packages
├── config/
│   ├── clickhouse/schema.sql  # ClickHouse tables
│   └── postgres/osint-schema.sql # PostgreSQL tables
└── docker-compose.yml         # Orchestration
```

## Development

### Running Services Locally

#### Go Services
```bash
cd apps/api
go run cmd/api/main.go          # API server
go run cmd/worker/main.go       # Temporal worker
```

#### Python Services
```bash
cd apps/parser  # or other Python service

# Install dependencies
uv sync

# Run service
uv run uvicorn main:app --reload --host 0.0.0.0 --port 8000
```

### Database Access

#### ClickHouse
```bash
# CLI access
docker exec -it quiethire-clickhouse clickhouse-client --database=quiethire

# Example queries
SELECT COUNT(*) FROM jobs;
SELECT company, COUNT(*) as job_count FROM jobs GROUP BY company;
SELECT AVG(real_score) FROM jobs WHERE is_active = 1;
```

#### PostgreSQL
```bash
# CLI access
docker exec -it quiethire-postgres psql -U quiethire -d quiethire

# Example queries
SELECT COUNT(*) FROM companies;
SELECT * FROM discovered_urls LIMIT 10;
SELECT * FROM crawl_queue WHERE status = 'pending';
```

#### Typesense
```bash
# Search jobs
curl "http://localhost:8108/collections/jobs/documents/search?q=engineer&query_by=title,description" \
  -H "X-TYPESENSE-API-KEY: your-api-key"
```

### Adding Dependencies

#### Go
```bash
cd apps/api
go get github.com/package/name
go mod tidy
```

#### Python
```bash
cd apps/parser
uv add package-name
uv sync
```

### Scaling Services

```bash
# Scale crawlers for higher throughput
docker-compose up -d --scale crawler-python=5

# Monitor logs
docker-compose logs -f crawler-python
```

### Code Quality

```bash
# Run linters
make lint

# Format code
make fmt

# Run pre-commit hooks
pre-commit run --all-files
```

## Temporal Workflows

QuietHire uses Temporal for reliable, scalable workflow orchestration. The worker service processes workflows and activities.

### Available Workflows

1. **CompanyDiscoveryWorkflow** - Discover companies and their career pages
2. **GoogleDorkDiscoveryWorkflow** - Use Google Dorks to find job postings
3. **CrawlCoordinatorWorkflow** - Coordinate distributed crawling
4. **ScheduledCrawlWorkflow** - Scheduled recurring crawls
5. **ContinuousDiscoveryWorkflow** - Continuous company discovery

### Registered Activities

#### Crawling Activities
- `DiscoverJobURLs` - Find job URLs from a page
- `CrawlJobBatch` - Crawl multiple URLs in batch
- `ParseJobActivity` - Parse job HTML into structured data
- `ScoreJobActivity` - Calculate authenticity score
- `ExtractHiringManagerActivity` - Extract hiring manager info

#### Discovery Activities
- `DiscoverCompaniesFromGitHub` - Find companies via GitHub
- `DiscoverCompaniesFromGoogleDorks` - Find via Google Dorks
- `DiscoverCareerPages` - Find career pages for a domain
- `EnumerateSubdomains` - Enumerate company subdomains
- `DetectATS` - Detect ATS platform and confidence
- `QueueURLsForCrawling` - Add URLs to crawl queue
- `GenerateDorkQueries` - Generate Google Dork queries
- `ExecuteDorkQuery` - Execute a dork query
- `DetectATSAndExtractDomain` - Combined ATS detection + domain extraction

### Monitoring Workflows

```bash
# Check worker logs
docker-compose logs -f worker

# Access Temporal UI
open http://localhost:8080

# View workflow executions, activity history, and errors in the UI
```

## Testing

### Run All Tests

```bash
# Go tests
cd apps/api
go test ./...

# Python tests
cd apps/parser
uv run pytest
```

### Manual Testing

```bash
# Test full flow: Discovery → Crawl → Parse
# 1. Discover career pages
curl -X POST http://localhost:8004/api/v1/discover/career-pages \
  -H "Content-Type: application/json" \
  -d '{"domain": "stripe.com"}'

# 2. Crawl discovered URLs
curl -X POST http://localhost:8002/crawl-batch \
  -H "Content-Type: application/json" \
  -d '["https://stripe.com/jobs"]'

# 3. Parse job HTML
curl -X POST http://localhost:8001/api/v1/parse \
  -H "Content-Type: application/json" \
  -d '{"html": "<html>...</html>", "url": "https://stripe.com/jobs/123"}'

# 4. Verify in database
docker exec quiethire-clickhouse clickhouse-client --database=quiethire \
  -q "SELECT * FROM jobs ORDER BY crawled_at DESC LIMIT 5;"
```

## Troubleshooting

### Services Won't Start

```bash
# Check logs for specific service
docker-compose logs api
docker-compose logs worker

# Restart service
docker-compose restart api

# Rebuild and restart
docker-compose up -d --build api

# Clean restart all services
docker-compose down -v
docker-compose up -d
```

### Database Connection Errors

```bash
# Check database health
docker-compose ps | grep -E "(postgres|clickhouse|typesense)"

# Verify containers are healthy
docker inspect quiethire-postgres | grep Status
docker inspect quiethire-clickhouse | grep Status

# Check database credentials in .env
cat .env | grep -E "(DB_|CLICKHOUSE_|TYPESENSE_)"

# Test database connections
docker exec quiethire-postgres psql -U quiethire -d quiethire -c "SELECT 1;"
docker exec quiethire-clickhouse clickhouse-client --database=quiethire -q "SELECT 1;"
curl -H "X-TYPESENSE-API-KEY: your-key" http://localhost:8108/health
```

### Worker Not Processing Workflows

```bash
# Check worker logs
docker-compose logs worker | tail -50

# Verify worker registered workflows and activities
docker-compose logs worker | grep "Registered"

# Check Temporal connection
docker-compose logs worker | grep "Temporal"

# Restart worker
docker-compose restart worker
```

### Crawler Issues

```bash
# Check if browser is ready
curl http://localhost:8002/health | jq .

# View crawler logs
docker-compose logs crawler-python

# Test direct crawl
curl -X POST http://localhost:8002/crawl-batch \
  -H "Content-Type: application/json" \
  -d '["https://example.com"]'
```

### Parser Not Working

```bash
# Check GROQ API key
echo $GROQ_API_KEY

# Test parser with sample data
curl -X POST http://localhost:8001/api/v1/parse \
  -H "Content-Type: application/json" \
  -d '{"html":"<html><head><script type=\"application/ld+json\">{\"@type\":\"JobPosting\",\"title\":\"Test Job\"}</script></head></html>","url":"https://test.com"}'
```

### ClickHouse Query Issues

```bash
# Always specify the database
docker exec quiethire-clickhouse clickhouse-client --database=quiethire

# Common issue: Using default database instead of quiethire
# WRONG: clickhouse-client -q "SELECT * FROM jobs"
# RIGHT:  clickhouse-client --database=quiethire -q "SELECT * FROM jobs"
```

## Performance Tuning

### Scaling Crawlers

```bash
# Increase crawler instances
docker-compose up -d --scale crawler-python=10

# Monitor resource usage
docker stats
```

### Database Optimization

```bash
# ClickHouse: Optimize tables
docker exec quiethire-clickhouse clickhouse-client --database=quiethire \
  -q "OPTIMIZE TABLE jobs FINAL;"

# PostgreSQL: Vacuum and analyze
docker exec quiethire-postgres psql -U quiethire -d quiethire \
  -c "VACUUM ANALYZE;"
```

## Roadmap

### Next Features (In Priority Order)

1. **Frontend Dashboard** - React/Next.js UI for job search and management
2. **Real-time Job Scoring** - Implement authenticity scoring algorithm
3. **Email Generation** - AI-powered personalized emails to hiring managers
4. **Manager Extraction** - Extract hiring manager contact info from LinkedIn/company pages
5. **Advanced Filtering** - Salary range, experience level, tech stack filters
6. **Job Alerts** - Email/webhook notifications for new matching jobs
7. **Company Profiles** - Track companies, funding, tech stack, hiring trends
8. **Analytics Dashboard** - Job market trends, salary insights, demand metrics

### Future Enhancements

- [ ] GraphQL API for flexible querying
- [ ] WebSocket support for real-time updates
- [ ] Machine learning for job quality prediction
- [ ] Browser extension for one-click job tracking
- [ ] Mobile app (React Native)
- [ ] Multi-tenant support for recruiters

## Contributing

This is currently a solo project. For questions or suggestions:
- Open an issue on GitHub
- Review `docs/` for detailed architecture and planning docs
