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

	"nafer/media/internal/handler"
)

type Server struct {
	app  *fiber.App
	port string
	log  *slog.Logger
}

func New(mediaHandler *handler.MediaHandler, port string, log *slog.Logger) *Server {
	app := fiber.New(fiber.Config{
		AppName:      "Nafer Media Service",
		BodyLimit:    500 * 1024 * 1024, // 500 MB max upload
		ReadTimeout:  120 * time.Second,
		WriteTimeout: 120 * time.Second,
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	app.Use(recover.New())
	app.Use(logger.New())

	api := app.Group("/api/v1/media")
	mediaHandler.RegisterRoutes(api)

	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "media"})
	})

	return &Server{app: app, port: port, log: log}
}

func (s *Server) Run() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		s.log.Info("media service starting", "port", s.port)
		if err := s.app.Listen(":" + s.port); err != nil {
			s.log.Error("server error", "error", err)
		}
	}()

	<-ctx.Done()
	s.log.Info("shutdown signal received...")
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	_ = s.app.ShutdownWithContext(shutdownCtx)
	s.log.Info("media service stopped cleanly")
}
