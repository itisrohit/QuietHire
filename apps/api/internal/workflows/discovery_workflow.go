// Package workflows contains Temporal workflow definitions for OSINT discovery orchestration
package workflows

import (
	"time"

	"go.temporal.io/sdk/temporal"
	"go.temporal.io/sdk/workflow"
)

// DiscoveryInput defines the input for discovery workflows
type DiscoveryInput struct {
	Query      string   // Search query or company name
	Sources    []string // Sources to use: github, google_dork, subdomains, etc.
	MaxResults int      // Maximum results to process
}

// DiscoveryResult defines the result of a discovery workflow
type DiscoveryResult struct {
	ATSPlatforms     map[string]int // 8 bytes - pointer (map header)
	Duration         time.Duration  // 8 bytes - int64
	CompaniesFound   int            // 8 bytes - int
	CareerPagesFound int            // 8 bytes - int
	TotalURLsQueued  int            // 8 bytes - int
}

// CompanyDiscoveryWorkflow discovers companies and their career pages
// This is the main OSINT discovery workflow
//
//nolint:gocyclo // workflow orchestrates complex discovery pipeline, complexity is acceptable
func CompanyDiscoveryWorkflow(ctx workflow.Context, input DiscoveryInput) (*DiscoveryResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting CompanyDiscoveryWorkflow", "query", input.Query, "sources", input.Sources)

	// Set workflow options
	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 15 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    2 * time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	startTime := workflow.Now(ctx)
	result := &DiscoveryResult{
		ATSPlatforms: make(map[string]int),
	}

	// Step 1: Discover companies from all sources in parallel
	var futures []workflow.Future

	for _, source := range input.Sources {
		switch source {
		case "github":
			future := workflow.ExecuteActivity(ctx, "DiscoverCompaniesFromGitHub", input.Query, input.MaxResults)
			futures = append(futures, future)
		case "google_dork":
			future := workflow.ExecuteActivity(ctx, "DiscoverCompaniesFromGoogleDorks", input.Query, input.MaxResults)
			futures = append(futures, future)
		case "manual":
			future := workflow.ExecuteActivity(ctx, "AddCompanyManually", input.Query)
			futures = append(futures, future)
		}
	}

	// Collect all discovered companies
	var allCompanies []CompanyInfo
	for _, future := range futures {
		var companies []CompanyInfo
		err := future.Get(ctx, &companies)
		if err != nil {
			logger.Error("Failed to discover companies from source", "error", err)
			continue
		}
		allCompanies = append(allCompanies, companies...)
	}

	result.CompaniesFound = len(allCompanies)
	logger.Info("Total companies discovered", "count", result.CompaniesFound)

	// Step 2: For each company, discover career pages (parallel processing)
	careerPageFutures := make([]workflow.Future, 0, len(allCompanies)*2)
	for _, company := range allCompanies {
		future := workflow.ExecuteActivity(ctx, "DiscoverCareerPages", company.Domain, company.Name)
		careerPageFutures = append(careerPageFutures, future)

		// Also enumerate subdomains
		future = workflow.ExecuteActivity(ctx, "EnumerateSubdomains", company.Domain)
		careerPageFutures = append(careerPageFutures, future)
	}

	// Collect all discovered career pages
	var allCareerPages []CareerPageInfo
	for _, future := range careerPageFutures {
		var pages []CareerPageInfo
		err := future.Get(ctx, &pages)
		if err != nil {
			logger.Error("Failed to discover career pages", "error", err)
			continue
		}
		allCareerPages = append(allCareerPages, pages...)
	}

	result.CareerPagesFound = len(allCareerPages)
	logger.Info("Total career pages discovered", "count", result.CareerPagesFound)

	// Step 3: Detect ATS platforms for each career page (parallel)
	atsDetectionFutures := make([]workflow.Future, 0, len(allCareerPages))
	for _, page := range allCareerPages {
		future := workflow.ExecuteActivity(ctx, "DetectATS", page.URL)
		atsDetectionFutures = append(atsDetectionFutures, future)
	}

	// Collect ATS detection results
	for _, future := range atsDetectionFutures {
		var atsInfo ATSInfo
		err := future.Get(ctx, &atsInfo)
		if err != nil {
			logger.Error("Failed to detect ATS", "error", err)
			continue
		}

		if atsInfo.IsATS && atsInfo.Platform != "" {
			result.ATSPlatforms[atsInfo.Platform]++
		}
	}

	// Step 4: Queue all discovered URLs for crawling (store in DB)
	queueFuture := workflow.ExecuteActivity(ctx, "QueueURLsForCrawling", allCareerPages)

	var queued int
	err := queueFuture.Get(ctx, &queued)
	if err != nil {
		logger.Error("Failed to queue URLs", "error", err)
	}
	result.TotalURLsQueued = queued

	// Step 5: Trigger CareerPageCrawlWorkflow for each discovered career page
	logger.Info("Triggering crawl workflows for career pages", "count", len(allCareerPages))

	// Create a map to group pages by company
	pagesByCompany := make(map[string][]CareerPageInfo)
	for _, page := range allCareerPages {
		pagesByCompany[page.Domain] = append(pagesByCompany[page.Domain], page)
	}

	var crawlFutures []workflow.ChildWorkflowFuture
	for domain, pages := range pagesByCompany {
		// Find the company name for this domain
		companyName := domain
		for _, company := range allCompanies {
			if company.Domain == domain {
				companyName = company.Name
				break
			}
		}

		// Start a crawl workflow for each career page
		for _, page := range pages {
			crawlInput := CareerPageCrawlInput{
				URL:         page.URL,
				CompanyName: companyName,
			}

			childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
				WorkflowID: "career-crawl-" + workflow.Now(ctx).Format("20060102150405") + "-" + page.Domain,
			})

			future := workflow.ExecuteChildWorkflow(childCtx, CareerPageCrawlWorkflow, crawlInput)
			crawlFutures = append(crawlFutures, future)
		}
	}

	// Wait for all crawl workflows to complete (don't block the main workflow)
	// We'll just count successes
	crawlSuccesses := 0
	for _, future := range crawlFutures {
		var crawlResult CareerPageCrawlResult
		if getErr := future.Get(ctx, &crawlResult); getErr != nil {
			logger.Error("Crawl workflow failed", "error", getErr)
		} else if crawlResult.Success {
			crawlSuccesses++
		}
	}

	logger.Info("Crawl workflows completed", "success", crawlSuccesses, "total", len(crawlFutures))

	result.Duration = workflow.Now(ctx).Sub(startTime)

	logger.Info("CompanyDiscoveryWorkflow completed",
		"companies", result.CompaniesFound,
		"career_pages", result.CareerPagesFound,
		"urls_queued", result.TotalURLsQueued,
		"crawls_triggered", len(crawlFutures),
		"crawls_successful", crawlSuccesses,
		"duration", result.Duration)

	return result, nil
}

