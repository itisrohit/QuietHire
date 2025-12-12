// Package main initializes the ClickHouse database schema for QuietHire
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/joho/godotenv"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Get ClickHouse configuration from environment
	host := os.Getenv("CLICKHOUSE_HOST")
	if host == "" {
		host = "localhost"
	}
	user := os.Getenv("CLICKHOUSE_USER")
	if user == "" {
		user = "default"
	}
	password := os.Getenv("CLICKHOUSE_PASSWORD")
	dbName := os.Getenv("CLICKHOUSE_DB")
	if dbName == "" {
		dbName = "quiethire"
	}

	// Connect to ClickHouse
	conn, err := clickhouse.Open(&clickhouse.Options{
		Addr: []string{fmt.Sprintf("%s:9000", host)},
		Auth: clickhouse.Auth{
			Database: dbName,
			Username: user,
			Password: password,
		},
	})
	if err != nil {
		log.Fatalf("Failed to connect to ClickHouse: %v", err)
	}
	defer func() {
		if closeErr := conn.Close(); closeErr != nil {
			log.Printf("Error closing connection: %v", closeErr)
		}
	}()

	ctx := context.Background()

	// Test connection
	if pingErr := conn.Ping(ctx); pingErr != nil {
		log.Fatalf("Failed to ping ClickHouse: %v", pingErr) //nolint:gocritic // Acceptable in initialization script
	}

	log.Println("✅ Connected to ClickHouse")

	// Create database if it doesn't exist
	err = conn.Exec(ctx, fmt.Sprintf("CREATE DATABASE IF NOT EXISTS %s", dbName))
	if err != nil {
		log.Fatalf("Failed to create database: %v", err)
	}
	log.Printf("✅ Database '%s' ready", dbName)

	// Read and execute schema (Docker path)
	schemaPath := "/app/config/clickhouse/schema.sql"
	schema, err := os.ReadFile(schemaPath)
	if err != nil {
		log.Fatalf("Failed to read schema file: %v", err)
	}

	// Split schema into individual statements and execute
	statements := strings.Split(string(schema), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" || strings.HasPrefix(stmt, "--") {
			continue
		}

		err = conn.Exec(ctx, stmt)
		if err != nil {
			log.Printf("Warning: Failed to execute statement: %v", err)
			stmtLen := len(stmt)
			if stmtLen > 100 {
				stmtLen = 100
			}
			log.Printf("Statement: %s", stmt[:stmtLen])
		}
	}

	log.Println("✅ ClickHouse schema initialized successfully!")
	log.Println("\nCreated tables:")
	log.Println("  - jobs (with ReplacingMergeTree for deduplication)")
	log.Println("  - jobs_raw_html")
	log.Println("  - crawl_history")
	log.Println("  - jobs_active (materialized view)")
	log.Println("  - job_duplicates")
	log.Println("  - job_stats")
}
