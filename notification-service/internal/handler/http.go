package handler

import (
	"bufio"
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"nafer/notification/internal/domain"
	"nafer/notification/internal/service"

	"github.com/redis/go-redis/v9"
)

// NotificationHandler handles HTTP + SSE requests for the notification service.
type NotificationHandler struct {
	svc   *service.NotificationService
	redis *redis.Client
	log   *slog.Logger
}

// NewNotificationHandler constructs the handler.
func NewNotificationHandler(svc *service.NotificationService, redis *redis.Client, log *slog.Logger) *NotificationHandler {
	return &NotificationHandler{svc: svc, redis: redis, log: log}
}

// RegisterRoutes mounts all notification routes on the given router.
func (h *NotificationHandler) RegisterRoutes(r fiber.Router) {
	r.Post("/", h.create)
	r.Get("/user/:userID", h.listForUser)
	r.Get("/user/:userID/unread-count", h.unreadCount)
	r.Put("/:id/read", h.markAsRead)
	r.Put("/user/:userID/read-all", h.markAllAsRead)
	// SSE endpoint: GET /api/v1/notifications/stream/:userID
	r.Get("/stream/:userID", h.stream)
}

// create handles POST /api/v1/notifications — internal endpoint for other services
func (h *NotificationHandler) create(c *fiber.Ctx) error {
	var body struct {
		UserID     string                   `json:"user_id"`
		ActorID    string                   `json:"actor_id"`
		Type       domain.NotificationType  `json:"type"`
		ResourceID string                   `json:"resource_id"`
		Message    string                   `json:"message"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "invalid request body"})
	}

	n, err := h.svc.CreateNotification(c.Context(), service.CreateNotificationInput{
		UserID:     body.UserID,
		ActorID:    body.ActorID,
		Type:       body.Type,
		ResourceID: body.ResourceID,
		Message:    body.Message,
	})
	if err != nil {
		h.log.Warn("create notification failed", "error", err.Error())
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": err.Error()})
	}

	return c.Status(fiber.StatusCreated).JSON(n)
}

// listForUser handles GET /api/v1/notifications/user/:userID
func (h *NotificationHandler) listForUser(c *fiber.Ctx) error {
	userID := c.Params("userID")
	limit, _ := strconv.Atoi(c.Query("limit", "30"))
	offset, _ := strconv.Atoi(c.Query("offset", "0"))

	notifications, err := h.svc.GetForUser(c.Context(), userID, limit, offset)
	if err != nil {
		h.log.Error("list notifications failed", "error", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}

	if notifications == nil {
		notifications = make([]*domain.Notification, 0)
	}

	return c.JSON(fiber.Map{"data": notifications, "count": len(notifications)})
}

// unreadCount handles GET /api/v1/notifications/user/:userID/unread-count
func (h *NotificationHandler) unreadCount(c *fiber.Ctx) error {
	userID := c.Params("userID")
	count, err := h.svc.CountUnread(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.JSON(fiber.Map{"unread_count": count})
}

// markAsRead handles PUT /api/v1/notifications/:id/read
func (h *NotificationHandler) markAsRead(c *fiber.Ctx) error {
	id := c.Params("id")
	userID := c.Query("user_id")
	if userID == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{"error": "user_id is required"})
	}

	if err := h.svc.MarkAsRead(c.Context(), id, userID); err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{"error": err.Error()})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// markAllAsRead handles PUT /api/v1/notifications/user/:userID/read-all
func (h *NotificationHandler) markAllAsRead(c *fiber.Ctx) error {
	userID := c.Params("userID")
	if err := h.svc.MarkAllAsRead(c.Context(), userID); err != nil {
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{"error": "internal error"})
	}
	return c.SendStatus(fiber.StatusNoContent)
}

// stream handles GET /api/v1/notifications/stream/:userID
// Implements Server-Sent Events (SSE) for real-time notification delivery.
// The client connects once and receives events as they are published to Redis.
func (h *NotificationHandler) stream(c *fiber.Ctx) error {
	userID := c.Params("userID")
	channel := fmt.Sprintf("notifications:%s", userID)

	// Set headers for SSE
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache")
	c.Set("Connection", "keep-alive")
	c.Set("X-Accel-Buffering", "no") // Disable nginx buffering

	// Create a cancellable context tied to this request
	ctx, cancel := context.WithCancel(c.Context())
	defer cancel()

	// Subscribe to the Redis Pub/Sub channel for this user
	pubsub := h.redis.Subscribe(ctx, channel)
	defer pubsub.Close()

	h.log.Info("sse client connected", "user_id", userID)

	// Use fasthttp BodyStreamWriter directly for maximum compatibility
	c.Context().SetBodyStreamWriter(func(w *bufio.Writer) {
		// Send initial connected event
		fmt.Fprintf(w, "event: connected\ndata: {\"status\":\"connected\"}\n\n")
		w.Flush()

		msgCh := pubsub.Channel()
		for {
			select {
			case <-ctx.Done():
				h.log.Info("sse client disconnected", "user_id", userID)
				return
			case msg, ok := <-msgCh:
				if !ok {
					return
				}
				fmt.Fprintf(w, "event: notification\ndata: %s\n\n", msg.Payload)
				if err := w.Flush(); err != nil {
					h.log.Info("sse flush error, client disconnected", "user_id", userID)
					return
				}
			}
		}
	})

	return nil
}
