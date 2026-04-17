package handler

import (
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"nafer/search/internal/domain"
	"nafer/search/internal/service"
)

// SearchHandler handles HTTP requests for the search service.
type SearchHandler struct {
	svc *service.SearchService
	log *slog.Logger
}

// NewSearchHandler constructs the handler.
func NewSearchHandler(svc *service.SearchService, log *slog.Logger) *SearchHandler {
	return &SearchHandler{svc: svc, log: log}
}

// RegisterRoutes mounts all search routes on the given router.
func (h *SearchHandler) RegisterRoutes(r fiber.Router) {
	r.Get("/", h.search)
	r.Post("/index", h.indexMedia)
	r.Delete("/index/:id", h.deleteMedia)
}

// search handles GET /api/v1/search?q=...&type=video&page=1&per_page=20
func (h *SearchHandler) search(c *fiber.Ctx) error {
	query := c.Query("q", "")
	mediaType := c.Query("type", "")
	page, _ := strconv.Atoi(c.Query("page", "1"))
	perPage, _ := strconv.Atoi(c.Query("per_page", "20"))

	result, err := h.svc.Search(c.Context(), query, mediaType, page, perPage)
	if err != nil {
		h.log.Error("search failed", "query", query, "error", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "search error"})
	}

	return c.JSON(result)
}

// indexMedia handles POST /api/v1/search/index — called by other services after upload
func (h *SearchHandler) indexMedia(c *fiber.Ctx) error {
	var body domain.IndexedMedia
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}
	if body.ID == "" || body.Title == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "id and title are required"})
	}

	if err := h.svc.IndexMedia(c.Context(), &body); err != nil {
		h.log.Error("index media failed", "id", body.ID, "error", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "indexing failed"})
	}

	return c.Status(fiber.StatusCreated).JSON(fiber.Map{"status": "indexed", "id": body.ID})
}

// deleteMedia handles DELETE /api/v1/search/index/:id
func (h *SearchHandler) deleteMedia(c *fiber.Ctx) error {
	id := c.Params("id")
	if err := h.svc.DeleteMedia(c.Context(), id); err != nil {
		h.log.Error("delete from index failed", "id", id, "error", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "delete failed"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}
