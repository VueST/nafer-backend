package handler

import (
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"nafer/comment/internal/domain"
	"nafer/comment/internal/service"
)

// CommentHandler handles HTTP requests for the comment service.
type CommentHandler struct {
	svc *service.CommentService
	log *slog.Logger
}

// NewCommentHandler constructs the handler with dependencies injected.
func NewCommentHandler(svc *service.CommentService, log *slog.Logger) *CommentHandler {
	return &CommentHandler{svc: svc, log: log}
}

// RegisterRoutes mounts all comment routes on the given router.
func (h *CommentHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/", h.create)
	r.Get("/media/:mediaID", h.listByMedia)
	r.Delete("/:id", h.delete)
}

// create handles POST /api/v1/comments
func (h *CommentHandler) create(c *fiber.Ctx) error {
	var body struct {
		MediaID  string  `json:"media_id"`
		UserID   string  `json:"user_id"`
		ParentID *string `json:"parent_id"`
		Body     string  `json:"body"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	comment, err := h.svc.CreateComment(c.Context(), service.CreateCommentInput{
		MediaID:  body.MediaID,
		UserID:   body.UserID,
		ParentID: body.ParentID,
		Body:     body.Body,
	})
	if err != nil {
		h.log.Warn("create comment failed", "error", err.Error())
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(comment)
}

// listByMedia handles GET /api/v1/comments/media/:mediaID
func (h *CommentHandler) listByMedia(c *fiber.Ctx) error {
	mediaID := c.Params("mediaID")
	limit, _ := strconv.Atoi(c.Query("limit", "50"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	comments, err := h.svc.ListByMedia(c.Context(), mediaID, limit, offset)
	if err != nil {
		h.log.Error("list comments failed", "error", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	if comments == nil {
		comments = make([]*domain.Comment, 0)
	}

	return c.JSON(fiber.Map{"data": comments, "count": len(comments)})
}

// delete handles DELETE /api/v1/comments/:id?user_id=xxx
// NOTE: In production, userID comes from JWT middleware, not query param.
func (h *CommentHandler) delete(c *fiber.Ctx) error {
	commentID := c.Params("id")
	userID := c.Query("user_id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id is required"})
	}

	if err := h.svc.DeleteComment(c.Context(), commentID, userID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
