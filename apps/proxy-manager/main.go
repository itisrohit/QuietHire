// Package main provides a proxy rotation and health checking service for QuietHire
package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/redis/go-redis/v9"
)

// Proxy represents a proxy server
type Proxy struct {
	LastUsed     time.Time `json:"last_used"`
	URL          string    `json:"url"`
	Host         string    `json:"host"`
	Protocol     string    `json:"protocol"` // http, https, socks5
	Username     string    `json:"username,omitempty"`
	Password     string    `json:"password,omitempty"`
	Country      string    `json:"country,omitempty"`
	Port         int       `json:"port"`
	FailCount    int       `json:"fail_count"`
	SuccessCount int       `json:"success_count"`
	AvgLatency   int       `json:"avg_latency_ms"`
	IsHealthy    bool      `json:"is_healthy"`
}

// ProxyManager manages proxy rotation and health checks
type ProxyManager struct {
	redis               *redis.Client
	proxies             []*Proxy
	mu                  sync.RWMutex
	healthCheckInterval time.Duration
	currentIndex        int
}

var (
	manager *ProxyManager
	ctx     = context.Background()
)

// NewProxyManager creates a new proxy manager
func NewProxyManager(redisClient *redis.Client) *ProxyManager {
	return &ProxyManager{
		proxies:             make([]*Proxy, 0),
		currentIndex:        0,
		redis:               redisClient,
		healthCheckInterval: 5 * time.Minute,
	}
}

// LoadProxiesFromEnv loads proxies from environment variable
func (pm *ProxyManager) LoadProxiesFromEnv() error {
	proxiesJSON := os.Getenv("PROXIES")
	if proxiesJSON == "" {
		log.Println("No proxies configured in PROXIES env var")
		return nil
	}

	var proxies []*Proxy
	if err := json.Unmarshal([]byte(proxiesJSON), &proxies); err != nil {
		return fmt.Errorf("failed to parse PROXIES: %w", err)
	}

	pm.mu.Lock()
	pm.proxies = proxies
	pm.mu.Unlock()

	log.Printf("Loaded %d proxies from environment", len(proxies))
	return nil
}

// LoadProxiesFromRedis loads proxies from Redis
func (pm *ProxyManager) LoadProxiesFromRedis() error {
	data, err := pm.redis.Get(ctx, "proxies:list").Result()
	if err == redis.Nil {
		log.Println("No proxies found in Redis")
		return nil
	}
	if err != nil {
		return fmt.Errorf("failed to load proxies from Redis: %w", err)
	}

	var proxies []*Proxy
	if err := json.Unmarshal([]byte(data), &proxies); err != nil {
		return fmt.Errorf("failed to parse proxies from Redis: %w", err)
	}

	pm.mu.Lock()
	pm.proxies = proxies
	pm.mu.Unlock()

	log.Printf("Loaded %d proxies from Redis", len(proxies))
	return nil
}

// SaveProxiesToRedis saves proxies to Redis
func (pm *ProxyManager) SaveProxiesToRedis() error {
	pm.mu.RLock()
	data, err := json.Marshal(pm.proxies)
	pm.mu.RUnlock()

	if err != nil {
		return fmt.Errorf("failed to marshal proxies: %w", err)
	}

	if err := pm.redis.Set(ctx, "proxies:list", data, 0).Err(); err != nil {
		return fmt.Errorf("failed to save proxies to Redis: %w", err)
	}

	return nil
}

// GetNextProxy returns the next available healthy proxy (round-robin)
func (pm *ProxyManager) GetNextProxy() (*Proxy, error) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	if len(pm.proxies) == 0 {
		return nil, fmt.Errorf("no proxies available")
	}

	// Find next healthy proxy
	attempts := 0
	maxAttempts := len(pm.proxies)

	for attempts < maxAttempts {
		pm.currentIndex = (pm.currentIndex + 1) % len(pm.proxies)
		proxy := pm.proxies[pm.currentIndex]

		if proxy.IsHealthy || proxy.FailCount < 3 {
			proxy.LastUsed = time.Now()
			return proxy, nil
		}

		attempts++
	}

	return nil, fmt.Errorf("no healthy proxies available")
}

