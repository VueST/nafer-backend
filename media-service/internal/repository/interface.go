package repository

import (
	"context"

	"nafer/media/internal/domain"
)

// MediaRepository defines what the service layer needs from the data store.
type MediaRepository interface {
	// Create persists a new media record.
	Create(ctx context.Context, media *domain.Media) (*domain.Media, error)

	// FindByID retrieves a media record by its ID.
	FindByID(ctx context.Context, id string) (*domain.Media, error)

	// UpdateStatus changes the status of a media record.
	UpdateStatus(ctx context.Context, id string, status domain.MediaStatus) error

	// ListByOwner returns all media records belonging to a user.
	ListByOwner(ctx context.Context, ownerID string) ([]*domain.Media, error)
}
