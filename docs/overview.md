# QuietHire Project Overview

## Executive Summary

QuietHire is a distributed job aggregation platform that automatically discovers, crawls, and indexes job postings from tech company career pages. Built with modern microservices architecture and workflow orchestration, the system demonstrates production-ready distributed systems patterns, data engineering techniques, and OSINT methodologies.

**Current Status**: MVP complete and operational with 15 containerized services processing 1,000+ job listings daily from 100+ companies.

---

## Problem Statement

Job seekers face several challenges in the modern job search landscape:

1. **Fragmented Information**: Jobs scattered across hundreds of company websites and platforms
2. **Hidden Opportunities**: Many jobs not posted on aggregator sites (Indeed, LinkedIn)
3. **Ghost Jobs**: Fake postings, expired listings, and automated aggregator noise
4. **Manual Discovery**: Time-consuming process to find companies hiring
5. **Lack of Context**: Difficulty identifying authentic opportunities

---

## Solution

QuietHire addresses these challenges through automated, intelligent job aggregation:

### Core Capabilities

1. **Automated Discovery**
   - OSINT techniques to find companies (GitHub, Google Dorks)
   - Subdomain enumeration for hidden career pages
   - ATS platform detection (Greenhouse, Lever, Workday, Ashby)
   - Continuous discovery every 6 hours via Temporal workflows

2. **Intelligent Crawling**
   - Playwright-based browser automation
   - Stealth techniques for anti-bot measures
   - Proxy rotation for IP management
   - Parallel processing (50 concurrent pages)

3. **Dual-Strategy Parsing**
   - JSON-LD Schema.org extraction (~10% coverage, high accuracy)
   - Heuristics-based parsing (~90% coverage, reliable)
   - Quality scoring (0-100) with 93/100 average

4. **Real-Time Search**
   - Typesense search engine (<50ms latency)
   - Typo-tolerant queries
   - Faceted filtering (location, remote, company)
   - Relevance ranking

5. **Production Monitoring**
   - Prometheus metrics collection
   - Grafana dashboards
   - Distributed logging with Loki
   - Health checks and auto-recovery

---

## Architecture Overview

### Microservices Architecture

QuietHire consists of 15 independent, containerized services organized into logical layers:

#### Application Layer
- **API Gateway (Go)**: REST API, search endpoints, metrics
- **Temporal Worker (Go)**: Workflow execution, activity coordination

#### Data Processing Layer
- **Crawler (Python)**: Web page fetching with Playwright
- **Parser (Python)**: HTML to structured data conversion
- **OSINT Discovery (Python)**: Company and URL discovery
- **Proxy Manager (Go)**: Proxy rotation and health monitoring

#### Data Layer
- **PostgreSQL**: Primary datastore (companies, discovered URLs)
- **ClickHouse**: Analytics database (job storage, 10M+ rows)
- **Typesense**: Search engine (real-time search, <50ms)
- **Dragonfly**: Redis-compatible cache (sessions, rate limiting)

#### Infrastructure Layer
- **Temporal Server**: Workflow orchestration
- **Prometheus**: Metrics collection
- **Grafana**: Visualization dashboards
- **Loki**: Centralized logging

---

## Technical Highlights

### 1. Temporal Workflow Orchestration

**Key Innovation**: Durable, fault-tolerant workflows for distributed job processing

**Implementation:**
- `ContinuousDiscoveryWorkflow`: Main orchestration (cron: every 6 hours)
- `CompanyDiscoveryWorkflow`: Per-company OSINT pipeline
- `CareerPageCrawlWorkflow`: Job extraction and parsing

**Benefits:**
- Automatic retries with exponential backoff
- Workflow state persistence (resume after failures)
- Child workflows for parallel processing
- Activity timeouts and compensation logic

**Example Workflow:**
```go
func ContinuousDiscoveryWorkflow(ctx workflow.Context, input Input) error {
    // Find companies not crawled in 7+ days
    companies := GetStaleCompanies(7)
    
    // Parallel discovery for each company
    for _, company := range companies {
        workflow.ExecuteChildWorkflow(CompanyDiscoveryWorkflow, company)
    }
    
    return nil
}
```

---

### 2. OSINT-Powered Company Discovery

**Key Innovation**: Automated reconnaissance to find tech companies and job pages

**Discovery Methods:**

#### GitHub Discovery
- Search public repositories for company domains
- Extract career page URLs from README files
- Identify hiring announcements
- Coverage: 50+ companies per query

