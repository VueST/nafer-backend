package server

import (
	"context"
	"log/slog"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/limiter"
	"github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"

	"nafer/identity/internal/handler"
	"nafer/identity/internal/handler/middleware"
	"nafer/identity/internal/service"
)

// Server wraps the Fiber app with graceful shutdown.
type Server struct {
	app  *fiber.App
	port string
	log  *slog.Logger
}

// New builds a fully wired HTTP server.
// All middleware and route registration happens here — one place, no surprises.
func New(
	authHandler *handler.AuthHandler,
	tokens *service.TokenService,
	jwtSecret string,
	port string,
	log *slog.Logger,
) *Server {
	app := fiber.New(fiber.Config{
		AppName:               "Nafer Identity Service",
		ReadTimeout:           10 * time.Second,
		WriteTimeout:          10 * time.Second,
		IdleTimeout:           120 * time.Second,
		DisableStartupMessage: false,
		// All errors return clean JSON — no HTML pages.
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			code := fiber.StatusInternalServerError
			if e, ok := err.(*fiber.Error); ok {
				code = e.Code
			}
			return c.Status(code).JSON(fiber.Map{"error": err.Error()})
		},
	})

	// ── Global Middleware ─────────────────────────────────────────────────────

	// Recover from panics — returns 500 instead of crashing.
	app.Use(recover.New())

	// Structured request logging.
	app.Use(logger.New(logger.Config{
		Format: "${time} | ${status} | ${latency} | ${method} ${path}\n",
	}))

	// ── Health Check (always public, no rate limiting) ────────────────────────
	app.Get("/health", func(c *fiber.Ctx) error {
		return c.JSON(fiber.Map{"status": "ok", "service": "identity"})
	})

	// ── API v1 Routes ─────────────────────────────────────────────────────────
	api := app.Group("/api/v1")

	// Rate limiter for auth endpoints: 20 req/min per IP.
	// Prevents brute-force and credential stuffing attacks.
	authRateLimiter := limiter.New(limiter.Config{
		Max:        20,
		Expiration: time.Minute,
		KeyGenerator: func(c *fiber.Ctx) string {
			return c.IP()
		},
		LimitReached: func(c *fiber.Ctx) error {
			return c.Status(fiber.StatusTooManyRequests).JSON(fiber.Map{
				"error": "too many requests — please wait before trying again",
			})
		},
	})

	jwtMid := middleware.JWTMiddleware(jwtSecret, tokens)

	authHandler.RegisterRoutes(api, jwtMid, authRateLimiter)

	return &Server{app: app, port: port, log: log}
}

// Run starts the server and blocks until SIGINT or SIGTERM is received.
func (s *Server) Run() {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	go func() {
		s.log.Info("server starting", "port", s.port)
		if err := s.app.Listen(":" + s.port); err != nil {
			s.log.Error("server listen error", "error", err)
		}
	}()

	<-ctx.Done()
	s.log.Info("shutdown signal received — draining connections")
	stop()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := s.app.ShutdownWithContext(shutdownCtx); err != nil {
		s.log.Error("graceful shutdown error", "error", err)
	}
	s.log.Info("server stopped cleanly")
}
