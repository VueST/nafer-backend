package service

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// Key prefixes — all Nafer auth keys are namespaced to avoid collisions.
const (
	accessDenylistPrefix = "nafer:auth:denylist:"  // nafer:auth:denylist:{jti}
	refreshTokenPrefix   = "nafer:auth:refresh:"   // nafer:auth:refresh:{uuid}
)

// TokenService manages JWT revocation and refresh token lifecycle in Redis.
// It is the single source of truth for token invalidation.
type TokenService struct {
	rdb        *redis.Client
	refreshTTL time.Duration
}

// NewTokenService constructs the service with its Redis client and refresh TTL.
func NewTokenService(rdb *redis.Client, refreshTTL time.Duration) *TokenService {
	return &TokenService{rdb: rdb, refreshTTL: refreshTTL}
}

// ── Access Token Denylist ─────────────────────────────────────────────────────

// DenyAccessToken adds a JWT JTI (token ID) to the denylist until it expires.
// After calling this, IsAccessTokenDenied will return true for this JTI.
// The Redis entry is automatically evicted when the token would have expired,
// so the denylist never grows unbounded.
func (s *TokenService) DenyAccessToken(ctx context.Context, jti string, expiresAt time.Time) error {
	ttl := time.Until(expiresAt)
	if ttl <= 0 {
		return nil // already expired — nothing to denylist
	}
	return s.rdb.Set(ctx, accessDenylistPrefix+jti, "1", ttl).Err()
}

// IsAccessTokenDenied returns true if the JTI is in the denylist.
// SECURITY: Fails closed — if Redis is unavailable, the token is denied.
// This prevents revoked tokens from being accepted when Redis is down.
func (s *TokenService) IsAccessTokenDenied(ctx context.Context, jti string) bool {
	n, err := s.rdb.Exists(ctx, accessDenylistPrefix+jti).Result()
	if err != nil {
		return true // fail closed — safer to deny than to accept
	}
	return n > 0
}

// ── Refresh Token Store ───────────────────────────────────────────────────────

// StoreRefreshToken persists an opaque refresh token (UUID → userID) with
// the configured TTL. The token is one-time use: ConsumeRefreshToken deletes it.
func (s *TokenService) StoreRefreshToken(ctx context.Context, tokenID, userID string) error {
	return s.rdb.Set(ctx, refreshTokenPrefix+tokenID, userID, s.refreshTTL).Err()
}

// ConsumeRefreshToken atomically reads and deletes the refresh token.
// Returns the associated userID on success.
// One-time use: after this call, the token is gone and cannot be reused.
// This prevents refresh token replay attacks.
func (s *TokenService) ConsumeRefreshToken(ctx context.Context, tokenID string) (string, error) {
	userID, err := s.rdb.GetDel(ctx, refreshTokenPrefix+tokenID).Result()
	if errors.Is(err, redis.Nil) {
		return "", fmt.Errorf("refresh token is invalid or has already been used")
	}
	if err != nil {
		return "", fmt.Errorf("consuming refresh token: %w", err)
	}
	return userID, nil
}

// RevokeRefreshToken removes a refresh token without reading it.
// Used during logout — best effort (errors are not fatal).
func (s *TokenService) RevokeRefreshToken(ctx context.Context, tokenID string) error {
	return s.rdb.Del(ctx, refreshTokenPrefix+tokenID).Err()
}
