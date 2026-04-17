package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"nafer/notification/internal/domain"
)

type postgresNotificationRepo struct {
	db *pgxpool.Pool
}

// NewPostgresNotificationRepository returns a Postgres-backed NotificationRepository.
func NewPostgresNotificationRepository(db *pgxpool.Pool) NotificationRepository {
	return &postgresNotificationRepo{db: db}
}

func (r *postgresNotificationRepo) Create(ctx context.Context, n *domain.Notification) (*domain.Notification, error) {
	created := &domain.Notification{}
	err := r.db.QueryRow(ctx, `
		INSERT INTO notifications (id, user_id, actor_id, type, resource_id, message, is_read, created_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING id, user_id, actor_id, type, resource_id, message, is_read, created_at
	`, n.ID, n.UserID, n.ActorID, n.Type, n.ResourceID, n.Message, n.IsRead, n.CreatedAt).
		Scan(&created.ID, &created.UserID, &created.ActorID, &created.Type,
			&created.ResourceID, &created.Message, &created.IsRead, &created.CreatedAt)
	if err != nil {
		return nil, err
	}
	return created, nil
}

func (r *postgresNotificationRepo) FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, error) {
	rows, err := r.db.Query(ctx, `
		SELECT id, user_id, actor_id, type, resource_id, message, is_read, created_at
		FROM notifications
		WHERE user_id = $1
		ORDER BY created_at DESC
		LIMIT $2 OFFSET $3
	`, userID, limit, offset)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var notifications []*domain.Notification
	for rows.Next() {
		n := &domain.Notification{}
		if err := rows.Scan(&n.ID, &n.UserID, &n.ActorID, &n.Type,
			&n.ResourceID, &n.Message, &n.IsRead, &n.CreatedAt); err != nil {
			return nil, err
		}
		notifications = append(notifications, n)
	}
	return notifications, rows.Err()
}

func (r *postgresNotificationRepo) MarkAsRead(ctx context.Context, id, userID string) error {
	tag, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true WHERE id = $1 AND user_id = $2
	`, id, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return errors.New("notification not found or not owned by user")
	}
	return nil
}

func (r *postgresNotificationRepo) MarkAllAsRead(ctx context.Context, userID string) error {
	_, err := r.db.Exec(ctx, `
		UPDATE notifications SET is_read = true WHERE user_id = $1 AND is_read = false
	`, userID)
	return err
}

func (r *postgresNotificationRepo) CountUnread(ctx context.Context, userID string) (int64, error) {
	var count int64
	err := r.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM notifications WHERE user_id = $1 AND is_read = false
	`, userID).Scan(&count)
	return count, err
}

func (r *postgresNotificationRepo) FindByID(ctx context.Context, id string) (*domain.Notification, error) {
	n := &domain.Notification{}
	err := r.db.QueryRow(ctx, `
		SELECT id, user_id, actor_id, type, resource_id, message, is_read, created_at
		FROM notifications WHERE id = $1
	`, id).Scan(&n.ID, &n.UserID, &n.ActorID, &n.Type,
		&n.ResourceID, &n.Message, &n.IsRead, &n.CreatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	return n, err
}
