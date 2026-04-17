package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/redis/go-redis/v9"

	"nafer/streaming/internal/config"
	"nafer/streaming/internal/queue"
	"nafer/streaming/internal/repository"
	"nafer/streaming/internal/transcoder"
)

// The worker process runs separately from the HTTP API server.
// It blocks on the Redis transcode queue (BRPOP) and processes jobs one at a time.
// Scale horizontally by launching multiple worker replicas.
func main() {
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
	log.Info("minio connected")

	// --- Dependency Wiring ---
	videoRepo := repository.NewPostgresVideoRepository(db)
	ffmpegTranscoder := transcoder.NewFFmpegTranscoder()
	worker := queue.NewWorker(redisClient, videoRepo, ffmpegTranscoder, minioClient, cfg.MinioBucket, log)

	// --- Graceful Shutdown ---
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("transcode worker process starting")

	// Run blocks until ctx is cancelled (SIGTERM received)
	worker.Run(ctx)

	log.Info("transcode worker process stopped cleanly")
}