// GetProxyByCountry returns a healthy proxy from a specific country
func (pm *ProxyManager) GetProxyByCountry(country string) (*Proxy, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	for _, proxy := range pm.proxies {
		if proxy.Country == country && (proxy.IsHealthy || proxy.FailCount < 3) {
			proxy.LastUsed = time.Now()
			return proxy, nil
		}
	}

	return nil, fmt.Errorf("no healthy proxies available for country: %s", country)
}

// MarkProxySuccess marks a proxy as successful
func (pm *ProxyManager) MarkProxySuccess(proxyURL string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, proxy := range pm.proxies {
		if proxy.URL == proxyURL {
			proxy.SuccessCount++
			proxy.FailCount = 0
			proxy.IsHealthy = true
			break
		}
	}

	// Save to Redis asynchronously
	go func() {
		if err := pm.SaveProxiesToRedis(); err != nil {
			log.Printf("Failed to save proxies to Redis: %v", err)
		}
	}()
}

// MarkProxyFailure marks a proxy as failed
func (pm *ProxyManager) MarkProxyFailure(proxyURL string) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for _, proxy := range pm.proxies {
		if proxy.URL == proxyURL {
			proxy.FailCount++
			if proxy.FailCount >= 3 {
				proxy.IsHealthy = false
			}
			break
		}
	}

	// Save to Redis asynchronously
	go func() {
		if err := pm.SaveProxiesToRedis(); err != nil {
			log.Printf("Failed to save proxies to Redis: %v", err)
		}
	}()
}

// CheckProxyHealth checks if a proxy is healthy
func (pm *ProxyManager) CheckProxyHealth(proxy *Proxy) bool {
	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	start := time.Now()
	resp, err := client.Get("https://httpbin.org/ip")
	if err != nil {
		log.Printf("Proxy %s health check failed: %v", proxy.URL, err)
		return false
	}
	defer func() {
		if closeErr := resp.Body.Close(); closeErr != nil {
			log.Printf("Failed to close response body: %v", closeErr)
		}
	}()

	latency := time.Since(start).Milliseconds()
	proxy.AvgLatency = int(latency)

	return resp.StatusCode == 200
}

// RunHealthChecks runs periodic health checks on all proxies
func (pm *ProxyManager) RunHealthChecks() {
	ticker := time.NewTicker(pm.healthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		log.Println("Running proxy health checks...")
		pm.mu.Lock()

		for _, proxy := range pm.proxies {
			go func(p *Proxy) {
				isHealthy := pm.CheckProxyHealth(p)
				p.IsHealthy = isHealthy
				if !isHealthy {
					p.FailCount++
				} else {
					p.FailCount = 0
				}
			}(proxy)
		}

		pm.mu.Unlock()

		// Save updated proxies to Redis
		time.Sleep(2 * time.Second) // Wait for health checks to complete
		if err := pm.SaveProxiesToRedis(); err != nil {
			log.Printf("Failed to save proxies to Redis: %v", err)
		}

		log.Println("Health checks completed")
	}
}

// AddProxy adds a new proxy
func (pm *ProxyManager) AddProxy(proxy *Proxy) {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	proxy.IsHealthy = true
	proxy.LastUsed = time.Now()
	pm.proxies = append(pm.proxies, proxy)

	go func() {
		if err := pm.SaveProxiesToRedis(); err != nil {
			log.Printf("Failed to save proxies to Redis: %v", err)
		}
	}()
}

// RemoveProxy removes a proxy by URL
func (pm *ProxyManager) RemoveProxy(proxyURL string) bool {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	for i, proxy := range pm.proxies {
		if proxy.URL == proxyURL {
			pm.proxies = append(pm.proxies[:i], pm.proxies[i+1:]...)
			go func() {
				if err := pm.SaveProxiesToRedis(); err != nil {
					log.Printf("Failed to save proxies to Redis: %v", err)
				}
			}()
			return true
		}
	}

	return false
}

// GetAllProxies returns all proxies
func (pm *ProxyManager) GetAllProxies() []*Proxy {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	proxies := make([]*Proxy, len(pm.proxies))
	copy(proxies, pm.proxies)
	return proxies
}

// GetStats returns proxy statistics
func (pm *ProxyManager) GetStats() map[string]interface{} {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	healthy := 0
	unhealthy := 0
	totalSuccess := 0
	totalFails := 0

	for _, proxy := range pm.proxies {
		if proxy.IsHealthy {
			healthy++
		} else {
			unhealthy++
		}
		totalSuccess += proxy.SuccessCount
		totalFails += proxy.FailCount
	}

	return map[string]interface{}{
		"total_proxies":     len(pm.proxies),
		"healthy_proxies":   healthy,
		"unhealthy_proxies": unhealthy,
		"total_success":     totalSuccess,
		"total_failures":    totalFails,
	}
}

