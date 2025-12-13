// Package main provides a CLI tool to schedule continuous discovery workflows with cron.
package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
)

func main() {
	log.Println("🚀 QuietHire Continuous Discovery Scheduler")
	log.Println("==========================================")

	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Get Temporal host from env or use default
	temporalHost := os.Getenv("TEMPORAL_HOST")
	if temporalHost == "" {
		temporalHost = "localhost:7233"
	}

	// Create Temporal client
	c, err := client.Dial(client.Options{
		HostPort: temporalHost,
	})
	if err != nil {
		log.Fatalf("❌ Failed to create Temporal client: %v", err)
	}

	log.Printf("✅ Connected to Temporal at %s\n", temporalHost)

	// Schedule continuous discovery workflow with cron
	scheduleID := "continuous-discovery-schedule"
	workflowID := "continuous-discovery-workflow"

	// Create workflow input
	input := map[string]interface{}{
		"StaleThresholdDays": 7,                                 // Re-crawl companies not seen in 7 days
		"RunGitHubDiscovery": true,                              // Discover new companies from GitHub
		"GitHubQuery":        "tech startup",                    // GitHub search query
		"RunDorkDiscovery":   true,                              // Discover via Google Dorks
		"DorkQuery":          "we are hiring software engineer", // Dork query
		"MaxNewCompanies":    50,                                // Max new companies per run
	}

	// Create schedule
	scheduleHandle, err := c.ScheduleClient().Create(context.Background(), client.ScheduleOptions{
		ID: scheduleID,
		Spec: client.ScheduleSpec{
			CronExpressions: []string{"0 */6 * * *"}, // Run every 6 hours
		},
		Action: &client.ScheduleWorkflowAction{
			ID:        workflowID,
			Workflow:  "ContinuousDiscoveryWorkflow",
			Args:      []interface{}{input},
			TaskQueue: "job-crawl-queue",
		},
	})

	if err != nil {
		c.Close()
		log.Fatalf("❌ Failed to create schedule: %v", err)
	}

	log.Printf("✅ Scheduled continuous discovery workflow\n")
	log.Printf("   Schedule ID: %s\n", scheduleID)
	log.Printf("   Cron: Every 6 hours (0 */6 * * *)\n")
	log.Printf("   Stale Threshold: 7 days\n")
	log.Printf("   Max New Companies: 50 per run\n")
	log.Println("")
	log.Println("📊 What happens every 6 hours:")
	log.Println("   1. Find companies not crawled in 7+ days")
	log.Println("   2. Re-discover their career pages and subdomains")
	log.Println("   3. Discover up to 50 new companies from GitHub")
	log.Println("   4. Discover new companies via Google Dorks")
	log.Println("   5. Queue all URLs for crawling")
	log.Println("")
	log.Println("💡 Next steps:")
	log.Printf("   View schedule: http://localhost:8080/namespaces/default/schedules/%s\n", scheduleID)
	log.Println("   Monitor workflows: docker-compose logs -f worker")
	log.Println("")
	log.Printf("✅ Schedule created successfully: %s\n", scheduleHandle.GetID())

	c.Close()
}
