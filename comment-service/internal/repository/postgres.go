package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"nafer/comment/internal/domain"
)

type postgresCommentRepo struct {
	db *pgxpool.Pool
}

// NewPostgresCommentRepository returns a Postgres-backed CommentRepository.
func NewPostgresCommentRepository(db *pgxpool.Pool) CommentRepository {
	return &postgresCommentRepo{db: db}
}

func (r *postgresCommentRepo) Create(ctx context.Context, c *domain.Comment) (*domain.Comment, error) {
	created := &domain.Comment{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO comments (id, media_id, user_id, parent_id, body, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING id, media_id, user_id, parent_id, body, created_at, updated_at
	`, c.ID, c.MediaID, c.UserID, c.ParentID, c.Body, c.CreatedAt, c.UpdatedAt).
		Scan(&created.ID, &created.MediaID, &created.UserID, &created.ParentID,
			&created.Body, &created.CreatedAt, &created.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (r *postgresCommentRepo) FindByMediaID(ctx context.Context, mediaID string, limit, offset int) ([]*domain.Comment, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, media_id, user_id, parent_id, body, created_at, updated_at
		FROM comments
		WHERE media_id = $1
		ORDER BY created_at ASC
		LIMIT $2 OFFSET $3
	`, mediaID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var comments []*domain.Comment
	for rows.Next() {
		c := &domain.Comment{}
		if err := rows.Scan(&c.ID, &c.MediaID, &c.UserID, &c.ParentID,
			&c.Body, &c.CreatedAt, &c.UpdatedAt); err != nil {
			return nil, err
		}
		comments = append(comments, c)
	}
	return comments, rows.Err()
}

func (r *postgresCommentRepo) FindByID(ctx context.Context, id string) (*domain.Comment, error) {
	c := &domain.Comment{}
	err := r.db.QueryRow(ctx, `
		SELECT id, media_id, user_id, parent_id, body, created_at, updated_at
		FROM comments WHERE id = $1
	`, id).Scan(&c.ID, &c.MediaID, &c.UserID, &c.ParentID,
		&c.Body, &c.CreatedAt, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return c, err
}

func (r *postgresCommentRepo) Delete(ctx context.Context, id, userID string) error {
	tag, err := r.db.Exec(ctx, `
		DELETE FROM comments WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("comment not found or not owned by user")
	}
	return nil
}
