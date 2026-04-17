package repository

import (
	"context"

	"nafer/comment/internal/domain"
)

// CommentRepository defines what the service layer needs from a data store.
// Any database (Postgres, MySQL, SQLite) can implement this interface.
// This is the "Port" in Ports & Adapters / Hexagonal Architecture.
type CommentRepository interface {
	// Create persists a new comment and returns it with DB-assigned fields.
	Create(ctx context.Context, comment *domain.Comment) (*domain.Comment, error)

	// FindByMediaID returns comments for a given media item, with pagination.
	FindByMediaID(ctx context.Context, mediaID string, limit, offset int) ([]*domain.Comment, error)

	// FindByID retrieves a single comment by its ID.
	// Returns nil, nil if no comment is found.
	FindByID(ctx context.Context, id string) (*domain.Comment, error)

	// Delete removes a comment by ID, enforcing ownership via userID.
	Delete(ctx context.Context, id, userID string) error
}
