package repository

import (
	"context"

	"nafer/notification/internal/domain"
)

// NotificationRepository defines what the service layer needs from a data store.
type NotificationRepository interface {
	// Create persists a new notification.
	Create(ctx context.Context, n *domain.Notification) (*domain.Notification, error)

	// FindByUserID returns notifications for a user, ordered by newest first.
	FindByUserID(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, error)

	// MarkAsRead marks a single notification as read for a given user.
	MarkAsRead(ctx context.Context, id, userID string) error

	// MarkAllAsRead marks all notifications for a user as read.
	MarkAllAsRead(ctx context.Context, userID string) error

	// CountUnread returns the count of unread notifications for a user.
	CountUnread(ctx context.Context, userID string) (int64, error)
}
