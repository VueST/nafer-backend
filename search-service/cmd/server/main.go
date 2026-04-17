package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/joho/godotenv"
	"github.com/meilisearch/meilisearch-go"
	"github.com/redis/go-redis/v9"

	"nafer/search/internal/config"
	"nafer/search/internal/handler"
	"nafer/search/internal/server"
	"nafer/search/internal/service"
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

	// --- Redis ---
	redisClient := redis.NewClient(&redis.Options{
		Addr: cfg.RedisURL,
	})
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	log.Info("redis connected")

	// --- Meilisearch ---
	meiliClient := meilisearch.NewClient(meilisearch.ClientConfig{
		Host:   cfg.MeiliURL,
		APIKey: cfg.MeiliKey,
	})

	// Verify Meilisearch connection
	if _, err := meiliClient.Health(); err != nil {
		log.Error("failed to connect to meilisearch", "error", err)
		os.Exit(1)
	}
	log.Info("meilisearch connected")

	// --- Dependency Wiring (Composition Root) ---
	// NewSearchService also configures the index (idempotent)
	searchSvc, err := service.NewSearchService(meiliClient, log)
	if err != nil {
		log.Error("failed to initialise search service", "error", err)
		os.Exit(1)
	}

	searchHandler := handler.NewSearchHandler(searchSvc, log)
	srv := server.New(searchHandler, cfg.Port, log)

	// --- Run (blocks until SIGTERM/SIGINT) ---
	srv.Run()
}