// ContinuousDiscoveryInput defines input for continuous discovery
type ContinuousDiscoveryInput struct {
	GitHubQuery        string // Query for GitHub discovery
	DorkQuery          string // Query for Dork discovery
	StaleThresholdDays int    // Days since last crawl to consider stale
	MaxNewCompanies    int    // Max new companies to discover per run
	RunGitHubDiscovery bool   // Whether to run GitHub discovery
	RunDorkDiscovery   bool   // Whether to run Google Dork discovery
}

// ContinuousDiscoveryWorkflow runs on a cron schedule to continuously discover companies and jobs
// This workflow is meant to be scheduled and run periodically
func ContinuousDiscoveryWorkflow(ctx workflow.Context, input ContinuousDiscoveryInput) error {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting ContinuousDiscoveryWorkflow", "days_threshold", input.StaleThresholdDays)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 30 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    5 * time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	// Step 1: Find stale companies that need re-crawling
	var staleCompanies []CompanyInfo
	err := workflow.ExecuteActivity(ctx, "GetStaleCompanies", input.StaleThresholdDays).Get(ctx, &staleCompanies)
	if err != nil {
		logger.Error("Failed to get stale companies", "error", err)
		return err
	}

	logger.Info("Found stale companies to re-crawl", "count", len(staleCompanies))

	// Step 2: Trigger discovery workflow for each stale company
	discoveryFutures := make([]workflow.ChildWorkflowFuture, 0, len(staleCompanies)+2)
	for _, company := range staleCompanies {
		discoveryInput := DiscoveryInput{
			Query:      company.Domain,
			Sources:    []string{"manual"},
			MaxResults: 10,
		}

		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: "stale-company-" + company.Domain + "-" + workflow.Now(ctx).Format("20060102-150405"),
		})

		future := workflow.ExecuteChildWorkflow(childCtx, CompanyDiscoveryWorkflow, discoveryInput)
		discoveryFutures = append(discoveryFutures, future)

		// Update last_crawled_at timestamp
		_ = workflow.ExecuteActivity(ctx, "UpdateCompanyLastCrawled", company.Domain)
	}

	// Step 3: Run discovery strategies based on configuration
	if input.RunGitHubDiscovery {
		logger.Info("Running GitHub discovery strategy")
		githubInput := DiscoveryInput{
			Query:      input.GitHubQuery,
			Sources:    []string{"github"},
			MaxResults: input.MaxNewCompanies,
		}

		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: "github-discovery-" + workflow.Now(ctx).Format("20060102-150405"),
		})

		future := workflow.ExecuteChildWorkflow(childCtx, CompanyDiscoveryWorkflow, githubInput)
		discoveryFutures = append(discoveryFutures, future)
	}

	if input.RunDorkDiscovery {
		logger.Info("Running Google Dork discovery strategy")
		dorkInput := DiscoveryInput{
			Query:      input.DorkQuery,
			Sources:    []string{"google_dork"},
			MaxResults: input.MaxNewCompanies,
		}

		childCtx := workflow.WithChildOptions(ctx, workflow.ChildWorkflowOptions{
			WorkflowID: "dork-discovery-" + workflow.Now(ctx).Format("20060102-150405"),
		})

		future := workflow.ExecuteChildWorkflow(childCtx, CompanyDiscoveryWorkflow, dorkInput)
		discoveryFutures = append(discoveryFutures, future)
	}

	// Wait for all discovery workflows to complete
	totalCompanies := 0
	totalURLs := 0
	for _, future := range discoveryFutures {
		var result DiscoveryResult
		if err := future.Get(ctx, &result); err != nil {
			logger.Error("Discovery workflow failed", "error", err)
			continue
		}
		totalCompanies += result.CompaniesFound
		totalURLs += result.TotalURLsQueued
	}

	logger.Info("ContinuousDiscoveryWorkflow completed",
		"stale_companies_processed", len(staleCompanies),
		"new_companies_found", totalCompanies,
		"total_urls_queued", totalURLs)

	return nil
}

