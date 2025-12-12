// Package config manages application configuration settings
package config

import (
	"fmt"
	"os"
	"strconv"
)

// Config holds all application configuration
type Config struct {
	ClickHouse ClickHouseConfig
	Database   DatabaseConfig
	Typesense  TypesenseConfig
	Services   ServicesConfig
	App        AppConfig
	Redis      RedisConfig
}

// AppConfig contains application-level settings
type AppConfig struct {
	Name        string
	Environment string
	Port        string
	LogLevel    string
}

// DatabaseConfig contains PostgreSQL database settings
type DatabaseConfig struct {
	Host     string
	User     string
	Password string
	Database string
	SSLMode  string
	Port     int
}

// ClickHouseConfig contains ClickHouse database settings
type ClickHouseConfig struct {
	Host     string
	Database string
	User     string
	Password string
	Port     int
}

// TypesenseConfig contains Typesense search engine settings
type TypesenseConfig struct {
	Host   string
	APIKey string
	Port   int
}

// RedisConfig contains Redis cache settings
type RedisConfig struct {
	Host string
	Port int
}

// ServicesConfig contains URLs for microservices
type ServicesConfig struct {
	CrawlerURL string
	ParserURL  string
	OSINTUrl   string
}

// Load reads configuration from environment variables
func Load() (*Config, error) {
	cfg := &Config{
		App: AppConfig{
			Name:        getEnv("APP_NAME", "QuietHire API"),
			Environment: getEnv("ENV", "development"),
			Port:        getEnv("API_PORT", "3000"),
			LogLevel:    getEnv("LOG_LEVEL", "info"),
		},
		Database: DatabaseConfig{
			Host:     getEnv("DB_HOST", "localhost"),
			Port:     getEnvAsInt("DB_PORT", 5432),
			User:     getEnv("DB_USER", "quiethire"),
			Password: getEnv("DB_PASSWORD", ""),
			Database: getEnv("DB_NAME", "quiethire"),
			SSLMode:  getEnv("DB_SSL_MODE", "disable"),
		},
		ClickHouse: ClickHouseConfig{
			Host:     getEnv("CLICKHOUSE_HOST", "localhost"),
			Port:     getEnvAsInt("CLICKHOUSE_PORT", 9000),
			Database: getEnv("CLICKHOUSE_DB", "quiethire"),
			User:     getEnv("CLICKHOUSE_USER", "default"),
			Password: getEnv("CLICKHOUSE_PASSWORD", ""),
		},
		Typesense: TypesenseConfig{
			Host:   getEnv("TYPESENSE_HOST", "localhost"),
			Port:   getEnvAsInt("TYPESENSE_PORT", 8108),
			APIKey: getEnv("TYPESENSE_API_KEY", ""),
		},
		Redis: RedisConfig{
			Host: getEnv("REDIS_HOST", "localhost"),
			Port: getEnvAsInt("REDIS_PORT", 6379),
		},
		Services: ServicesConfig{
			CrawlerURL: getEnv("CRAWLER_SERVICE_URL", "http://localhost:8002"),
			ParserURL:  getEnv("PARSER_SERVICE_URL", "http://localhost:8001"),
			OSINTUrl:   getEnv("OSINT_SERVICE_URL", "http://localhost:8004"),
		},
	}
	if cfg.Database.Password == "" {
		return nil, fmt.Errorf("DB_PASSWORD is required")
	}
	if cfg.Typesense.APIKey == "" {
		return nil, fmt.Errorf("TYPESENSE_API_KEY is required")
	}
	return cfg, nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvAsInt(key string, defaultValue int) int {
	valueStr := os.Getenv(key)
	if value, err := strconv.Atoi(valueStr); err == nil {
		return value
	}
	return defaultValue
}
