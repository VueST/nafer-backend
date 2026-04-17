package main

import (
	"context"
	"log/slog"
	"os"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"nafer/identity/internal/config"
	"nafer/identity/internal/handler"
	"nafer/identity/internal/repository"
	"nafer/identity/internal/server"
	"nafer/identity/internal/service"
)

func main() {
	// Structured JSON logging — identical format in dev and production.
	log := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelInfo,
	}))

	// Load .env in development — Docker injects env vars directly in production.
	_ = godotenv.Load()

	// ── Config: Fail Fast ─────────────────────────────────────────────────────
	cfg, err := config.Load()
	if err != nil {
		log.Error("invalid configuration — cannot start", "error", err)
		os.Exit(1)
	}
	log.Info("configuration loaded",
		"port", cfg.Port,
		"access_token_ttl", cfg.AccessTokenTTL,
		"refresh_token_ttl", cfg.RefreshTokenTTL,
	)

	// ── PostgreSQL ────────────────────────────────────────────────────────────
	// Docker Compose healthcheck ensures Postgres is ready before this process starts.
	db, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to create postgres pool", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Error("failed to ping postgres", "error", err)
		os.Exit(1)
	}
	log.Info("postgres connected")

	// ── Redis ─────────────────────────────────────────────────────────────────
	rdb := redis.NewClient(&redis.Options{
		Addr:         cfg.RedisAddr,
		DialTimeout:  5 * time.Second,
		ReadTimeout:  3 * time.Second,
		WriteTimeout: 3 * time.Second,
	})
	defer rdb.Close()

	if err := rdb.Ping(context.Background()).Err(); err != nil {
		log.Error("failed to ping redis", "addr", cfg.RedisAddr, "error", err)
		os.Exit(1)
	}
	log.Info("redis connected", "addr", cfg.RedisAddr)

	// ── Parse TTLs ────────────────────────────────────────────────────────────
	accessTTL, err := time.ParseDuration(cfg.AccessTokenTTL)
	if err != nil {
		log.Error("invalid ACCESS_TOKEN_TTL format", "value", cfg.AccessTokenTTL)
		os.Exit(1)
	}
	refreshTTL, err := time.ParseDuration(cfg.RefreshTokenTTL)
	if err != nil {
		log.Error("invalid REFRESH_TOKEN_TTL format", "value", cfg.RefreshTokenTTL)
		os.Exit(1)
	}

	// ── Dependency Wiring (Composition Root) ──────────────────────────────────
	// All dependencies are wired here and only here.
	// Adding a new service means adding one line here — nothing else changes.
	userRepo    := repository.NewPostgresUserRepository(db)
	tokenSvc    := service.NewTokenService(rdb, refreshTTL)
	authSvc     := service.NewAuthService(userRepo, tokenSvc, cfg.JWTSecret, accessTTL)
	authHandler := handler.NewAuthHandler(authSvc, log)
	srv         := server.New(authHandler, tokenSvc, cfg.JWTSecret, cfg.Port, log)

	// ── Run ───────────────────────────────────────────────────────────────────
	srv.Run()
}
