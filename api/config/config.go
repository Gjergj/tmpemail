package config

import (
	"os"
	"strconv"
	"strings"
	"time"
)

// Config holds the application configuration
type Config struct {
	// Database
	DBPath string

	// Server
	Port string

	// Domain
	EmailDomain string

	// Storage
	StoragePath string

	// Expiration
	DefaultExpiration time.Duration

	// Rate limiting
	RateLimitGenerate int // Rate limit for /api/v1/generate (per minute)
	RateLimitAPI      int // Rate limit for other API endpoints (per minute)
	RateLimitWS       int // Rate limit for WebSocket connections (per minute)

	// CORS
	AllowedOrigins []string

	// Cleanup
	CleanupInterval time.Duration

	// Storage quota
	StorageQuotaPerAddress int64 // Max storage per address in bytes (0 = unlimited)
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		DBPath:                 getEnv("TMPEMAIL_DB_PATH", "/var/lib/tmpemail/tmpemail.db"),
		Port:                   getEnv("TMPEMAIL_PORT", "8080"),
		EmailDomain:            getEnv("TMPEMAIL_DOMAIN", "tmpemail.xyz"),
		StoragePath:            getEnv("TMPEMAIL_STORAGE_PATH", "/var/mail/tmpemail"),
		DefaultExpiration:      getDurationEnv("TMPEMAIL_DEFAULT_EXPIRATION", 1*time.Hour),
		RateLimitGenerate:      getIntEnv("TMPEMAIL_RATE_LIMIT_GENERATE", 10), // 10 req/min for generate
		RateLimitAPI:           getIntEnv("TMPEMAIL_RATE_LIMIT_API", 60),      // 60 req/min for email retrieval
		RateLimitWS:            getIntEnv("TMPEMAIL_RATE_LIMIT_WS", 5),        // 5 connections/min for WebSocket
		AllowedOrigins:         getEnvList("TMPEMAIL_ALLOWED_ORIGINS", []string{"http://localhost:5173", "http://localhost:3000"}),
		CleanupInterval:        getDurationEnv("TMPEMAIL_CLEANUP_INTERVAL", 5*time.Minute),
		StorageQuotaPerAddress: getInt64Env("TMPEMAIL_STORAGE_QUOTA", 50*1024*1024), // 50MB default
	}
}

// getEnvList retrieves a comma-separated list from environment variable or returns default
func getEnvList(key string, defaultValue []string) []string {
	if value := os.Getenv(key); value != "" {
		parts := strings.Split(value, ",")
		result := make([]string, 0, len(parts))
		for _, p := range parts {
			if trimmed := strings.TrimSpace(p); trimmed != "" {
				result = append(result, trimmed)
			}
		}
		if len(result) > 0 {
			return result
		}
	}
	return defaultValue
}

// getEnv retrieves an environment variable or returns a default value
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getIntEnv retrieves an integer environment variable or returns a default value
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getInt64Env retrieves an int64 environment variable or returns a default value
func getInt64Env(key string, defaultValue int64) int64 {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.ParseInt(value, 10, 64); err == nil {
			return intVal
		}
	}
	return defaultValue
}

// getDurationEnv retrieves a duration environment variable or returns a default value
func getDurationEnv(key string, defaultValue time.Duration) time.Duration {
	if value := os.Getenv(key); value != "" {
		if duration, err := time.ParseDuration(value); err == nil {
			return duration
		}
	}
	return defaultValue
}
