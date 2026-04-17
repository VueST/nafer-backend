package config

import (
	"fmt"
	"os"
)

type Config struct {
	Port     string
	DatabaseURL string
	RedisURL    string

	// MinIO / Storage
	MinioEndpoint  string
	MinioAccessKey string
	MinioSecretKey string
	MinioBucket    string
	MinioUseSSL    bool
}

func Load() (*Config, error) {
	cfg := &Config{
		Port:           getOrDefault("PORT", "8080"),
		DatabaseURL:    os.Getenv("DATABASE_URL"),
		RedisURL:       os.Getenv("REDIS_URL"),
		MinioEndpoint:  os.Getenv("MINIO_ENDPOINT"),
		MinioAccessKey: os.Getenv("MINIO_ROOT_USER"),
		MinioSecretKey: os.Getenv("MINIO_ROOT_PASSWORD"),
		MinioBucket:    getOrDefault("MEDIA_BUCKET", "nafer-media"),
		MinioUseSSL:    false,
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.MinioEndpoint == "" {
		return nil, fmt.Errorf("MINIO_ENDPOINT is required")
	}
	if cfg.MinioAccessKey == "" || cfg.MinioSecretKey == "" {
		return nil, fmt.Errorf("MINIO_ROOT_USER and MINIO_ROOT_PASSWORD are required")
	}

	return cfg, nil
}

func getOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
