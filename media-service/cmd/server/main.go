package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"

	"nafer/media/internal/config"
	"nafer/media/internal/handler"
	"nafer/media/internal/repository"
	"nafer/media/internal/server"
	"nafer/media/internal/service"
	"nafer/media/internal/storage"
)

func main() {
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

	_ = godotenv.Load()

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

	// --- Object Storage (MinIO) ---
	store, err := storage.NewMinioProvider(
		cfg.MinioEndpoint,
		cfg.MinioAccessKey,
		cfg.MinioSecretKey,
		cfg.MinioBucket,
		cfg.MinioUseSSL,
	)
	if err != nil {
		log.Error("failed to create minio client", "error", err)
		os.Exit(1)
	}

	// Ensure the bucket exists before accepting traffic
	if err := store.EnsureBucket(context.Background(), cfg.MinioBucket); err != nil {
		log.Error("failed to ensure storage bucket", "bucket", cfg.MinioBucket, "error", err)
		os.Exit(1)
	}
	log.Info("storage ready", "bucket", cfg.MinioBucket)

	// --- Dependency Wiring ---
	mediaRepo    := repository.NewPostgresMediaRepository(db)
	uploadSvc    := service.NewUploadService(mediaRepo, store, cfg.MinioBucket)
	mediaHandler := handler.NewMediaHandler(uploadSvc, log)
	srv          := server.New(mediaHandler, cfg.Port, log)

	srv.Run()
}
