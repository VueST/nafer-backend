package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"nafer/streaming/internal/domain"
	"nafer/streaming/internal/repository"
)

const transcodeQueue = "nafer:transcode:queue"

// VideoService contains business logic for video management and job dispatch.
type VideoService struct {
	videos repository.VideoRepository
	redis  *redis.Client
	log    *slog.Logger
}

// NewVideoService constructs the service with its dependencies.
func NewVideoService(videos repository.VideoRepository, redis *redis.Client, log *slog.Logger) *VideoService {
	return &VideoService{videos: videos, redis: redis, log: log}
}

// CreateVideoInput holds the data needed to register a new video upload.
type CreateVideoInput struct {
	UploaderID  string
	Title       string
	Description string
	SourcePath  string // MinIO object key of the uploaded source file
}

// CreateVideo registers a new video and dispatches a transcoding job to the queue.
func (s *VideoService) CreateVideo(ctx context.Context, input CreateVideoInput) (*domain.Video, error) {
	if input.UploaderID == "" || input.SourcePath == "" {
		return nil, fmt.Errorf("uploader_id and source_path are required")
	}

	now := time.Now().UTC()
	v := &domain.Video{
		ID:          uuid.NewString(),
		UploaderID:  input.UploaderID,
		Title:       input.Title,
		Description: input.Description,
		SourcePath:  input.SourcePath,
		Status:      domain.VideoStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	created, err := s.videos.Create(ctx, v)
	if err != nil {
		return nil, fmt.Errorf("creating video record: %w", err)
	}

	// Dispatch transcoding job to Redis queue (LPUSH for FIFO processing)
	job := domain.TranscodeJob{
		VideoID:    created.ID,
		SourcePath: created.SourcePath,
		OutputDir:  fmt.Sprintf("hls/%s", created.ID),
	}
	payload, _ := json.Marshal(job)
	if err := s.redis.LPush(ctx, transcodeQueue, payload).Err(); err != nil {
		// Non-fatal for the API — log and return. The video is in "pending" state.
		// A separate reconciler or retry mechanism can re-enqueue failed jobs.
		s.log.Error("failed to enqueue transcode job", "video_id", created.ID, "error", err)
	} else {
		s.log.Info("transcode job enqueued", "video_id", created.ID)
	}

	return created, nil
}

// GetVideo retrieves a single video by ID.
func (s *VideoService) GetVideo(ctx context.Context, id string) (*domain.Video, error) {
	v, err := s.videos.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("finding video: %w", err)
	}
	if v == nil {
		return nil, fmt.Errorf("video not found")
	}
	return v, nil
}

// ListVideos retrieves paginated videos, optionally filtered by uploader.
func (s *VideoService) ListVideos(ctx context.Context, uploaderID string, limit, offset int) ([]*domain.Video, error) {
	if limit <= 0 || limit > 100 {
		limit = 20
	}
	return s.videos.List(ctx, uploaderID, limit, offset)
}
