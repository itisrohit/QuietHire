// Package main provides a web crawler using Playwright for QuietHire
package main

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/joho/godotenv"
	"github.com/playwright-community/playwright-go"
)

type Crawler struct {
	pw      *playwright.Playwright
	browser playwright.Browser
}

type CrawlResult struct {
	CrawledAt time.Time
	URL       string
	HTML      string
	Title     string
	Error     string
	Success   bool
}

func NewCrawler() (*Crawler, error) {
	// Install playwright browsers if not already installed
	err := playwright.Install(&playwright.RunOptions{
		Verbose: false,
	})
	if err != nil {
		return nil, fmt.Errorf("could not install playwright: %w", err)
	}

	pw, err := playwright.Run()
	if err != nil {
		return nil, fmt.Errorf("could not start playwright: %w", err)
	}

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args: []string{
			"--disable-blink-features=AutomationControlled",
			"--disable-dev-shm-usage",
			"--no-sandbox",
		},
	})
	if err != nil {
		if stopErr := pw.Stop(); stopErr != nil {
			log.Printf("Error stopping playwright: %v", stopErr)
		}
		return nil, fmt.Errorf("could not launch browser: %w", err)
	}

	return &Crawler{
		pw:      pw,
		browser: browser,
	}, nil
}

func (c *Crawler) Close() error {
	if c.browser != nil {
		if err := c.browser.Close(); err != nil {
			return err
		}
	}
	if c.pw != nil {
		if err := c.pw.Stop(); err != nil {
			return err
		}
	}
	return nil
}

func (c *Crawler) CrawlURL(_ context.Context, url string) (*CrawlResult, error) {
	result := &CrawlResult{
		URL:       url,
		CrawledAt: time.Now(),
	}

	// Create a new page
	page, err := c.browser.NewPage()
	if err != nil {
		result.Error = fmt.Sprintf("could not create page: %v", err)
		return result, err
	}
	defer func() {
		if closeErr := page.Close(); closeErr != nil {
			log.Printf("Error closing page: %v", closeErr)
		}
	}()

	// Set user agent to look like a real browser
	if headersErr := page.SetExtraHTTPHeaders(map[string]string{
		"User-Agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"Accept":     "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,*/*;q=0.8",
	}); headersErr != nil {
		result.Error = fmt.Sprintf("could not set headers: %v", headersErr)
		return result, headersErr
	}

	// Navigate to the URL
	_, err = page.Goto(url, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
		Timeout:   playwright.Float(30000),
	})
	if err != nil {
		result.Error = fmt.Sprintf("could not navigate: %v", err)
		return result, err
	}

	// Wait for the page to load
	time.Sleep(2 * time.Second)

	// Get the page title
	title, err := page.Title()
	if err == nil {
		result.Title = title
	}

	// Get the HTML content
	html, err := page.Content()
	if err != nil {
		result.Error = fmt.Sprintf("could not get content: %v", err)
		return result, err
	}

	result.HTML = html
	result.Success = true

	log.Printf("Successfully crawled: %s (title: %s)", url, title)

	return result, nil
}

// CrawlBatch crawls multiple URLs sequentially
func (c *Crawler) CrawlBatch(ctx context.Context, urls []string, delayMs int) []*CrawlResult {
	results := make([]*CrawlResult, 0, len(urls))

	for _, url := range urls {
		// Check if context is canceled
		select {
		case <-ctx.Done():
			log.Println("Context canceled, stopping batch crawl")
			return results
		default:
		}

		result, err := c.CrawlURL(ctx, url)
		if err != nil {
			log.Printf("Error crawling %s: %v", url, err)
		}
		results = append(results, result)

		// Rate limiting delay between requests
		if delayMs > 0 {
			time.Sleep(time.Duration(delayMs) * time.Millisecond)
		}
	}

	return results
}

// GenerateJobID generates a unique ID for a job based on URL using SHA256
func GenerateJobID(url string) string {
	hash := sha256.Sum256([]byte(url))
	return hex.EncodeToString(hash[:])
}

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	log.Println("🕷️  QuietHire Go Crawler starting...")

	// Create crawler
	crawler, err := NewCrawler()
	if err != nil {
		log.Fatalf("Failed to create crawler: %v", err)
	}
	defer func() {
		if closeErr := crawler.Close(); closeErr != nil {
			log.Printf("Error closing crawler: %v", closeErr)
		}
	}()

	log.Println("✅ Crawler initialized with Playwright")

	// Example: Crawl a test URL
	testURL := os.Getenv("TEST_CRAWL_URL")
	if testURL == "" {
		testURL = "https://example.com"
	}

	ctx := context.Background()
	result, err := crawler.CrawlURL(ctx, testURL)
	if err != nil {
		log.Printf("Crawl failed: %v", err)
	} else {
		log.Printf("Crawl successful!")
		log.Printf("  URL: %s", result.URL)
		log.Printf("  Title: %s", result.Title)
		log.Printf("  HTML length: %d bytes", len(result.HTML))
		log.Printf("  Job ID: %s", GenerateJobID(result.URL))
	}

	log.Println("✅ Crawler test complete")
}
