package repository

import (
	"context"

	"nafer/streaming/internal/domain"
)

// VideoRepository defines what the service layer needs from a data store.
type VideoRepository interface {
	// Create persists a new video record.
	Create(ctx context.Context, v *domain.Video) (*domain.Video, error)

	// FindByID retrieves a video by its ID.
	// Returns nil, nil if not found.
	FindByID(ctx context.Context, id string) (*domain.Video, error)

	// UpdateStatus updates the processing status (and optional error message) of a video.
	UpdateStatus(ctx context.Context, id string, status domain.VideoStatus, errMsg string) error

	// UpdateHLSPath stores the final HLS playlist path after transcoding succeeds.
	UpdateHLSPath(ctx context.Context, id, hlsPath string) error

	// List returns videos with pagination, optionally filtered by uploaderID.
	List(ctx context.Context, uploaderID string, limit, offset int) ([]*domain.Video, error)
}
