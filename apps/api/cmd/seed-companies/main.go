// Package main provides a CLI tool to seed initial companies into the database.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"os"

	_ "github.com/lib/pq"
)

// Company represents a company to add to the database
type Company struct {
	Name        string
	Domain      string
	Description string
	Website     string
}

// List of tech companies to seed
var companies = []Company{
	{
		Name:        "Stripe",
		Domain:      "stripe.com",
		Description: "Financial infrastructure for the internet",
		Website:     "https://stripe.com/jobs",
	},
	{
		Name:        "Shopify",
		Domain:      "shopify.com",
		Description: "E-commerce platform for online stores",
		Website:     "https://www.shopify.com/careers",
	},
	{
		Name:        "GitHub",
		Domain:      "github.com",
		Description: "Code hosting platform for version control and collaboration",
		Website:     "https://github.com/about/careers",
	},
	{
		Name:        "GitLab",
		Domain:      "gitlab.com",
		Description: "DevOps platform for software development lifecycle",
		Website:     "https://about.gitlab.com/jobs/",
	},
	{
		Name:        "Atlassian",
		Domain:      "atlassian.com",
		Description: "Team collaboration and productivity software",
		Website:     "https://www.atlassian.com/company/careers",
	},
	{
		Name:        "Notion",
		Domain:      "notion.so",
		Description: "All-in-one workspace for notes, docs, and collaboration",
		Website:     "https://www.notion.so/careers",
	},
	{
		Name:        "Figma",
		Domain:      "figma.com",
		Description: "Collaborative design and prototyping platform",
		Website:     "https://www.figma.com/careers/",
	},
	{
		Name:        "Vercel",
		Domain:      "vercel.com",
		Description: "Platform for frontend developers",
		Website:     "https://vercel.com/careers",
	},
	{
		Name:        "Linear",
		Domain:      "linear.app",
		Description: "Modern issue tracking for software teams",
		Website:     "https://linear.app/careers",
	},
	{
		Name:        "Canva",
		Domain:      "canva.com",
		Description: "Online graphic design platform",
		Website:     "https://www.canva.com/careers/",
	},
}

func main() {
	log.Println("🏢 Seeding QuietHire with Tech Companies")
	log.Println("========================================")

	// Get database connection string from env
	dbHost := getEnv("DB_HOST", "localhost")
	dbPort := getEnv("DB_PORT", "5432")
	dbUser := getEnv("DB_USER", "quiethire")
	dbPassword := getEnv("DB_PASSWORD", "quiethire_password")
	dbName := getEnv("DB_NAME", "quiethire")

	connStr := fmt.Sprintf("host=%s port=%s user=%s password=%s dbname=%s sslmode=disable",
		dbHost, dbPort, dbUser, dbPassword, dbName)

	// Connect to PostgreSQL
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		log.Fatalf("❌ Failed to connect to PostgreSQL: %v", err)
	}
	defer func() {
		if closeErr := db.Close(); closeErr != nil {
			log.Printf("⚠️  Failed to close database: %v", closeErr)
		}
	}()

	// Test connection
	if err := db.Ping(); err != nil {
		log.Printf("❌ Failed to ping PostgreSQL: %v", err)
		return
	}

	log.Printf("✅ Connected to PostgreSQL at %s:%s\n\n", dbHost, dbPort)

	successCount := 0
	skipCount := 0
	failCount := 0

	for i, company := range companies {
		log.Printf("[%d/%d] Adding: %s (%s)", i+1, len(companies), company.Name, company.Domain)

		// Check if company already exists
		var existingID int
		err := db.QueryRow(
			"SELECT id FROM companies WHERE domain = $1",
			company.Domain,
		).Scan(&existingID)

		if err == nil {
			log.Printf("   ⏭️  Already exists (ID: %d)\n", existingID)
			skipCount++
			continue
		} else if err != sql.ErrNoRows {
			log.Printf("   ❌ Error checking existence: %v\n", err)
			failCount++
			continue
		}

		// Insert company
		var id int
		err = db.QueryRowContext(context.Background(), `
			INSERT INTO companies (name, domain, description, source, metadata)
			VALUES ($1, $2, $3, $4, $5)
			RETURNING id
		`, company.Name, company.Domain, company.Description, "manual_seed",
			fmt.Sprintf(`{"website": "%s"}`, company.Website)).Scan(&id)

		if err != nil {
			log.Printf("   ❌ Failed to insert: %v\n", err)
			failCount++
			continue
		}

		log.Printf("   ✅ Added successfully (ID: %d)\n", id)
		successCount++
	}

	log.Println("\n========================================")
	log.Printf("📊 Summary:\n")
	log.Printf("   Added:   %d/%d\n", successCount, len(companies))
	log.Printf("   Skipped: %d/%d (already exist)\n", skipCount, len(companies))
	log.Printf("   Failed:  %d/%d\n", failCount, len(companies))
	log.Println("")

	if successCount > 0 || skipCount > 0 {
		log.Println("✨ Companies ready for discovery!")
		log.Println("")
		log.Println("💡 Next step:")
		log.Println("   Run: go run cmd/trigger-discovery/main.go")
		log.Println("   Or build: cd apps/api && go build -o trigger-discovery cmd/trigger-discovery/main.go")
	}
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
