// Package main provides a test CLI for continuous discovery workflow
package main

import (
	"context"
	"log"
	"os"

	"github.com/joho/godotenv"
	"go.temporal.io/sdk/client"
)

func main() {
	log.Println("🧪 Testing Continuous Discovery Workflow")
	log.Println("=========================================")

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

	// Test with only stale companies (no GitHub/Dork discovery for testing)
	input := map[string]interface{}{
		"StaleThresholdDays": 7,     // Find companies not crawled in 7 days
		"RunGitHubDiscovery": false, // Skip GitHub for testing
		"GitHubQuery":        "",
		"RunDorkDiscovery":   false, // Skip Dork for testing
		"DorkQuery":          "",
		"MaxNewCompanies":    5,
	}

	workflowOptions := client.StartWorkflowOptions{
		ID:        "continuous-discovery-test",
		TaskQueue: "job-crawl-queue",
	}

	we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, "ContinuousDiscoveryWorkflow", input)
	if err != nil {
		c.Close()
		log.Fatalf("❌ Failed to start workflow: %v", err)
	}

	log.Printf("✅ Started workflow successfully\n")
	log.Printf("   Workflow ID: %s\n", we.GetID())
	log.Printf("   Run ID: %s\n", we.GetRunID())
	log.Println("")
	log.Println("📊 Test Configuration:")
	log.Println("   - Finding companies not crawled in 7+ days")
	log.Println("   - Skipping GitHub discovery (test mode)")
	log.Println("   - Skipping Dork discovery (test mode)")
	log.Println("")
	log.Println("💡 Monitor progress:")
	log.Printf("   Temporal UI: http://localhost:8080/namespaces/default/workflows/%s\n", we.GetID())
	log.Println("   Worker logs: docker-compose logs -f worker")
	log.Println("")
	log.Println("⏳ Waiting for workflow to complete...")

	// Wait for result
	var result interface{}
	err = we.Get(context.Background(), &result)
	if err != nil {
		log.Printf("❌ Workflow failed: %v\n", err)
	} else {
		log.Printf("✅ Workflow completed successfully!\n")
		log.Printf("   Result: %v\n", result)
	}

	c.Close()
}
