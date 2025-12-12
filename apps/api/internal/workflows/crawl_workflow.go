// Package workflows contains Temporal workflow definitions for job crawling orchestration
package workflows

import (
	"fmt"
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// JobCrawlInput defines the input for a job crawl workflow
type JobCrawlInput struct {
	Platform string   // "indeed", "linkedin", etc.
	URLs     []string // List of URLs to crawl
	MaxJobs  int      // Maximum jobs to crawl in this session
}

// JobCrawlResult defines the result of a crawl workflow
type JobCrawlResult struct {
	TotalCrawled int
	Successful   int
	Failed       int
	Duration     time.Duration
}

// CrawlCoordinatorWorkflow orchestrates the entire crawling process
// This is the main workflow that coordinates all crawlers
func CrawlCoordinatorWorkflow(ctx workflow.Context, input JobCrawlInput) (*JobCrawlResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting CrawlCoordinatorWorkflow", "platform", input.Platform)

	// Set workflow options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 10 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	startTime := workflow.Now(ctx)
	result := &JobCrawlResult{}

	// Activity 1: Discover job URLs from the platform
	var urls []string
	err := workflow.ExecuteActivity(ctx, "DiscoverJobURLs", input.Platform, input.URLs).Get(ctx, &urls)
	if err != nil {
		logger.Error("Failed to discover URLs", "error", err)
		return nil, err
	}

	logger.Info("Discovered URLs", "count", len(urls))

	// Activity 2: Crawl jobs in parallel (fan-out/fan-in pattern)
	// Split URLs into batches for parallel processing
	batchSize := 10
	totalBatches := (len(urls) + batchSize - 1) / batchSize

	var futures []workflow.Future
	for i := 0; i < totalBatches; i++ {
		start := i * batchSize
		end := start + batchSize
		if end > len(urls) {
			end = len(urls)
		}
		batch := urls[start:end]

		// Execute crawl activity for each batch in parallel
		future := workflow.ExecuteActivity(ctx, "CrawlJobBatch", batch, input.Platform)
		futures = append(futures, future)
	}

	// Wait for all batches to complete
	for _, future := range futures {
		var batchResult struct {
			Successful int
			Failed     int
		}
		err := future.Get(ctx, &batchResult)
		if err != nil {
			logger.Error("Batch crawl failed", "error", err)
			result.Failed++
		} else {
			result.Successful += batchResult.Successful
			result.Failed += batchResult.Failed
		}
	}

	result.TotalCrawled = result.Successful + result.Failed
	result.Duration = workflow.Now(ctx).Sub(startTime)

	logger.Info("CrawlCoordinatorWorkflow completed",
		"total", result.TotalCrawled,
		"successful", result.Successful,
		"failed", result.Failed,
		"duration", result.Duration)

	return result, nil
}

// ScheduledCrawlWorkflow runs on a schedule (e.g., every 6 hours)
// to continuously crawl all configured platforms
func ScheduledCrawlWorkflow(ctx workflow.Context) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting ScheduledCrawlWorkflow")

	// List of platforms to crawl
	platforms := []struct {
		Name string
		URLs []string
	}{
		{Name: "indeed", URLs: []string{"https://www.indeed.com/jobs?q=software+engineer"}},
		{Name: "linkedin", URLs: []string{"https://www.linkedin.com/jobs/search/?keywords=software%20engineer"}},
		// Add more platforms as needed
	}

	// Start a separate workflow for each platform
	for _, platform := range platforms {
		input := JobCrawlInput{
			Platform: platform.Name,
			URLs:     platform.URLs,
			MaxJobs:  1000,
		}

		// Start child workflow for each platform
		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: "crawl-" + platform.Name + "-" + workflow.Now(ctx).Format("20060102-150405"),
		})

		var result JobCrawlResult
		err := workflow.ExecuteChildWorkflow(childCtx, CrawlCoordinatorWorkflow, input).Get(childCtx, &result)
		if err != nil {
			logger.Error("Platform crawl failed", "platform", platform.Name, "error", err)
		} else {
			logger.Info("Platform crawl completed", "platform", platform.Name, "result", result)
		}
	}

	return nil
}

// CareerPageCrawlInput defines input for crawling a discovered career page
type CareerPageCrawlInput struct {
	URL         string
	CompanyName string
	CompanyID   int
}

// CareerPageCrawlResult defines the result of crawling a career page
type CareerPageCrawlResult struct {
	URL          string
	ErrorMessage string
	Duration     time.Duration
	JobsFound    int
	JobsStored   int
	Success      bool
}

