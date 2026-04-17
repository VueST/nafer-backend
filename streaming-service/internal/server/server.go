package server

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"nafer/streaming/internal/handler"
)

// Server wraps the Fiber app with graceful shutdown logic.
type Server struct {
	app  *fiber.App
	port string
	log  *slog.Logger
}

// New constructs a fully configured HTTP server for the streaming API.
func New(videoHandler *handler.VideoHandler, port string, log *slog.Logger) *Server {
	app := fiber.New(fiber.Config{
		AppName:      "Nafer Streaming Service",
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 10 * time.Second,
		// Allow large body for upload metadata requests
		BodyLimit: 1 * 1024 * 1024, // 1MB
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(recover.New())
	app.Use(logger.New(logger.Config{
		Format: "${time} ${status} ${latency} ${method} ${path}\n",
	}))

	// Health check (used by Docker Compose healthcheck)
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "streaming"})
	})

	api := app.Group("/api/v1/videos")
	videoHandler.RegisterRoutes(api)

	return &Server{app: app, port: port, log: log}
}

// Run starts the server and blocks until a shutdown signal is received.
func (s *Server) Run() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		s.log.Info("server starting", "port", s.port)
		if err := s.app.Listen(":" + s.port); err != nil {
			s.log.Error("server error", "error", err)
		}
	}()

	<-ctx.Done()
	s.log.Info("shutdown signal received, draining connections...")
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.app.ShutdownWithContext(shutdownCtx); err != nil {
		s.log.Error("shutdown error", "error", err)
	}
	s.log.Info("server stopped cleanly")
}
