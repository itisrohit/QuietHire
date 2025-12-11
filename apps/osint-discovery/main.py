import asyncio
import logging
import os
from typing import Any

import httpx
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel, Field

# Configure logging
logging.basicConfig(
    level=os.getenv("LOG_LEVEL", "INFO").upper(),
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

app = FastAPI(title="OSINT Discovery Service", version="1.0.0")


# ============================================================================
# Data Models
# ============================================================================


class CompanyDiscoveryRequest(BaseModel):
    source: str = Field(..., description="Source to search: github, crunchbase, manual")
    query: str = Field(..., description="Search query or company name")
    limit: int = Field(100, description="Maximum results to return")


class Company(BaseModel):
    name: str
    domain: str | None = None
    description: str | None = None
    industry: str | None = None
    source: str
    metadata: dict[str, Any] = Field(default_factory=dict)


class CompanyDiscoveryResponse(BaseModel):
    companies: list[Company]
    total_found: int
    source: str


class CareerPageRequest(BaseModel):
    domain: str = Field(..., description="Company domain to search")
    company_name: str | None = None


class CareerPage(BaseModel):
    url: str
    page_type: str  # careers, jobs, opportunities, etc.
    confidence: float  # 0.0 to 1.0
    ats_platform: str | None = None
    discovered_via: str  # subdomain, google, sitemap, etc.


class CareerPageResponse(BaseModel):
    career_pages: list[CareerPage]
    domain: str
    total_found: int


class ATSDetectionRequest(BaseModel):
    url: str


class ATSDetectionResponse(BaseModel):
    url: str
    is_ats: bool
    platform: str | None = None  # greenhouse, ashby, lever, workday, etc.
    confidence: float
    job_listing_urls: list[str] = Field(default_factory=list)


class SubdomainEnumerationRequest(BaseModel):
    domain: str
    methods: list[str] = Field(
        ["dns", "crt"],
        description="Methods to use: dns, crt (Certificate Transparency)",
    )


class Subdomain(BaseModel):
    subdomain: str
    method: str  # dns, crt
    is_job_related: bool  # True if contains careers/jobs/hiring/etc.


class SubdomainEnumerationResponse(BaseModel):
    subdomains: list[Subdomain]
    domain: str
    total_found: int


class GoogleDorkRequest(BaseModel):
    query: str = Field(..., description="Google dork query")
    num_results: int = Field(100, description="Number of results to fetch")


class DorkResult(BaseModel):
    url: str
    title: str
    snippet: str
    rank: int


class GoogleDorkResponse(BaseModel):
    results: list[DorkResult]
    query: str
    total_found: int


# ============================================================================
# OSINT Discovery Modules
# ============================================================================


class CompanyFinder:
    """Discover companies from various sources"""

    def __init__(self):
        self.github_token = os.getenv("GITHUB_TOKEN")
        self.crunchbase_key = os.getenv("CRUNCHBASE_API_KEY")

    async def discover_from_github(self, query: str, limit: int = 100) -> list[Company]:
        """
        Find companies by searching GitHub organizations and repositories.
        Looks for orgs with career pages, HIRING.md, or job-related repos.
        """
        companies = []

        if not self.github_token:
            logger.warning("GITHUB_TOKEN not set, skipping GitHub discovery")
            return companies

        headers = {
            "Authorization": f"token {self.github_token}",
            "Accept": "application/vnd.github.v3+json",
        }

        async with httpx.AsyncClient() as client:
            # Search for organizations with "careers" or "jobs" in name/description
            search_queries = [
                f"{query} in:name,description type:org",
                f"{query} hiring in:name,description type:org",
            ]

            for search_query in search_queries:
                try:
                    response = await client.get(
                        "https://api.github.com/search/users",
                        headers=headers,
                        params={"q": search_query, "per_page": min(limit, 100)},
                        timeout=30.0,
                    )

                    if response.status_code == 200:
                        data = response.json()
                        for org in data.get("items", [])[:limit]:
                            # Fetch org details
                            org_response = await client.get(
                                org["url"], headers=headers, timeout=30.0
                            )
                            if org_response.status_code == 200:
                                org_data = org_response.json()
                                companies.append(
                                    Company(
                                        name=org_data.get("name") or org_data["login"],
                                        domain=org_data.get("blog"),
                                        description=org_data.get("bio"),
                                        source="github",
                                        metadata={
                                            "github_url": org_data["html_url"],
                                            "repos": org_data.get("public_repos", 0),
                                            "followers": org_data.get("followers", 0),
                                        },
                                    )
                                )

                            if len(companies) >= limit:
                                break

                    await asyncio.sleep(0.5)  # Rate limiting

                except Exception as e:
                    logger.error("Error searching GitHub: %s", e)

        return companies[:limit]

    async def discover_from_crunchbase(
        self, query: str, limit: int = 100
    ) -> list[Company]:
        """
        Find companies from Crunchbase API.
        Requires CRUNCHBASE_API_KEY.
        """
        companies = []

        if not self.crunchbase_key:
            logger.warning("CRUNCHBASE_API_KEY not set, skipping Crunchbase discovery")
            return companies

        # TODO: Implement Crunchbase API integration
        # Crunchbase has rate limits and requires paid API access
        logger.info("Crunchbase integration not yet implemented")
        return companies

    async def discover_manual(self, company_name: str) -> list[Company]:
        """
        Manually add a company for discovery.
        Used when user provides specific company names.
        """
        return [
            Company(
                name=company_name,
                domain=None,  # Will be resolved later
                source="manual",
                metadata={"added_manually": True},
            )
        ]


class CareerPageFinder:
    """Find career pages for companies"""

    JOB_KEYWORDS = [
        "careers",
        "jobs",
        "hiring",
        "opportunities",
        "positions",
        "openings",
        "join",
        "work-with-us",
        "job-openings",
    ]

    async def find_career_pages(
        self, domain: str, company_name: str | None = None
    ) -> list[CareerPage]:
        """
        Find career pages for a given domain using multiple techniques:
        1. Subdomain enumeration (careers.*, jobs.*)
        2. Common career page paths (/careers, /jobs, etc.)
        3. Sitemap analysis
        4. Google dorking
        """
        pages = []

        # 1. Check common subdomain patterns
        subdomain_pages = await self._check_job_subdomains(domain)
        pages.extend(subdomain_pages)

        # 2. Check common career page paths
        path_pages = await self._check_career_paths(domain)
        pages.extend(path_pages)

        return pages

    async def _check_job_subdomains(self, domain: str) -> list[CareerPage]:
        """Check for job-related subdomains"""
        pages = []
        subdomains = [f"{keyword}.{domain}" for keyword in self.JOB_KEYWORDS]

        async with httpx.AsyncClient(follow_redirects=True) as client:
            for subdomain in subdomains:
                try:
                    response = await client.get(
                        f"https://{subdomain}", timeout=10.0, follow_redirects=True
                    )
                    if response.status_code == 200:
                        # Check if it's actually a career page
                        content = response.text.lower()
                        job_indicator_count = sum(
                            1 for kw in self.JOB_KEYWORDS if kw in content
                        )

                        if job_indicator_count >= 2:
                            keyword = subdomain.split(".")[0]
                            pages.append(
                                CareerPage(
                                    url=str(response.url),
                                    page_type=keyword,
                                    confidence=min(
                                        0.5 + (job_indicator_count * 0.1), 1.0
                                    ),
                                    discovered_via="subdomain",
                                )
                            )
                except Exception as e:
                    logger.debug("Subdomain %s not accessible: %s", subdomain, e)

        return pages

    async def _check_career_paths(self, domain: str) -> list[CareerPage]:
        """Check common career page URL paths"""
        pages = []
        paths = [f"/{keyword}" for keyword in self.JOB_KEYWORDS]

        async with httpx.AsyncClient(follow_redirects=True) as client:
            for path in paths:
                url = f"https://{domain}{path}"
                try:
                    response = await client.get(
                        url, timeout=10.0, follow_redirects=True
                    )
                    if response.status_code == 200:
                        content = response.text.lower()
                        job_indicator_count = sum(
                            1 for kw in self.JOB_KEYWORDS if kw in content
                        )

                        if job_indicator_count >= 2:
                            keyword = path.strip("/")
                            pages.append(
                                CareerPage(
                                    url=str(response.url),
                                    page_type=keyword,
                                    confidence=min(
                                        0.6 + (job_indicator_count * 0.1), 1.0
                                    ),
                                    discovered_via="path",
                                )
                            )
                except Exception as e:
                    logger.debug("Path %s not accessible: %s", url, e)

        return pages


class ATSDetector:
    """Detect Applicant Tracking Systems (ATS) platforms"""

    ATS_PATTERNS = {
        "greenhouse": {
            "domains": ["greenhouse.io", "boards.greenhouse.io"],
            "url_patterns": ["/jobs/"],
            "content_indicators": ["greenhouse", "powered by greenhouse"],
        },
        "ashby": {
            "domains": ["jobs.ashbyhq.com"],
            "url_patterns": ["/jobs/"],
            "content_indicators": ["ashby", "ashbyhq"],
        },
        "lever": {
            "domains": ["jobs.lever.co"],
            "url_patterns": ["/jobs/"],
            "content_indicators": ["lever", "jobs.lever.co"],
        },
        "workday": {
            "domains": ["myworkdayjobs.com"],
            "url_patterns": ["/jobs/"],
            "content_indicators": ["workday", "myworkdayjobs"],
        },
        "indeed": {
            "domains": ["indeed.com"],
            "url_patterns": ["/viewjob", "/cmp/", "/jobs"],
            "content_indicators": ["indeed"],
        },
        "linkedin": {
            "domains": ["linkedin.com"],
            "url_patterns": ["/jobs/view/", "/jobs/collections/"],
            "content_indicators": ["linkedin jobs"],
        },
        "bamboohr": {
            "domains": ["bamboohr.com"],
            "url_patterns": ["/jobs/"],
            "content_indicators": ["bamboohr"],
        },
        "icims": {
            "domains": ["icims.com"],
            "url_patterns": ["/jobs/"],
            "content_indicators": ["icims"],
        },
    }

    async def detect_ats(self, url: str) -> ATSDetectionResponse:
        """
        Detect if a URL is using an ATS platform and identify which one.
        """
        url_lower = url.lower()

        # Check URL patterns
        for platform, patterns in self.ATS_PATTERNS.items():
            # Check domain match
            if any(domain in url_lower for domain in patterns["domains"]):
                return ATSDetectionResponse(
                    url=url,
                    is_ats=True,
                    platform=platform,
                    confidence=0.95,
                    job_listing_urls=[],
                )

            # Check URL path patterns
            if any(pattern in url_lower for pattern in patterns["url_patterns"]):
                return ATSDetectionResponse(
                    url=url,
                    is_ats=True,
                    platform=platform,
                    confidence=0.75,
                    job_listing_urls=[],
                )

        # If no pattern match, fetch page and check content
        try:
            async with httpx.AsyncClient() as client:
                response = await client.get(url, timeout=15.0, follow_redirects=True)
                if response.status_code == 200:
                    content = response.text.lower()

                    for platform, patterns in self.ATS_PATTERNS.items():
                        if any(
                            indicator in content
                            for indicator in patterns["content_indicators"]
                        ):
                            return ATSDetectionResponse(
                                url=url,
                                is_ats=True,
                                platform=platform,
                                confidence=0.6,
                                job_listing_urls=[],
                            )
        except Exception as e:
            logger.error("Error fetching URL for ATS detection: %s", e)

        return ATSDetectionResponse(
            url=url, is_ats=False, platform=None, confidence=0.0, job_listing_urls=[]
        )


class SubdomainEnumerator:
    """Enumerate subdomains for job discovery"""

    JOB_RELATED_KEYWORDS = [
        "careers",
        "jobs",
        "hiring",
        "opportunities",
        "positions",
        "openings",
        "join",
        "work",
        "talent",
        "recruit",
    ]

    async def enumerate_subdomains(
        self, domain: str, methods: list[str]
    ) -> list[Subdomain]:
        """
        Enumerate subdomains using multiple methods:
        - dns: DNS enumeration
        - crt: Certificate Transparency logs
        """
        subdomains = []

        if "crt" in methods:
            crt_subs = await self._enumerate_crt_sh(domain)
            subdomains.extend(crt_subs)

        if "dns" in methods:
            dns_subs = await self._enumerate_dns(domain)
            subdomains.extend(dns_subs)

        # Deduplicate
        seen = set()
        unique_subdomains = []
        for sub in subdomains:
            if sub.subdomain not in seen:
                seen.add(sub.subdomain)
                unique_subdomains.append(sub)

        return unique_subdomains

    async def _enumerate_crt_sh(self, domain: str) -> list[Subdomain]:
        """Enumerate subdomains using Certificate Transparency logs (crt.sh)"""
        subdomains = []

        try:
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    "https://crt.sh/",
                    params={"q": f"%.{domain}", "output": "json"},
                    timeout=30.0,
                )

                if response.status_code == 200:
                    data = response.json()
                    for entry in data:
                        name_value = entry.get("name_value", "")
                        # Handle wildcard and multiple subdomains
                        for subdomain in name_value.split("\n"):
                            subdomain = subdomain.strip().replace("*.", "")
                            if subdomain and subdomain.endswith(domain):
                                is_job_related = any(
                                    kw in subdomain.lower()
                                    for kw in self.JOB_RELATED_KEYWORDS
                                )
                                subdomains.append(
                                    Subdomain(
                                        subdomain=subdomain,
                                        method="crt",
                                        is_job_related=is_job_related,
                                    )
                                )
        except Exception as e:
            logger.error("Error enumerating crt.sh for %s: %s", domain, e)

        return subdomains

    async def _enumerate_dns(self, domain: str) -> list[Subdomain]:
        """Enumerate subdomains using DNS queries for common job-related subdomains"""
        subdomains = []

        # Try common job-related subdomains
        candidates = [f"{kw}.{domain}" for kw in self.JOB_RELATED_KEYWORDS]

        async with httpx.AsyncClient() as client:
            for subdomain in candidates:
                try:
                    # Quick check if subdomain resolves
                    response = await client.get(
                        f"https://{subdomain}", timeout=5.0, follow_redirects=False
                    )
                    # If we get any response, subdomain exists
                    if response.status_code < 500:
                        subdomains.append(
                            Subdomain(
                                subdomain=subdomain, method="dns", is_job_related=True
                            )
                        )
                except Exception:
                    pass  # Subdomain doesn't exist or not accessible

        return subdomains


