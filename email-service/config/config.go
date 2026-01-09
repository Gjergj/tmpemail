package config

import (
	"os"
	"strconv"
)

// Config holds the email service configuration
type Config struct {
	// SMTP Server
	SMTPPort string
	SMTPHost string

	// Health check HTTP server
	HealthPort string

	// Storage
	StoragePath string

	// API Service
	APIServiceURL string

	// Email limits
	MaxEmailSize int // in bytes

	// TLS Settings
	TLSEnabled  bool   // Enable TLS/STARTTLS
	TLSCertPath string // Path to TLS certificate file
	TLSKeyPath  string // Path to TLS private key file
}

// Load loads configuration from environment variables with defaults
func Load() *Config {
	return &Config{
		SMTPPort:      getEnv("TMPEMAIL_SMTP_PORT", "2525"),
		SMTPHost:      getEnv("TMPEMAIL_SMTP_HOST", "0.0.0.0"),
		HealthPort:    getEnv("TMPEMAIL_HEALTH_PORT", "8081"),
		StoragePath:   getEnv("TMPEMAIL_STORAGE_PATH", "./mail"),
		APIServiceURL: getEnv("TMPEMAIL_API_URL", "http://localhost:8080"),
		MaxEmailSize:  getIntEnv("TMPEMAIL_MAX_EMAIL_SIZE", 20*1024*1024), // 20MB default
		TLSEnabled:    getBoolEnv("TMPEMAIL_TLS_ENABLED", false),
		TLSCertPath:   getEnv("TMPEMAIL_TLS_CERT_PATH", "./certs/smtp.crt"),
		TLSKeyPath:    getEnv("TMPEMAIL_TLS_KEY_PATH", "./certs/smtp.key"),
	}
}

// getBoolEnv retrieves a bool environment variable or returns a default value
func getBoolEnv(key string, defaultValue bool) bool {
	if value := os.Getenv(key); value != "" {
		return value == "true" || value == "1" || value == "yes"
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

// getIntEnv retrieves an int environment variable or returns a default value
func getIntEnv(key string, defaultValue int) int {
	if value := os.Getenv(key); value != "" {
		if intVal, err := strconv.Atoi(value); err == nil {
			return intVal
		}
	}
	return defaultValue
}
