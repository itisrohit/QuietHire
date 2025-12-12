// Package main provides a CLI tool to index jobs from ClickHouse to Typesense.
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"
	"time"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/itisrohit/quiethire/apps/api/internal/config"
	"github.com/joho/godotenv"
	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
	"github.com/typesense/typesense-go/typesense/api/pointer"
)

// Job represents a job posting with all metadata.
//
//nolint:govet // Field order optimized for readability over memory alignment
type Job struct {
	ID                 string   `json:"id"`
	Title              string   `json:"title"`
	Company            string   `json:"company"`
	Description        string   `json:"description"`
	Location           string   `json:"location"`
	JobType            string   `json:"job_type"`
	SourceURL          string   `json:"source_url"`
	SourcePlatform     string   `json:"source_platform"`
	Tags               []string `json:"tags,omitempty"`
	PostedAt           int64    `json:"posted_at"`
	UpdatedAt          int64    `json:"updated_at"`
	RealScore          int32    `json:"real_score"`
	SalaryMin          *int32   `json:"salary_min,omitempty"`
	SalaryMax          *int32   `json:"salary_max,omitempty"`
	Currency           *string  `json:"currency,omitempty"`
	ExperienceLevel    *string  `json:"experience_level,omitempty"`
	HiringManagerName  *string  `json:"hiring_manager_name,omitempty"`
	HiringManagerEmail *string  `json:"hiring_manager_email,omitempty"`
	Remote             bool     `json:"remote"`
}

func main() {
	if err := run(); err != nil {
		log.Fatalf("Error: %v", err)
	}
}

