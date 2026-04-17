package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the search service.
// The service WILL NOT START if any required value is missing (Fail Fast).
type Config struct {
	Port     string
	RedisURL string
	MeiliURL string
	MeiliKey string
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Port:     getOrDefault("PORT", "8080"),
		RedisURL: os.Getenv("REDIS_URL"),
		MeiliURL: getOrDefault("MEILI_URL", "http://meilisearch:7700"),
		MeiliKey: os.Getenv("MEILI_KEY"),
	}

	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required")
	}
	if cfg.MeiliKey == "" {
		return nil, fmt.Errorf("MEILI_KEY is required")
	}

	return cfg, nil
}

func getOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