class GoogleDorker:
    """Execute Google dork queries for job discovery"""

    def __init__(self):
        self.serpapi_key = os.getenv("SERPAPI_API_KEY")

    async def execute_dork(
        self, query: str, num_results: int = 100
    ) -> list[DorkResult]:
        """
        Execute a Google dork query using SerpAPI.
        """
        if not self.serpapi_key:
            logger.warning("SERPAPI_API_KEY not set, cannot execute dork queries")
            return []

        results = []

        try:
            # SerpAPI has a Python client, but we'll use direct HTTP for simplicity
            async with httpx.AsyncClient() as client:
                response = await client.get(
                    "https://serpapi.com/search",
                    params={
                        "q": query,
                        "api_key": self.serpapi_key,
                        "num": min(num_results, 100),  # SerpAPI max per request
                        "engine": "google",
                    },
                    timeout=30.0,
                )

                if response.status_code == 200:
                    data = response.json()
                    organic_results = data.get("organic_results", [])

                    for i, result in enumerate(organic_results[:num_results]):
                        results.append(
                            DorkResult(
                                url=result.get("link", ""),
                                title=result.get("title", ""),
                                snippet=result.get("snippet", ""),
                                rank=i + 1,
                            )
                        )
        except Exception as e:
            logger.error("Error executing dork query: %s", e)

        return results

    def generate_job_dorks(self, keyword: str | None = None) -> list[str]:
        """
        Generate useful Google dork queries for job discovery.
        """
        base_dorks = [
            'site:greenhouse.io OR site:lever.co OR site:ashbyhq.com "software engineer"',
            'inurl:careers OR inurl:jobs "we are hiring"',
            'intitle:"job openings" OR intitle:"careers" site:*.com',
            '"apply now" (inurl:jobs OR inurl:careers)',
        ]

        if keyword:
            base_dorks.append(f'"{keyword}" (inurl:jobs OR inurl:careers)')
            base_dorks.append(f'site:greenhouse.io "{keyword}"')

        return base_dorks


