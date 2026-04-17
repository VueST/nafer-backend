package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"nafer/comment/internal/domain"
	"nafer/comment/internal/repository"
)

// CommentService contains all business logic for comments.
// It depends ONLY on the repository interface — not on Postgres, not on HTTP.
type CommentService struct {
	comments repository.CommentRepository
}

// NewCommentService constructs the service with its dependencies injected.
func NewCommentService(comments repository.CommentRepository) *CommentService {
	return &CommentService{comments: comments}
}

// CreateCommentInput holds the validated data needed to create a comment.
type CreateCommentInput struct {
	MediaID  string
	UserID   string
	ParentID *string // nil for top-level comments
	Body     string
}

// CreateComment validates input and persists a new comment.
func (s *CommentService) CreateComment(ctx context.Context, input CreateCommentInput) (*domain.Comment, error) {
	if input.MediaID == "" {
		return nil, fmt.Errorf("media_id is required")
	}
	if input.UserID == "" {
		return nil, fmt.Errorf("user_id is required")
	}
	if input.Body == "" {
		return nil, fmt.Errorf("comment body cannot be empty")
	}
	if len(input.Body) > 2000 {
		return nil, fmt.Errorf("comment body exceeds maximum length of 2000 characters")
	}

	now := time.Now().UTC()
	c := &domain.Comment{
		ID:        uuid.NewString(),
		MediaID:   input.MediaID,
		UserID:    input.UserID,
		ParentID:  input.ParentID,
		Body:      input.Body,
		CreatedAt: now,
		UpdatedAt: now,
	}
	return s.comments.Create(ctx, c)
}

// ListByMedia returns a paginated list of comments for a media item.
func (s *CommentService) ListByMedia(ctx context.Context, mediaID string, limit, offset int) ([]*domain.Comment, error) {
	if limit <= 0 || limit > 100 {
		limit = 50
	}
	if offset < 0 {
		offset = 0
	}
	return s.comments.FindByMediaID(ctx, mediaID, limit, offset)
}

// DeleteComment removes a comment, enforcing ownership.
func (s *CommentService) DeleteComment(ctx context.Context, commentID, userID string) error {
	if commentID == "" || userID == "" {
		return fmt.Errorf("comment_id and user_id are required")
	}
	return s.comments.Delete(ctx, commentID, userID)
}
