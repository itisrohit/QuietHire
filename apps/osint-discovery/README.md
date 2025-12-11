# OSINT Discovery Service

OSINT-powered job discovery service that finds companies, career pages, and ATS platforms across the internet.

## Features

- **Company Discovery**: Find companies from GitHub, Crunchbase, AngelList, and other sources
- **Google Dorking**: Advanced search queries to discover career pages
- **ATS Detection**: Identify and classify ATS platforms (Greenhouse, Ashby, Lever, Workday, etc.)
- **Subdomain Enumeration**: Discover job/career subdomains via DNS and Certificate Transparency logs
- **Career Page Detection**: Identify career pages using ML and heuristics
- **URL Queue Management**: Feed discovered URLs to crawler services

## API Endpoints

- `GET /health` - Health check
- `POST /api/v1/discover/companies` - Discover companies from various sources
- `POST /api/v1/discover/career-pages` - Find career pages for a company
- `POST /api/v1/detect/ats` - Detect ATS platform from URL
- `POST /api/v1/enumerate/subdomains` - Enumerate job-related subdomains
- `POST /api/v1/search/dork` - Execute Google dork queries

## Environment Variables

- `SERPAPI_API_KEY` - SerpAPI key for Google searches
- `GITHUB_TOKEN` - GitHub API token
- `CRUNCHBASE_API_KEY` - Crunchbase API key (optional)
- `LOG_LEVEL` - Logging level (default: INFO)

## Usage

```bash
uvicorn main:app --host 0.0.0.0 --port 8000
```
