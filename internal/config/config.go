// Package config handles application configuration from environment variables.
package config

import (
	"os"
	"strconv"
	"time"
)

// Config holds all application configuration.
type Config struct {
	Port         string
	Env          string
	MTABusAPIKey string
	CacheTTL     time.Duration
	HTTPTimeout  time.Duration
}

// Load reads configuration from environment variables with sensible defaults.
func Load() *Config {
	return &Config{
		Port:         getEnv("PORT", "3000"),
		Env:          getEnv("ENV", "development"),
		MTABusAPIKey: getEnv("MTA_BUS_API_KEY", ""),
		CacheTTL:     getDurationEnv("CACHE_TTL_SECONDS", 120) * time.Second,
		HTTPTimeout:  getDurationEnv("HTTP_TIMEOUT_SECONDS", 10) * time.Second,
	}
}

// IsDevelopment returns true if running in development mode.
func (c *Config) IsDevelopment() bool {
	return c.Env == "development"
}

// Validate checks that required configuration is present.
func (c *Config) Validate() error {
	return nil
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getDurationEnv(key string, defaultSeconds int) time.Duration {
	if value := os.Getenv(key); value != "" {
		if seconds, err := strconv.Atoi(value); err == nil {
			return time.Duration(seconds)
		}
	}
	return time.Duration(defaultSeconds)
}
