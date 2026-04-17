package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the streaming service.
// The service WILL NOT START if any required value is missing (Fail Fast).
type Config struct {
	Port              string
	DatabaseURL       string
	RedisURL          string
	MinioEndpoint     string
	MinioRootUser     string
	MinioRootPassword string
	MinioBucket       string
	MinioUseSSL       bool
}

// Load reads configuration from environment variables.
func Load() (*Config, error) {
	cfg := &Config{
		Port:              getOrDefault("PORT", "8080"),
		DatabaseURL:       os.Getenv("DATABASE_URL"),
		RedisURL:          os.Getenv("REDIS_URL"),
		MinioEndpoint:     getOrDefault("MINIO_ENDPOINT", "minio:9000"),
		MinioRootUser:     os.Getenv("MINIO_ROOT_USER"),
		MinioRootPassword: os.Getenv("MINIO_ROOT_PASSWORD"),
		MinioBucket:       getOrDefault("STREAMING_BUCKET", "nafer-streaming"),
		MinioUseSSL:       false,
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.RedisURL == "" {
		return nil, fmt.Errorf("REDIS_URL is required")
	}
	if cfg.MinioRootUser == "" {
		return nil, fmt.Errorf("MINIO_ROOT_USER is required")
	}
	if cfg.MinioRootPassword == "" {
		return nil, fmt.Errorf("MINIO_ROOT_PASSWORD is required")
	}

	return cfg, nil
}

func getOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