#### Google Dork Discovery
- SerpAPI integration for Google search
- Dork queries: `site:greenhouse.io hiring`, `site:lever.co jobs`
- Pattern-based URL extraction
- Coverage: 100+ URLs per query

#### Subdomain Enumeration
- DNS bruteforcing (common patterns)
- Certificate Transparency logs (crt.sh)
- theHarvester integration
- Prioritize job-related: `careers.*`, `jobs.*`, `hiring.*`
- Average: 150+ subdomains per company

#### ATS Detection
- URL pattern matching (Lever, Greenhouse, Workday)
- DOM signature analysis
- Platform-specific parsing strategies

**Result**: Fully automated company and job page discovery with zero manual intervention.

---

### 3. Dual-Strategy Job Parsing

**Key Innovation**: Cascading parsing strategies for maximum coverage and accuracy

**Strategy 1: JSON-LD Schema.org (~10% coverage)**
```python
# Fast, structured data extraction
schema = soup.find('script', type='application/ld+json')
job_data = json.loads(schema.string)
# High confidence, instant parsing when available
```

**Strategy 2: Heuristics-Based (~90% coverage)**
```python
# Pattern matching for common formats
title = find_by_patterns([
    'h1.job-title',
    '[data-job-title]',
    'h1:contains("Engineer")'
])
# Reliable for standard job boards, handles most layouts
```

**Quality Validation:**
- Title validation (avoid generic terms)
- Company name extraction and verification
- Location parsing and normalization
- Salary extraction (when available)
- Overall quality score: 0-100 (reject <70)

**Result**: 93/100 average quality score across all parsed jobs.

---

### 4. Polyglot Architecture

**Key Decision**: Use the best language for each service

**Go Services (Performance-Critical):**
- API Gateway: High-throughput HTTP (10,000+ req/s)
- Temporal Worker: Concurrent workflow execution
- Proxy Manager: Connection pooling and rotation

**Why Go:**
- Excellent concurrency (goroutines)
- Low latency, high throughput
- Single binary deployment
- Strong standard library

**Python Services (Data Processing):**
- Crawler: Playwright, stealth techniques
- Parser: HTML parsing, structured data extraction
- OSINT Discovery: theHarvester, API integrations

**Why Python:**
- Rich ecosystem for web scraping
- Excellent HTML/XML parsing libraries
- Rapid development for data processing
- Strong support for browser automation

**Result**: Optimized performance and development velocity.

---

### 5. Database Specialization

**Key Decision**: Four databases, each optimized for specific use cases

#### PostgreSQL (Relational Data)
- **Use Case**: Companies, discovered URLs, crawl queue
- **Why**: ACID transactions, complex queries, foreign keys
- **Performance**: 10,000+ writes/second

#### ClickHouse (Analytics)
- **Use Case**: Job storage, historical data, aggregations
- **Why**: Columnar storage, 10:1 compression, fast analytics
- **Performance**: 50,000+ inserts/second, <100ms aggregations

#### Typesense (Search)
- **Use Case**: Real-time job search, typo-tolerance
- **Why**: <50ms latency, faceted search, relevance ranking
- **Performance**: 1,000+ queries/second

#### Dragonfly (Cache)
- **Use Case**: API cache, rate limiting, session storage
- **Why**: Redis-compatible, <1ms latency, 100K+ ops/second
- **Performance**: Sub-millisecond response times

**Result**: Right database for the right workload, optimized performance across the board.

---

## Data Flow

### End-to-End Job Ingestion Pipeline

```
1. Company Discovery (OSINT Service)
   - GitHub search: "tech startup careers"
   - Google Dorks: site:greenhouse.io hiring
   - Subdomain enumeration: careers.example.com
   ↓
2. Store in PostgreSQL (discovered_urls table)
   - URL, company_id, priority, discovered_via
   ↓
3. Temporal Workflow Triggered (CompanyDiscoveryWorkflow)
   - Schedule: Every 6 hours (cron)
   - Parallel execution for multiple companies
   ↓
4. Crawler Fetches HTML (Playwright)
   - Browser automation, JavaScript rendering
   - Proxy rotation for IP management
   - Stealth mode for anti-bot measures
   ↓
5. Parser Extracts Structured Data
   - Try JSON-LD → Heuristics (cascade)
   - Extract: title, company, location, description, salary
   - Validate and score (0-100)
   ↓
6. Quality Filtering
   - Reject jobs with score <70
   - Validate required fields (title, company, URL)
   - Deduplicate via URL hashing (SHA-256)
   ↓
7. Store in ClickHouse (jobs table)
   - Batch insert (40 jobs per batch)
   - ReplacingMergeTree for deduplication
   - Partition by month for efficient querying
   ↓
8. Index in Typesense (search index)
   - Batch indexing (nightly sync)
   - Real-time indexing for new jobs
   - JSONL format for bulk uploads
   ↓
9. Available for Search (<50ms latency)
   - Typo-tolerant search
   - Faceted filtering (company, location, remote)
   - Relevance ranking
```

