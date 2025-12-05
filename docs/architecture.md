# QuietHire System Architecture

## Table of Contents
- [Overview](#overview)
- [Architecture Principles](#architecture-principles)
- [System Context](#system-context)
- [Container Architecture](#container-architecture)
- [Component Breakdown](#component-breakdown)
- [Data Flow](#data-flow)
- [Technology Stack](#technology-stack)
- [Deployment Architecture](#deployment-architecture)
- [Scalability & Performance](#scalability--performance)

---

## Overview

QuietHire is a real-time job search engine built as a Docker-first monorepo application. The system is designed to:
- Index authentic job openings (public and hidden)
- Automatically filter ghost/fake postings
- Extract hiring manager contact information
- Enable direct outreach to decision-makers

**Core Value Proposition:** Type any role → instantly see only real jobs + the exact person to message.

---

## Architecture Principles

### 1. **Docker-First Design**
- Every service runs in a container from day one
- Single `docker-compose.yml` orchestrates the entire stack
- Easy transition to Kubernetes/Docker Swarm when needed

### 2. **Monorepo Structure**
- All code lives in one repository
- Shared tooling and dependencies
- Simplified deployment and versioning

### 3. **Microservices Architecture**
- Services communicate via APIs and message queues
- Each service has a single responsibility
- Independent scaling of components

### 4. **Polyglot Approach**
- Go for high-performance APIs and crawlers
- Python for ML/AI workloads and complex parsing
- Choose the right tool for each job

### 5. **Observability-First**
- Comprehensive logging, metrics, and tracing from the start
- Grafana stack for unified monitoring
- Proactive error detection with Sentry

---

## System Context

```mermaid
C4Context
    title System Context Diagram - QuietHire

    Person(user, "Job Seeker", "Searches for authentic jobs and contacts hiring managers")
    
    System(quiethire, "QuietHire Platform", "Real-time job search engine with authenticity scoring and hiring manager extraction")
    
    System_Ext(jobboards, "Job Boards", "Public job boards (Indeed, LinkedIn, etc.)")
    System_Ext(ats, "ATS Platforms", "Ashby, Greenhouse, Workday, Lever")
    System_Ext(notion, "Notion Pages", "Company career pages on Notion")
    System_Ext(llm, "LLM Services", "Groq, Llama models for parsing and email generation")
    System_Ext(proxy, "Proxy Services", "Residential and datacenter proxies")
    System_Ext(email, "Email Service", "SMTP/Email delivery service")
    
    Rel(user, quiethire, "Searches jobs, saves searches, sends emails")
    Rel(quiethire, jobboards, "Crawls job postings")
    Rel(quiethire, ats, "Crawls hidden jobs")
    Rel(quiethire, notion, "Extracts career pages")
    Rel(quiethire, llm, "Parses jobs, scores authenticity, generates emails")
    Rel(quiethire, proxy, "Routes requests through")
    Rel(quiethire, email, "Sends outreach emails")
```

---

## Container Architecture

```mermaid
C4Container
    title Container Diagram - QuietHire Platform

    Person(user, "Job Seeker")

    Container_Boundary(frontend, "Frontend Layer") {
        Container(web, "Web Application", "Next.js/HTMX", "Search interface, job listings, user dashboard")
    }

    Container_Boundary(api_layer, "API Layer") {
        Container(api, "Main API", "Go Fiber", "Search, user management, job retrieval")
    }

    Container_Boundary(processing, "Processing Services") {
        Container(crawler_go, "Go Crawler", "Go + Playwright", "Fast crawling for static pages")
        Container(crawler_py, "Python Crawler", "Python + Undetected Playwright", "Stealth crawling for protected sites")
        Container(parser, "Parser Service", "Python FastAPI + Unstructured + Groq", "Converts HTML to structured job data")
        Container(realscore, "RealScore Engine", "Python FastAPI", "Authenticity scoring (0-100)")
        Container(manager_ext, "Manager Extractor", "Python", "Extracts hiring manager info")
        Container(email_writer, "Email Writer", "Python FastAPI + Llama", "Generates personalized emails")
        Container(proxy_mgr, "Proxy Manager", "Go", "Manages proxy rotation")
    }

    Container_Boundary(orchestration, "Orchestration") {
        Container(temporal, "Temporal Server", "Temporal + PostgreSQL", "Workflow orchestration")
        Container(worker, "Temporal Workers", "Go", "Execute crawling workflows")
    }

    Container_Boundary(data_layer, "Data Layer") {
        ContainerDb(typesense, "Typesense", "Search Engine", "Indexed job search")
        ContainerDb(clickhouse, "ClickHouse", "Column Store", "Job storage and deduplication")
        ContainerDb(postgres, "PostgreSQL", "Relational DB", "Users, payments, saved searches")
        ContainerDb(dragonfly, "Dragonfly", "Redis-compatible", "Cache and sessions")
    }

    Container_Boundary(observability, "Observability") {
        Container(grafana, "Grafana Stack", "Loki + Prometheus + Tempo", "Monitoring and logging")
        Container(sentry, "Sentry", "Error Tracking", "Exception monitoring")
    }

    Rel(user, web, "Uses", "HTTPS")
    Rel(web, api, "API calls", "REST/JSON")
    Rel(api, typesense, "Searches", "HTTP")
    Rel(api, postgres, "Reads/Writes", "SQL")
    Rel(api, dragonfly, "Caches", "Redis Protocol")
    
    Rel(temporal, worker, "Schedules tasks")
    Rel(worker, crawler_go, "Triggers")
    Rel(worker, crawler_py, "Triggers")
    Rel(crawler_go, proxy_mgr, "Gets proxies")
    Rel(crawler_py, proxy_mgr, "Gets proxies")
    
    Rel(crawler_go, parser, "Sends HTML")
    Rel(crawler_py, parser, "Sends HTML")
    Rel(parser, realscore, "Sends structured job")
    Rel(realscore, manager_ext, "Sends validated job")
    Rel(manager_ext, clickhouse, "Stores job")
    Rel(clickhouse, typesense, "Indexes nightly")
    
    Rel(api, email_writer, "Requests email")
    Rel(email_writer, user, "Sends via SMTP")
```

---

## Component Breakdown

### Frontend Components

```mermaid
graph TB
    subgraph "Web Application"
        A[Search Interface]
        B[Job Listings]
        C[Job Detail View]
        D[User Dashboard]
        E[Saved Searches]
        F[Email Composer]
        G[Auth Pages]
    end
    
    A --> B
    B --> C
    C --> F
    D --> E
    G --> D
```

### API Service Components

```mermaid
graph LR
    subgraph "Go API Service"
        A[HTTP Router]
        B[Search Handler]
        C[User Handler]
        D[Job Handler]
        E[Auth Middleware]
        F[Rate Limiter]
        G[Cache Layer]
    end
    
    A --> E
    E --> F
    F --> B
    F --> C
    F --> D
    B --> G
    C --> G
    D --> G
```

### Crawler Components

```mermaid
graph TB
    subgraph "Crawling System"
        A[Temporal Coordinator]
        B[URL Queue]
        C[Go Crawler Pool]
        D[Python Crawler Pool]
        E[Proxy Manager]
        F[Rate Limiter]
        G[HTML Storage]
    end
    
    A --> B
    B --> C
    B --> D
    C --> E
    D --> E
    C --> F
    D --> F
    C --> G
    D --> G
```

### Processing Pipeline Components

```mermaid
graph LR
    subgraph "Job Processing Pipeline"
        A[Raw HTML]
        B[Parser Service]
        C[Structured Job]
        D[RealScore Engine]
        E[Scored Job]
        F[Manager Extractor]
        G[Complete Job]
        H[ClickHouse]
    end
    
    A --> B
    B --> C
    C --> D
    D --> E
    E --> F
    F --> G
    G --> H
```

---

## Data Flow

### End-to-End User Search Flow

```mermaid
sequenceDiagram
    actor User
    participant Web
    participant API
    participant Cache
    participant Typesense
    participant ClickHouse

    User->>Web: Enter search query
    Web->>API: GET /search?q=software+engineer
    API->>Cache: Check cache
    
    alt Cache Hit
        Cache-->>API: Return cached results
    else Cache Miss
        API->>Typesense: Search query
        Typesense->>ClickHouse: Fetch full job details
        ClickHouse-->>Typesense: Job data
        Typesense-->>API: Search results
        API->>Cache: Store results
    end
    
    API-->>Web: JSON response
    Web-->>User: Display job listings
```

### Crawling and Ingestion Flow

```mermaid
sequenceDiagram
    participant Temporal
    participant Worker
    participant Crawler
    participant Proxy
    participant Parser
    participant RealScore
    participant ManagerExt
    participant ClickHouse
    participant Typesense

    Temporal->>Worker: Schedule crawl job
    Worker->>Crawler: Start crawling
    Crawler->>Proxy: Request proxy
    Proxy-->>Crawler: Proxy credentials
    Crawler->>Crawler: Fetch job page
    Crawler->>Parser: Send raw HTML
    
    Parser->>Parser: Extract structured data
    Parser->>RealScore: Send job data
    RealScore->>RealScore: Calculate authenticity score
    
    alt Score >= 70
        RealScore->>ManagerExt: Send for manager extraction
        ManagerExt->>ManagerExt: Extract hiring manager
        ManagerExt->>ClickHouse: Store complete job
        
        Note over ClickHouse,Typesense: Nightly batch indexing
        ClickHouse->>Typesense: Index new jobs
    else Score < 70
        RealScore->>RealScore: Discard ghost job
    end
```

### Email Generation Flow

```mermaid
sequenceDiagram
    actor User
    participant Web
    participant API
    participant EmailWriter
    participant LLM
    participant SMTP

    User->>Web: Click "Write Email for Me"
    Web->>API: POST /email/generate
    API->>EmailWriter: Request email generation
    
    EmailWriter->>EmailWriter: Extract job context
    EmailWriter->>EmailWriter: Load user profile
    EmailWriter->>LLM: Generate personalized email
    LLM-->>EmailWriter: Email content
    EmailWriter-->>API: Generated email
    API-->>Web: Email preview
    Web-->>User: Show editable email
    
    User->>Web: Click "Send"
    Web->>API: POST /email/send
    API->>SMTP: Send email
    SMTP-->>API: Delivery confirmation
    API-->>Web: Success
    Web-->>User: Email sent notification
```

### Data Deduplication Flow

```mermaid
flowchart TD
    A[New Job Crawled] --> B{Generate Job Hash}
    B --> C[URL + Title + Company]
    C --> D{Check ClickHouse}
    
    D -->|Hash Exists| E{Compare Fields}
    E -->|Identical| F[Skip - Duplicate]
    E -->|Different| G[Update Existing Record]
    
    D -->|Hash Not Found| H[Insert New Job]
    
    G --> I[Mark as Updated]
    H --> I
    I --> J[Queue for Indexing]
```

---

### Languages & Frameworks

```mermaid
%%{init: {'theme':'base', 'themeVariables': { 'primaryColor':'#1e88e5','primaryTextColor':'#fff','primaryBorderColor':'#0d47a1','lineColor':'#42a5f5','secondaryColor':'#43a047','tertiaryColor':'#fb8c00','background':'#ffffff','mainBkg':'#1e88e5','secondaryBkg':'#43a047','tertiaryBkg':'#fb8c00'}}}%%
graph TB
    ROOT[QuietHire Tech Stack]
    
    subgraph Backend
        GO[Go]
        GO_FIBER[Fiber Framework]
        GO_PLAY[Playwright-go]
        GO_STD[Standard Library]
        
        PY[Python]
        PY_FAST[FastAPI]
        PY_PLAY[Undetected Playwright]
        PY_UNST[Unstructured]
    end
    
    subgraph Frontend
        NEXT[Next.js]
        HTMX[HTMX Alternative]
        CSS[Vanilla CSS]
    end
    
    subgraph Databases
        TS[Typesense]
        CH[ClickHouse]
        PG[PostgreSQL]
        DF[Dragonfly Redis]
    end
    
    subgraph AI_ML[AI/ML]
        GROQ[Groq API]
        LLAMA[Llama 3.3 70B]
        CUSTOM[Custom Scoring Models]
    end
    
    subgraph Infrastructure
        DOCKER[Docker]
        COMPOSE[Docker Compose]
        TEMPORAL[Temporal]
        GRAFANA[Grafana Stack]
    end
    
    ROOT --> Backend
    ROOT --> Frontend
    ROOT --> Databases
    ROOT --> AI_ML
    ROOT --> Infrastructure
    
    GO --> GO_FIBER
    GO --> GO_PLAY
    GO --> GO_STD
    
    PY --> PY_FAST
    PY --> PY_PLAY
    PY --> PY_UNST
    
    style ROOT fill:#1e88e5,stroke:#0d47a1,stroke-width:3px,color:#fff
    style GO fill:#00ADD8,stroke:#00758f,color:#fff
    style PY fill:#3776AB,stroke:#1e5a8e,color:#fff
    style NEXT fill:#000000,stroke:#333,color:#fff
    style HTMX fill:#3d72d7,stroke:#2557b8,color:#fff
    style TS fill:#e34234,stroke:#c02d1f,color:#fff
    style CH fill:#FFCC01,stroke:#d9ad00,color:#000
    style PG fill:#336791,stroke:#1e4a6b,color:#fff
    style DF fill:#DC382D,stroke:#b52d23,color:#fff
    style GROQ fill:#f55036,stroke:#d13d27,color:#fff
    style LLAMA fill:#0467DF,stroke:#0352b5,color:#fff
    style DOCKER fill:#2496ED,stroke:#1a7cc4,color:#fff
    style TEMPORAL fill:#000000,stroke:#333,color:#fff
```

### Service Technology Mapping

| Service | Language | Framework/Library | Purpose |
|---------|----------|-------------------|---------|
| **Main API** | Go | Fiber | High-performance REST API |
| **Go Crawler** | Go | Playwright-go | Fast static page crawling |
| **Python Crawler** | Python | Undetected Playwright | Stealth crawling |
| **Parser** | Python | FastAPI + Unstructured + Groq | HTML to structured data |
| **RealScore** | Python | FastAPI + Custom ML | Authenticity scoring |
| **Manager Extractor** | Python | Custom + NLP | Contact extraction |
| **Email Writer** | Python | FastAPI + Llama 3.3 | Email generation |
| **Proxy Manager** | Go | Standard Library | Proxy rotation |
| **Temporal Workers** | Go | Temporal SDK | Workflow execution |
| **Web Frontend** | JavaScript | Next.js/HTMX | User interface |

---

## Deployment Architecture

### Local Development

```mermaid
graph TB
    subgraph "Developer Machine"
        A[Docker Compose]
        
        subgraph "Application Services"
            B[API]
            C[Web]
            D[Crawlers]
            E[Processing Services]
        end
        
        subgraph "Data Services"
            F[PostgreSQL]
            G[ClickHouse]
            H[Typesense]
            I[Dragonfly]
        end
        
        subgraph "Infrastructure"
            J[Temporal]
            K[Grafana Stack]
        end
    end
    
    A --> B
    A --> C
    A --> D
    A --> E
    A --> F
    A --> G
    A --> H
    A --> I
    A --> J
    A --> K
```

### Scaling Strategy

```mermaid
flowchart LR
    A[Single Docker Compose] --> B{Traffic Growth}
    B -->|Low Traffic| C[Scale with --scale flag]
    C --> D[docker compose up --scale crawler-python=8]
    
    B -->|Medium Traffic| E[Docker Swarm]
    E --> F[Multi-node deployment]
    
    B -->|High Traffic| G[Kubernetes]
    G --> H[Auto-scaling pods]
    G --> I[Load balancing]
    G --> J[Service mesh]
```

### Service Scaling Configuration

```yaml
# Example scaling scenarios
# Low traffic (1-1000 users)
- api: 2 instances
- crawler-go: 4 instances
- crawler-python: 4 instances
- typesense: 1 node

# Medium traffic (1000-10000 users)
- api: 4-6 instances
- crawler-go: 6-8 instances
- crawler-python: 8-12 instances
- typesense: 3 nodes (cluster)

# High traffic (10000+ users)
- api: 10+ instances (auto-scale)
- crawler-go: 10+ instances
- crawler-python: 12+ instances
- typesense: 3-5 nodes
- clickhouse: cluster mode
```

---

## Scalability & Performance

### Performance Targets

```mermaid
graph LR
    subgraph "Performance SLAs"
        A[Search Latency<br/>< 200ms p95]
        B[API Response<br/>< 100ms p95]
        C[Email Generation<br/>< 400ms]
        D[Crawler Throughput<br/>100k+ jobs/day]
        E[System Uptime<br/>99.9%]
    end
```

### Caching Strategy

```mermaid
flowchart TD
    A[User Request] --> B{Check Cache Layer}
    
    B -->|Hit| C[Return from Dragonfly]
    C --> D[Response < 10ms]
    
    B -->|Miss| E{Check Typesense}
    E -->|Found| F[Return from Search Engine]
    F --> G[Cache in Dragonfly]
    G --> H[Response < 200ms]
    
    E -->|Not Found| I[Query ClickHouse]
    I --> J[Index in Typesense]
    J --> G
```

### Database Optimization

```mermaid
graph TB
    subgraph "Data Storage Strategy"
        A[Hot Data<br/>Dragonfly Cache<br/>TTL: 5 min]
        B[Warm Data<br/>Typesense Index<br/>Recent 6 months]
        C[Cold Data<br/>ClickHouse<br/>All historical data]
    end
    
    A --> B
    B --> C
    
    style A fill:#ff6b6b,stroke:#c92a2a,color:#fff
    style B fill:#ffd93d,stroke:#d9ad00,color:#000
    style C fill:#6bcf7f,stroke:#37b24d,color:#000
```

### Horizontal Scaling Points

```mermaid
graph TD
    A[Load Balancer] --> B[API Instances]
    A --> C[API Instances]
    A --> D[API Instances]
    
    E[Temporal] --> F[Worker Pool 1]
    E --> G[Worker Pool 2]
    E --> H[Worker Pool N]
    
    F --> I[Crawler Go 1-4]
    F --> J[Crawler Py 1-4]
    
    G --> K[Crawler Go 5-8]
    G --> L[Crawler Py 5-8]
    
    I --> M[Proxy Pool]
    J --> M
    K --> M
    L --> M
```

---

## Security Architecture

### Authentication Flow

```mermaid
sequenceDiagram
    actor User
    participant Web
    participant API
    participant Auth
    participant DB
    participant Session

    User->>Web: Login credentials
    Web->>API: POST /auth/login
    API->>Auth: Validate credentials
    Auth->>DB: Check user
    DB-->>Auth: User data
    Auth->>Auth: Hash & verify password
    Auth->>Session: Create session token
    Session-->>Auth: Token
    Auth-->>API: JWT token
    API-->>Web: Set secure cookie
    Web-->>User: Redirect to dashboard
```

### Data Protection Layers

```mermaid
graph TB
    subgraph "Security Layers"
        A[HTTPS/TLS]
        B[Rate Limiting]
        C[API Authentication]
        D[Input Validation]
        E[SQL Injection Prevention]
        F[XSS Protection]
        G[CSRF Tokens]
        H[Data Encryption at Rest]
    end
    
    A --> B
    B --> C
    C --> D
    D --> E
    D --> F
    D --> G
    E --> H
    F --> H
```

---

## Monitoring & Observability

### Observability Stack

```mermaid
graph TB
    subgraph "Application Services"
        A[API]
        B[Crawlers]
        C[Processing Services]
    end
    
    subgraph "Observability Stack"
        D[Prometheus<br/>Metrics]
        E[Loki<br/>Logs]
        F[Tempo<br/>Traces]
        G[Grafana<br/>Visualization]
        H[Sentry<br/>Errors]
    end
    
    A --> D
    A --> E
    A --> F
    A --> H
    
    B --> D
    B --> E
    B --> F
    B --> H
    
    C --> D
    C --> E
    C --> F
    C --> H
    
    D --> G
    E --> G
    F --> G
```

### Key Metrics Dashboard

```mermaid
graph LR
    subgraph "Monitored Metrics"
        A[Request Rate<br/>req/sec]
        B[Error Rate<br/>%]
        C[Latency<br/>p50, p95, p99]
        D[Crawler Success<br/>%]
        E[Database Connections<br/>active/max]
        F[Cache Hit Rate<br/>%]
        G[Queue Depth<br/>pending jobs]
        H[System Resources<br/>CPU, Memory, Disk]
    end
```

---

## Disaster Recovery

### Backup Strategy

```mermaid
flowchart TD
    A[Data Sources] --> B{Backup Type}
    
    B -->|Critical| C[PostgreSQL]
    C --> D[Daily Full Backup]
    C --> E[Hourly Incremental]
    
    B -->|Important| F[ClickHouse]
    F --> G[Daily Backup]
    
    B -->|Rebuildable| H[Typesense]
    H --> I[Weekly Snapshot]
    
    D --> J[S3/Cloud Storage]
    E --> J
    G --> J
    I --> J
    
    J --> K[30-day Retention]
```

### Failure Recovery

```mermaid
sequenceDiagram
    participant Monitor
    participant Alert
    participant OnCall
    participant System
    participant Backup

    Monitor->>Monitor: Detect failure
    Monitor->>Alert: Trigger alert
    Alert->>OnCall: Notify (PagerDuty/Email)
    
    alt Automatic Recovery
        Monitor->>System: Restart service
        System-->>Monitor: Health check OK
    else Manual Recovery Required
        OnCall->>System: Investigate
        OnCall->>Backup: Restore if needed
        Backup-->>System: Data restored
        OnCall->>System: Restart services
    end
    
    System->>Monitor: Resume monitoring
```

---

## Future Architecture Considerations

### Phase 1: Current State (Weeks 1-30)
- Single Docker Compose deployment
- Manual scaling with `--scale` flag
- Single-region deployment

### Phase 2: Growth (Months 10-12)
- Migrate to Docker Swarm or Kubernetes
- Multi-region deployment
- Auto-scaling based on metrics
- CDN for static assets

### Phase 3: Scale (Year 2+)
- Global deployment across multiple regions
- Advanced caching with edge computing
- Machine learning model improvements
- Real-time collaboration features

```mermaid
timeline
    title Architecture Evolution
    Phase 1 (Months 1-7) : Docker Compose
                          : Single Server
                          : Manual Scaling
    Phase 2 (Months 8-12) : Docker Swarm/K8s
                           : Multi-node Cluster
                           : Auto-scaling
    Phase 3 (Year 2+) : Multi-region
                      : Edge Computing
                      : Advanced ML
```

---

## Conclusion

This architecture is designed for:
- **Simplicity**: Start with Docker Compose, scale when needed
- **Flexibility**: Polyglot approach using the best tool for each job
- **Observability**: Comprehensive monitoring from day one
- **Scalability**: Clear path from single server to global deployment
- **Maintainability**: Solo developer can understand and manage the entire system

The modular design allows incremental development while maintaining system integrity. Each service can be developed, tested, and deployed independently, making it ideal for solo development with the ability to scale as the platform grows.
