package storage

import (
	"context"
	"io"
)

// UploadInput contains everything needed to store a file.
type UploadInput struct {
	Key         string    // Storage object key e.g. "user123/video.mp4"
	Reader      io.Reader // File content
	Size        int64     // Content size in bytes (-1 if unknown)
	ContentType string    // MIME type
}

// StorageProvider defines the interface for object storage.
// Currently backed by MinIO — could be replaced with S3 without changing service code.
type StorageProvider interface {
	// Upload stores a file and returns its public/presigned URL.
	Upload(ctx context.Context, input UploadInput) (string, error)

	// Delete removes a file from storage.
	Delete(ctx context.Context, key string) error

	// GetURL returns a presigned URL for downloading a private object.
	GetURL(ctx context.Context, key string) (string, error)

	// EnsureBucket creates the bucket if it does not exist.
	EnsureBucket(ctx context.Context, bucket string) error
}