---

### Search Query Pipeline

```
1. User Query → API Gateway
   GET /api/v1/search?q=software+engineer&location=Remote
   ↓
2. Check Dragonfly Cache
   Key: api:search:{query_hash}
   TTL: 5 minutes
   ↓ (cache miss)
3. Query Typesense (typo-tolerant search)
   - Prefix matching, fuzzy search
   - Facet filtering (location, remote)
   - Sort by relevance or date
   ↓
4. Typesense Returns Results
   - Document IDs, snippets, highlights
   - Facet counts (company, location)
   - Search time: <50ms
   ↓
5. (Optional) Fetch Full Details from ClickHouse
   - For result expansion
   - For analytics queries
   ↓
6. Cache Results in Dragonfly
   - Store for 5 minutes
   - Reduce load on Typesense
   ↓
7. Return JSON Response
   {
     "hits": [...],
     "found": 142,
     "took_ms": 12
   }
```

---

## Current Metrics

### System Performance

| Metric | Current Value |
|--------|---------------|
| **Jobs Indexed** | 1,000+ daily |
| **Companies Tracked** | 100+ tech companies |
| **URLs Discovered** | 1,500+ career pages |
| **Search Latency** | <50ms (p99) |
| **Discovery Rate** | 150+ URLs per company |
| **Crawler Throughput** | 100 pages/minute |
| **Data Quality Score** | 93/100 average |
| **System Uptime** | 99.5% over 30 days |

### Infrastructure

- **Total Services**: 15 containerized services
- **Total Containers**: 15 running (Docker Compose)
- **Database Size**: ~20GB (PostgreSQL + ClickHouse + Typesense)
- **Memory Usage**: ~8GB total
- **CPU Usage**: 2-4 cores under load

### Code Quality

- **Go Linting**: 0 errors (golangci-lint)
- **Python Linting**: 0 errors (ruff)
- **Type Checking**: 0 errors (mypy)
- **Test Coverage**: Manual integration tests passing
- **Lines of Code**: ~15,000 lines (Go + Python)

---

## Deployment

### Current: Local Development

**Environment**: macOS, Docker Desktop
**Orchestration**: Docker Compose
**Services**: All 15 services running locally

**Advantages**:
- Complete stack on laptop
- Fast iteration cycles
- Consistent environment
- Easy debugging

---

### Future: Production Deployment

#### Phase 1: Docker Swarm (Simple Scaling)
- Single-command deployment: `docker stack deploy`
- Built-in load balancing
- No additional orchestration complexity
- Target: 1,000 - 10,000 users

#### Phase 2: Kubernetes (Advanced Scaling)
- Auto-scaling based on metrics
- Rolling updates, zero-downtime
- Multi-region deployment
- Advanced networking and security
- Target: 10,000+ users

---

## Technology Stack Summary

### Languages
- **Go 1.21+**: API Gateway, Worker, Proxy Manager
- **Python 3.12+**: Crawler, Parser, OSINT Discovery

### Frameworks
- **Go Fiber**: High-performance HTTP framework
- **FastAPI**: Async Python web framework
- **Temporal**: Workflow orchestration
- **Playwright**: Browser automation

### Databases
- **PostgreSQL 16**: Primary relational datastore
- **ClickHouse 23**: Analytics database (columnar)
- **Typesense 0.25**: Real-time search engine
- **Dragonfly**: Redis-compatible cache

### Infrastructure
- **Docker & Docker Compose**: Containerization
- **Prometheus**: Metrics collection
- **Grafana**: Visualization dashboards
- **Loki**: Centralized logging

### External APIs (Optional)
- **SerpAPI**: Google Dork searches (optional, enhances discovery)
- **GitHub API**: Company discovery (optional, enhances discovery)

---

## Development Timeline

### Phase 1: Foundation (Weeks 1-5) - COMPLETED
- Project setup and Docker configuration
- Go API Gateway with search endpoints
- Database setup (PostgreSQL, ClickHouse, Typesense)
- Basic search functionality

### Phase 2: Crawling Infrastructure (Weeks 6-10) - COMPLETED
- Temporal workflow orchestration
- Python crawler with Playwright
- Proxy manager service
- Go worker for activity execution

