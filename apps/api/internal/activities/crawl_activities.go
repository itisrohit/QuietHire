// Package activities contains all Temporal workflow activities for job crawling
package activities

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log"
	"time"
)

// CrawlActivities contains all crawling-related activities
type CrawlActivities struct {
	// Dependencies will be injected here (DB, HTTP client, etc.)
}

// JobData represents a crawled job
type JobData struct {
	CrawledAt   time.Time
	ID          string
	Title       string
	Company     string
	Description string
	Location    string
	URL         string
	Platform    string
	HTML        string
}

// DiscoverJobURLs discovers job listing URLs from a platform
func (a *CrawlActivities) DiscoverJobURLs(_ context.Context, platform string, seedURLs []string) ([]string, error) {
	log.Printf("Discovering URLs for platform: %s", platform)

	// TODO: Implement actual URL discovery logic
	// This is a placeholder that returns the seed URLs
	// In production, this would:
	// 1. Fetch the search page
	// 2. Parse pagination
	// 3. Extract individual job URLs
	// 4. Return list of job detail page URLs

	discoveredURLs := make([]string, 0)

	// Placeholder: simulate discovering 50 job URLs per seed URL
	for _, seedURL := range seedURLs {
		for i := 1; i <= 50; i++ {
			jobURL := fmt.Sprintf("%s&job=%d", seedURL, i)
			discoveredURLs = append(discoveredURLs, jobURL)
		}
	}

	log.Printf("Discovered %d URLs for platform %s", len(discoveredURLs), platform)
	return discoveredURLs, nil
}

// CrawlJobBatch crawls a batch of job URLs
func (a *CrawlActivities) CrawlJobBatch(ctx context.Context, urls []string, platform string) (map[string]interface{}, error) {
	log.Printf("Crawling batch of %d URLs for platform: %s", len(urls), platform)

	successful := 0
	failed := 0

	for _, url := range urls {
		// Check if context is canceled
		select {
		case <-ctx.Done():
			return map[string]interface{}{
				"Successful": successful,
				"Failed":     failed,
			}, ctx.Err()
		default:
		}

		// Crawl individual job
		jobData, err := a.crawlSingleJob(ctx, url, platform)
		if err != nil {
			log.Printf("Failed to crawl %s: %v", url, err)
			failed++
			continue
		}

		// Store raw HTML and job data
		err = a.storeJobData(ctx, jobData)
		if err != nil {
			log.Printf("Failed to store job data for %s: %v", url, err)
			failed++
			continue
		}

		successful++

		// Rate limiting - sleep between requests
		time.Sleep(1 * time.Second)
	}

	log.Printf("Batch complete: %d successful, %d failed", successful, failed)

	return map[string]interface{}{
		"Successful": successful,
		"Failed":     failed,
	}, nil
}

// crawlSingleJob crawls a single job URL
//
//nolint:unparam // Placeholder function - will return errors in production
func (a *CrawlActivities) crawlSingleJob(_ context.Context, url, platform string) (*JobData, error) {
	// TODO: Implement actual crawling logic with playwright-go
	// This is a placeholder

	// Generate a unique ID based on URL using SHA256
	hash := sha256.Sum256([]byte(url))
	id := hex.EncodeToString(hash[:])

	jobData := &JobData{
		ID:          id,
		Title:       "Software Engineer (Placeholder)",
		Company:     "Example Company",
		Description: "This is a placeholder job description",
		Location:    "Remote",
		URL:         url,
		Platform:    platform,
		HTML:        "<html>Placeholder HTML</html>",
		CrawledAt:   time.Now(),
	}

	return jobData, nil
}

// storeJobData stores crawled job data in ClickHouse
//
//nolint:unparam // Placeholder function - will return errors in production
func (a *CrawlActivities) storeJobData(_ context.Context, job *JobData) error {
	// TODO: Implement actual storage logic
	// This should:
	// 1. Store raw HTML in jobs_raw_html table
	// 2. Queue the job for parsing
	// 3. Update crawl_history

	log.Printf("Storing job data: %s - %s", job.ID, job.Title)
	return nil
}

// ParseJobActivity parses raw HTML into structured job data
func (a *CrawlActivities) ParseJobActivity(_ context.Context, jobID string, _ string) (map[string]interface{}, error) {
	log.Printf("Parsing job: %s", jobID)

	// TODO: Call the Parser service API
	// This should send HTML to the parser service and get back structured data

	// Placeholder response
	return map[string]interface{}{
		"title":       "Parsed Job Title",
		"company":     "Parsed Company",
		"description": "Parsed description",
		"location":    "Parsed location",
	}, nil
}

// ScoreJobActivity calculates authenticity score for a job
func (a *CrawlActivities) ScoreJobActivity(_ context.Context, jobData map[string]interface{}) (int, error) {
	log.Printf("Scoring job: %v", jobData["title"])

	// TODO: Call the RealScore service API
	// This should analyze the job and return a score 0-100

	// Placeholder: return a random score between 70-95
	return 85, nil
}

// ExtractHiringManagerActivity extracts hiring manager info
func (a *CrawlActivities) ExtractHiringManagerActivity(_ context.Context, jobData map[string]interface{}) (map[string]string, error) {
	log.Printf("Extracting hiring manager for: %v", jobData["title"])

	// TODO: Call the Manager Extractor service API

	// Placeholder response
	return map[string]string{
		"name":  "John Doe",
		"email": "john.doe@example.com",
	}, nil
}
