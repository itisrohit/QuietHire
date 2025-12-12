// Package main provides a CLI tool to trigger company discovery workflows.
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"time"

	"go.temporal.io/sdk/client"
)

// Company represents a company to discover jobs for
type Company struct {
	Name   string
	Domain string
}

// List of tech companies with accessible job pages
var companies = []Company{
	{Name: "Stripe", Domain: "stripe.com"},
	{Name: "Shopify", Domain: "shopify.com"},
	{Name: "GitHub", Domain: "github.com"},
	{Name: "GitLab", Domain: "gitlab.com"},
	{Name: "Atlassian", Domain: "atlassian.com"},
	{Name: "Notion", Domain: "notion.so"},
	{Name: "Figma", Domain: "figma.com"},
	{Name: "Vercel", Domain: "vercel.com"},
	{Name: "Linear", Domain: "linear.app"},
	{Name: "Canva", Domain: "canva.com"},
}

func main() {
	log.Println("🚀 QuietHire Discovery Workflow Trigger")
	log.Println("========================================")

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
	defer c.Close()

	log.Printf("✅ Connected to Temporal at %s\n", temporalHost)
	log.Printf("📋 Triggering discovery for %d companies\n\n", len(companies))

	successCount := 0
	failCount := 0

	for i, company := range companies {
		log.Printf("[%d/%d] Processing: %s (%s)", i+1, len(companies), company.Name, company.Domain)

		// Create workflow input
		input := map[string]interface{}{
			"Query":      company.Domain,
			"Sources":    []string{"manual"},
			"MaxResults": 10,
		}

		// Create unique workflow ID
		workflowID := fmt.Sprintf("company-discovery-%s-%d", company.Domain, time.Now().Unix())

		// Workflow options
		workflowOptions := client.StartWorkflowOptions{
			ID:        workflowID,
			TaskQueue: "job-crawl-queue",
		}

		// Start the workflow
		we, err := c.ExecuteWorkflow(context.Background(), workflowOptions, "CompanyDiscoveryWorkflow", input)
		if err != nil {
			log.Printf("   ❌ Failed to start workflow: %v\n", err)
			failCount++
			continue
		}

		log.Printf("   ✅ Workflow started: ID=%s, RunID=%s\n", we.GetID(), we.GetRunID())
		successCount++

		// Small delay to avoid overwhelming the system
		time.Sleep(500 * time.Millisecond)
	}

	log.Println("\n========================================")
	log.Printf("📊 Summary:\n")
	log.Printf("   Success: %d/%d\n", successCount, len(companies))
	log.Printf("   Failed:  %d/%d\n", failCount, len(companies))
	log.Println("")
	log.Println("💡 Next steps:")
	log.Println("   1. Check workflow status: docker-compose logs -f worker")
	log.Println("   2. View Temporal UI: http://localhost:8080")
	log.Println("   3. Monitor jobs in database:")
	log.Println("      docker exec quiethire-clickhouse clickhouse-client --database=quiethire -q \"SELECT COUNT(*) FROM jobs\"")
	log.Println("")
	log.Println("⏳ Workflows are running in the background. Check logs for progress.")
}
