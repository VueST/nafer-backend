package middleware

import (
	"fmt"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/golang-jwt/jwt/v5"

	"nafer/identity/internal/service"
)

// contextKey constants for Fiber locals — prevents typo bugs in handler code.
const (
	LocalUserID    = "userID"
	LocalUserEmail = "userEmail"
	LocalUserRole  = "userRole"
	LocalTokenJTI  = "tokenJTI"
	LocalTokenExp  = "tokenExp"
)

// JWTMiddleware validates Bearer tokens on every protected route.
// On success, it injects user claims into Fiber's request context.
// SECURITY: Fails closed on Redis errors — revoked tokens are always denied.
func JWTMiddleware(jwtSecret string, tokens *service.TokenService) fiber.Handler {
	keyFunc := func(t *jwt.Token) (interface{}, error) {
		// Reject tokens signed with an unexpected algorithm.
		// This prevents the "alg:none" attack.
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing algorithm: %v", t.Header["alg"])
		}
		return []byte(jwtSecret), nil
	}

	return func(c *fiber.Ctx) error {
		// 1. Extract token from Authorization header.
		auth := c.Get("Authorization")
		if !strings.HasPrefix(auth, "Bearer ") {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "authorization token is missing or malformed",
			})
		}
		tokenStr := strings.TrimPrefix(auth, "Bearer ")

		// 2. Parse and cryptographically verify the JWT.
		claims := &service.Claims{}
		token, err := jwt.ParseWithClaims(tokenStr, claims, keyFunc)
		if err != nil || !token.Valid {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "invalid or expired access token",
			})
		}

		// 3. Check the denylist — ensures logged-out tokens are rejected.
		// Fails closed: if Redis is down, the token is treated as revoked.
		if tokens.IsAccessTokenDenied(c.Context(), claims.ID) {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"error": "access token has been revoked",
			})
		}

		// 4. Inject validated claims into Fiber locals for downstream handlers.
		c.Locals(LocalUserID, claims.Subject)
		c.Locals(LocalUserEmail, claims.Email)
		c.Locals(LocalUserRole, claims.Role)
		c.Locals(LocalTokenJTI, claims.ID)
		c.Locals(LocalTokenExp, claims.ExpiresAt.Time)

		return c.Next()
	}
}

// MustUserID extracts the validated user ID from Fiber locals.
// Panics if called outside of a JWT-protected route — intentional.
func MustUserID(c *fiber.Ctx) string {
	return c.Locals(LocalUserID).(string)
}

// MustTokenJTI extracts the JWT ID from Fiber locals.
func MustTokenJTI(c *fiber.Ctx) string {
	return c.Locals(LocalTokenJTI).(string)
}

// MustTokenExp extracts the token expiry time from Fiber locals.
func MustTokenExp(c *fiber.Ctx) time.Time {
	return c.Locals(LocalTokenExp).(time.Time)
}
