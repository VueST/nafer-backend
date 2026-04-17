package main

import (
	"context"
	"log/slog"
	"os"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"

	"nafer/streaming/internal/config"
	"nafer/streaming/internal/handler"
	"nafer/streaming/internal/repository"
	"nafer/streaming/internal/server"
	"nafer/streaming/internal/service"
)

func main() {
	// --- Structured Logger ---
	log := slog.New(slog.NewJSONHandler(os.Stdout, nil))

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
	redisClient := redis.NewClient(&redis.Options{Addr: cfg.RedisURL})
	defer redisClient.Close()

	if err := redisClient.Ping(context.Background()).Err(); err != nil {
		log.Error("failed to connect to redis", "error", err)
		os.Exit(1)
	}
	log.Info("redis connected")

	// --- MinIO ---
	minioClient, err := minio.New(cfg.MinioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.MinioRootUser, cfg.MinioRootPassword, ""),
		Secure: cfg.MinioUseSSL,
	})
	if err != nil {
		log.Error("failed to create minio client", "error", err)
		os.Exit(1)
	}

	// Ensure streaming bucket exists
	ctx := context.Background()
	exists, err := minioClient.BucketExists(ctx, cfg.MinioBucket)
	if err != nil {
		log.Error("failed to check minio bucket", "error", err)
		os.Exit(1)
	}
	if !exists {
		if err := minioClient.MakeBucket(ctx, cfg.MinioBucket, minio.MakeBucketOptions{}); err != nil {
			log.Error("failed to create minio bucket", "bucket", cfg.MinioBucket, "error", err)
			os.Exit(1)
		}
		log.Info("created minio bucket", "bucket", cfg.MinioBucket)
	}
	log.Info("minio connected", "bucket", cfg.MinioBucket)

	// --- Dependency Wiring (Composition Root) ---
	videoRepo := repository.NewPostgresVideoRepository(db)
	videoSvc := service.NewVideoService(videoRepo, redisClient, log)
	videoHandler := handler.NewVideoHandler(videoSvc, log)
	srv := server.New(videoHandler, cfg.Port, log)

	// --- Run (blocks until SIGTERM/SIGINT) ---
	srv.Run()
}
