package handler

import (
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"nafer/streaming/internal/domain"
	"nafer/streaming/internal/service"
)

// VideoHandler handles HTTP requests for the streaming service API.
type VideoHandler struct {
	svc *service.VideoService
	log *slog.Logger
}

// NewVideoHandler constructs the handler.
func NewVideoHandler(svc *service.VideoService, log *slog.Logger) *VideoHandler {
	return &VideoHandler{svc: svc, log: log}
}

// RegisterRoutes mounts all video routes on the given router.
func (h *VideoHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/", h.create)
	r.Get("/", h.list)
	r.Get("/:id", h.get)
}

// create handles POST /api/v1/videos — registers a video for transcoding
func (h *VideoHandler) create(c *fiber.Ctx) error {
	var body struct {
		UploaderID  string `json:"uploader_id"`
		Title       string `json:"title"`
		Description string `json:"description"`
		SourcePath  string `json:"source_path"` // MinIO object key
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	video, err := h.svc.CreateVideo(c.Context(), service.CreateVideoInput{
		UploaderID:  body.UploaderID,
		Title:       body.Title,
		Description: body.Description,
		SourcePath:  body.SourcePath,
	})
	if err != nil {
		h.log.Warn("create video failed", "error", err.Error())
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(video)
}

// get handles GET /api/v1/videos/:id
func (h *VideoHandler) get(c *fiber.Ctx) error {
	id := c.Params("id")
	video, err := h.svc.GetVideo(c.Context(), id)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.JSON(video)
}

// list handles GET /api/v1/videos?uploader_id=xxx&limit=20&offset=0
func (h *VideoHandler) list(c *fiber.Ctx) error {
	uploaderID := c.Query("uploader_id", "")
	limit, _ := strconv.Atoi(c.Query("limit", "20"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	videos, err := h.svc.ListVideos(c.Context(), uploaderID, limit, offset)
	if err != nil {
		h.log.Error("list videos failed", "error", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	if videos == nil {
		videos = make([]*domain.Video, 0)
	}

	return c.JSON(fiber.Map{"data": videos, "count": len(videos)})
}
