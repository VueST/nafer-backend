package config

import (
	"fmt"
	"os"
)

// Config holds all configuration for the identity service.
// All values come from environment variables. The service will NOT START
// if any required value is missing — fail fast, fail loudly.
type Config struct {
	// HTTP server
	Port string

	// PostgreSQL
	DatabaseURL string

	// Redis — used for JWT denylist + refresh token storage
	RedisAddr string

	// JWT
	JWTSecret       string
	AccessTokenTTL  string // e.g. "15m"
	RefreshTokenTTL string // e.g. "168h" (7 days)
}

// Load reads configuration from environment variables and validates them.
func Load() (*Config, error) {
	cfg := &Config{
		Port:            getOrDefault("PORT", "8080"),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		RedisAddr:       getOrDefault("REDIS_URL", "localhost:6379"), // REDIS_URL kept for backward compat
		JWTSecret:       os.Getenv("JWT_SECRET"),
		AccessTokenTTL:  getOrDefault("ACCESS_TOKEN_TTL", "15m"),
		RefreshTokenTTL: getOrDefault("REFRESH_TOKEN_TTL", "168h"),
	}

	if cfg.DatabaseURL == "" {
		return nil, fmt.Errorf("DATABASE_URL is required")
	}
	if cfg.JWTSecret == "" {
		return nil, fmt.Errorf("JWT_SECRET is required")
	}
	if len(cfg.JWTSecret) < 32 {
		return nil, fmt.Errorf("JWT_SECRET must be at least 32 characters long")
	}

	return cfg, nil
}

func getOrDefault(key, defaultValue string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return defaultValue
}