//nolint:gocyclo // run function handles complete indexing workflow, complexity is acceptable
func run() error {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Initialize ClickHouse connection
	chConn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:%d", cfg.ClickHouse.Host, cfg.ClickHouse.Port)},
		Auth: clickhouse.Auth{
			Database: cfg.ClickHouse.Database,
			Username: cfg.ClickHouse.User,
			Password: cfg.ClickHouse.Password,
		},
	})
	if err != nil {
		return fmt.Errorf("failed to connect to ClickHouse: %w", err)
	}
	defer func() {
		if closeErr := chConn.Close(); closeErr != nil {
			log.Printf("Warning: Failed to close ClickHouse connection: %v", closeErr)
		}
	}()

	if pingErr := chConn.Ping(context.Background()); pingErr != nil {
		return fmt.Errorf("clickHouse ping failed: %w", pingErr)
	}
	log.Println("✅ Connected to ClickHouse")

	// Initialize Typesense client
	tsClient := typesense.NewClient(
		typesense.WithServer(fmt.Sprintf("http://%s:%d", cfg.Typesense.Host, cfg.Typesense.Port)),
		typesense.WithAPIKey(cfg.Typesense.APIKey),
	)
	log.Println("✅ Typesense client initialized")

	// Fetch all jobs from ClickHouse
	log.Println("📦 Fetching jobs from ClickHouse...")
	query := `
		SELECT 
			id, title, company, description, location, remote,
			salary_min, salary_max, currency, job_type, experience_level,
			real_score, hiring_manager_name, hiring_manager_email,
			source_url, source_platform, tags,
			toUnixTimestamp(posted_at) as posted_at,
			toUnixTimestamp(updated_at) as updated_at
		FROM jobs
		ORDER BY posted_at DESC
	`

	rows, err := chConn.Query(context.Background(), query)
	if err != nil {
		return fmt.Errorf("query failed: %w", err)
	}
	defer func() {
		if closeErr := rows.Close(); closeErr != nil {
			log.Printf("Warning: Failed to close rows: %v", closeErr)
		}
	}()

	var jobs []Job
	for rows.Next() {
		var job Job
		var remote uint8
		var postedAt uint32
		var updatedAt uint32

		scanErr := rows.Scan(
			&job.ID,
			&job.Title,
			&job.Company,
			&job.Description,
			&job.Location,
			&remote,
			&job.SalaryMin,
			&job.SalaryMax,
			&job.Currency,
			&job.JobType,
			&job.ExperienceLevel,
			&job.RealScore,
			&job.HiringManagerName,
			&job.HiringManagerEmail,
			&job.SourceURL,
			&job.SourcePlatform,
			&job.Tags,
			&postedAt,
			&updatedAt,
		)
		if scanErr != nil {
			log.Printf("⚠️  Scan error: %v", scanErr)
			continue
		}

		job.Remote = remote == 1
		job.PostedAt = int64(postedAt)
		job.UpdatedAt = int64(updatedAt)
		jobs = append(jobs, job)
	}

	log.Printf("📊 Found %d jobs in ClickHouse\n", len(jobs))

	if len(jobs) == 0 {
		log.Println("✅ No jobs to index")
		return nil
	}

	// Index jobs to Typesense in batches using JSONL format
	batchSize := 40
	totalIndexed := 0
	totalErrors := 0

	log.Println("🚀 Indexing jobs to Typesense...")
	for i := 0; i < len(jobs); i += batchSize {
		end := i + batchSize
		if end > len(jobs) {
			end = len(jobs)
		}

		batch := jobs[i:end]
		log.Printf("   Batch %d-%d of %d...", i+1, end, len(jobs))

		// Convert jobs to JSONL (newline-delimited JSON)
		var buf bytes.Buffer
		for _, job := range batch {
			jobJSON, marshalErr := json.Marshal(job)
			if marshalErr != nil {
				log.Printf("⚠️  JSON marshal error: %v", marshalErr)
				totalErrors++
				continue
			}
			buf.Write(jobJSON)
			buf.WriteString("\n")
		}

		// Import batch to Typesense
		action := "upsert"
		params := &api.ImportDocumentsParams{
			Action:    &action,
			BatchSize: pointer.Int(40),
		}

		resp, importErr := tsClient.Collection("jobs").Documents().ImportJsonl(
			context.Background(),
			bytes.NewReader(buf.Bytes()),
			params,
		)
		if importErr != nil {
			log.Printf("⚠️  Import error: %v", importErr)
			totalErrors += len(batch)
			continue
		}

		// Read response body
		var respBuf bytes.Buffer
		if _, readErr := respBuf.ReadFrom(resp); readErr != nil {
			log.Printf("⚠️  Failed to read response: %v", readErr)
			_ = resp.Close()
			totalErrors += len(batch)
			continue
		}

		if closeErr := resp.Close(); closeErr != nil {
			log.Printf("Warning: Failed to close response: %v", closeErr)
		}

		// Parse results - response is JSONL with one result per line
		successCount := 0
		errorCount := 0

		lines := strings.Split(strings.TrimSpace(respBuf.String()), "\n")
		for _, line := range lines {
			if line == "" {
				continue
			}
			var res map[string]interface{}
			if unmarshalErr := json.Unmarshal([]byte(line), &res); unmarshalErr != nil {
				log.Printf("⚠️  Parse error: %v", unmarshalErr)
				errorCount++
				continue
			}
			if success, ok := res["success"].(bool); ok && success {
				successCount++
			} else {
				errorCount++
				if errMsg, ok := res["error"].(string); ok {
					log.Printf("   ⚠️  Document error: %s", errMsg)
				}
			}
		}

		totalIndexed += successCount
		totalErrors += errorCount

		log.Printf("   ✓ Indexed %d documents (%d errors)", successCount, errorCount)

		// Small delay to avoid overwhelming Typesense
		time.Sleep(100 * time.Millisecond)
	}

	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")
	log.Printf("✅ Indexing complete!")
	log.Printf("   Total jobs: %d", len(jobs))
	log.Printf("   Indexed: %d", totalIndexed)
	log.Printf("   Errors: %d", totalErrors)
	log.Println("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━")

	// Verify index count
	collection, err := tsClient.Collection("jobs").Retrieve(context.Background())
	if err != nil {
		return fmt.Errorf("could not verify collection: %w", err)
	}

	log.Printf("📊 Typesense 'jobs' collection now has %d documents\n", *collection.NumDocuments)
	log.Println("✅ All done! You can now search jobs via /api/v1/search endpoint")

	return nil
}
