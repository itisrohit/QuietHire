// Package activities contains all Temporal workflow activities for job crawling
package activities

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"time"
)

// CrawlActivities contains all crawling-related activities
type CrawlActivities struct {
	// HTTP client for calling microservices
	HTTPClient *http.Client

	// Service URLs
	CrawlerURL string
	ParserURL  string
	OSINTUrl   string
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

// crawlSingleJob crawls a single job URL using the Python Crawler service
func (a *CrawlActivities) crawlSingleJob(ctx context.Context, url, platform string) (*JobData, error) {
	// Call the Python crawler service
	payload := map[string]interface{}{
		"urls": []string{url},
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.CrawlerURL+"/crawl/batch", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call crawler service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("crawler service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result struct {
		Results []struct {
			URL     string `json:"url"`
			HTML    string `json:"html"`
			Success bool   `json:"success"`
			Error   string `json:"error"`
		} `json:"results"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(result.Results) == 0 || !result.Results[0].Success {
		errMsg := "unknown error"
		if len(result.Results) > 0 {
			errMsg = result.Results[0].Error
		}
		return nil, fmt.Errorf("crawl failed: %s", errMsg)
	}

	// Generate a unique ID based on URL using SHA256
	hash := sha256.Sum256([]byte(url))
	id := hex.EncodeToString(hash[:])

	jobData := &JobData{
		ID:        id,
		URL:       url,
		Platform:  platform,
		HTML:      result.Results[0].HTML,
		CrawledAt: time.Now(),
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

// ParseJobActivity parses raw HTML into structured job data using Parser service
func (a *CrawlActivities) ParseJobActivity(ctx context.Context, jobID string, html string) (map[string]interface{}, error) {
	log.Printf("Parsing job: %s", jobID)

	// Call the Parser service
	payload := map[string]string{
		"html": html,
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", a.ParserURL+"/parse", bytes.NewBuffer(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := a.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to call parser service: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == 422 {
		return nil, fmt.Errorf("no structured data found in HTML (requires JSON-LD JobPosting schema)")
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("parser service returned status %d: %s", resp.StatusCode, string(bodyBytes))
	}

	var result map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return result, nil
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
