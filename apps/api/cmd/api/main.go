// Package main provides the main HTTP API server for QuietHire
package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strconv"

	clickhouse "github.com/ClickHouse/clickhouse-go/v2"
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/itisrohit/quiethire/apps/api/internal/config"
	"github.com/joho/godotenv"
	"github.com/typesense/typesense-go/typesense"
	tsapi "github.com/typesense/typesense-go/typesense/api"
)

func main() {
	// Load environment variables
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found, using system environment variables")
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("Failed to load configuration: %v", err)
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
		log.Printf("Warning: Failed to connect to ClickHouse: %v", err)
		log.Println("API will start but database features will be limited")
	} else {
		// Test connection
		if err := chConn.Ping(context.Background()); err != nil {
			log.Printf("Warning: ClickHouse ping failed: %v", err)
		} else {
			log.Println("✅ Connected to ClickHouse")
		}
	}

	// Initialize Typesense client
	tsClient := typesense.NewClient(
		typesense.WithServer(fmt.Sprintf("http://%s:%d", cfg.Typesense.Host, cfg.Typesense.Port)),
		typesense.WithAPIKey(cfg.Typesense.APIKey),
	)
	log.Println("✅ Typesense client initialized")

	// Create Fiber app
	app := fiber.New(fiber.Config{
		AppName:      cfg.App.Name,
		ErrorHandler: customErrorHandler,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "[${time}] ${status} - ${method} ${path} - ${latency}\n",
	}))
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE,OPTIONS",
		AllowHeaders: "Origin, Content-Type, Accept, Authorization",
	}))

	// Health check endpoint
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "ok",
			"service": "QuietHire API",
			"version": "0.1.0",
		})
	})

	// API v1 routes
	api := app.Group("/api/v1")

	// Search endpoint with Typesense
	api.Get("/search", func(c *fiber.Ctx) error {
		query := c.Query("q")
		if query == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "query parameter 'q' is required",
			})
		}

		// Get pagination parameters
		page, _ := strconv.Atoi(c.Query("page", "1"))
		perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

		// Search in Typesense
		searchParams := &tsapi.SearchCollectionParams{
			Q:       query,
			QueryBy: "title,company,description",
			Page:    &page,
			PerPage: &perPage,
		}

		results, err := tsClient.Collection("jobs").Documents().Search(context.Background(), searchParams)
		if err != nil {
			log.Printf("Search error: %v", err)
			// Return empty results if Typesense is not set up yet
			return c.JSON(fiber.Map{
				"query":   query,
				"results": []interface{}{},
				"found":   0,
				"page":    page,
				"message": "Typesense collection not initialized yet",
			})
		}

		return c.JSON(fiber.Map{
			"query":   query,
			"results": results,
			"page":    page,
		})
	})

	// Get job by ID from ClickHouse
	api.Get("/jobs/:id", func(c *fiber.Ctx) error {
		jobID := c.Params("id")

		if chConn == nil {
			return c.Status(503).JSON(fiber.Map{
				"error": "Database connection not available",
			})
		}

		var job struct {
			ID                 string   `ch:"id"`
			Title              string   `ch:"title"`
			Company            string   `ch:"company"`
			Description        string   `ch:"description"`
			Location           string   `ch:"location"`
			Remote             uint8    `ch:"remote"`
			SalaryMin          *int32   `ch:"salary_min"`
			SalaryMax          *int32   `ch:"salary_max"`
			Currency           *string  `ch:"currency"`
			JobType            string   `ch:"job_type"`
			ExperienceLevel    *string  `ch:"experience_level"`
			RealScore          int32    `ch:"real_score"`
			HiringManagerName  *string  `ch:"hiring_manager_name"`
			HiringManagerEmail *string  `ch:"hiring_manager_email"`
			SourceURL          string   `ch:"source_url"`
			SourcePlatform     string   `ch:"source_platform"`
			Tags               []string `ch:"tags"`
			PostedAt           string   `ch:"posted_at"`
			UpdatedAt          string   `ch:"updated_at"`
			CrawledAt          string   `ch:"crawled_at"`
		}

		err := chConn.QueryRow(context.Background(), `
			SELECT id, title, company, description, location, remote,
			       salary_min, salary_max, currency, job_type, experience_level,
			       real_score, hiring_manager_name, hiring_manager_email,
			       source_url, source_platform, tags,
			       toString(posted_at) as posted_at,
			       toString(updated_at) as updated_at,
			       toString(crawled_at) as crawled_at
			FROM jobs
			WHERE id = ?
			LIMIT 1
		`, jobID).ScanStruct(&job)

		if err != nil {
			log.Printf("Query error: %v", err)
			return c.Status(404).JSON(fiber.Map{
				"error": "Job not found",
			})
		}

		return c.JSON(job)
	})

	// List jobs with pagination
	api.Get("/jobs", func(c *fiber.Ctx) error {
		if chConn == nil {
			return c.Status(503).JSON(fiber.Map{
				"error": "Database connection not available",
			})
		}

		// Get pagination parameters
		limit, _ := strconv.Atoi(c.Query("limit", "20"))
		offset, _ := strconv.Atoi(c.Query("offset", "0"))

		// Query filters
		company := c.Query("company")
		location := c.Query("location")
		minScore := c.Query("min_score", "70")

		query := `
			SELECT id, title, company, location, remote,
			       salary_min, salary_max, currency,
			       real_score, source_platform,
			       toString(posted_at) as posted_at
			FROM jobs
			WHERE real_score >= ?
		`
		args := []interface{}{minScore}

		if company != "" {
			query += " AND company = ?"
			args = append(args, company)
		}
		if location != "" {
			query += " AND location = ?"
			args = append(args, location)
		}

		query += " ORDER BY posted_at DESC LIMIT ? OFFSET ?"
		args = append(args, limit, offset)

		rows, err := chConn.Query(context.Background(), query, args...)
		if err != nil {
			log.Printf("Query error: %v", err)
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to fetch jobs",
			})
		}
		defer rows.Close()

		var jobs []map[string]interface{}
		for rows.Next() {
			var (
				id             string
				title          string
				company        string
				location       string
				remote         uint8
				salaryMin      *int32
				salaryMax      *int32
				currency       *string
				realScore      int32
				sourcePlatform string
				postedAt       string
			)

			if err := rows.Scan(&id, &title, &company, &location, &remote,
				&salaryMin, &salaryMax, &currency, &realScore, &sourcePlatform, &postedAt); err != nil {
				log.Printf("Scan error: %v", err)
				continue
			}

			jobs = append(jobs, map[string]interface{}{
				"id":              id,
				"title":           title,
				"company":         company,
				"location":        location,
				"remote":          remote == 1,
				"salary_min":      salaryMin,
				"salary_max":      salaryMax,
				"currency":        currency,
				"real_score":      realScore,
				"source_platform": sourcePlatform,
				"posted_at":       postedAt,
			})
		}

		return c.JSON(fiber.Map{
			"jobs":   jobs,
			"count":  len(jobs),
			"limit":  limit,
			"offset": offset,
		})
	})

	// Stats endpoint
	api.Get("/stats", func(c *fiber.Ctx) error {
		if chConn == nil {
			return c.Status(503).JSON(fiber.Map{
				"error": "Database connection not available",
			})
		}

		var stats struct {
			TotalJobs     int64   `ch:"total_jobs"`
			ActiveJobs    int64   `ch:"active_jobs"`
			Companies     int64   `ch:"companies"`
			AvgRealScore  float64 `ch:"avg_real_score"`
			LastCrawledAt string  `ch:"last_crawled_at"`
		}

		err := chConn.QueryRow(context.Background(), `
			SELECT 
				count() as total_jobs,
				countIf(real_score >= 70 AND posted_at >= now() - INTERVAL 90 DAY) as active_jobs,
				uniq(company) as companies,
				avg(real_score) as avg_real_score,
				toString(max(crawled_at)) as last_crawled_at
			FROM jobs
		`).ScanStruct(&stats)

		if err != nil {
			log.Printf("Stats query error: %v", err)
			return c.Status(500).JSON(fiber.Map{
				"error": "Failed to fetch stats",
			})
		}

		return c.JSON(stats)
	})

	// Start server
	port := os.Getenv("API_PORT")
	if port == "" {
		port = "3000"
	}

	log.Printf("Starting %s on port %s", cfg.App.Name, port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}

func customErrorHandler(c *fiber.Ctx, err error) error {
	code := fiber.StatusInternalServerError

	if e, ok := err.(*fiber.Error); ok {
		code = e.Code
	}

	return c.Status(code).JSON(fiber.Map{
		"error":  err.Error(),
		"code":   code,
		"path":   c.Path(),
		"method": c.Method(),
	})
}
