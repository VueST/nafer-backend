package repository

import (
	"context"

	"nafer/identity/internal/domain"
)

// UserRepository defines what the service layer needs from a data store.
// Any database (Postgres, MySQL, SQLite) can implement this interface.
// This is the "Port" in Ports & Adapters / Hexagonal Architecture.
type UserRepository interface {
	// Create persists a new user and returns the created user with DB-assigned fields.
	Create(ctx context.Context, user *domain.User) (*domain.User, error)

	// FindByEmail retrieves a user by their email address.
	// Returns nil, nil if no user is found (not an error).
	FindByEmail(ctx context.Context, email string) (*domain.User, error)

	// FindByID retrieves a user by their unique ID.
	FindByID(ctx context.Context, id string) (*domain.User, error)
}
