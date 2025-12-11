// Package main runs the Temporal worker for job crawling workflows
package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/itisrohit/quiethire/apps/api/internal/activities"
	"github.com/itisrohit/quiethire/apps/api/internal/config"
	"github.com/itisrohit/quiethire/apps/api/internal/workflows"
	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Printf("Warning: Failed to load full configuration: %v", err)
		log.Println("Continuing with partial configuration...")
	}

	// Get Temporal configuration
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalln("Unable to create Temporal client", err)
	}
	defer func() {
		c.Close()
	}()

	// Create HTTP client for calling microservices
	httpClient := &http.Client{
		Timeout: 60 * time.Second,
	}

	// Create worker
	w := worker.New(c, "job-crawl-queue", worker.Options{})

	// Register workflows
	w.RegisterWorkflow(workflows.CrawlCoordinatorWorkflow)
	w.RegisterWorkflow(workflows.ScheduledCrawlWorkflow)
	w.RegisterWorkflow(workflows.CompanyDiscoveryWorkflow)
	w.RegisterWorkflow(workflows.ContinuousDiscoveryWorkflow)
	w.RegisterWorkflow(workflows.GoogleDorkDiscoveryWorkflow)

	// Initialize and register crawl activities
	crawlActivities := &activities.CrawlActivities{
		HTTPClient: httpClient,
		CrawlerURL: cfg.Services.CrawlerURL,
		ParserURL:  cfg.Services.ParserURL,
		OSINTUrl:   cfg.Services.OSINTUrl,
	}
	w.RegisterActivity(crawlActivities.DiscoverJobURLs)
	w.RegisterActivity(crawlActivities.CrawlJobBatch)
	w.RegisterActivity(crawlActivities.ParseJobActivity)
	w.RegisterActivity(crawlActivities.ScoreJobActivity)
	w.RegisterActivity(crawlActivities.ExtractHiringManagerActivity)

	// Initialize and register discovery activities
	discoveryActivities := &activities.DiscoveryActivities{
		HTTPClient: httpClient,
		OSINTUrl:   cfg.Services.OSINTUrl,
	}
	w.RegisterActivity(discoveryActivities.DiscoverCompaniesFromGitHub)
	w.RegisterActivity(discoveryActivities.DiscoverCompaniesFromGoogleDorks)
	w.RegisterActivity(discoveryActivities.AddCompanyManually)
	w.RegisterActivity(discoveryActivities.DiscoverCareerPages)
	w.RegisterActivity(discoveryActivities.EnumerateSubdomains)
	w.RegisterActivity(discoveryActivities.DetectATS)
	w.RegisterActivity(discoveryActivities.QueueURLsForCrawling)
	w.RegisterActivity(discoveryActivities.GenerateDorkQueries)
	w.RegisterActivity(discoveryActivities.ExecuteDorkQuery)
	w.RegisterActivity(discoveryActivities.DetectATSAndExtractDomain)

	log.Println("✅ Temporal worker started")
	log.Println("Task Queue: job-crawl-queue")
	log.Println("Registered Workflows:")
	log.Println("  - CrawlCoordinatorWorkflow")
	log.Println("  - ScheduledCrawlWorkflow")
	log.Println("  - CompanyDiscoveryWorkflow")
	log.Println("  - ContinuousDiscoveryWorkflow")
	log.Println("  - GoogleDorkDiscoveryWorkflow")
	log.Println("Registered Activities:")
	log.Println("  Crawl Activities:")
	log.Println("    - DiscoverJobURLs")
	log.Println("    - CrawlJobBatch")
	log.Println("    - ParseJobActivity")
	log.Println("    - ScoreJobActivity")
	log.Println("    - ExtractHiringManagerActivity")
	log.Println("  Discovery Activities:")
	log.Println("    - DiscoverCompaniesFromGitHub")
	log.Println("    - DiscoverCompaniesFromGoogleDorks")
	log.Println("    - AddCompanyManually")
	log.Println("    - DiscoverCareerPages")
	log.Println("    - EnumerateSubdomains")
	log.Println("    - DetectATS")
	log.Println("    - QueueURLsForCrawling")
	log.Println("    - GenerateDorkQueries")
	log.Println("    - ExecuteDorkQuery")
	log.Println("    - DetectATSAndExtractDomain")

	// Start listening to the Task Queue
	if err := w.Run(worker.InterruptCh()); err != nil {
		log.Fatalln("Unable to start worker", err) //nolint:gocritic // Acceptable pattern for worker exit
	}
}
