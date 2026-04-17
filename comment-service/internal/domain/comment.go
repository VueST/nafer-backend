package domain

import "time"

// Comment represents a user comment on a media item.
// Pure Go struct — zero external dependencies.
// Supports threaded replies via ParentID.
type Comment struct {
	ID        string    `json:"id"`
	MediaID   string    `json:"media_id"`
	UserID    string    `json:"user_id"`
	ParentID  *string   `json:"parent_id,omitempty"` // nil = top-level comment
	Body      string    `json:"body"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
