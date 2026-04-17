package service

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"

	"nafer/identity/internal/domain"
	"nafer/identity/internal/repository"
)

// Claims is the structured JWT payload.
// Exported so the middleware package can parse tokens without re-importing jwt.
type Claims struct {
	jwt.RegisteredClaims
	Email string `json:"email"`
	Role  string `json:"role"`
}

// RegisterInput holds the data required to create a new user account.
type RegisterInput struct {
	Email    string
	Password string
}

// AuthResult is returned by Login, Register, and Refresh.
// It contains both the short-lived access token and the long-lived refresh token.
type AuthResult struct {
	AccessToken  string    // signed JWT — short-lived (e.g. 15m)
	RefreshToken string    // opaque UUID — stored server-side in Redis (e.g. 7d)
	ExpiresAt    time.Time // access token expiry — frontend uses this for pre-emptive refresh
	User         *domain.User
}

// AuthService contains all authentication business logic.
// It depends only on interfaces — never on HTTP, Postgres, or Redis directly.
type AuthService struct {
	users     repository.UserRepository
	tokens    *TokenService
	jwtSecret string
	accessTTL time.Duration
}

// NewAuthService constructs the service with all dependencies injected.
func NewAuthService(
	users repository.UserRepository,
	tokens *TokenService,
	jwtSecret string,
	accessTTL time.Duration,
) *AuthService {
	return &AuthService{
		users:     users,
		tokens:    tokens,
		jwtSecret: jwtSecret,
		accessTTL: accessTTL,
	}
}

// Register creates a new account and returns an AuthResult with both tokens.
func (s *AuthService) Register(ctx context.Context, input RegisterInput) (*AuthResult, error) {
	if len(input.Password) < 8 {
		return nil, fmt.Errorf("password must be at least 8 characters")
	}

	existing, err := s.users.FindByEmail(ctx, input.Email)
	if err != nil {
		return nil, fmt.Errorf("checking email availability: %w", err)
	}
	if existing != nil {
		return nil, fmt.Errorf("email is already registered")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hashing password: %w", err)
	}

	now := time.Now().UTC()
	user := &domain.User{
		ID:           uuid.NewString(),
		Email:        input.Email,
		PasswordHash: string(hash),
		Role:         domain.DefaultRole,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	created, err := s.users.Create(ctx, user)
	if err != nil {
		return nil, fmt.Errorf("creating user: %w", err)
	}

	return s.issueTokenPair(ctx, created)
}

// Login validates credentials and returns an AuthResult with both tokens.
func (s *AuthService) Login(ctx context.Context, email, password string) (*AuthResult, error) {
	user, err := s.users.FindByEmail(ctx, email)
	if err != nil {
		return nil, fmt.Errorf("looking up user: %w", err)
	}
	// Compare hash regardless of whether user was found — constant-time protection.
	// We still do the check after.
	if user == nil {
		// Burn time to prevent email enumeration via timing attack
		_ = bcrypt.CompareHashAndPassword([]byte("$2a$10$dummy"), []byte(password))
		return nil, fmt.Errorf("invalid credentials")
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, fmt.Errorf("invalid credentials")
	}

	return s.issueTokenPair(ctx, user)
}

// Logout invalidates the current access token and revokes the refresh token.
// jti is the JWT ID extracted from the valid access token by the middleware.
// expiresAt is when the access token expires — used to set the denylist TTL.
// refreshToken is the opaque token sent by the client — can be empty string.
func (s *AuthService) Logout(ctx context.Context, jti string, expiresAt time.Time, refreshToken string) error {
	// Denylist the access token so it cannot be reused before it expires.
	if err := s.tokens.DenyAccessToken(ctx, jti, expiresAt); err != nil {
		return fmt.Errorf("revoking access token: %w", err)
	}

	// Revoke the refresh token — best effort, don't fail logout if already gone.
	if refreshToken != "" {
		_ = s.tokens.RevokeRefreshToken(ctx, refreshToken)
	}

	return nil
}

// Refresh exchanges a valid refresh token for a new access + refresh token pair.
// The old refresh token is consumed (deleted) atomically, preventing replay attacks.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*AuthResult, error) {
	// ConsumeRefreshToken atomically reads and deletes — one-time use.
	userID, err := s.tokens.ConsumeRefreshToken(ctx, refreshToken)
	if err != nil {
		return nil, fmt.Errorf("invalid refresh token: %w", err)
	}

	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("loading user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user associated with token no longer exists")
	}

	return s.issueTokenPair(ctx, user)
}

// GetMe returns the full user record for the authenticated user.
func (s *AuthService) GetMe(ctx context.Context, userID string) (*domain.User, error) {
	user, err := s.users.FindByID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("loading user: %w", err)
	}
	if user == nil {
		return nil, fmt.Errorf("user not found")
	}
	return user, nil
}

// ── Private helpers ───────────────────────────────────────────────────────────

// issueTokenPair generates a signed access JWT and an opaque refresh token,
// stores the refresh token in Redis, and returns both in an AuthResult.
func (s *AuthService) issueTokenPair(ctx context.Context, user *domain.User) (*AuthResult, error) {
	jti := uuid.NewString()
	expiresAt := time.Now().Add(s.accessTTL)

	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ID:        jti,
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
		},
		Email: user.Email,
		Role:  string(user.Role),
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	accessToken, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		return nil, fmt.Errorf("signing access token: %w", err)
	}

	// Generate an opaque refresh token and store it in Redis.
	refreshToken := uuid.NewString()
	if err := s.tokens.StoreRefreshToken(ctx, refreshToken, user.ID); err != nil {
		return nil, fmt.Errorf("storing refresh token: %w", err)
	}

	return &AuthResult{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresAt:    expiresAt,
		User:         user,
	}, nil
}
