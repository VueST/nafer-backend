package domain

import "time"

// UserRole is a type-safe alias for role strings.
// All role checks must use these constants — never raw strings in business logic.
type UserRole string

const (
	RoleUser    UserRole = "user"
	RolePremium UserRole = "premium"
	RoleMod     UserRole = "mod"
	RoleAdmin   UserRole = "admin"
)

// DefaultRole is the tier assigned to every newly registered user.
const DefaultRole = RoleUser

// User is the core entity of the identity service.
// This struct has NO external dependencies — it is pure Go.
// Business rules that belong to the user live here as methods.
type User struct {
	ID           string
	Email        string
	PasswordHash string
	Role         UserRole
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// IsActive returns whether the user account should be considered active.
func (u *User) IsActive() bool {
	return u.ID != ""
}

// HasPremiumAccess returns true if the user is on a paid or elevated tier.
// premium, mod and admin all have access to premium content.
func (u *User) HasPremiumAccess() bool {
	return u.Role == RolePremium || u.Role == RoleMod || u.Role == RoleAdmin
}

// CanModerate returns true if the user can perform moderation actions
// (delete comments, issue temporary bans).
func (u *User) CanModerate() bool {
	return u.Role == RoleMod || u.Role == RoleAdmin
}

// CanUploadContent returns true if the user can push or remove media.
func (u *User) CanUploadContent() bool {
	return u.Role == RoleAdmin
}

// CanChangeRole returns true if the user can elevate or demote other users.
func (u *User) CanChangeRole() bool {
	return u.Role == RoleAdmin
}
