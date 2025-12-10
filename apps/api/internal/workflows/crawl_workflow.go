// Package workflows contains Temporal workflow definitions for job crawling orchestration
package workflows

import (
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