# ============================================================================
# Initialize Services
# ============================================================================

company_finder = CompanyFinder()
career_page_finder = CareerPageFinder()
ats_detector = ATSDetector()
subdomain_enumerator = SubdomainEnumerator()
google_dorker = GoogleDorker()


# ============================================================================
# API Endpoints
# ============================================================================


@app.get("/health")
async def health_check() -> dict[str, str]:
    """Health check endpoint"""
    return {"status": "healthy", "service": "osint-discovery", "version": "1.0.0"}


@app.post("/api/v1/discover/companies", response_model=CompanyDiscoveryResponse)
async def discover_companies(
    request: CompanyDiscoveryRequest,
) -> CompanyDiscoveryResponse:
    """
    Discover companies from various sources (GitHub, Crunchbase, etc.)
    """
    try:
        logger.info(
            "Discovering companies: source=%s, query=%s", request.source, request.query
        )

        companies = []

        if request.source == "github":
            companies = await company_finder.discover_from_github(
                request.query, request.limit
            )
        elif request.source == "crunchbase":
            companies = await company_finder.discover_from_crunchbase(
                request.query, request.limit
            )
        elif request.source == "manual":
            companies = await company_finder.discover_manual(request.query)
        else:
            raise HTTPException(
                status_code=400,
                detail=f"Unknown source: {request.source}. Supported: github, crunchbase, manual",
            )

        return CompanyDiscoveryResponse(
            companies=companies, total_found=len(companies), source=request.source
        )

    except HTTPException:
        raise
    except Exception as e:
        logger.exception("Error discovering companies")
        raise HTTPException(
            status_code=500, detail=f"Failed to discover companies: {e!s}"
        ) from e


