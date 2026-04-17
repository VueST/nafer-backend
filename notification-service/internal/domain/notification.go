package domain

import "time"

// NotificationType enumerates all possible notification kinds.
type NotificationType string

const (
	NotificationTypeComment NotificationType = "comment"
	NotificationTypeLike    NotificationType = "like"
	NotificationTypeFollow  NotificationType = "follow"
	NotificationTypeSystem  NotificationType = "system"
)

// Notification represents a user notification.
// Pure Go struct — zero external dependencies.
type Notification struct {
	ID         string           `json:"id"`
	UserID     string           `json:"user_id"`     // recipient
	ActorID    string           `json:"actor_id"`    // who triggered it
	Type       NotificationType `json:"type"`
	ResourceID string           `json:"resource_id"` // e.g. comment_id, media_id
	Message    string           `json:"message"`
	IsRead     bool             `json:"is_read"`
	CreatedAt  time.Time        `json:"created_at"`
}