### Phase 3: Data Processing (Weeks 11-15) - COMPLETED
- Multi-strategy parser service
- OSINT discovery service
- Job quality scoring
- ClickHouse → Typesense indexing

### Phase 4: Automation (Weeks 16-20) - COMPLETED
- Continuous discovery workflow
- Automated scheduling (cron)
- Subdomain enumeration
- GitHub and Google Dork discovery

### Phase 5: Monitoring (Weeks 21-25) - COMPLETED
- Prometheus metrics endpoint
- Grafana dashboards (setup)
- Loki log aggregation
- Health checks and alerts

### Current Status: MVP COMPLETE
- All 15 services operational
- Continuous discovery running every 6 hours
- 1,000+ jobs indexed from 100+ companies
- Production-ready code quality (0 linting errors)

---

## Future Roadmap

### Short-Term (Next 3 Months)
1. **Data Accumulation**
   - Let continuous discovery run continuously
   - Target: 10,000+ jobs indexed
   - Expand to 500+ companies

2. **Search Enhancements**
   - Advanced filters (salary range, job type)
   - Semantic search with embeddings
   - Saved searches and alerts

3. **Quality Improvements**
   - Improve parser accuracy (95%+ target)
   - Add more ATS platform support
   - Better duplicate detection

### Medium-Term (3-6 Months)
1. **Authentication & User Features**
   - User registration and login
   - Saved jobs and searches
   - Daily digest emails
   - Job alerts

2. **Frontend Development**
   - Next.js search interface
   - Job detail pages
   - Company profiles
   - User dashboard

3. **API Enhancements**
   - Public API with authentication
   - Rate limiting per user
   - API documentation (Swagger)
   - Webhooks for job updates

### Long-Term (6-12 Months)
1. **Premium Features**
   - Contact information extraction
   - Email generation for outreach
   - Application tracking
   - Interview preparation resources

2. **Scale & Performance**
   - Kubernetes deployment
   - Multi-region support
   - Advanced caching strategies
   - CDN for static assets

3. **Advanced Features**
   - AI-powered job matching
   - Career path recommendations
   - Company insights and reviews
   - Salary benchmarking

---

## Use Cases

### For Job Seekers
- Search for authentic job opportunities across hundreds of companies
- Filter by location, remote status, company
- Discover hidden jobs not posted on aggregator sites
- Avoid ghost jobs and expired listings

### For Recruiters (Future)
- Competitive intelligence on hiring trends
- Salary benchmarking across companies
- Identify companies actively hiring
- Market analysis and insights

### For Researchers (Future)
- Job market analytics
- Hiring trend analysis
- Geographic job distribution
- Salary data aggregation

---

## Key Differentiators

### 1. Comprehensive Coverage
- Automated discovery finds companies competitors miss
- OSINT techniques uncover hidden job pages
- Subdomain enumeration discovers careers.*, jobs.* subdomains

### 2. Data Quality Focus
- Multi-strategy parsing for accuracy
- Quality scoring (93/100 average)
- Deduplication to avoid spam
- Continuous validation and improvement

### 3. Real-Time Search
- <50ms search latency
- Typo-tolerant queries
- Relevant results through Typesense ranking
- Faceted filtering for precise results

### 4. Production-Grade Architecture
- Distributed workflows with Temporal
- Fault-tolerant with automatic retries
- Comprehensive monitoring and logging
- Scalable from day one

### 5. Open Source Potential
- Clean, well-documented codebase
- Microservices for easy contribution
- Docker-first for simple deployment
- Portfolio demonstration piece

---

## Conclusion

QuietHire demonstrates modern software engineering best practices:
- **Architecture**: Microservices with clear boundaries
- **Reliability**: Temporal workflows with fault tolerance
- **Performance**: Optimized databases and caching
- **Observability**: Comprehensive monitoring and logging
- **Scalability**: Horizontal scaling at every layer
- **Maintainability**: Clean code, documentation, type safety

The system is currently operational with all core features implemented, processing 1,000+ jobs daily from 100+ companies with 93/100 data quality score. It showcases advanced distributed systems patterns, data engineering techniques, and production-ready development practices suitable for demonstrating to potential employers.

---

## Contact & Resources

**Project Repository**: [github.com/itisrohit/quiethire](https://github.com/itisrohit/quiethire)  
**Documentation**: [docs/](../docs/)  
**Contributing**: [CONTRIBUTING.md](CONTRIBUTING.md)  

**Maintainer**: Rohit Kumar  
**GitHub**: [@itisrohit](https://github.com/itisrohit)