func main() {
	// Initialize Redis client
	redisHost := os.Getenv("REDIS_HOST")
	if redisHost == "" {
		redisHost = "localhost"
	}
	redisPort := os.Getenv("REDIS_PORT")
	if redisPort == "" {
		redisPort = "6379"
	}
	redisAddr := fmt.Sprintf("%s:%s", redisHost, redisPort)

	redisClient := redis.NewClient(&redis.Options{
		Addr:     redisAddr,
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       0,
	})

	// Test Redis connection
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v", err)
	}
	log.Println("Connected to Redis")

	// Initialize proxy manager
	manager = NewProxyManager(redisClient)

	// Load proxies
	if err := manager.LoadProxiesFromRedis(); err != nil {
		log.Printf("Warning: Failed to load proxies from Redis: %v", err)
	}

	if err := manager.LoadProxiesFromEnv(); err != nil {
		log.Fatalf("Failed to load proxies from environment: %v", err)
	}

	// Start health checks in background
	go manager.RunHealthChecks()

	// Initialize Fiber app
	app := fiber.New(fiber.Config{
		AppName:               "Proxy Manager Service",
		DisableStartupMessage: false,
	})

	// Middleware
	app.Use(recover.New())
	app.Use(logger.New())
	app.Use(cors.New(cors.Config{
		AllowOrigins: "*",
		AllowMethods: "GET,POST,PUT,DELETE",
	}))

	// Health check
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{
			"status":  "healthy",
			"service": "proxy-manager",
		})
	})

	// Get next proxy
	app.Get("/api/v1/proxy/next", func(c *fiber.Ctx) error {
		proxy, err := manager.GetNextProxy()
		if err != nil {
			return c.Status(fiber.StatusServiceUnavailable).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(proxy)
	})

	// Get proxy by country
	app.Get("/api/v1/proxy/country/:country", func(c *fiber.Ctx) error {
		country := c.Params("country")
		proxy, err := manager.GetProxyByCountry(country)
		if err != nil {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
				"error": err.Error(),
			})
		}
		return c.JSON(proxy)
	})

	// Get all proxies
	app.Get("/api/v1/proxies", func(c *fiber.Ctx) error {
		proxies := manager.GetAllProxies()
		return c.JSON(proxies)
	})

	// Get proxy stats
	app.Get("/api/v1/proxies/stats", func(c *fiber.Ctx) error {
		stats := manager.GetStats()
		return c.JSON(stats)
	})

	// Add proxy
	app.Post("/api/v1/proxies", func(c *fiber.Ctx) error {
		var proxy Proxy
		if err := c.BodyParser(&proxy); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		manager.AddProxy(&proxy)
		return c.Status(fiber.StatusCreated).JSON(proxy)
	})

	// Remove proxy
	app.Delete("/api/v1/proxies/:url", func(c *fiber.Ctx) error {
		proxyURL := c.Params("url")
		if manager.RemoveProxy(proxyURL) {
			return c.JSON(fiber.Map{"message": "Proxy removed"})
		}
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "Proxy not found",
		})
	})

	// Mark proxy success
	app.Post("/api/v1/proxy/success", func(c *fiber.Ctx) error {
		var req struct {
			ProxyURL string `json:"proxy_url"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		manager.MarkProxySuccess(req.ProxyURL)
		return c.JSON(fiber.Map{"message": "Proxy marked as successful"})
	})

	// Mark proxy failure
	app.Post("/api/v1/proxy/failure", func(c *fiber.Ctx) error {
		var req struct {
			ProxyURL string `json:"proxy_url"`
		}
		if err := c.BodyParser(&req); err != nil {
			return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
				"error": "Invalid request body",
			})
		}

		manager.MarkProxyFailure(req.ProxyURL)
		return c.JSON(fiber.Map{"message": "Proxy marked as failed"})
	})

	// Start server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8000"
	}

	log.Printf("Proxy Manager listening on port %s", port)
	if err := app.Listen(":" + port); err != nil {
		log.Fatalf("Failed to start server: %v", err)
	}
}
