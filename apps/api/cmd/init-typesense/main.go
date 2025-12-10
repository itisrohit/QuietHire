// Package main initializes the Typesense schema for QuietHire
package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/typesense/typesense-go/typesense"
	"github.com/typesense/typesense-go/typesense/api"
	"github.com/typesense/typesense-go/typesense/api/pointer"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	}

	// Get Typesense configuration from environment
	host := os.Getenv("TYPESENSE_HOST")
	if host == "" {
		host = "localhost"
	}
	apiKey := os.Getenv("TYPESENSE_API_KEY")
	if apiKey == "" {
		log.Fatal("TYPESENSE_API_KEY is required")
	}

	// Create Typesense client
	client := typesense.NewClient(
		typesense.WithServer(fmt.Sprintf("http://%s:8108", host)),
		typesense.WithAPIKey(apiKey),
	)

	// Define the jobs collection schema
	schema := &api.CollectionSchema{
		Name: "jobs",
		Fields: []api.Field{
			{
				Name: "id",
				Type: "string",
			},
			{
				Name: "title",
				Type: "string",
			},
			{
				Name:  "company",
				Type:  "string",
				Facet: pointer.True(),
			},
			{
				Name: "description",
				Type: "string",
			},
			{
				Name:  "location",
				Type:  "string",
				Facet: pointer.True(),
			},
			{
				Name:  "remote",
				Type:  "bool",
				Facet: pointer.True(),
			},
			{
				Name:     "salary_min",
				Type:     "int32",
				Optional: pointer.True(),
			},
			{
				Name:     "salary_max",
				Type:     "int32",
				Optional: pointer.True(),
			},
			{
				Name:     "currency",
				Type:     "string",
				Facet:    pointer.True(),
				Optional: pointer.True(),
			},
			{
				Name:  "job_type",
				Type:  "string",
				Facet: pointer.True(),
			},
			{
				Name:     "experience_level",
				Type:     "string",
				Facet:    pointer.True(),
				Optional: pointer.True(),
			},
			{
				Name: "real_score",
				Type: "int32",
			},
			{
				Name:     "hiring_manager_name",
				Type:     "string",
				Optional: pointer.True(),
			},
			{
				Name:     "hiring_manager_email",
				Type:     "string",
				Optional: pointer.True(),
			},
			{
				Name: "posted_at",
				Type: "int64",
			},
			{
				Name: "updated_at",
				Type: "int64",
			},
			{
				Name: "source_url",
				Type: "string",
			},
			{
				Name:  "source_platform",
				Type:  "string",
				Facet: pointer.True(),
			},
			{
				Name:     "tags",
				Type:     "string[]",
				Facet:    pointer.True(),
				Optional: pointer.True(),
			},
		},
		DefaultSortingField: pointer.String("posted_at"),
	}

	// Try to delete existing collection (if it exists)
	ctx := context.Background()
	_, err := client.Collection("jobs").Delete(ctx)
	if err != nil {
		log.Printf("Note: Could not delete existing collection (may not exist): %v", err)
	}

	// Create the collection
	collection, err := client.Collections().Create(ctx, schema)
	if err != nil {
		log.Fatalf("Failed to create collection: %v", err)
	}

	log.Printf("Successfully created collection: %s", collection.Name)
	log.Printf("Collection has %d fields", len(collection.Fields))
	log.Println("\nTypesense schema initialized successfully!")
	log.Println("Jobs can now be indexed with the following fields:")
	for _, field := range collection.Fields {
		log.Printf("  - %s (%s)", field.Name, field.Type)
	}
}
