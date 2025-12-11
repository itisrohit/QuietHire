import json
import logging
import os

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
    """Job posting parser using structured data extraction (JSON-LD, schema.org)"""

    def __init__(self):
        pass

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

        return {
            "title": schema.get("title", ""),
            "description": schema.get("description", ""),
            "company": hiring_org.get("name", "")
            if isinstance(hiring_org, dict)
            else str(hiring_org),
            "location": location,
            "salary": salary,
            "job_type": schema.get("employmentType", ""),
            "date_posted": schema.get("datePosted"),
            "valid_through": schema.get("validThrough"),
        }

    async def parse(self, html: str, url: str) -> ParsedJob:
        """
        Parse job posting using structured data extraction (JSON-LD, schema.org).
        This is fast, accurate, and works for any site that implements JobPosting schema.
        """
        # Extract structured data (JSON-LD)
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

        # If no structured data found, raise error
        # This forces us to target sites that implement proper schema.org markup
        raise ValueError(
            f"No structured data (JSON-LD JobPosting schema) found at {url}"
        )


# Initialize parser
job_parser = JobParser()


@app.get("/health")
async def health_check() -> dict[str, str]:
    """Health check endpoint"""
    return {
        "status": "healthy",
        "service": "parser",
        "version": "1.0.0",
        "parser_type": "structured_data_only",
    }


@app.post("/api/v1/parse", response_model=ParsedJob)
async def parse_job_posting(request: ParseRequest) -> ParsedJob:
    """
    Parse a job posting HTML and extract structured data using JSON-LD schema.org.
    Only works on sites that implement proper JobPosting structured data markup.
    """
    try:
        logger.info("Parsing job from URL: %s", request.url)
        result = await job_parser.parse(request.html, request.url)
        logger.info("Successfully parsed job: %s at %s", result.title, result.company)
        return result

    except ValueError as e:
        logger.warning("No structured data found: %s", e)
        raise HTTPException(
            status_code=422, detail=f"No structured data found: {e!s}"
        ) from e
    except Exception as e:
        logger.exception("Error parsing job posting")
        raise HTTPException(
            status_code=500, detail=f"Failed to parse job posting: {e!s}"
        ) from e


def main() -> None:
    pass


if __name__ == "__main__":
    main()


def main() -> None:
    pass


if __name__ == "__main__":
    main()
