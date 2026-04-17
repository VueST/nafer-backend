package queue

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/redis/go-redis/v9"

	"nafer/streaming/internal/domain"
	"nafer/streaming/internal/repository"
	"nafer/streaming/internal/transcoder"
)

const (
	transcodeQueue  = "nafer:transcode:queue"
	blockTimeout    = 5 * time.Second // BRPOP block duration
	tempBaseDir     = "/tmp/nafer-transcode"
)

// Worker is a long-running background process that:
//  1. Blocks on a Redis list (BRPOP) waiting for transcode jobs
//  2. Downloads the source video from MinIO
//  3. Runs FFmpeg transcoding to HLS
//  4. Uploads HLS segments back to MinIO
//  5. Updates the video status in Postgres
type Worker struct {
	redis      *redis.Client
	videos     repository.VideoRepository
	transcoder transcoder.Transcoder
	minio      *minio.Client
	bucket     string
	log        *slog.Logger
}

// NewWorker constructs the background worker with all dependencies injected.
func NewWorker(
	redis *redis.Client,
	videos repository.VideoRepository,
	transcoder transcoder.Transcoder,
	minio *minio.Client,
	bucket string,
	log *slog.Logger,
) *Worker {
	return &Worker{
		redis:      redis,
		videos:     videos,
		transcoder: transcoder,
		minio:      minio,
		bucket:     bucket,
		log:        log,
	}
}

// Run starts the worker loop. It processes jobs until ctx is cancelled.
// This is designed to run as a separate process (cmd/worker/main.go).
func (w *Worker) Run(ctx context.Context) {
	w.log.Info("transcode worker started", "queue", transcodeQueue)

	for {
		select {
		case <-ctx.Done():
			w.log.Info("transcode worker stopping")
			return
		default:
			w.processNext(ctx)
		}
	}
}

// processNext blocks for one job and processes it.
func (w *Worker) processNext(ctx context.Context) {
	// BRPOP blocks until a job arrives or the timeout elapses.
	// This avoids busy-waiting / CPU-spinning.
	result, err := w.redis.BRPop(ctx, blockTimeout, transcodeQueue).Result()
	if err != nil {
		if err == redis.Nil {
			return // Timeout, no job — loop again
		}
		w.log.Error("brpop error", "error", err)
		return
	}

	// result[0] = queue name, result[1] = payload
	var job domain.TranscodeJob
	if err := json.Unmarshal([]byte(result[1]), &job); err != nil {
		w.log.Error("invalid job payload", "error", err)
		return
	}

	w.log.Info("processing transcode job", "video_id", job.VideoID)

	if err := w.processJob(ctx, job); err != nil {
		w.log.Error("transcode job failed", "video_id", job.VideoID, "error", err)
		_ = w.videos.UpdateStatus(ctx, job.VideoID, domain.VideoStatusFailed, err.Error())
	}
}

// processJob executes the full transcode pipeline for one job.
func (w *Worker) processJob(ctx context.Context, job domain.TranscodeJob) error {
	// Mark as processing
	if err := w.videos.UpdateStatus(ctx, job.VideoID, domain.VideoStatusProcessing, ""); err != nil {
		return fmt.Errorf("updating status to processing: %w", err)
	}

	// Create temp working directory for this job
	workDir := filepath.Join(tempBaseDir, job.VideoID)
	defer os.RemoveAll(workDir) // Always clean up
	if err := os.MkdirAll(workDir, 0755); err != nil {
		return fmt.Errorf("creating work dir: %w", err)
	}

	// Download source from MinIO
	sourceLocalPath := filepath.Join(workDir, "source"+filepath.Ext(job.SourcePath))
	if err := w.minio.FGetObject(ctx, w.bucket, job.SourcePath, sourceLocalPath, minio.GetObjectOptions{}); err != nil {
		return fmt.Errorf("downloading source from minio: %w", err)
	}

	// Run FFmpeg
	outputDir := filepath.Join(workDir, "hls")
	masterPath, err := w.transcoder.Transcode(ctx, sourceLocalPath, outputDir)
	if err != nil {
		return fmt.Errorf("transcoding: %w", err)
	}

	// Upload all HLS files back to MinIO
	hlsMinioPrefix := job.OutputDir
	if err := w.uploadDir(ctx, outputDir, hlsMinioPrefix); err != nil {
		return fmt.Errorf("uploading hls to minio: %w", err)
	}

	// Compute the MinIO path to master.m3u8
	relMaster, _ := filepath.Rel(outputDir, masterPath)
	hlsMasterPath := hlsMinioPrefix + "/" + filepath.ToSlash(relMaster)

	// Update DB with HLS path — this also sets status = "ready"
	if err := w.videos.UpdateHLSPath(ctx, job.VideoID, hlsMasterPath); err != nil {
		return fmt.Errorf("updating hls path: %w", err)
	}

	w.log.Info("transcode complete", "video_id", job.VideoID, "hls_path", hlsMasterPath)
	return nil
}

// uploadDir recursively uploads all files in localDir to MinIO under the given prefix.
func (w *Worker) uploadDir(ctx context.Context, localDir, minioPrefix string) error {
	return filepath.Walk(localDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		rel, _ := filepath.Rel(localDir, path)
		objectName := minioPrefix + "/" + filepath.ToSlash(rel)
		_, err = w.minio.FPutObject(ctx, w.bucket, objectName, path, minio.PutObjectOptions{})
		return err
	})
}
