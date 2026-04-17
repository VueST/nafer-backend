package handler

import (
	"log/slog"

	"github.com/gofiber/fiber/v2"

	"nafer/identity/internal/domain"
	"nafer/identity/internal/handler/middleware"
	"nafer/identity/internal/service"
)

// AuthHandler handles all authentication HTTP requests.
type AuthHandler struct {
	auth *service.AuthService
	log  *slog.Logger
}

// NewAuthHandler constructs the handler with its dependencies injected.
func NewAuthHandler(auth *service.AuthService, log *slog.Logger) *AuthHandler {
	return &AuthHandler{auth: auth, log: log}
}

// RegisterRoutes mounts all auth routes onto the given router.
// The JWT middleware is applied PER-ROUTE on protected endpoints.
// This avoids Fiber v2's group.Use() pitfall where Group("", middleware)
// acts as app.Use(prefix, middleware) and applies to ALL sub-routes.
func (h *AuthHandler) RegisterRoutes(api fiber.Router, jwtMiddleware fiber.Handler, rateLimiter fiber.Handler) {
	// ── Public routes (rate-limited only) ────────────────────────────────────
	api.Post("/auth/register", rateLimiter, h.register)
	api.Post("/auth/login", rateLimiter, h.login)
	api.Post("/auth/refresh", rateLimiter, h.refresh)

	// ── Protected routes (rate-limited + JWT validated) ───────────────────────
	api.Post("/auth/logout", rateLimiter, jwtMiddleware, h.logout)
	api.Get("/users/me", jwtMiddleware, h.getMe)
}

// ── Request handlers ──────────────────────────────────────────────────────────

// POST /api/v1/auth/register
func (h *AuthHandler) register(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if body.Email == "" || body.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "email and password are required",
		})
	}

	result, err := h.auth.Register(c.Context(), service.RegisterInput{
		Email:    body.Email,
		Password: body.Password,
	})
	if err != nil {
		h.log.Warn("register failed", "error", err.Error())
		return c.Status(fiber.StatusConflict).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	return c.Status(fiber.StatusCreated).JSON(toAuthResponse(result))
}

// POST /api/v1/auth/login
func (h *AuthHandler) login(c *fiber.Ctx) error {
	var body struct {
		Email    string `json:"email"`
		Password string `json:"password"`
	}
	if err := c.BodyParser(&body); err != nil {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}
	if body.Email == "" || body.Password == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "email and password are required",
		})
	}

	result, err := h.auth.Login(c.Context(), body.Email, body.Password)
	if err != nil {
		h.log.Warn("login failed", "email", body.Email)
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "invalid credentials",
		})
	}

	return c.JSON(toAuthResponse(result))
}

// POST /api/v1/auth/refresh
func (h *AuthHandler) refresh(c *fiber.Ctx) error {
	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	if err := c.BodyParser(&body); err != nil || body.RefreshToken == "" {
		return c.Status(fiber.StatusBadRequest).JSON(fiber.Map{
			"error": "refresh_token is required",
		})
	}

	result, err := h.auth.Refresh(c.Context(), body.RefreshToken)
	if err != nil {
		h.log.Warn("refresh failed", "error", err.Error())
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
			"error": "refresh token is invalid or expired",
		})
	}

	return c.JSON(toAuthResponse(result))
}

// POST /api/v1/auth/logout  (protected)
func (h *AuthHandler) logout(c *fiber.Ctx) error {
	jti := middleware.MustTokenJTI(c)
	exp := middleware.MustTokenExp(c)

	var body struct {
		RefreshToken string `json:"refresh_token"`
	}
	_ = c.BodyParser(&body)

	if err := h.auth.Logout(c.Context(), jti, exp, body.RefreshToken); err != nil {
		h.log.Error("logout failed", "jti", jti, "error", err.Error())
		return c.Status(fiber.StatusInternalServerError).JSON(fiber.Map{
			"error": "logout failed",
		})
	}

	return c.SendStatus(fiber.StatusNoContent)
}

// GET /api/v1/users/me  (protected)
func (h *AuthHandler) getMe(c *fiber.Ctx) error {
	userID := middleware.MustUserID(c)

	user, err := h.auth.GetMe(c.Context(), userID)
	if err != nil {
		return c.Status(fiber.StatusNotFound).JSON(fiber.Map{
			"error": "user not found",
		})
	}

	return c.JSON(toUserResponse(user))
}

// ── Response DTOs ─────────────────────────────────────────────────────────────

type authResponse struct {
	AccessToken  string       `json:"access_token"`
	RefreshToken string       `json:"refresh_token"`
	ExpiresAt    int64        `json:"expires_at"`
	User         userResponse `json:"user"`
}

type userResponse struct {
	ID    string `json:"id"`
	Email string `json:"email"`
	Role  string `json:"role"`
}

func toAuthResponse(r *service.AuthResult) authResponse {
	return authResponse{
		AccessToken:  r.AccessToken,
		RefreshToken: r.RefreshToken,
		ExpiresAt:    r.ExpiresAt.Unix(),
		User:         toUserResponse(r.User),
	}
}

func toUserResponse(u *domain.User) userResponse {
	return userResponse{ID: u.ID, Email: u.Email, Role: string(u.Role)}
}
