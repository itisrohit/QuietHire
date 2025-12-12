"""
Python Stealth Crawler Service
Uses undetected-playwright for anti-detection crawling of job sites
"""

import asyncio
import logging
from collections.abc import AsyncGenerator
from contextlib import asynccontextmanager

import uvicorn
from fastapi import FastAPI, HTTPException
from playwright.async_api import Browser, Page, ProxySettings, async_playwright
from pydantic import BaseModel, field_validator

# Configure logging
logging.basicConfig(
    level=logging.INFO, format="%(asctime)s - %(name)s - %(levelname)s - %(message)s"
)
logger = logging.getLogger(__name__)

# Global browser instance
browser: Browser | None = None


class CrawlRequest(BaseModel):
    url: str
    wait_for_selector: str | None = None
    wait_time: int | None = 2000  # milliseconds
    use_stealth: bool = True
    proxy_url: str | None = None

    @field_validator("url")
    @classmethod
    def validate_url(cls, v: str) -> str:
        """Validate URL format"""
        if not v.startswith(("http://", "https://")):
            error_msg = "URL must start with http:// or https://"
            raise ValueError(error_msg)
        return v


class CrawlResponse(BaseModel):
    url: str
    html: str
    status: int
    success: bool
    error: str | None = None


@asynccontextmanager
async def lifespan(_app: FastAPI) -> AsyncGenerator[None, None]:
    """Initialize and cleanup browser on startup/shutdown"""
    global browser  # noqa: PLW0603
    logger.info("Starting browser...")

    try:
        playwright = await async_playwright().start()
        browser = await playwright.chromium.launch(
            headless=True,
            args=[
                "--no-sandbox",
                "--disable-setuid-sandbox",
                "--disable-dev-shm-usage",
                "--disable-accelerated-2d-canvas",
                "--disable-gpu",
                "--window-size=1920x1080",
                "--disable-blink-features=AutomationControlled",
            ],
        )
        logger.info("Browser started successfully")
        yield
    finally:
        if browser:
            logger.info("Closing browser...")
            await browser.close()
            logger.info("Browser closed")


app = FastAPI(
    title="Stealth Crawler Service",
    description="Anti-detection web crawler using undetected-playwright",
    version="1.0.0",
    lifespan=lifespan,
)


async def apply_stealth(page: Page) -> None:
    """Apply stealth techniques to avoid detection"""
    # Additional stealth scripts (Malenia requires BrowserContext, so we handle manually)
    await page.add_init_script("""
        // Override navigator.webdriver
        Object.defineProperty(navigator, 'webdriver', {
            get: () => undefined
        });

        // Override chrome detection
        window.chrome = {
            runtime: {}
        };

        // Override permissions
        const originalQuery = window.navigator.permissions.query;
        window.navigator.permissions.query = (parameters) => (
            parameters.name === 'notifications' ?
                Promise.resolve({ state: Notification.permission }) :
                originalQuery(parameters)
        );

        // Add plugins
        Object.defineProperty(navigator, 'plugins', {
            get: () => [1, 2, 3, 4, 5]
        });

        // Add languages
        Object.defineProperty(navigator, 'languages', {
            get: () => ['en-US', 'en']
        });
    """)


@app.get("/health")
async def health_check() -> dict[str, str | bool]:
    """Health check endpoint"""
    return {
        "status": "healthy",
        "service": "crawler-python",
        "browser_ready": browser is not None,
    }


@app.post("/crawl", response_model=CrawlResponse)
async def crawl_url(request: CrawlRequest) -> CrawlResponse:
    """
    Crawl a URL with stealth techniques

    Args:
        request: CrawlRequest with URL and options

    Returns:
        CrawlResponse with HTML content and metadata
    """
    if not browser:
        raise HTTPException(status_code=503, detail="Browser not initialized")

    context = None
    page = None

    try:
        logger.info("Crawling URL: %s", request.url)

        # Create browser context with proxy if provided
        proxy: ProxySettings | None = None
        if request.proxy_url:
            proxy = {"server": request.proxy_url}

        context = await browser.new_context(
            viewport={"width": 1920, "height": 1080},
            user_agent="Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
            locale="en-US",
            timezone_id="America/New_York",
            proxy=proxy,
        )
        page = await context.new_page()

        # Apply stealth if requested
        if request.use_stealth:
            await apply_stealth(page)

        # Navigate to URL
        response = await page.goto(
            str(request.url), wait_until="domcontentloaded", timeout=30000
        )

        if not response:
            raise HTTPException(status_code=500, detail="No response from page")  # noqa: TRY301

        # Wait for specific selector if provided
        if request.wait_for_selector:
            await page.wait_for_selector(request.wait_for_selector, timeout=10000)

        # Additional wait time for JS to render
        if request.wait_time:
            await asyncio.sleep(request.wait_time / 1000)

        # Get HTML content
        html = await page.content()
        status = response.status

        logger.info("Successfully crawled %s - Status: %d", request.url, status)

        return CrawlResponse(
            url=str(request.url),
            html=html,
            status=status,
            success=status < 400,
            error=None,
        )

    except Exception as e:
        logger.exception("Error crawling %s", request.url)
        return CrawlResponse(
            url=str(request.url), html="", status=500, success=False, error=str(e)
        )

    finally:
        if page:
            await page.close()
        if context:
            await context.close()


@app.post("/crawl-batch")
async def crawl_batch(urls: list[str], use_stealth: bool = True) -> list[CrawlResponse]:
    """
    Crawl multiple URLs in parallel

    Args:
        urls: List of URLs to crawl
        use_stealth: Whether to use stealth techniques

    Returns:
        List of CrawlResponse objects
    """
    tasks = []
    for url in urls:
        request = CrawlRequest(url=url, use_stealth=use_stealth)
        tasks.append(crawl_url(request))

    results = await asyncio.gather(*tasks, return_exceptions=True)

    # Convert exceptions to error responses
    responses: list[CrawlResponse] = []
    for i, result in enumerate(results):
        if isinstance(result, Exception):
            responses.append(
                CrawlResponse(
                    url=urls[i], html="", status=500, success=False, error=str(result)
                )
            )
        elif isinstance(result, CrawlResponse):
            responses.append(result)

    return responses


def main() -> None:
    """Run the FastAPI application"""
    uvicorn.run("main:app", host="0.0.0.0", port=8000, log_level="info", reload=False)


if __name__ == "__main__":
    main()
