package repository

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"

	"nafer/media/internal/domain"
)

type postgresMediaRepository struct {
	db *pgxpool.Pool
}

func NewPostgresMediaRepository(db *pgxpool.Pool) MediaRepository {
	return &postgresMediaRepository{db: db}
}

func (r *postgresMediaRepository) Create(ctx context.Context, media *domain.Media) (*domain.Media, error) {
	query := `
		INSERT INTO media (id, owner_id, filename, content_type, size_bytes, storage_key, status, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
		RETURNING id, owner_id, filename, content_type, size_bytes, storage_key, status, created_at, updated_at
	`
	row := r.db.QueryRow(ctx, query,
		media.ID, media.OwnerID, media.Filename,
		media.ContentType, media.SizeBytes, media.StorageKey,
		media.Status, media.CreatedAt, media.UpdatedAt,
	)

	created := &domain.Media{}
	err := row.Scan(
		&created.ID, &created.OwnerID, &created.Filename,
		&created.ContentType, &created.SizeBytes, &created.StorageKey,
		&created.Status, &created.CreatedAt, &created.UpdatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("create media: %w", err)
	}
	return created, nil
}

func (r *postgresMediaRepository) FindByID(ctx context.Context, id string) (*domain.Media, error) {
	query := `
		SELECT id, owner_id, filename, content_type, size_bytes, storage_key, status, created_at, updated_at
		FROM media WHERE id = $1
	`
	row := r.db.QueryRow(ctx, query, id)
	media := &domain.Media{}
	err := row.Scan(
		&media.ID, &media.OwnerID, &media.Filename,
		&media.ContentType, &media.SizeBytes, &media.StorageKey,
		&media.Status, &media.CreatedAt, &media.UpdatedAt,
	)
	if err != nil {
		return nil, nil
	}
	return media, nil
}

func (r *postgresMediaRepository) UpdateStatus(ctx context.Context, id string, status domain.MediaStatus) error {
	query := `UPDATE media SET status = $1, updated_at = $2 WHERE id = $3`
	_, err := r.db.Exec(ctx, query, status, time.Now().UTC(), id)
	if err != nil {
		return fmt.Errorf("update media status: %w", err)
	}
	return nil
}

func (r *postgresMediaRepository) ListByOwner(ctx context.Context, ownerID string) ([]*domain.Media, error) {
	query := `
		SELECT id, owner_id, filename, content_type, size_bytes, storage_key, status, created_at, updated_at
		FROM media WHERE owner_id = $1 ORDER BY created_at DESC
	`
	rows, err := r.db.Query(ctx, query, ownerID)
	if err != nil {
		return nil, fmt.Errorf("list media by owner: %w", err)
	}
	defer rows.Close()

	var results []*domain.Media
	for rows.Next() {
		m := &domain.Media{}
		if err := rows.Scan(
			&m.ID, &m.OwnerID, &m.Filename,
			&m.ContentType, &m.SizeBytes, &m.StorageKey,
			&m.Status, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan media row: %w", err)
		}
		results = append(results, m)
	}
	return results, nil
}
