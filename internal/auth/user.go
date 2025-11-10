package auth

import (
	"time"
)

// User represents a system user
type User struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Password  string    `json:"-"` // Never expose password in JSON
	FirstName string    `json:"firstName,omitempty"`
	LastName  string    `json:"lastName,omitempty"`
	Role      string    `json:"role"` // admin, member, viewer
	Active    bool      `json:"active"`
	Settings  UserSettings `json:"settings,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// UserSettings represents user preferences
type UserSettings struct {
	Theme       string `json:"theme,omitempty"`       // light, dark
	Language    string `json:"language,omitempty"`    // en, de, es, etc.
	Timezone    string `json:"timezone,omitempty"`    // UTC, America/New_York, etc.
	Preferences map[string]interface{} `json:"preferences,omitempty"`
}

// UserCredentials represents login credentials
type UserCredentials struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

// UserRegistration represents user registration data
type UserRegistration struct {
	Email     string `json:"email"`
	Password  string `json:"password"`
	FirstName string `json:"firstName,omitempty"`
	LastName  string `json:"lastName,omitempty"`
	InviteToken string `json:"inviteToken,omitempty"`
}

// UserUpdate represents user update data
type UserUpdate struct {
	FirstName *string `json:"firstName,omitempty"`
	LastName  *string `json:"lastName,omitempty"`
	Email     *string `json:"email,omitempty"`
	Password  *string `json:"password,omitempty"`
	Role      *string `json:"role,omitempty"`
	Active    *bool   `json:"active,omitempty"`
	Settings  *UserSettings `json:"settings,omitempty"`
}

// Session represents an active user session
type Session struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	CreatedAt time.Time `json:"createdAt"`
	LastUsed  time.Time `json:"lastUsed"`
	IPAddress string    `json:"ipAddress,omitempty"`
	UserAgent string    `json:"userAgent,omitempty"`
}

// APIKey represents an API key for authentication
type APIKey struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Name      string    `json:"name"`
	Key       string    `json:"key"` // Hashed in storage
	Prefix    string    `json:"prefix"` // First 8 chars for identification
	ExpiresAt *time.Time `json:"expiresAt,omitempty"`
	LastUsed  *time.Time `json:"lastUsed,omitempty"`
	Active    bool      `json:"active"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// InviteToken represents an invitation token
type InviteToken struct {
	ID        string    `json:"id"`
	Email     string    `json:"email"`
	Token     string    `json:"token"`
	Role      string    `json:"role"`
	InvitedBy string    `json:"invitedBy"` // User ID
	ExpiresAt time.Time `json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// PasswordResetToken represents a password reset token
type PasswordResetToken struct {
	ID        string    `json:"id"`
	UserID    string    `json:"userId"`
	Token     string    `json:"token"`
	ExpiresAt time.Time `json:"expiresAt"`
	UsedAt    *time.Time `json:"usedAt,omitempty"`
	CreatedAt time.Time `json:"createdAt"`
}

// UserRole defines available user roles
type UserRole string

const (
	RoleAdmin  UserRole = "admin"
	RoleMember UserRole = "member"
	RoleViewer UserRole = "viewer"
)

// IsValid checks if a role is valid
func (r UserRole) IsValid() bool {
	switch r {
	case RoleAdmin, RoleMember, RoleViewer:
		return true
	default:
		return false
	}
}

// HasPermission checks if a role has a specific permission
func (r UserRole) HasPermission(permission string) bool {
	switch r {
	case RoleAdmin:
		return true // Admin has all permissions
	case RoleMember:
		// Members can read and write but not delete users
		switch permission {
		case "workflow:read", "workflow:write", "workflow:execute",
			"execution:read", "credential:read", "credential:write":
			return true
		default:
			return false
		}
	case RoleViewer:
		// Viewers can only read
		switch permission {
		case "workflow:read", "execution:read":
			return true
		default:
			return false
		}
	default:
		return false
	}
}

// Sanitize removes sensitive data from user
func (u *User) Sanitize() *User {
	copy := *u
	copy.Password = ""
	return &copy
}
