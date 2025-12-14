# QuietHire API Documentation

## Table of Contents
- [Overview](#overview)
- [Base URLs](#base-urls)
- [Authentication](#authentication)
- [Rate Limiting](#rate-limiting)
- [API Gateway Endpoints](#api-gateway-endpoints)
- [Microservice Endpoints](#microservice-endpoints)
- [Response Formats](#response-formats)
- [Error Handling](#error-handling)
- [Examples](#examples)

---

## Overview

QuietHire exposes REST APIs for job search and system integration. The API is organized into two layers:

1. **Public API (Port 3000)**: User-facing endpoints for job search and statistics
2. **Internal APIs (Ports 8001-8004)**: Microservice endpoints for data processing

**Current Version**: v1  
**Protocol**: HTTP/HTTPS  
**Data Format**: JSON  
**Character Encoding**: UTF-8

---

## Base URLs

### Local Development
```
API Gateway:     http://localhost:3000
Parser Service:  http://localhost:8001
Crawler Service: http://localhost:8002
Proxy Manager:   http://localhost:8003
OSINT Discovery: http://localhost:8004
```

### Production (Planned)
```
API Gateway:     https://api.quiethire.com
```

---

## Authentication

**Current Status**: No authentication required (public search)

**Planned**:
- JWT Bearer tokens for authenticated requests
- API keys for programmatic access
- OAuth2 for third-party integrations

---

## Rate Limiting

**Current**: No rate limiting implemented

**Planned**:
```
Free Tier:     100 requests/minute
Authenticated: 1,000 requests/minute
Premium:       10,000 requests/minute
```

**Headers** (future):
```
X-RateLimit-Limit: 100
X-RateLimit-Remaining: 95
X-RateLimit-Reset: 1702540800
```

---

## API Gateway Endpoints

### Health Check

Check API server health and connectivity.

**Endpoint**: `GET /health`

**Response**:
```json
{
  "status": "healthy",
  "timestamp": "2024-12-14T10:30:00Z",
  "services": {
    "typesense": "connected",
    "clickhouse": "connected",
    "postgres": "connected",
    "dragonfly": "connected"
  }
}
```

**Status Codes**:
- `200 OK`: All services healthy
- `503 Service Unavailable`: One or more services down

---

### Search Jobs

Search for jobs with typo-tolerant, faceted search.

**Endpoint**: `GET /api/v1/search`

**Query Parameters**:

| Parameter | Type | Required | Description | Default |
|-----------|------|----------|-------------|---------|
| `q` | string | Yes | Search query (title, company, description) | - |
| `location` | string | No | Filter by location (e.g., "San Francisco" or "Remote") | - |
| `remote` | boolean | No | Filter for remote jobs only | false |
| `company` | string | No | Filter by company name | - |
| `job_type` | string | No | Filter by job type (full-time, part-time, contract) | - |
| `salary_min` | integer | No | Minimum salary filter | - |
| `salary_max` | integer | No | Maximum salary filter | - |
| `limit` | integer | No | Number of results (max 250) | 20 |
| `offset` | integer | No | Pagination offset | 0 |
| `sort_by` | string | No | Sort field (relevance, posted_at, salary) | relevance |

**Example Request**:
```bash
GET /api/v1/search?q=software+engineer&location=Remote&limit=10&sort_by=posted_at
```

**Response**:
```json
{
  "hits": [
    {
      "id": "550e8400-e29b-41d4-a716-446655440000",
      "title": "Senior Software Engineer",
      "company": "Linear",
      "location": "Remote",
      "description": "Build amazing developer tools with a small, focused team...",
      "requirements": [
        "5+ years of software engineering experience",
        "Strong TypeScript and React skills",
        "Experience with distributed systems"
      ],
      "salary_min": 150000,
      "salary_max": 200000,
      "salary_currency": "USD",
      "remote": true,
      "job_type": "full-time",
      "url": "https://linear.app/careers/engineer",
      "application_url": "https://jobs.ashbyhq.com/linear/...",
      "posted_at": "2024-12-14T00:00:00Z",
      "quality_score": 95
    }
  ],
  "found": 142,
  "page": 1,
  "limit": 10,
  "took_ms": 12,
  "facets": {
    "company": {
      "Linear": 15,
      "Vercel": 12,
      "Stripe": 10
    },
    "location": {
      "Remote": 85,
      "San Francisco": 30,
      "New York": 27
    },
    "remote": {
      "true": 85,
      "false": 57
    }
  }
}
```

**Status Codes**:
- `200 OK`: Search successful
- `400 Bad Request`: Invalid query parameters
- `500 Internal Server Error`: Search engine failure

---

### List Jobs

Retrieve jobs with filtering and pagination (alternative to search).

**Endpoint**: `GET /api/v1/jobs`

**Query Parameters**:

| Parameter | Type | Required | Description | Default |
|-----------|------|----------|-------------|---------|
| `location` | string | No | Filter by location | - |
| `remote` | boolean | No | Filter for remote jobs | - |
| `company` | string | No | Filter by company name | - |
| `job_type` | string | No | full-time, part-time, contract | - |
| `posted_after` | string | No | ISO 8601 date (e.g., 2024-12-01) | - |
| `quality_min` | integer | No | Minimum quality score (0-100) | 70 |
| `limit` | integer | No | Number of results (max 100) | 20 |
| `offset` | integer | No | Pagination offset | 0 |

**Example Request**:
```bash
GET /api/v1/jobs?remote=true&quality_min=90&limit=50&offset=0
```

**Response**:
```json
{
  "jobs": [
    {
      "id": "uuid",
      "title": "Backend Engineer",
      "company": "Notion",
      "location": "Remote",
      "remote": true,
      "job_type": "full-time",
      "salary_min": 140000,
      "salary_max": 180000,
      "url": "https://notion.so/careers",
      "posted_at": "2024-12-13T00:00:00Z",
      "quality_score": 92
    }
  ],
  "total": 237,
  "limit": 50,
  "offset": 0,
  "has_more": true
}
```

**Status Codes**:
- `200 OK`: Query successful
- `400 Bad Request`: Invalid parameters
- `500 Internal Server Error`: Database error

---

### Get Job Details

Retrieve full details for a specific job.

**Endpoint**: `GET /api/v1/jobs/{id}`

**Path Parameters**:
- `id` (string, required): Job UUID

**Example Request**:
```bash
GET /api/v1/jobs/550e8400-e29b-41d4-a716-446655440000
```

**Response**:
```json
{
  "id": "550e8400-e29b-41d4-a716-446655440000",
  "url": "https://linear.app/careers/engineer",
  "title": "Senior Software Engineer",
  "company_name": "Linear",
  "location": "Remote",
  "description": "Full job description with HTML formatting...",
  "requirements": [
    "5+ years of software engineering experience",
    "Strong TypeScript and React skills",
    "Experience with distributed systems"
  ],
  "salary_min": 150000,
  "salary_max": 200000,
  "salary_currency": "USD",
  "remote": true,
  "job_type": "full-time",
  "application_url": "https://jobs.ashbyhq.com/linear/...",
  "posted_at": "2024-12-14T00:00:00Z",
  "quality_score": 95,
  "parsed_at": "2024-12-14T01:00:00Z",
  "crawled_at": "2024-12-14T00:30:00Z"
}
```

**Status Codes**:
- `200 OK`: Job found
- `404 Not Found`: Job ID doesn't exist
- `500 Internal Server Error`: Database error

---

### Get Statistics

Retrieve system-wide statistics.

**Endpoint**: `GET /api/v1/stats`

**Response**:
```json
{
  "total_jobs": 1247,
  "active_jobs": 1189,
  "companies": 87,
  "avg_quality_score": 93.5,
  "last_crawled_at": "2024-12-14T01:30:00Z",
  "jobs_by_type": {
    "full-time": 980,
    "part-time": 45,
    "contract": 164
  },
  "jobs_by_location": {
    "Remote": 654,
    "San Francisco": 213,
    "New York": 145
  },
  "remote_jobs_percentage": 52.4
}
```

**Status Codes**:
- `200 OK`: Statistics retrieved
- `500 Internal Server Error`: Database error

---

### Prometheus Metrics

Retrieve system metrics in Prometheus format.

**Endpoint**: `GET /metrics`

**Response** (text/plain):
```
# HELP quiethire_jobs_total Total number of jobs in database
# TYPE quiethire_jobs_total gauge
quiethire_jobs_total 1247

# HELP go_goroutines Number of goroutines
# TYPE go_goroutines gauge
go_goroutines 42

# HELP process_cpu_seconds_total Total user and system CPU time
# TYPE process_cpu_seconds_total counter
process_cpu_seconds_total 123.45

# Standard Go runtime metrics...
```

**Status Codes**:
- `200 OK`: Metrics retrieved

---

## Microservice Endpoints

### Parser Service (Port 8001)

#### Parse Job HTML

Convert raw HTML to structured job data.

**Endpoint**: `POST /api/v1/parse`

**Request Body**:
```json
{
  "html": "<html><body>Job posting HTML...</body></html>",
  "url": "https://example.com/job/senior-engineer"
}
```

**Response**:
```json
{
  "title": "Senior Software Engineer",
  "company": "Example Corp",
  "location": "San Francisco, CA",
  "description": "We are looking for an experienced engineer...",
  "requirements": [
    "5+ years experience",
    "Python/Go expertise",
    "Distributed systems knowledge"
  ],
  "salary_min": 150000,
  "salary_max": 200000,
  "salary_currency": "USD",
  "remote": false,
  "job_type": "full-time",
  "application_url": "https://example.com/apply/123",
  "posted_at": "2024-12-10T00:00:00Z",
  "quality_score": 92,
  "parsing_strategy": "heuristics",
  "parsing_time_ms": 234
}
```

**Status Codes**:
- `200 OK`: Parsing successful
- `400 Bad Request`: Invalid HTML or URL
- `422 Unprocessable Entity`: Parsing failed
- `500 Internal Server Error`: Service error

---

#### Extract Job Links

Extract job URLs from a career page.

**Endpoint**: `POST /api/v1/extract-job-links`

**Request Body**:
```json
{
  "html": "<html>Career page HTML with job listings...</html>",
  "url": "https://example.com/careers"
}
```

**Response**:
```json
{
  "job_links": [
    {
      "url": "https://example.com/jobs/senior-engineer",
      "title": "Senior Engineer",
      "location": "San Francisco"
    },
    {
      "url": "https://example.com/jobs/product-manager",
      "title": "Product Manager",
      "location": "Remote"
    }
  ],
  "total_links": 15,
  "extraction_time_ms": 123
}
```

**Status Codes**:
- `200 OK`: Extraction successful
- `400 Bad Request`: Invalid HTML or URL
- `500 Internal Server Error`: Service error

---

### Crawler Service (Port 8002)

#### Batch Crawl URLs

Fetch HTML for multiple URLs in parallel.

**Endpoint**: `POST /crawl-batch`

**Request Body**:
```json
[
  "https://example.com/jobs/role-1",
  "https://example.com/jobs/role-2",
  "https://example.com/jobs/role-3"
]
```

**Response**:
```json
{
  "results": [
    {
      "url": "https://example.com/jobs/role-1",
      "html": "<html>...</html>",
      "status_code": 200,
      "success": true
    },
    {
      "url": "https://example.com/jobs/role-2",
      "html": "<html>...</html>",
      "status_code": 200,
      "success": true
    },
    {
      "url": "https://example.com/jobs/role-3",
      "error": "Timeout after 30s",
      "status_code": 0,
      "success": false
    }
  ],
  "total_requested": 3,
  "successful": 2,
  "failed": 1,
  "duration_ms": 4567
}
```

**Status Codes**:
- `200 OK`: Batch crawl completed (check individual results)
- `400 Bad Request`: Invalid URL list
- `500 Internal Server Error`: Service error

---

### Proxy Manager (Port 8003)

#### Get Proxy

Retrieve next available proxy from rotation pool.

**Endpoint**: `GET /proxy`

**Response**:
```json
{
  "host": "192.168.1.100",
  "port": 8080,
  "username": "user123",
  "password": "pass456",
  "protocol": "http",
  "type": "residential"
}
```

**Status Codes**:
- `200 OK`: Proxy available
- `503 Service Unavailable`: No healthy proxies available

---

#### Report Proxy Health

Report proxy health status after use.

**Endpoint**: `POST /proxy/health`

**Request Body**:
```json
{
  "proxy": "192.168.1.100:8080",
  "status": "success",
  "response_time_ms": 234
}
```

**Response**:
```json
{
  "acknowledged": true
}
```

**Status Codes**:
- `200 OK`: Health report recorded
- `400 Bad Request`: Invalid proxy identifier

---

#### Get Proxy Statistics

Retrieve proxy pool statistics.

**Endpoint**: `GET /proxy/stats`

**Response**:
```json
{
  "total_proxies": 50,
  "healthy_proxies": 45,
  "failed_proxies": 5,
  "avg_response_time_ms": 312,
  "requests_last_hour": 2456,
  "success_rate": 95.2
}
```

**Status Codes**:
- `200 OK`: Statistics retrieved

---

### OSINT Discovery (Port 8004)

#### Discover Career Pages

Find career page URLs for a company domain.

**Endpoint**: `POST /api/v1/discover/career-pages`

**Request Body**:
```json
{
  "domain": "linear.app"
}
```

**Response**:
```json
{
  "domain": "linear.app",
  "career_pages": [
    {
      "url": "https://linear.app/careers",
      "status": "active",
      "method": "common_path"
    },
    {
      "url": "https://jobs.ashbyhq.com/linear",
      "status": "active",
      "method": "ats_detection"
    }
  ],
  "total_found": 2,
  "discovery_time_ms": 1234
}
```

**Status Codes**:
- `200 OK`: Discovery completed
- `400 Bad Request`: Invalid domain
- `500 Internal Server Error`: Service error

---

#### Enumerate Subdomains

Find subdomains for a company domain.

**Endpoint**: `POST /api/v1/enumerate/subdomains`

**Request Body**:
```json
{
  "domain": "example.com",
  "methods": ["dns", "crt", "harvester"]
}
```

**Response**:
```json
{
  "domain": "example.com",
  "subdomains": [
    {
      "subdomain": "careers.example.com",
      "method": "dns",
      "is_job_related": true,
      "priority": 10
    },
    {
      "subdomain": "jobs.example.com",
      "method": "crt",
      "is_job_related": true,
      "priority": 10
    },
    {
      "subdomain": "api.example.com",
      "method": "dns",
      "is_job_related": false,
      "priority": 1
    }
  ],
  "total_found": 127,
  "job_related": 5,
  "enumeration_time_ms": 15234
}
```

**Status Codes**:
- `200 OK`: Enumeration completed
- `400 Bad Request`: Invalid domain or methods
- `500 Internal Server Error`: Service error

---

#### Detect ATS Platform

Identify the ATS platform for a job board URL.

**Endpoint**: `POST /api/v1/detect/ats`

**Request Body**:
```json
{
  "url": "https://jobs.ashbyhq.com/linear"
}
```

**Response**:
```json
{
  "url": "https://jobs.ashbyhq.com/linear",
  "ats_platform": "ashby",
  "confidence": 100,
  "detection_method": "url_pattern",
  "features": {
    "job_feed_url": "https://jobs.ashbyhq.com/linear/jobs.json",
    "api_available": true,
    "pagination_type": "offset"
  }
}
```

**Supported ATS Platforms**:
- Ashby
- Greenhouse
- Lever
- Workday
- Bamboo HR
- Jazz HR
- SmartRecruiters
- Taleo

**Status Codes**:
- `200 OK`: Detection completed
- `400 Bad Request`: Invalid URL
- `404 Not Found`: No ATS detected
- `500 Internal Server Error`: Service error

---

#### GitHub Company Discovery

Search GitHub for companies with career pages.

**Endpoint**: `POST /api/v1/discover/github`

**Request Body**:
```json
{
  "query": "tech startup hiring",
  "max_results": 50
}
```

**Response**:
```json
{
  "query": "tech startup hiring",
  "companies": [
    {
      "domain": "linear.app",
      "name": "Linear",
      "repository": "linear/linear",
      "career_url": "https://linear.app/careers",
      "source": "README.md"
    }
  ],
  "total_found": 42,
  "search_time_ms": 3456
}
```

**Status Codes**:
- `200 OK`: Search completed
- `400 Bad Request`: Invalid query
- `500 Internal Server Error`: GitHub API error

---

#### Google Dork Discovery

Search for job boards using Google Dorks.

**Endpoint**: `POST /api/v1/discover/dork`

**Request Body**:
```json
{
  "query": "site:greenhouse.io software engineer",
  "max_results": 100
}
```

**Response**:
```json
{
  "query": "site:greenhouse.io software engineer",
  "results": [
    {
      "url": "https://boards.greenhouse.io/linear/jobs/123",
      "title": "Senior Software Engineer - Linear",
      "snippet": "We're hiring a senior engineer to build...",
      "rank": 1
    }
  ],
  "total_found": 87,
  "search_time_ms": 2345
}
```

**Status Codes**:
- `200 OK`: Search completed
- `400 Bad Request`: Invalid query
- `500 Internal Server Error`: SerpAPI error

---

## Response Formats

### Success Response

All successful responses follow this structure:

```json
{
  "data": { },
  "metadata": {
    "timestamp": "2024-12-14T10:30:00Z",
    "took_ms": 123
  }
}
```

For list endpoints:
```json
{
  "results": [],
  "total": 100,
  "limit": 20,
  "offset": 0,
  "has_more": true
}
```

---

### Error Response

All error responses follow this structure:

```json
{
  "error": {
    "code": "INVALID_QUERY",
    "message": "Search query must not be empty",
    "details": {
      "field": "q",
      "reason": "required field missing"
    }
  },
  "timestamp": "2024-12-14T10:30:00Z"
}
```

---

## Error Handling

### HTTP Status Codes

- `200 OK`: Request successful
- `400 Bad Request`: Invalid request parameters
- `401 Unauthorized`: Authentication required (future)
- `403 Forbidden`: Insufficient permissions (future)
- `404 Not Found`: Resource not found
- `422 Unprocessable Entity`: Validation failed
- `429 Too Many Requests`: Rate limit exceeded (future)
- `500 Internal Server Error`: Server error
- `503 Service Unavailable`: Service temporarily unavailable

### Error Codes

| Code | Description |
|------|-------------|
| `INVALID_QUERY` | Search query is invalid or empty |
| `INVALID_PARAMETER` | Request parameter is invalid |
| `NOT_FOUND` | Requested resource doesn't exist |
| `PARSE_FAILED` | Job parsing failed |
| `CRAWL_FAILED` | Web crawling failed |
| `DATABASE_ERROR` | Database operation failed |
| `SERVICE_UNAVAILABLE` | External service unavailable |
| `RATE_LIMIT_EXCEEDED` | Too many requests (future) |

---

## Examples

### Example 1: Search for Remote Software Engineering Jobs

**Request**:
```bash
curl -X GET "http://localhost:3000/api/v1/search?q=software+engineer&remote=true&limit=5" \
  -H "Accept: application/json"
```

**Response**:
```json
{
  "hits": [
    {
      "id": "uuid-1",
      "title": "Senior Software Engineer",
      "company": "Linear",
      "location": "Remote",
      "remote": true,
      "salary_min": 150000,
      "salary_max": 200000,
      "url": "https://linear.app/careers/engineer"
    }
  ],
  "found": 85,
  "took_ms": 8
}
```

---

### Example 2: Parse Job HTML

**Request**:
```bash
curl -X POST "http://localhost:8001/api/v1/parse" \
  -H "Content-Type: application/json" \
  -d '{
    "html": "<html><body><h1>Senior Engineer</h1>...</body></html>",
    "url": "https://example.com/job/123"
  }'
```

**Response**:
```json
{
  "title": "Senior Engineer",
  "company": "Example Corp",
  "location": "San Francisco",
  "quality_score": 92
}
```

---

### Example 3: Discover Career Pages

**Request**:
```bash
curl -X POST "http://localhost:8004/api/v1/discover/career-pages" \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "vercel.com"
  }'
```

**Response**:
```json
{
  "domain": "vercel.com",
  "career_pages": [
    {
      "url": "https://vercel.com/careers",
      "status": "active"
    }
  ],
  "total_found": 1
}
```

---

### Example 4: Enumerate Subdomains

**Request**:
```bash
curl -X POST "http://localhost:8004/api/v1/enumerate/subdomains" \
  -H "Content-Type: application/json" \
  -d '{
    "domain": "stripe.com",
    "methods": ["dns", "crt"]
  }'
```

**Response**:
```json
{
  "domain": "stripe.com",
  "subdomains": [
    {
      "subdomain": "careers.stripe.com",
      "is_job_related": true,
      "priority": 10
    }
  ],
  "total_found": 234,
  "job_related": 2
}
```

---

## API Versioning

**Current Version**: v1

API versioning is included in the URL path: `/api/v1/...`

**Breaking Changes Policy**:
- New versions introduced for breaking changes
- Old versions supported for 6 months after deprecation
- Deprecation notices in response headers

---

## Contact & Support

**Project Repository**: [github.com/itisrohit/quiethire](https://github.com/itisrohit/quiethire)  
**Issues**: [github.com/itisrohit/quiethire/issues](https://github.com/itisrohit/quiethire/issues)  
**Contributing**: [CONTRIBUTING.md](CONTRIBUTING.md)  

**Maintainer**: Rohit Kumar ([@itisrohit](https://github.com/itisrohit))  

---

**Last Updated**: December 2024  
**API Version**: v1.0.0
