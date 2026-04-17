package middleware

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"nafer/identity/internal/domain"
)

// RequireRole returns a Fiber middleware that enforces role-based access control.
// It reads the Bearer token from the Authorization header, validates it,
// and only proceeds if the embedded role is in the allowedRoles list.
//
// Usage:
//
//	r.Delete("/comment/:id", middleware.RequireRole(jwtSecret, domain.RoleMod, domain.RoleAdmin), handler)
func RequireRole(jwtSecret string, allowedRoles ...domain.UserRole) fiber.Handler {
	// Build a set for O(1) lookups — no linear scanning on every request.
	allowed := make(map[domain.UserRole]struct{}, len(allowedRoles))
	for _, r := range allowedRoles {
		allowed[r] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		// 1. Extract Bearer token from Authorization header
		authHeader := c.Get("Authorization")
		if authHeader == "" || !strings.HasPrefix(authHeader, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "missing or malformed authorization header",
			})
		}
		tokenStr := strings.TrimPrefix(authHeader, "Bearer ")

		// 2. Parse and validate the JWT
		token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fiber.ErrUnauthorized
			}
			return []byte(jwtSecret), nil
		})
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired token",
			})
		}

		// 3. Extract the role claim
		claims, ok := token.Claims.(jwt.MapClaims)
		if !ok {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "malformed token claims",
			})
		}

		roleStr, ok := claims["role"].(string)
		if !ok || roleStr == "" {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "role claim missing from token",
			})
		}

		// 4. Check permission
		role := domain.UserRole(roleStr)
		if _, permitted := allowed[role]; !permitted {
			return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
				"error": "insufficient permissions",
				"required_one_of": func() []string {
					roles := make([]string, 0, len(allowedRoles))
					for _, r := range allowedRoles {
						roles = append(roles, string(r))
					}
					return roles
				}(),
			})
		}

		// 5. Store validated user info in context for downstream handlers
		c.Locals("userID", claims["sub"])
		c.Locals("userRole", role)
		c.Locals("userEmail", claims["email"])

		return c.Next()
	}
}