// CareerPageCrawlWorkflow orchestrates crawling a single career page and extracting jobs
//
//nolint:gocyclo // workflow orchestrates multiple steps sequentially, complexity is inherent
func CareerPageCrawlWorkflow(ctx workflow.Context, input CareerPageCrawlInput) (*CareerPageCrawlResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting CareerPageCrawlWorkflow",
		"url", input.URL,
		"company", input.CompanyName)

	startTime := workflow.Now(ctx)
	result := &CareerPageCrawlResult{
		URL:     input.URL,
		Success: false,
	}

	// Set activity options with retries
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 5 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    30 * time.Second,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Crawl the career page to get HTML
	logger.Info("Step 1: Crawling career page", "url", input.URL)
	var crawlResult struct {
		URL     string
		HTML    string
		Error   string
		Success bool
	}

	crawlInput := map[string]interface{}{
		"URL":         input.URL,
		"CompanyName": input.CompanyName,
	}

	err := workflow.ExecuteActivity(ctx, "CrawlCareerPage", crawlInput).Get(ctx, &crawlResult)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to crawl page: %v", err)
		logger.Error("Crawl failed", "error", err)
		result.Duration = workflow.Now(ctx).Sub(startTime)
		return result, nil // Return result instead of error to mark workflow complete
	}

	if !crawlResult.Success {
		result.ErrorMessage = fmt.Sprintf("crawl unsuccessful: %s", crawlResult.Error)
		logger.Warn("Crawl unsuccessful", "error", crawlResult.Error)
		result.Duration = workflow.Now(ctx).Sub(startTime)
		return result, nil
	}

	logger.Info("Successfully crawled page", "html_size", len(crawlResult.HTML))

	// Step 2: Extract job links from the career page HTML
	logger.Info("Step 2: Extracting job links from HTML")
	var jobLinks []struct {
		URL   string `json:"url"`
		Title string `json:"title"`
	}
	err = workflow.ExecuteActivity(ctx, "ExtractJobLinks", input.URL, crawlResult.HTML).Get(ctx, &jobLinks)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to extract job links: %v", err)
		logger.Error("Extract job links failed", "error", err)
		result.Duration = workflow.Now(ctx).Sub(startTime)
		return result, nil
	}

	logger.Info("Extracted job links", "count", len(jobLinks))

	if len(jobLinks) == 0 {
		logger.Info("No job links found on career page")
		result.Success = true
		result.Duration = workflow.Now(ctx).Sub(startTime)
		return result, nil
	}

	// Step 3: Crawl each individual job page (limit to first 5 for MVP)
	maxJobsToCrawl := 5
	if len(jobLinks) > maxJobsToCrawl {
		logger.Info("Limiting job crawl", "total_links", len(jobLinks), "max", maxJobsToCrawl)
		jobLinks = jobLinks[:maxJobsToCrawl]
	}

	result.JobsFound = len(jobLinks)
	jobs := make([]map[string]interface{}, 0, len(jobLinks))

	for i, link := range jobLinks {
		logger.Info("Step 3: Crawling individual job page",
			"index", i+1,
			"total", len(jobLinks),
			"url", link.URL)

		// Crawl the job detail page
		var jobPageCrawlResult struct {
			URL     string
			HTML    string
			Error   string
			Success bool
		}

		jobCrawlInput := map[string]interface{}{
			"URL":         link.URL,
			"CompanyName": input.CompanyName,
		}

		err = workflow.ExecuteActivity(ctx, "CrawlCareerPage", jobCrawlInput).Get(ctx, &jobPageCrawlResult)
		if err != nil || !jobPageCrawlResult.Success {
			logger.Warn("Failed to crawl job page", "url", link.URL, "error", err)
			continue
		}

		logger.Info("Successfully crawled job page", "url", link.URL, "html_size", len(jobPageCrawlResult.HTML))

		// Parse the job page for JSON-LD structured data
		var parsedJob map[string]interface{}
		err = workflow.ExecuteActivity(ctx, "ParseJobPage", link.URL, jobPageCrawlResult.HTML, input.CompanyName).Get(ctx, &parsedJob)
		if err != nil {
			logger.Warn("Failed to parse job page", "url", link.URL, "error", err)
			continue
		}

		// If parsedJob is nil, it means no structured data was found (422 response)
		if parsedJob == nil {
			logger.Info("No structured data found on job page", "url", link.URL)
			continue
		}

		// Add source URL if not present
		if parsedJob["source_url"] == nil || parsedJob["source_url"] == "" {
			parsedJob["source_url"] = link.URL
		}
		if parsedJob["source_platform"] == nil || parsedJob["source_platform"] == "" {
			parsedJob["source_platform"] = "career_page"
		}

		jobs = append(jobs, parsedJob)
		logger.Info("Successfully parsed job", "title", parsedJob["title"], "company", parsedJob["company"])
	}

	logger.Info("Parsed jobs from individual pages", "count", len(jobs))

	if len(jobs) == 0 {
		logger.Info("No jobs with structured data found")
		result.Success = true
		result.Duration = workflow.Now(ctx).Sub(startTime)
		return result, nil
	}

	// Step 4: Store jobs in ClickHouse
	logger.Info("Step 4: Storing jobs in ClickHouse", "count", len(jobs))
	var storedCount int
	err = workflow.ExecuteActivity(ctx, "StoreJobsInClickHouse", jobs, input.URL).Get(ctx, &storedCount)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("failed to store jobs: %v", err)
		logger.Error("Storage failed", "error", err)
		result.Duration = workflow.Now(ctx).Sub(startTime)
		return result, nil
	}

	result.JobsStored = storedCount
	result.Success = true
	result.Duration = workflow.Now(ctx).Sub(startTime)

	logger.Info("CareerPageCrawlWorkflow completed",
		"jobs_found", result.JobsFound,
		"jobs_stored", result.JobsStored,
		"duration", result.Duration)

	return result, nil
}
