package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"nafer/streaming/internal/domain"
)

type postgresVideoRepo struct {
	db *pgxpool.Pool
}

// NewPostgresVideoRepository returns a Postgres-backed VideoRepository.
func NewPostgresVideoRepository(db *pgxpool.Pool) VideoRepository {
	return &postgresVideoRepo{db: db}
}

func (r *postgresVideoRepo) Create(ctx context.Context, v *domain.Video) (*domain.Video, error) {
	created := &domain.Video{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO videos (id, uploader_id, title, description, source_path, hls_path, thumbnail_url, duration_sec, status, error_msg, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12)
		RETURNING id, uploader_id, title, description, source_path, hls_path, thumbnail_url, duration_sec, status, error_msg, created_at, updated_at
	`, v.ID, v.UploaderID, v.Title, v.Description, v.SourcePath, v.HLSPath,
		v.ThumbnailURL, v.DurationSec, v.Status, v.ErrorMsg, v.CreatedAt, v.UpdatedAt).
		Scan(&created.ID, &created.UploaderID, &created.Title, &created.Description,
			&created.SourcePath, &created.HLSPath, &created.ThumbnailURL, &created.DurationSec,
			&created.Status, &created.ErrorMsg, &created.CreatedAt, &created.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (r *postgresVideoRepo) FindByID(ctx context.Context, id string) (*domain.Video, error) {
	v := &domain.Video{}
	err := r.db.QueryRow(ctx, `
		SELECT id, uploader_id, title, description, source_path, hls_path, thumbnail_url, duration_sec, status, error_msg, created_at, updated_at
		FROM videos WHERE id = $1
	`, id).Scan(&v.ID, &v.UploaderID, &v.Title, &v.Description,
		&v.SourcePath, &v.HLSPath, &v.ThumbnailURL, &v.DurationSec,
		&v.Status, &v.ErrorMsg, &v.CreatedAt, &v.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return v, err
}

func (r *postgresVideoRepo) UpdateStatus(ctx context.Context, id string, status domain.VideoStatus, errMsg string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE videos SET status = $1, error_msg = $2, updated_at = NOW() WHERE id = $3
	`, status, errMsg, id)
	return err
}

func (r *postgresVideoRepo) UpdateHLSPath(ctx context.Context, id, hlsPath string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE videos SET hls_path = $1, status = $2, updated_at = NOW() WHERE id = $3
	`, hlsPath, domain.VideoStatusReady, id)
	return err
}

func (r *postgresVideoRepo) List(ctx context.Context, uploaderID string, limit, offset int) ([]*domain.Video, error) {
	query := `
		SELECT id, uploader_id, title, description, source_path, hls_path, thumbnail_url, duration_sec, status, error_msg, created_at, updated_at
		FROM videos
	`
	args := []any{limit, offset}
	if uploaderID != "" {
		query += ` WHERE uploader_id = $3`
		args = append(args, uploaderID)
		query += ` ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	} else {
		query += ` ORDER BY created_at DESC LIMIT $1 OFFSET $2`
	}

	rows, err := r.db.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var videos []*domain.Video
	for rows.Next() {
		v := &domain.Video{}
		if err := rows.Scan(&v.ID, &v.UploaderID, &v.Title, &v.Description,
			&v.SourcePath, &v.HLSPath, &v.ThumbnailURL, &v.DurationSec,
			&v.Status, &v.ErrorMsg, &v.CreatedAt, &v.UpdatedAt); err != nil {
			return nil, err
		}
		videos = append(videos, v)
	}
	return videos, rows.Err()
}
