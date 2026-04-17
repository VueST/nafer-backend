package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"nafer/media/internal/service"
)

type MediaHandler struct {
	upload *service.UploadService
	log    *slog.Logger
}

func NewMediaHandler(upload *service.UploadService, log *slog.Logger) *MediaHandler {
	return &MediaHandler{upload: upload, log: log}
}

func (h *MediaHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/upload", h.upload_file)
	r.Get("/:id", h.getMedia)
	r.Get("/health", h.health)
}

// upload_file handles POST /api/v1/media/upload
// Accepts multipart/form-data with a "file" field
func (h *MediaHandler) upload_file(c *fiber.Ctx) error {
	// Get owner from header (set by API Gateway / auth middleware)
	ownerID := c.Get("X-User-ID", "anonymous")

	fileHeader, err := c.FormFile("file")
	if err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "file is required (multipart field: 'file')",
		})
	}

	file, err := fileHeader.Open()
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "failed to open uploaded file",
		})
	}
	defer file.Close()

	result, err := h.upload.Upload(c.Context(), service.UploadInput{
		OwnerID:     ownerID,
		Filename:    fileHeader.Filename,
		ContentType: fileHeader.Header.Get("Content-Type"),
		Size:        fileHeader.Size,
		Reader:      file,
	})
	if err != nil {
		h.log.Error("upload failed", "error", err, "owner", ownerID)
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "upload failed",
		})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{
		"id":       result.Media.ID,
		"filename": result.Media.Filename,
		"size":     result.Media.SizeBytes,
		"status":   result.Media.Status,
		"url":      result.URL,
	})
}

// getMedia handles GET /api/v1/media/:id
func (h *MediaHandler) getMedia(c *fiber.Ctx) error {
	id := c.Params("id")

	result, err := h.upload.GetByID(c.Context(), id)
	if err != nil {
		if err.Error() == "media not found" {
			return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": "media not found"})
		}
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	return c.JSON(fiber.Map{
		"id":       result.Media.ID,
		"filename": result.Media.Filename,
		"size":     result.Media.SizeBytes,
		"status":   result.Media.Status,
		"url":      result.URL,
	})
}

func (h *MediaHandler) health(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "ok", "service": "media"})
}
