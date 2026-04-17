package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"nafer/comment/internal/config"
	"nafer/comment/internal/handler"
	"nafer/comment/internal/repository"
	"nafer/comment/internal/server"
	"nafer/comment/internal/service"
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
	// No retry loop here.
	// Docker Compose healthcheck ensures Postgres is ready before this starts.
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

	// --- Dependency Wiring (Composition Root) ---
	commentRepo := repository.NewPostgresCommentRepository(db)
	commentSvc := service.NewCommentService(commentRepo)
	commentHandler := handler.NewCommentHandler(commentSvc, log)
	srv := server.New(commentHandler, cfg.Port, log)

	// --- Run (blocks until SIGTERM/SIGINT) ---
	srv.Run()
}