// GoogleDorkDiscoveryWorkflow specifically for Google dork-based discovery
// Executes pre-configured dork queries to find job postings
func GoogleDorkDiscoveryWorkflow(ctx workflow.Context, keyword string) (*DiscoveryResult, error) {
	logger := workflow.GetLogger(ctx)
	logger.Info("Starting GoogleDorkDiscoveryWorkflow", "keyword", keyword)

	ao := workflow.ActivityOptions{
		StartToCloseTimeout: 20 * time.Minute,
		RetryPolicy: &temporal.RetryPolicy{
			InitialInterval:    2 * time.Second,
			BackoffCoefficient: 2.0,
			MaximumInterval:    3 * time.Minute,
			MaximumAttempts:    3,
		},
	}
	ctx = workflow.WithActivityOptions(ctx, ao)

	startTime := workflow.Now(ctx)
	result := &DiscoveryResult{
		ATSPlatforms: make(map[string]int),
	}

	// Step 1: Generate dork queries
	var dorkQueries []string
	err := workflow.ExecuteActivity(ctx, "GenerateDorkQueries", keyword).Get(ctx, &dorkQueries)
	if err != nil {
		logger.Error("Failed to generate dork queries", "error", err)
		return nil, err
	}

	logger.Info("Generated dork queries", "count", len(dorkQueries))

	// Step 2: Execute each dork query in parallel
	futures := make([]workflow.Future, 0, len(dorkQueries))
	for _, query := range dorkQueries {
		future := workflow.ExecuteActivity(ctx, "ExecuteDorkQuery", query, 100)
		futures = append(futures, future)
	}

	// Step 3: Collect all results
	var allURLs []string
	for _, future := range futures {
		var urls []string
		if getErr := future.Get(ctx, &urls); getErr != nil {
			logger.Error("Dork query failed", "error", getErr)
			continue
		}
		allURLs = append(allURLs, urls...)
	}

	logger.Info("Total URLs found from dorks", "count", len(allURLs))

	// Step 4: Detect ATS and extract domains
	detectionFutures := make([]workflow.Future, 0, len(allURLs))
	for _, url := range allURLs {
		future := workflow.ExecuteActivity(ctx, "DetectATSAndExtractDomain", url)
		detectionFutures = append(detectionFutures, future)
	}

	// Step 5: Collect detection results
	discoveredPages := make([]CareerPageInfo, 0, len(allURLs))
	for _, future := range detectionFutures {
		var pageInfo CareerPageInfo
		if getErr := future.Get(ctx, &pageInfo); getErr != nil {
			continue
		}
		discoveredPages = append(discoveredPages, pageInfo)

		if pageInfo.ATSPlatform != "" {
			result.ATSPlatforms[pageInfo.ATSPlatform]++
		}
	}

	result.CareerPagesFound = len(discoveredPages)

	// Step 6: Queue for crawling
	var queued int
	err = workflow.ExecuteActivity(ctx, "QueueURLsForCrawling", discoveredPages).Get(ctx, &queued)
	if err != nil {
		logger.Error("Failed to queue URLs", "error", err)
	}
	result.TotalURLsQueued = queued

	result.Duration = workflow.Now(ctx).Sub(startTime)

	logger.Info("GoogleDorkDiscoveryWorkflow completed",
		"career_pages", result.CareerPagesFound,
		"urls_queued", result.TotalURLsQueued,
		"ats_platforms", result.ATSPlatforms,
		"duration", result.Duration)

	return result, nil
}

// CompanyInfo contains basic information about a discovered company.
type CompanyInfo struct {
	Name        string
	Domain      string
	Description string
	Source      string
}

// CareerPageInfo contains information about a discovered career page.
type CareerPageInfo struct {
	URL         string
	Domain      string
	PageType    string
	ATSPlatform string
	Confidence  float64
	Priority    int
}

// ATSInfo contains information about a detected ATS platform.
type ATSInfo struct {
	URL        string
	Platform   string
	Confidence float64
	IsATS      bool
}
