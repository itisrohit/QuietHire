// Package main provides the main HTTP API server for QuietHire
package main

import (
	"log"
	"os"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/itisrohit/quiethire/apps/api/internal/config"
	"github.com/joho/godotenv"
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

	// Search endpoint (placeholder)
	api.Get("/search", func(c *fiber.Ctx) error {
		query := c.Query("q")
		if query == "" {
			return c.Status(400).JSON(fiber.Map{
				"error": "query parameter 'q' is required",
			})
		}

		// TODO: Implement Typesense search
		return c.JSON(fiber.Map{
			"query":   query,
			"results": []fiber.Map{},
			"total":   0,
			"message": "Search endpoint placeholder - will be implemented with Typesense",
		})
	})

	// Jobs endpoint (placeholder)
	api.Get("/jobs/:id", func(c *fiber.Ctx) error {
		jobID := c.Params("id")

		// TODO: Fetch job from ClickHouse
		return c.JSON(fiber.Map{
			"id":      jobID,
			"message": "Job detail endpoint placeholder - will fetch from ClickHouse",
		})
	})

	// Stats endpoint
	api.Get("/stats", func(c *fiber.Ctx) error {
		// TODO: Get real stats from database
		return c.JSON(fiber.Map{
			"total_jobs":      0,
			"active_jobs":     0,
			"companies":       0,
			"avg_real_score":  0,
			"last_crawled_at": nil,
		})
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
