package service

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"nafer/notification/internal/domain"
	"nafer/notification/internal/repository"
)

// NotificationService contains all business logic for notifications.
// It also consumes events from Redis Pub/Sub to create notifications reactively.
type NotificationService struct {
	notifications repository.NotificationRepository
	redis         *redis.Client
	log           *slog.Logger
}

// NewNotificationService constructs the service with its dependencies.
func NewNotificationService(
	notifications repository.NotificationRepository,
	redis *redis.Client,
	log *slog.Logger,
) *NotificationService {
	return &NotificationService{
		notifications: notifications,
		redis:         redis,
		log:           log,
	}
}

// CreateNotificationInput holds the data to create a notification programmatically.
type CreateNotificationInput struct {
	UserID     string
	ActorID    string
	Type       domain.NotificationType
	ResourceID string
	Message    string
}

// CreateNotification persists a notification and publishes it to Redis for real-time delivery.
func (s *NotificationService) CreateNotification(ctx context.Context, input CreateNotificationInput) (*domain.Notification, error) {
	if input.UserID == "" || input.ActorID == "" || input.Message == "" {
		return nil, fmt.Errorf("user_id, actor_id, and message are required")
	}

	n := &domain.Notification{
		ID:         uuid.NewString(),
		UserID:     input.UserID,
		ActorID:    input.ActorID,
		Type:       input.Type,
		ResourceID: input.ResourceID,
		Message:    input.Message,
		IsRead:     false,
		CreatedAt:  time.Now().UTC(),
	}

	created, err := s.notifications.Create(ctx, n)
	if err != nil {
		return nil, fmt.Errorf("persisting notification: %w", err)
	}

	// Publish to Redis channel for real-time SSE delivery.
	// The channel key is per-user: "notifications:{userID}"
	payload, _ := json.Marshal(created)
	channel := fmt.Sprintf("notifications:%s", created.UserID)
	if err := s.redis.Publish(ctx, channel, payload).Err(); err != nil {
		// Non-fatal: notification is already persisted
		s.log.Warn("failed to publish notification to redis", "error", err)
	}

	return created, nil
}

// GetForUser returns paginated notifications for a user.
func (s *NotificationService) GetForUser(ctx context.Context, userID string, limit, offset int) ([]*domain.Notification, error) {
	if limit <= 0 || limit > 100 {
		limit = 30
	}
	return s.notifications.FindByUserID(ctx, userID, limit, offset)
}

// MarkAsRead marks a single notification as read.
func (s *NotificationService) MarkAsRead(ctx context.Context, id, userID string) error {
	return s.notifications.MarkAsRead(ctx, id, userID)
}

// MarkAllAsRead marks all notifications for a user as read.
func (s *NotificationService) MarkAllAsRead(ctx context.Context, userID string) error {
	return s.notifications.MarkAllAsRead(ctx, userID)
}

// CountUnread returns the unread notification count for a user.
func (s *NotificationService) CountUnread(ctx context.Context, userID string) (int64, error) {
	return s.notifications.CountUnread(ctx, userID)
}
