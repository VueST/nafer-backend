package repository

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"nafer/identity/internal/domain"
)

// postgresUserRepository is the PostgreSQL implementation of UserRepository.
// Only this file knows about SQL — the rest of the codebase is DB-agnostic.
type postgresUserRepository struct {
	db *pgxpool.Pool
}

// NewPostgresUserRepository constructs a Postgres implementation.
// Returns the interface so callers never depend on the concrete type.
func NewPostgresUserRepository(db *pgxpool.Pool) UserRepository {
	return &postgresUserRepository{db: db}
}

// Create persists a new user and returns the created record.
func (r *postgresUserRepository) Create(ctx context.Context, user *domain.User) (*domain.User, error) {
	const query = `
		INSERT INTO users (id, email, password_hash, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6)
		RETURNING id, email, password_hash, role, created_at, updated_at
	`
	row := r.db.QueryRow(ctx, query,
		user.ID,
		user.Email,
		user.PasswordHash,
		string(user.Role),
		user.CreatedAt,
		user.UpdatedAt,
	)

	created := &domain.User{}
	if err := row.Scan(
		&created.ID,
		&created.Email,
		&created.PasswordHash,
		&created.Role,
		&created.CreatedAt,
		&created.UpdatedAt,
	); err != nil {
		return nil, fmt.Errorf("create user: %w", err)
	}
	return created, nil
}

// FindByEmail retrieves a user by email address.
// Returns (nil, nil) when no user is found — that is not an error.
// Returns (nil, err) only when a real database error occurs.
func (r *postgresUserRepository) FindByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users WHERE email = $1
	`
	row := r.db.QueryRow(ctx, query, email)

	user := &domain.User{}
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // not found — caller decides what to do
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return user, nil
}

// FindByID retrieves a user by their unique identifier.
// Returns (nil, nil) when no user is found — that is not an error.
// Returns (nil, err) only when a real database error occurs.
func (r *postgresUserRepository) FindByID(ctx context.Context, id string) (*domain.User, error) {
	const query = `
		SELECT id, email, password_hash, role, created_at, updated_at
		FROM users WHERE id = $1
	`
	row := r.db.QueryRow(ctx, query, id)

	user := &domain.User{}
	err := row.Scan(
		&user.ID,
		&user.Email,
		&user.PasswordHash,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // not found — caller decides what to do
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return user, nil
}