@app.post("/api/v1/discover/career-pages", response_model=CareerPageResponse)
async def discover_career_pages(request: CareerPageRequest) -> CareerPageResponse:
    """
    Find career pages for a given company domain.
    """
    try:
        logger.info("Finding career pages for domain: %s", request.domain)

        pages = await career_page_finder.find_career_pages(
            request.domain, request.company_name
        )

        return CareerPageResponse(
            career_pages=pages, domain=request.domain, total_found=len(pages)
        )

    except Exception as e:
        logger.exception("Error finding career pages")
        raise HTTPException(
            status_code=500, detail=f"Failed to find career pages: {e!s}"
        ) from e


@app.post("/api/v1/detect/ats", response_model=ATSDetectionResponse)
async def detect_ats(request: ATSDetectionRequest) -> ATSDetectionResponse:
    """
    Detect if a URL is using an ATS platform and identify which one.
    """
    try:
        logger.info("Detecting ATS for URL: %s", request.url)

        result = await ats_detector.detect_ats(request.url)
        return result

    except Exception as e:
        logger.exception("Error detecting ATS")
        raise HTTPException(
            status_code=500, detail=f"Failed to detect ATS: {e!s}"
        ) from e


@app.post("/api/v1/enumerate/subdomains", response_model=SubdomainEnumerationResponse)
async def enumerate_subdomains(
    request: SubdomainEnumerationRequest,
) -> SubdomainEnumerationResponse:
    """
    Enumerate job-related subdomains for a domain.
    """
    try:
        logger.info("Enumerating subdomains for domain: %s", request.domain)

        subdomains = await subdomain_enumerator.enumerate_subdomains(
            request.domain, request.methods
        )

        return SubdomainEnumerationResponse(
            subdomains=subdomains, domain=request.domain, total_found=len(subdomains)
        )

    except Exception as e:
        logger.exception("Error enumerating subdomains")
        raise HTTPException(
            status_code=500, detail=f"Failed to enumerate subdomains: {e!s}"
        ) from e


@app.post("/api/v1/search/dork", response_model=GoogleDorkResponse)
async def execute_google_dork(request: GoogleDorkRequest) -> GoogleDorkResponse:
    """
    Execute a Google dork query for job discovery.
    """
    try:
        logger.info("Executing dork query: %s", request.query)

        results = await google_dorker.execute_dork(request.query, request.num_results)

        return GoogleDorkResponse(
            results=results, query=request.query, total_found=len(results)
        )

    except Exception as e:
        logger.exception("Error executing dork query")
        raise HTTPException(
            status_code=500, detail=f"Failed to execute dork query: {e!s}"
        ) from e


@app.get("/api/v1/dorks/templates")
async def get_dork_templates(keyword: str | None = None) -> dict[str, list[str]]:
    """
    Get pre-built Google dork query templates for job discovery.
    """
    dorks = google_dorker.generate_job_dorks(keyword)
    return {"dorks": dorks, "keyword": keyword}


def main() -> None:
    pass


if __name__ == "__main__":
    main()
