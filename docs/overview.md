## Project Overview 
A real-time search engine that indexes every authentic job opening in the world — public and hidden — removes ghost/fake postings automatically, extracts the actual hiring manager’s name + verified corporate email, and lets users apply directly to the human who owns the role.

Core promise:  
“Type any role → instantly see only jobs that are real right now + the exact person to message.”

No login required to search. Login only for saves, outreach, and exports.

## Final Architecture (Docker-First)

Everything lives in a single monorepo and is deployed with Docker Compose (later Coolify / Docker Swarm / Kubernetes — zero code changes).

```
quiethire/
├── apps/
│   ├── api/                → Go (Fiber) – public API + search endpoint
│   ├── web/                → Next.js or HTMX frontend (optional, can be static)
│   ├── crawler-go/         → Go + playwright-go (fast 80% of pages)
│   ├── crawler-python/     → Python undetected-playwright + stealth (hard 20%)
│   ├── parser/             → Python FastAPI + Unstructured + Groq
│   ├── realscore/          → Python FastAPI – final ghost-job filter & 0–100 score
│   ├── email-writer/       → Python FastAPI – “Write Email for Me” (Llama-3.3-70B)
│   ├── manager-extractor/  → Python – extracts name/email from PDFs, signatures, Notion
│   └── proxy-manager/      → Go – hands out fresh residential/datacenter proxies
├── services/
│   ├── typesense/          → Typesense cluster (3 nodes)
│   ├── clickhouse/         → ClickHouse (single node → cluster later)
│   ├── temporal/           → Temporal Server + PostgreSQL
│   ├── postgres/           → Main relational DB
│   ├── dragonfly/          → Redis-compatible cache
│   └── grafana-stack/      → Loki + Prometheus + Tempo
└── docker-compose.yml      → one-command full stack
```

### Docker Compose Services (Final MVP)

```yaml
services:
  api:              # Go Fiber – main search API
  typesense:        # search engine (3 replicas)
  clickhouse:       # job storage + deduplication
  postgres:         # users, payments, saved searches
  temporal:         # workflow orchestration
  temporal-worker-go:   # Go workers (coordinator + fast crawler)
  crawler-go:       # 4–10 instances – fast playwright-go
  crawler-python:   # 4–12 instances – stealth + residential proxies
  parser:           # Python – turns raw HTML → clean structured job
  realscore:        # Python – final 0–100 authenticity score
  manager-extractor:# Python – pulls hiring manager + verified email
  email-writer:     # Python – generates perfect cold email in <400ms
  proxy-manager:    # Go – proxy rotation logic
  dragonfly:        # cache + Temporal visibility
  grafana:          # observability (Loki + Prometheus + Tempo)
  sentry:           # error tracking
```

### Data Flow (15-second user experience under the hood)

1. User searches → Go API → Typesense (instant filtered results)
2. Results come from ClickHouse (ingested jobs) → indexed nightly into Typesense
3. Crawlers (Go + Python) run 24/7 via Temporal workflows
   - Go crawler handles static + simple JS pages
   - Python stealth crawler handles Ashby, Greenhouse, Workday, Notion, etc.
4. Raw HTML → parser (Python + Unstructured + Groq) → structured job
5. Structured job → realscore service → 0–100 score + ghost filter
6. manager-extractor runs on every job → finds real hiring manager + email
7. Final validated job → ClickHouse → Typesense index

### Build & Execution Plan (16–20 Weeks, Solo Developer)

| Phase | Weeks | Goal | Key Deliverable |
|-------|-------|------|-----------------|
| 1     | 1-2   | Project setup + Go API + Typesense | Working search bar returning dummy jobs |
| 2     | 3-5   | Temporal + Go crawler (public jobs) | 100k+ public jobs ingested & searchable |
| 3     | 6-8   | Python stealth crawler + proxy rotation | First hidden Ashby/Greenhouse jobs appear |
| 4     | 9-11  | Parser + manager extraction pipeline | Clean JDs + real hiring manager names/emails |
| 5     | 12-14 | Real-Score engine (rules + LLM) | Only real jobs survive, 0–100 score visible |
| 6     | 15-16 | "Write Email for Me" + one-click send | Full outreach flow working |
| 7     | 17-18 | Auth, saves, daily digests, payments | Login + paid tier ready |
| 8     | 19-20 | Polish, rate limiting, monitoring, launch | quiethire.com live |

All services are Dockerized from day 1.  
One `docker compose up --scale crawler-python=8 --scale crawler-go=6` = full crawling fleet.