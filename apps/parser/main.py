import json
import logging
import os
import re
from urllib.parse import urljoin, urlparse

from bs4 import BeautifulSoup
from fastapi import FastAPI, HTTPException
from pydantic import BaseModel

# Configure logging
logging.basicConfig(
    level=os.getenv("LOG_LEVEL", "INFO").upper(),
    format="%(asctime)s - %(name)s - %(levelname)s - %(message)s",
)
logger = logging.getLogger(__name__)

app = FastAPI(title="Parser Service", version="1.0.0")


class ParseRequest(BaseModel):
    url: str
    html: str


class ExtractJobLinksRequest(BaseModel):
    url: str
    html: str


class JobLink(BaseModel):
    url: str
    title: str | None = None


class ExtractJobLinksResponse(BaseModel):
    job_links: list[JobLink]
    total_count: int


class ParsedJob(BaseModel):
    title: str
    description: str
    company: str
    location: str | None = None
    salary: str | None = None
    job_type: str | None = None
    experience_level: str | None = None
    requirements: list[str] = []
    benefits: list[str] = []


class JobParser:
    """Job posting parser using structured data extraction (JSON-LD, schema.org) and LLM fallback"""

    def __init__(self):
        pass

    def extract_text_from_html(self, html: str) -> str:
        """Extract clean text content from HTML for LLM processing"""
        soup = BeautifulSoup(html, "html.parser")
        
        # Remove script and style elements
        for script in soup(["script", "style", "nav", "header", "footer"]):
            script.decompose()
        
        # Get text
        text = soup.get_text()
        
        # Break into lines and remove leading/trailing space
        lines = (line.strip() for line in text.splitlines())
        # Break multi-headlines into a line each
        chunks = (phrase.strip() for line in lines for phrase in line.split("  "))
        # Drop blank lines
        text = '\n'.join(chunk for chunk in chunks if chunk)
        
        # Limit to first 8000 characters to avoid token limits
        return text[:8000]

    def _is_valid_job_title(self, title: str) -> bool:
        """Check if title looks like a valid job posting"""
        if not title or len(title) < 3 or len(title) > 150:
            return False
        
        # Filter out bad patterns
        bad_patterns = [
            "logo", "jobs", "career", "culture", "benefit", "principle",
            "your job", "find your", "we're hiring", "join", "everyone at",
            "you're not", "saved jobs", "dream job"
        ]
        title_lower = title.lower()
        
        for pattern in bad_patterns:
            if pattern in title_lower:
                return False
        
        # Must have some word characters (not just symbols/numbers)
        if not re.search(r'[a-zA-Z]{3,}', title):
            return False
            
        return True
    
    def _extract_company_from_url(self, url: str) -> str:
        """Extract company name from URL domain"""
        try:
            parsed = urlparse(url)
            domain = parsed.netloc or parsed.path
            # Remove common prefixes/suffixes
            domain = domain.replace('www.', '').replace('careers.', '').replace('jobs.', '')
            # Get main domain name
            parts = domain.split('.')
            if len(parts) >= 2:
                company = parts[0]
                # Capitalize first letter
                return company.capitalize()
            return domain.capitalize()
        except:
            return "Unknown Company"

    def extract_with_simple_heuristics(self, html: str, url: str) -> dict | None:
        """
        Extract job data using simple heuristics (for when LLM is not available).
        This is a fallback that works reasonably well for most job pages.
        """
        soup = BeautifulSoup(html, "html.parser")
        
        # Try to find job title (prioritize h1 in main content)
        title = None
        for selector in ["h1[class*='job']", "h1[class*='title']", "h1", "[class*='job-title']", "[class*='position-title']"]:
            elem = soup.select_one(selector)
            if elem:
                title = elem.get_text(strip=True)
                if self._is_valid_job_title(title):
                    break
        
        # If no valid title found, this is likely not a job page
        if not title or not self._is_valid_job_title(title):
            return None
        
        # Try to find company name from multiple sources
        company = None
        for selector in ["[class*='company-name']", "[class*='company']", "[class*='organization']", "[class*='employer']", "meta[property='og:site_name']"]:
            elem = soup.select_one(selector)
            if elem:
                if elem.name == 'meta':
                    company = elem.get('content', '')
                else:
                    company = elem.get_text(strip=True)
                if company and len(company) > 2 and len(company) < 100 and not any(x in company.lower() for x in ['logo', 'jobs', 'career']):
                    break
        
        # Fallback: extract company from URL domain
        if not company:
            company = self._extract_company_from_url(url)
        
        # Try to find location (more specific selectors)
        location = None
        for selector in ["[class*='job-location']", "[class*='location']", "[data-test*='location']", "[class*='city']", "[class*='office']"]:
            elem = soup.select_one(selector)
            if elem:
                loc_text = elem.get_text(strip=True)
                # Filter out noise
                if loc_text and len(loc_text) > 2 and len(loc_text) < 100:
                    # Check if it looks like a location (contains city/country keywords or has state/country codes)
                    if re.search(r'\b(remote|hybrid|onsite|usa|uk|ca|us|europe|americas|asia)\b', loc_text.lower()) or ',' in loc_text:
                        location = loc_text
                        break
        
        # Try to find description (prioritize job-specific containers)
        description = None
        for selector in ["[class*='job-description']", "[class*='description']", "[class*='job-content']", "[class*='content']", "main", "article"]:
            elem = soup.select_one(selector)
            if elem:
                description = elem.get_text(strip=True)
                if description and len(description) > 100:  # Increased minimum length
                    # Limit description length
                    description = description[:2000]
                    break
        
        # Quality check: must have meaningful description
        if not description or len(description) < 100:
            return None
        
        return {
            "title": title,
            "description": description,
            "company": company,
            "location": location,
            "salary": None,
            "job_type": None,
        }

    def extract_structured_data(self, html: str) -> dict | None:
        """
        Try to extract structured data (JSON-LD, schema.org) from HTML.
        Many job sites include structured data that's easier to parse.
        """
        soup = BeautifulSoup(html, "html.parser")

        # Look for JSON-LD
        json_ld_tags = soup.find_all("script", type="application/ld+json")

        for tag in json_ld_tags:
            try:
                data = json.loads(tag.string)

                # Check if it's a JobPosting schema
                if isinstance(data, dict):
                    if data.get("@type") == "JobPosting":
                        return self._parse_job_posting_schema(data)

                    # Sometimes it's wrapped in a graph
                    if "@graph" in data:
                        for item in data["@graph"]:
                            if item.get("@type") == "JobPosting":
                                return self._parse_job_posting_schema(item)

            except json.JSONDecodeError:
                continue

        return None

    def _parse_job_posting_schema(self, schema: dict) -> dict:
        """Parse schema.org JobPosting structured data"""
        hiring_org = schema.get("hiringOrganization", {})
        location_data = schema.get("jobLocation", {})
        salary_data = schema.get("baseSalary", {})

        # Extract location
        location = None
        if isinstance(location_data, dict):
            address = location_data.get("address", {})
            if isinstance(address, dict):
                city = address.get("addressLocality", "")
                state = address.get("addressRegion", "")
                location = f"{city}, {state}".strip(", ")
            else:
                location = location_data.get("name")

        # Extract salary
        salary = None
        if isinstance(salary_data, dict):
            value = salary_data.get("value", {})
            if isinstance(value, dict):
                min_val = value.get("minValue")
                max_val = value.get("maxValue")
                currency = value.get("currency", "USD")
                if min_val and max_val:
                    salary = f"${min_val:,} - ${max_val:,} {currency}"

        # Extract job type (can be string or array)
        job_type = schema.get("employmentType", "")
        if isinstance(job_type, list):
            job_type = job_type[0] if job_type else ""
        
        return {
            "title": schema.get("title", ""),
            "description": schema.get("description", ""),
            "company": hiring_org.get("name", "")
            if isinstance(hiring_org, dict)
            else str(hiring_org),
            "location": location,
            "salary": salary,
            "job_type": job_type,
            "date_posted": schema.get("datePosted"),
            "valid_through": schema.get("validThrough"),
        }

    async def parse(self, html: str, url: str) -> ParsedJob:
        """
        Parse job posting using multiple strategies:
        1. Try JSON-LD structured data (fast, accurate)
        2. Fall back to simple heuristics (reliable, works for most sites)
        """
        # Strategy 1: Try JSON-LD structured data
        structured_data = self.extract_structured_data(html)
        if structured_data:
            logger.info("Successfully extracted structured data for %s", url)
            return ParsedJob(
                title=structured_data.get("title", "Unknown Title"),
                description=structured_data.get("description", ""),
                company=structured_data.get("company", "Unknown Company"),
                location=structured_data.get("location"),
                salary=structured_data.get("salary"),
                job_type=structured_data.get("job_type"),
                experience_level=None,
                requirements=[],
                benefits=[],
            )
        
        # Strategy 2: Try simple heuristics
        heuristic_data = self.extract_with_simple_heuristics(html, url)
        if heuristic_data:
            logger.info("Successfully extracted job data using heuristics for %s", url)
            return ParsedJob(
                title=heuristic_data.get("title", "Unknown Title"),
                description=heuristic_data.get("description", ""),
                company=heuristic_data.get("company", "Unknown Company"),
                location=heuristic_data.get("location"),
                salary=heuristic_data.get("salary"),
                job_type=heuristic_data.get("job_type"),
                experience_level=None,
                requirements=[],
                benefits=[],
            )
        
        # No data found
        raise ValueError(
            f"Could not extract job data from {url} using any available method"
        )


