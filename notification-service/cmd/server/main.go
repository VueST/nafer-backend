package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/redis/go-redis/v9"

	"nafer/notification/internal/config"
	"nafer/notification/internal/handler"
	"nafer/notification/internal/repository"
	"nafer/notification/internal/server"
	"nafer/notification/internal/service"
)

func main() {
	// --- Structured Logger (JSON output) ---
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	// --- Load .env (ignored in production — env vars come from Docker) ---
	_ = godotenv.Load()

	// --- Config: Fail Fast ---
	cfg, err := config.Load()
	if err != nil {
		log.Error("invalid configuration", "error", err)
		os.Exit(1)
	}

	// --- Database ---
	db, err := pgxpool.New(context.Background(), cfg.DatabaseURL)
	if err != nil {
		log.Error("failed to create db pool", "error", err)
		os.Exit(1)
	}
	defer db.Close()

	if err := db.Ping(context.Background()); err != nil {
		log.Error("failed to ping database", "error", err)
		os.Exit(1)
	}
	log.Info("database connected")

	// --- Redis ---
	// Parse "host:port" format from REDIS_URL env var
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	log.Info("redis connected")

	// --- Dependency Wiring (Composition Root) ---
	notifRepo := repository.NewPostgresNotificationRepository(db)
	notifSvc := service.NewNotificationService(notifRepo, redisClient, log)
	notifHandler := handler.NewNotificationHandler(notifSvc, redisClient, log)
	srv := server.New(notifHandler, cfg.Port, log)

	// --- Run (blocks until SIGTERM/SIGINT) ---
	srv.Run()
}
