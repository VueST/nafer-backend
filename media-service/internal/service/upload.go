package service

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"

	"nafer/media/internal/domain"
	"nafer/media/internal/repository"
	"nafer/media/internal/storage"
)

// UploadService handles all file upload business logic.
// Depends only on interfaces — not on MinIO or Postgres directly.
type UploadService struct {
	media   repository.MediaRepository
	storage storage.StorageProvider
	bucket  string
}

func NewUploadService(media repository.MediaRepository, store storage.StorageProvider, bucket string) *UploadService {
	return &UploadService{media: media, storage: store, bucket: bucket}
}

// UploadInput contains the data needed to upload a file.
type UploadInput struct {
	OwnerID     string
	Filename    string
	ContentType string
	Size        int64
	Reader      io.Reader
}

// UploadResult is returned after a successful upload.
type UploadResult struct {
	Media *domain.Media
	URL   string
}

// Upload stores a file and creates a media record.
func (s *UploadService) Upload(ctx context.Context, input UploadInput) (*UploadResult, error) {
	id := uuid.NewString()
	// Use a namespaced key to avoid collisions: ownerID/uuid/filename
	storageKey := fmt.Sprintf("%s/%s/%s", input.OwnerID, id, input.Filename)

	now := time.Now().UTC()
	media := &domain.Media{
		ID:          id,
		OwnerID:     input.OwnerID,
		Filename:    input.Filename,
		ContentType: input.ContentType,
		SizeBytes:   input.Size,
		StorageKey:  storageKey,
		Status:      domain.MediaStatusPending,
		CreatedAt:   now,
		UpdatedAt:   now,
	}

	// Persist metadata first (before uploading)
	created, err := s.media.Create(ctx, media)
	if err != nil {
		return nil, fmt.Errorf("create media record: %w", err)
	}

	// Upload to object storage
	url, err := s.storage.Upload(ctx, storage.UploadInput{
		Key:         storageKey,
		Reader:      input.Reader,
		Size:        input.Size,
		ContentType: input.ContentType,
	})
	if err != nil {
		// Mark as failed if upload fails
		_ = s.media.UpdateStatus(ctx, created.ID, domain.MediaStatusFailed)
		return nil, fmt.Errorf("upload to storage: %w", err)
	}

	// Mark as uploaded
	if err := s.media.UpdateStatus(ctx, created.ID, domain.MediaStatusUploaded); err != nil {
		return nil, fmt.Errorf("update media status: %w", err)
	}
	created.Status = domain.MediaStatusUploaded

	return &UploadResult{Media: created, URL: url}, nil
}

// GetByID retrieves a media record and its presigned download URL.
func (s *UploadService) GetByID(ctx context.Context, id string) (*UploadResult, error) {
	media, err := s.media.FindByID(ctx, id)
	if err != nil {
		return nil, fmt.Errorf("find media: %w", err)
	}
	if media == nil {
		return nil, fmt.Errorf("media not found")
	}

	url, err := s.storage.GetURL(ctx, media.StorageKey)
	if err != nil {
		return nil, fmt.Errorf("get url: %w", err)
	}

	return &UploadResult{Media: media, URL: url}, nil
}