# Initialize parser
job_parser = JobParser()


@app.get("/health")
async def health_check() -> dict[str, str]:
    """Health check endpoint"""
    return {
        "status": "healthy",
        "service": "parser",
        "version": "2.0.0",
        "parser_type": "multi_strategy (JSON-LD + heuristics)",
    }


@app.post("/api/v1/parse", response_model=ParsedJob)
async def parse_job_posting(request: ParseRequest) -> ParsedJob:
    """
    Parse a job posting HTML and extract structured data.
    Tries multiple strategies: JSON-LD schema.org first, then heuristics.
    """
    try:
        logger.info("Parsing job from URL: %s", request.url)
        result = await job_parser.parse(request.html, request.url)
        logger.info("Successfully parsed job: %s at %s", result.title, result.company)
        return result

    except ValueError as e:
        logger.warning("Could not parse job: %s", e)
        raise HTTPException(
            status_code=422, detail=f"Could not extract job data: {e!s}"
        ) from e
    except Exception as e:
        logger.exception("Error parsing job posting")
        raise HTTPException(
            status_code=500, detail=f"Failed to parse job posting: {e!s}"
        ) from e


@app.post("/api/v1/extract-job-links", response_model=ExtractJobLinksResponse)
async def extract_job_links(request: ExtractJobLinksRequest) -> ExtractJobLinksResponse:
    """
    Extract individual job posting links from a career page listing.
    Uses heuristics to identify job links (e.g., /jobs/, /careers/, job titles in href).
    """
    try:
        logger.info("Extracting job links from URL: %s", request.url)
        soup = BeautifulSoup(request.html, "html.parser")
        base_url = f"{urlparse(request.url).scheme}://{urlparse(request.url).netloc}"
        
        job_links = []
        seen_urls = set()
        
        # Find all links
        for link in soup.find_all("a", href=True):
            href = link.get("href")
            if not href:
                continue
            
            # Make URL absolute
            absolute_url = urljoin(request.url, href)
            
            # Skip if already seen
            if absolute_url in seen_urls:
                continue
            
            # Skip non-http links (mailto, tel, javascript, etc.)
            if not absolute_url.startswith(("http://", "https://")):
                continue
            
            # Skip external domains (only keep same domain)
            if urlparse(absolute_url).netloc != urlparse(request.url).netloc:
                continue
            
            # Heuristics to identify job posting URLs
            url_lower = absolute_url.lower()
            is_job_link = False
            
            # Skip generic career pages
            skip_patterns = [
                "/benefits", "/culture", "/life-at-", "/about", "/team",
                "/principles", "/values", "/diversity", "/inclusion",
                "/extraordinary", "/saved-jobs", "/all-jobs", "/accessibility",
                "/interview-process", "/disciplines", "/university"
            ]
            
            should_skip = any(pattern in url_lower for pattern in skip_patterns)
            if should_skip:
                continue
            
            # Check if URL contains job-related paths
            job_patterns = [
                "/job/", "/jobs/", "/career/", "/careers/", 
                "/position/", "/positions/", "/opening/", "/openings/",
                "/vacancy/", "/vacancies/", "/role/", "/roles/"
            ]
            
            for pattern in job_patterns:
                if pattern in url_lower:
                    # Avoid listing pages (they usually have few path segments)
                    path_parts = urlparse(absolute_url).path.strip("/").split("/")
                    # Job detail pages typically have ID or slug after /jobs/
                    # Must have either: UUID, numeric ID, or at least 3 path segments
                    has_id = any(char.isdigit() for char in absolute_url) or len(path_parts) >= 3
                    if has_id:
                        is_job_link = True
                        break
            
            if is_job_link:
                # Extract title from link text
                title = link.get_text(strip=True)
                # Must have reasonable title length
                if title and len(title) >= 5 and len(title) <= 200:
                    job_links.append(JobLink(url=absolute_url, title=title))
                    seen_urls.add(absolute_url)
        
        logger.info("Found %d job links from %s", len(job_links), request.url)
        return ExtractJobLinksResponse(job_links=job_links, total_count=len(job_links))
    
    except Exception as e:
        logger.exception("Error extracting job links")
        raise HTTPException(
            status_code=500, detail=f"Failed to extract job links: {e!s}"
        ) from e


def main() -> None:
    pass


if __name__ == "__main__":
    main()


def main() -> None:
    pass


if __name__ == "__main__":
    main()
