# QuietHire System Architecture

## Table of Contents
- [Overview](#overview)
- [Architecture Principles](#architecture-principles)
- [System Components](#system-components)
- [Data Architecture](#data-architecture)
- [Workflow Architecture](#workflow-architecture)
- [Service Communication](#service-communication)
- [Technology Stack](#technology-stack)
- [Deployment Architecture](#deployment-architecture)
- [Scalability Strategy](#scalability-strategy)
- [Security Architecture](#security-architecture)
- [Monitoring & Observability](#monitoring--observability)

---

## Overview

QuietHire is a distributed job aggregation platform built using microservices architecture and workflow orchestration. The system automatically discovers companies, crawls career pages, extracts job postings, and indexes them for real-time search.

### Core Architecture Characteristics

- **Microservices-based**: 15 independent, containerized services
- **Event-driven**: Temporal workflows orchestrate asynchronous operations
- **Polyglot**: Go for performance-critical services, Python for data processing
- **Cloud-native**: Docker Compose for local development, ready for Kubernetes deployment
- **Observable**: Comprehensive monitoring with Prometheus, Grafana, and Loki

---

## Architecture Principles

### 1. Single Responsibility Principle

Each microservice has a well-defined, singular purpose:
- **API**: User-facing REST endpoints and search interface
- **Crawler**: Web page fetching and HTML extraction
- **Parser**: HTML to structured data conversion
- **OSINT Discovery**: Company and URL discovery
- **Worker**: Temporal workflow execution
- **Proxy Manager**: Proxy rotation and management

### 2. Separation of Concerns

Clear boundaries between different system layers:
- **Presentation Layer**: API Gateway (Go Fiber)
- **Application Layer**: Temporal Workflows and Activities
- **Data Processing Layer**: Crawler, Parser, OSINT services
- **Data Layer**: PostgreSQL, ClickHouse, Typesense, Dragonfly
- **Infrastructure Layer**: Docker, Temporal Server, Monitoring

### 3. Fault Tolerance

System designed for reliability:
- Automatic retries with exponential backoff (Temporal)
- Activity timeouts and compensation logic
- Database transaction management
- Health checks for all services
- Graceful degradation when services are unavailable

### 4. Scalability First

Horizontal scaling at every layer:
- Stateless services for easy replication
- Database sharding strategies planned
- Distributed workflow execution
- Parallel processing with configurable concurrency

### 5. Observability by Design

Monitoring and logging built-in from day one:
- Structured logging across all services
- Custom Prometheus metrics
- Distributed tracing with Temporal
- Real-time dashboards with Grafana

---

## System Components

### API Gateway (Go Fiber)

**Responsibilities:**
- Handle HTTP requests from users
- Execute search queries against Typesense
- Serve Prometheus metrics endpoint
- Health check endpoint for monitoring

**Key Features:**
- High-performance HTTP routing
- Connection pooling for databases
- Request logging and error handling
- CORS support for web clients

**Technology:**
- Language: Go 1.21+
- Framework: Fiber v2
- Port: 3000

**Endpoints:**
- `GET /api/v1/search` - Job search
- `GET /api/v1/jobs` - List jobs with filters
- `GET /api/v1/stats` - System statistics
- `GET /metrics` - Prometheus metrics
- `GET /health` - Health check

---

### Temporal Worker (Go)

**Responsibilities:**
- Execute workflow and activity code
- Manage workflow state and history
- Handle retries and error recovery
- Coordinate distributed operations

**Registered Workflows:**
1. `ContinuousDiscoveryWorkflow` - Main orchestration loop
2. `CompanyDiscoveryWorkflow` - Per-company discovery
3. `CareerPageCrawlWorkflow` - Job extraction pipeline

**Registered Activities:**
- `GetStaleCompanies` - Find companies needing re-crawl
- `UpdateCompanyLastCrawled` - Update crawl timestamps
- `DiscoverCareerPages` - OSINT career page discovery
- `EnumerateSubdomains` - Subdomain enumeration
- `DetectATS` - ATS platform detection
- `CrawlCareerPage` - Fetch career page HTML
- `ExtractJobLinks` - Parse job URLs from HTML
- `CrawlJobPages` - Batch fetch job pages
- `ParseJobData` - Extract structured job data
- `StoreJobsInClickHouse` - Persist jobs to database

**Technology:**
- Language: Go 1.21+
- Framework: Temporal Go SDK
- Task Queue: `quiethire-task-queue`

---

### Crawler Service (Python)

**Responsibilities:**
- Fetch web pages using Playwright
- Handle JavaScript rendering
- Execute browser automation
- Return raw HTML for processing

**Key Features:**
- Batch URL crawling
- Stealth mode for anti-bot measures
- Proxy support for IP rotation
- Screenshot capture for debugging
- Configurable timeouts and retries

**Technology:**
- Language: Python 3.12+
- Framework: FastAPI
- Browser: Playwright (Chromium)
- Port: 8002

**Endpoints:**
- `POST /crawl-batch` - Crawl multiple URLs

---

### Parser Service (Python)

**Responsibilities:**
- Convert raw HTML to structured job data
- Extract job links from career pages
- Apply multiple parsing strategies
- Validate and clean extracted data

**Parsing Strategies:**
1. **JSON-LD Schema.org** (~10% coverage)
   - Fast, structured data extraction
   - High confidence when available
   - Parses JobPosting schema directly
   
2. **Heuristics-based** (~90% coverage)
   - Pattern matching for common formats
   - Reliable for standard job boards
   - CSS selector-based extraction

**Technology:**
- Language: Python 3.12+
- Framework: FastAPI
- Libraries: BeautifulSoup4, lxml
- Port: 8001

**Endpoints:**
- `POST /api/v1/parse` - Parse job HTML
- `POST /api/v1/extract-job-links` - Extract job URLs

**Output Schema:**
```json
{
  "title": "Senior Software Engineer",
  "company": "Example Corp",
  "location": "San Francisco, CA",
  "description": "Full job description...",
  "requirements": ["5+ years experience", "..."],
  "salary_min": 150000,
  "salary_max": 200000,
  "remote": true,
  "posted_at": "2024-12-14T00:00:00Z",
  "application_url": "https://...",
  "quality_score": 95
}
```

---

### OSINT Discovery Service (Python)

**Responsibilities:**
- Discover company career pages
- Enumerate subdomains for job boards
- Detect ATS platforms
- GitHub-based company discovery
- Google Dork-based URL discovery

**Discovery Methods:**
1. **Career Page Discovery**
   - Common patterns: `/careers`, `/jobs`, `/about/jobs`
   - HTTP probing with status code validation
   - Content-based verification
   
2. **Subdomain Enumeration**
   - DNS bruteforcing
   - crt.sh certificate transparency logs
   - theHarvester integration
   - Prioritize job-related subdomains: `careers.*`, `jobs.*`, `hiring.*`
   
3. **ATS Detection**
   - URL pattern matching (Lever, Greenhouse, Workday, etc.)
   - DOM signature analysis
   - Platform-specific identifiers

4. **GitHub Discovery**
   - Search public repositories for company domains
   - Extract URLs from README files
   - Identify hiring announcements

5. **Google Dork Discovery**
   - SerpAPI integration
   - Queries: "site:greenhouse.io hiring", "site:lever.co jobs"
   - Pagination and result filtering

**Technology:**
- Language: Python 3.12+
- Framework: FastAPI
- Libraries: theHarvester, requests, beautifulsoup4
- Port: 8004

**Endpoints:**
- `POST /api/v1/discover/career-pages` - Find career pages
- `POST /api/v1/enumerate/subdomains` - Enumerate subdomains
- `POST /api/v1/detect/ats` - Detect ATS platform
- `POST /api/v1/discover/github` - GitHub company discovery
- `POST /api/v1/discover/dork` - Google Dork discovery

---

### Proxy Manager (Go)

**Responsibilities:**
- Manage pool of residential and datacenter proxies
- Rotate proxies to avoid rate limiting
- Monitor proxy health and performance
- Provide fresh proxies to crawler services

**Key Features:**
- Automatic proxy rotation
- Health check monitoring
- Failed proxy removal
- Performance tracking
- Load balancing across proxy pool

**Technology:**
- Language: Go 1.21+
- Framework: Standard library (net/http)
- Port: 8003

**Endpoints:**
- `GET /proxy` - Get next available proxy
- `POST /proxy/health` - Report proxy health
- `GET /proxy/stats` - Proxy statistics

---

## Data Architecture

### Database Layer

QuietHire uses four specialized databases, each optimized for specific use cases:

#### 1. PostgreSQL (Primary Datastore)

**Purpose**: Relational data, OSINT discovery data, crawl queue

**Schema: OSINT Discovery**
```sql
CREATE TABLE companies (
    id UUID PRIMARY KEY,
    domain VARCHAR(255) UNIQUE NOT NULL,
    name VARCHAR(255),
    last_crawled_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW(),
    updated_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE discovered_urls (
    id UUID PRIMARY KEY,
    company_id UUID REFERENCES companies(id),
    url TEXT NOT NULL,
    url_type VARCHAR(50), -- career_page, job_board, ats
    discovered_via VARCHAR(50), -- osint, github, dork, subdomain
    priority INTEGER DEFAULT 5,
    crawled BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE discovered_subdomains (
    id UUID PRIMARY KEY,
    company_id UUID REFERENCES companies(id),
    subdomain VARCHAR(255) NOT NULL,
    method VARCHAR(50), -- dns, crt, harvester
    is_job_related BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE dork_results (
    id UUID PRIMARY KEY,
    query TEXT NOT NULL,
    url TEXT NOT NULL,
    title TEXT,
    snippet TEXT,
    rank INTEGER,
    discovered_at TIMESTAMP DEFAULT NOW()
);

CREATE TABLE crawl_queue (
    id UUID PRIMARY KEY,
    url TEXT NOT NULL,
    priority INTEGER DEFAULT 5,
    status VARCHAR(50) DEFAULT 'pending', -- pending, processing, completed, failed
    attempts INTEGER DEFAULT 0,
    last_attempt_at TIMESTAMP,
    created_at TIMESTAMP DEFAULT NOW()
);
```

**Indexes:**
- `idx_companies_domain` on `companies(domain)`
- `idx_companies_last_crawled` on `companies(last_crawled_at)`
- `idx_discovered_urls_company` on `discovered_urls(company_id)`
- `idx_discovered_urls_crawled` on `discovered_urls(crawled)`
- `idx_crawl_queue_status` on `crawl_queue(status, priority)`

**Connection Configuration:**
- Host: localhost
- Port: 5432
- Database: quiethire
- Max Connections: 100

---

#### 2. ClickHouse (Analytics Database)

**Purpose**: High-volume job storage, analytics, historical data

**Schema: Jobs Table**
```sql
CREATE TABLE jobs (
    id UUID,
    url String,
    url_hash String,
    title String,
    company_name String,
    location String,
    description String,
    requirements Array(String),
    salary_min Nullable(Int64),
    salary_max Nullable(Int64),
    salary_currency Nullable(String),
    remote Boolean,
    job_type String, -- full-time, part-time, contract
    posted_at Nullable(DateTime),
    application_url String,
    quality_score Int8,
    parsed_at DateTime,
    crawled_at DateTime,
    created_at DateTime DEFAULT now(),
    updated_at DateTime DEFAULT now()
)
ENGINE = ReplacingMergeTree(updated_at)
PARTITION BY toYYYYMM(created_at)
ORDER BY (company_name, url_hash, created_at);
```

**Key Design Decisions:**
- `ReplacingMergeTree` for automatic deduplication
- Partition by month for efficient querying
- `url_hash` for deduplication (SHA-256)
- Arrays for multi-value fields (requirements)
- Nullable for optional fields (salary)

**Performance Characteristics:**
- Insert throughput: 50,000+ rows/second
- Query latency: <100ms for aggregations
- Compression: 10:1 ratio on text fields
- Storage: ~1KB per job record (compressed)

**Connection Configuration:**
- Host: localhost
- Port: 9000
- Database: quiethire
- Max Connections: 50

---

#### 3. Typesense (Search Engine)

**Purpose**: Real-time job search with typo-tolerance and faceting

**Schema: Jobs Collection**
```json
{
  "name": "jobs",
  "fields": [
    {"name": "id", "type": "string"},
    {"name": "title", "type": "string"},
    {"name": "company", "type": "string", "facet": true},
    {"name": "location", "type": "string", "facet": true},
    {"name": "description", "type": "string"},
    {"name": "remote", "type": "bool", "facet": true},
    {"name": "job_type", "type": "string", "facet": true},
    {"name": "salary_min", "type": "int32", "optional": true},
    {"name": "salary_max", "type": "int32", "optional": true},
    {"name": "quality_score", "type": "int32"},
    {"name": "posted_at", "type": "int64"}
  ],
  "default_sorting_field": "posted_at"
}
```

**Search Features:**
- Typo tolerance (2 typos allowed)
- Prefix matching
- Faceted search (company, location, remote)
- Sorting by relevance or date
- Filtering by salary range
- Geo-search (planned)

**Indexing Strategy:**
- Batch indexing from ClickHouse (nightly)
- Real-time indexing for new jobs
- JSONL format for batch uploads (40 jobs/batch)
- Automatic schema validation

**Performance:**
- Search latency: <50ms (p99)
- Index size: ~2KB per job
- Throughput: 1,000+ queries/second

**Connection Configuration:**
- Host: localhost
- Port: 7108
- API Key: (from environment)
- Protocol: HTTP

---

#### 4. Dragonfly (Cache & Session Store)

**Purpose**: Redis-compatible cache for high-frequency data

**Use Cases:**
1. **Activity Queue** (Temporal visibility)
   - Key: `temporal:activity:{id}`
   - TTL: 1 hour
   
2. **API Response Cache**
   - Key: `api:search:{query_hash}`
   - TTL: 5 minutes
   
3. **Crawler State**
   - Key: `crawler:state:{workflow_id}`
   - TTL: 24 hours
   
4. **Rate Limiting**
   - Key: `ratelimit:{ip}:{endpoint}`
   - TTL: 1 minute

**Performance:**
- Latency: <1ms (p99)
- Throughput: 100,000+ ops/second
- Memory: 2GB allocated

**Connection Configuration:**
- Host: localhost
- Port: 6380
- Protocol: Redis

---

### Data Flow Architecture

#### Ingestion Pipeline

```
1. Company Discovery (OSINT Service)
   ↓
2. Store in PostgreSQL (discovered_urls table)
   ↓
3. Temporal Workflow Triggered
   ↓
4. Crawler Fetches HTML (Playwright)
   ↓
5. Parser Extracts Structured Data
   ↓
6. Quality Validation (score 0-100)
   ↓
7. Store in ClickHouse (jobs table)
   ↓
8. Index in Typesense (batch or real-time)
   ↓
9. Available for Search
```

#### Search Pipeline

```
1. User Query → API Gateway
   ↓
2. Check Dragonfly Cache
   ↓ (cache miss)
3. Query Typesense (typo-tolerant search)
   ↓
4. Typesense Returns Document IDs + Snippets
   ↓
5. Fetch Full Details from ClickHouse (optional)
   ↓
6. Cache Results in Dragonfly
   ↓
7. Return JSON Response to User
```

---

## Workflow Architecture

### Temporal Workflows

QuietHire uses Temporal for durable, fault-tolerant workflow execution. Workflows are written in Go and executed by worker processes.

#### 1. ContinuousDiscoveryWorkflow

**Purpose**: Main orchestration loop for automated job discovery

**Schedule**: Cron - Every 6 hours

**Input Parameters:**
```go
type ContinuousDiscoveryInput struct {
    GitHubQuery         string  // GitHub search query
    DorkQuery           string  // Google Dork query
    StaleThresholdDays  int     // Re-crawl threshold (default: 7)
    MaxNewCompanies     int     // Max new companies per run (default: 50)
    RunGitHubDiscovery  bool    // Enable GitHub discovery
    RunDorkDiscovery    bool    // Enable Google Dork discovery
}
```

**Workflow Logic:**
```go
func ContinuousDiscoveryWorkflow(ctx workflow.Context, input ContinuousDiscoveryInput) error {
    // 1. Find stale companies (not crawled in N days)
    companies := GetStaleCompanies(staleThresholdDays)
    
    // 2. Trigger discovery for each company (parallel)
    for _, company := range companies {
        workflow.ExecuteChildWorkflow(CompanyDiscoveryWorkflow, company)
        UpdateCompanyLastCrawled(company.Domain)
    }
    
    // 3. (Optional) Discover new companies via GitHub
    if input.RunGitHubDiscovery {
        newCompanies := DiscoverFromGitHub(input.GitHubQuery, input.MaxNewCompanies)
        // Store in PostgreSQL
    }
    
    // 4. (Optional) Discover new companies via Google Dorks
    if input.RunDorkDiscovery {
        newUrls := DiscoverFromDorks(input.DorkQuery, input.MaxNewCompanies)
        // Store in PostgreSQL
    }
    
    return nil
}
```

**Error Handling:**
- Retry policy: Exponential backoff, max 3 attempts
- Timeout: 2 hours per workflow execution
- Compensation: Log failures, continue with remaining companies

---

#### 2. CompanyDiscoveryWorkflow

**Purpose**: Discover all job-related URLs for a single company

**Input:**
```go
type CompanyDiscoveryInput struct {
    Domain string
}
```

**Workflow Logic:**
```go
func CompanyDiscoveryWorkflow(ctx workflow.Context, input CompanyDiscoveryInput) error {
    // 1. Discover career pages
    careerPages := DiscoverCareerPages(input.Domain)
    
    // 2. Enumerate subdomains
    subdomains := EnumerateSubdomains(input.Domain)
    
    // 3. Detect ATS platforms
    for _, url := range careerPages {
        ats := DetectATS(url)
        // Store ATS type for specialized parsing
    }
    
    // 4. Store all discovered URLs in PostgreSQL
    StoreDiscoveredURLs(careerPages, subdomains)
    
    // 5. Trigger crawl workflows for high-priority URLs
    for _, url := range careerPages {
        workflow.ExecuteChildWorkflow(CareerPageCrawlWorkflow, url)
    }
    
    return nil
}
```

**Parallelization:**
- Subdomain enumeration runs in parallel (3 methods)
- ATS detection runs concurrently for all URLs
- Child workflows execute asynchronously

---

#### 3. CareerPageCrawlWorkflow

**Purpose**: Extract and parse all jobs from a career page

**Input:**
```go
type CareerPageCrawlInput struct {
    URL string
}
```

**Workflow Logic:**
```go
func CareerPageCrawlWorkflow(ctx workflow.Context, input CareerPageCrawlInput) error {
    // 1. Crawl career page HTML
    html := CrawlCareerPage(input.URL)
    
    // 2. Extract job links
    jobLinks := ExtractJobLinks(html, input.URL)
    
    // 3. Batch crawl job pages (50 concurrent)
    jobHTMLs := CrawlJobPages(jobLinks, batchSize=50)
    
    // 4. Parse each job page
    jobs := []Job{}
    for _, jobHTML := range jobHTMLs {
        job := ParseJobData(jobHTML)
        if job.QualityScore >= 70 {
            jobs = append(jobs, job)
        }
    }
    
    // 5. Store jobs in ClickHouse (batch insert)
    StoreJobsInClickHouse(jobs)
    
    return nil
}
```

**Performance Optimizations:**
- Parallel job page crawling (50 concurrent)
- Batch database inserts (40 jobs/batch)
- Quality filtering before storage (score >= 70)
- Deduplication via URL hashing

---

### Activity Design Patterns

#### Idempotent Activities

All activities are designed to be idempotent (can be retried safely):
```go
func CrawlCareerPage(ctx context.Context, url string) (string, error) {
    // Check cache first (idempotency)
    if cached := getFromCache(url); cached != "" {
        return cached, nil
    }
    
    // Perform crawl
    html, err := crawler.Fetch(url)
    if err != nil {
        return "", err
    }
    
    // Cache result
    cacheResult(url, html, ttl=1hour)
    
    return html, nil
}
```

#### Activity Timeouts

Each activity has appropriate timeouts:
- Fast activities (DB queries): 10 seconds
- Medium activities (API calls): 30 seconds
- Slow activities (crawling): 2 minutes

#### Retry Policies

```go
retryPolicy := &temporal.RetryPolicy{
    InitialInterval:    1 * time.Second,
    BackoffCoefficient: 2.0,
    MaximumInterval:    60 * time.Second,
    MaximumAttempts:    3,
}
```

---

## Service Communication

### Communication Patterns

#### 1. Synchronous HTTP (REST)

Used for request-response interactions:
- API Gateway → Typesense (search queries)
- Worker → Parser Service (job parsing)
- Worker → Crawler Service (page fetching)
- Worker → OSINT Service (discovery)

**Example:**
```go
// Worker calls Parser Service
resp, err := http.Post("http://parser:8001/api/v1/parse", 
    "application/json", 
    bytes.NewBuffer(payload))
```

#### 2. Asynchronous Workflows (Temporal)

Used for long-running, distributed operations:
- Continuous discovery orchestration
- Multi-step job extraction pipeline
- Error recovery and retries

**Example:**
```go
// Start child workflow
childWorkflow := workflow.ExecuteChildWorkflow(
    ctx, 
    CompanyDiscoveryWorkflow, 
    input,
)
```

#### 3. Database as Message Queue

Used for decoupled processing:
- `crawl_queue` table in PostgreSQL
- Workers poll for pending URLs
- Status updates for coordination

---

### Service Discovery

**Local Development:**
- Docker Compose DNS: Service names resolve to container IPs
- Example: `http://parser:8001`

**Production:**
- Kubernetes Service Discovery
- Environment variables for endpoints
- Health checks for availability

---

## Technology Stack

### Backend Languages

#### Go (Golang)

**Used For:**
- API Gateway (high-performance HTTP)
- Temporal Worker (workflow execution)
- Proxy Manager (concurrent connections)

**Rationale:**
- Excellent concurrency primitives (goroutines)
- Low latency, high throughput
- Strong standard library
- Fast compilation and deployment

**Key Libraries:**
- Fiber v2: Web framework
- Temporal Go SDK: Workflow orchestration
- pgx: PostgreSQL driver
- ClickHouse Go client

---

#### Python

**Used For:**
- Crawler (Playwright, stealth techniques)
- Parser (HTML parsing, structured data extraction)
- OSINT Discovery (theHarvester, APIs)

**Rationale:**
- Rich ecosystem for web scraping
- Excellent HTML/XML parsing libraries
- Rapid development for data processing
- Strong support for browser automation

**Key Libraries:**
- FastAPI: Async web framework
- Playwright: Browser automation
- BeautifulSoup4: HTML parsing
- lxml: Fast XML/HTML processing
- theHarvester: OSINT tool

---

### Infrastructure

#### Docker & Docker Compose

**Configuration:**
- 15 services defined in `docker-compose.yml`
- Shared networks for service communication
- Volume mounts for persistence
- Health checks for all services

**Scaling:**
```bash
# Scale crawler instances
docker-compose up -d --scale crawler-python=10
```

---

#### Temporal

**Configuration:**
- Server: Temporal OSS
- Persistence: PostgreSQL
- UI: Web interface on port 8080

**Task Queues:**
- `quiethire-task-queue`: Default queue for all workflows

**Namespaces:**
- `default`: All workflows run in default namespace

---

## Deployment Architecture

### Local Development

```
Developer Machine
├── Docker Compose (15 containers)
│   ├── Application Services (API, Worker, Crawler, Parser, OSINT, Proxy)
│   ├── Data Services (PostgreSQL, ClickHouse, Typesense, Dragonfly)
│   └── Infrastructure (Temporal, Prometheus, Grafana, Loki)
└── Source Code (mounted volumes for hot-reload)
```

**Advantages:**
- Complete stack on laptop
- Fast iteration cycles
- Consistent environment

---

### Production Deployment (Planned)

#### Docker Swarm (Simple Scaling)

```bash
# Initialize swarm
docker swarm init

# Deploy stack
docker stack deploy -c docker-compose.prod.yml quiethire

# Scale services
docker service scale quiethire_crawler-python=20
```

**Advantages:**
- Simple migration from Compose
- Built-in load balancing
- No additional orchestration complexity

---

#### Kubernetes (Advanced Scaling)

```
Kubernetes Cluster
├── Ingress Controller (nginx)
├── Namespaces
│   ├── quiethire-prod
│   └── quiethire-staging
├── Deployments
│   ├── api (3 replicas)
│   ├── worker (5 replicas)
│   ├── crawler-python (10 replicas)
│   └── parser (5 replicas)
├── StatefulSets
│   ├── postgres
│   ├── clickhouse
│   └── typesense (3 nodes)
├── Services (ClusterIP, LoadBalancer)
└── ConfigMaps & Secrets
```

**Advantages:**
- Auto-scaling based on metrics
- Rolling updates, zero-downtime
- Advanced networking and security
- Multi-region deployment

---

## Scalability Strategy

### Horizontal Scaling Targets

| Service | Current | 1K Users | 10K Users | 100K Users |
|---------|---------|----------|-----------|------------|
| **API** | 1 | 2 | 5 | 20 |
| **Worker** | 1 | 2 | 5 | 10 |
| **Crawler** | 1 | 4 | 10 | 30 |
| **Parser** | 1 | 2 | 5 | 15 |
| **OSINT** | 1 | 1 | 2 | 5 |
| **Proxy Manager** | 1 | 1 | 2 | 5 |

### Database Scaling

#### PostgreSQL
- **1K users**: Single instance
- **10K users**: Read replicas (1 primary, 2 replicas)
- **100K users**: Sharding by company domain

#### ClickHouse
- **1K users**: Single node
- **10K users**: 3-node cluster
- **100K users**: Distributed table across 10 nodes

#### Typesense
- **1K users**: Single node
- **10K users**: 3-node cluster
- **100K users**: 5-node cluster with replication

---

## Security Architecture

### Authentication & Authorization

**Current State:**
- No authentication (public search)

**Planned:**
- JWT-based authentication
- OAuth2 for social login
- API key authentication for programmatic access
- Role-based access control (RBAC)

---

### Data Protection

**In Transit:**
- TLS 1.3 for all external communications
- Encrypted connections to databases

**At Rest:**
- Database encryption (PostgreSQL, ClickHouse)
- Encrypted secrets in Docker/K8s

---

### Security Best Practices

1. **Input Validation**: All user inputs sanitized
2. **SQL Injection Prevention**: Parameterized queries only
3. **Rate Limiting**: Per-IP and per-user limits
4. **CORS**: Restricted to allowed origins
5. **Secrets Management**: Environment variables, never hardcoded

---

## Monitoring & Observability

### Metrics (Prometheus)

**Custom Metrics:**
```
quiethire_jobs_total          # Total jobs indexed
quiethire_crawler_requests    # Crawler requests per second
quiethire_parser_duration     # Parser latency histogram
quiethire_workflow_duration   # Workflow execution time
quiethire_error_total         # Error count by service
```

**System Metrics:**
- CPU usage per service
- Memory consumption
- Goroutine count (Go services)
- Database connections
- HTTP request latency

**Scrape Configuration:**
```yaml
scrape_configs:
  - job_name: 'api'
    static_configs:
      - targets: ['api:3000']
  - job_name: 'worker'
    static_configs:
      - targets: ['worker:9090']
```

---

### Logging (Loki)

**Structured Logging Format:**
```json
{
  "timestamp": "2024-12-14T10:30:00Z",
  "level": "info",
  "service": "worker",
  "workflow_id": "abc123",
  "message": "Started CompanyDiscoveryWorkflow",
  "company": "example.com"
}
```

**Log Aggregation:**
- All services send logs to Loki
- Grafana for log querying and visualization
- Retention: 30 days

---

### Dashboards (Grafana)

**System Overview Dashboard:**
- Total jobs indexed (gauge)
- Jobs indexed per day (graph)
- Crawler success rate (gauge)
- API request rate (graph)
- Error rate (graph)

**Workflow Dashboard:**
- Active workflows (gauge)
- Workflow success rate (gauge)
- Workflow duration (histogram)
- Failed workflows (list)

**Database Dashboard:**
- Query latency (graph)
- Connection pool usage (gauge)
- Database size (gauge)
- Slow queries (list)

---

## Conclusion

QuietHire's architecture is designed for:
- **Reliability**: Temporal workflows with automatic retries
- **Scalability**: Horizontal scaling at every layer
- **Maintainability**: Clear service boundaries, comprehensive monitoring
- **Performance**: Optimized databases, caching, parallel processing
- **Flexibility**: Polyglot architecture using best tools for each job

The system is currently running in local development with all 15 services containerized. The architecture supports seamless migration to production environments (Docker Swarm or Kubernetes) with minimal code changes.

Key architectural decisions prioritize simplicity for solo development while maintaining production-ready patterns for future scaling.
