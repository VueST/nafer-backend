package domain

import "time"

// MediaStatus represents the lifecycle of an uploaded file.
type MediaStatus string

const (
	MediaStatusPending   MediaStatus = "pending"
	MediaStatusUploaded  MediaStatus = "uploaded"
	MediaStatusFailed    MediaStatus = "failed"
)

// Media is the core entity for the media service.
// No external dependencies — pure Go.
type Media struct {
	ID          string
	OwnerID     string
	Filename    string
	ContentType string
	SizeBytes   int64
	StorageKey  string      // The object key in MinIO/S3
	Status      MediaStatus
	CreatedAt   time.Time
	UpdatedAt   time.Time
}

// IsReady returns true if the media is fully uploaded and accessible.
func (m *Media) IsReady() bool {
	return m.Status == MediaStatusUploaded
}
