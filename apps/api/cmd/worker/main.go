// Package main runs the Temporal worker for job crawling workflows
package main

import (
	"database/sql"
	"fmt"
	"log"
	"net/http"
	"os"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/itisrohit/quiethire/apps/api/internal/activities"
	"github.com/itisrohit/quiethire/apps/api/internal/workflows"
	"github.com/joho/godotenv"
	_ "github.com/lib/pq"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Helper function to get environment variable with default
	getEnv := func(key, defaultValue string) string {
		if value := os.Getenv(key); value != "" {
			return value
		}
		return defaultValue
	}

	// Get Temporal configuration
	temporalHost := getEnv("TEMPORAL_HOST", "localhost:7233")

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

	// Initialize ClickHouse connection using environment variables directly
	var chConn clickhouse.Conn
	clickhouseHost := getEnv("CLICKHOUSE_HOST", "localhost")
	clickhousePort := getEnv("CLICKHOUSE_PORT", "9000")
	clickhouseDB := getEnv("CLICKHOUSE_DB", "quiethire")
	clickhouseUser := getEnv("CLICKHOUSE_USER", "default")
	clickhousePassword := getEnv("CLICKHOUSE_PASSWORD", "")

	chConn, err = clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%s", clickhouseHost, clickhousePort)},
		Auth: clickhouse.Auth{
			Database: clickhouseDB,
			Username: clickhouseUser,
			Password: clickhousePassword,
		},
	})
	if err != nil {
		log.Printf("Warning: Failed to connect to ClickHouse: %v", err)
	} else {
		log.Println("✅ Connected to ClickHouse")
	}

	// Initialize PostgreSQL connection using environment variables directly
	var pgConn *sql.DB
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "quiethire")
	dbPassword := getEnv("DB_PASSWORD", "")
	dbName := getEnv("DB_NAME", "quiethire")
	dbSSLMode := getEnv("DB_SSL_MODE", "disable")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=%s",
		dbHost, dbPort, dbUser, dbPassword, dbName, dbSSLMode)

	pgConn, err = sql.Open("postgres", connStr)
	if err != nil {
		log.Printf("Warning: Failed to connect to PostgreSQL: %v", err)
	} else {
		if err := pgConn.Ping(); err != nil {
			log.Printf("Warning: PostgreSQL ping failed: %v", err)
		} else {
			log.Println("✅ Connected to PostgreSQL")
		}
	}

	// Create worker
	w := worker.New(c, "job-crawl-queue", worker.Options{})

	// Register workflows
	w.RegisterWorkflow(workflows.CrawlCoordinatorWorkflow)
	w.RegisterWorkflow(workflows.ScheduledCrawlWorkflow)
	w.RegisterWorkflow(workflows.CompanyDiscoveryWorkflow)
	w.RegisterWorkflow(workflows.ContinuousDiscoveryWorkflow)
	w.RegisterWorkflow(workflows.GoogleDorkDiscoveryWorkflow)

	// Get service URLs from environment or config
	crawlerURL := getEnv("CRAWLER_SERVICE_URL", "http://localhost:8002")
	parserURL := getEnv("PARSER_SERVICE_URL", "http://localhost:8001")
	osintURL := getEnv("OSINT_SERVICE_URL", "http://localhost:8004")

	// Initialize and register crawl activities
	crawlActivities := &activities.CrawlActivities{
		HTTPClient: httpClient,
		CrawlerURL: crawlerURL,
		ParserURL:  parserURL,
		OSINTUrl:   osintURL,
		ClickHouse: chConn,
	}
	w.RegisterActivity(crawlActivities.DiscoverJobURLs)
	w.RegisterActivity(crawlActivities.CrawlJobBatch)
	w.RegisterActivity(crawlActivities.ParseJobActivity)
	w.RegisterActivity(crawlActivities.ScoreJobActivity)
	w.RegisterActivity(crawlActivities.ExtractHiringManagerActivity)

	// Initialize and register discovery activities
	discoveryActivities := &activities.DiscoveryActivities{
		HTTPClient: httpClient,
		OSINTUrl:   osintURL,
		PostgreSQL: pgConn,
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
