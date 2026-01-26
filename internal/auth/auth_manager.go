package auth

import (
	"errors"
	"fmt"
	"log"
	"time"
)

// AuthManager handles authentication and user management
type AuthManager struct {
	userStorage    UserStorage
	jwtManager     *JWTManager
	passwordHasher *PasswordHasher
}

// NewAuthManager creates a new authentication manager
func NewAuthManager(userStorage UserStorage, jwtSecretKey string, tokenDuration time.Duration) *AuthManager {
	return &AuthManager{
		userStorage:    userStorage,
		jwtManager:     NewJWTManager(jwtSecretKey, tokenDuration, "m9m"),
		passwordHasher: NewPasswordHasher(),
	}
}

// Login authenticates a user and returns a JWT token
func (m *AuthManager) Login(credentials *UserCredentials) (string, *User, error) {
	// Validate credentials
	if credentials.Email == "" || credentials.Password == "" {
		return "", nil, errors.New("email and password are required")
	}

	// Get user by email
	user, err := m.userStorage.GetUserByEmail(credentials.Email)
	if err != nil {
		log.Printf("Login failed for %s: user not found", credentials.Email)
		return "", nil, errors.New("invalid email or password")
	}

	// Check if user is active
	if !user.Active {
		return "", nil, errors.New("user account is deactivated")
	}

	// Verify password
	if err := m.passwordHasher.VerifyPassword(credentials.Password, user.Password); err != nil {
		log.Printf("Login failed for %s: invalid password", credentials.Email)
		return "", nil, errors.New("invalid email or password")
	}

	// Generate JWT token
	token, err := m.jwtManager.GenerateToken(user)
	if err != nil {
		return "", nil, fmt.Errorf("failed to generate token: %w", err)
	}

	// Create session
	session := &Session{
		ID:        generateID("sess"),
		UserID:    user.ID,
		Token:     token,
		ExpiresAt: time.Now().Add(24 * time.Hour),
		CreatedAt: time.Now(),
		LastUsed:  time.Now(),
	}

	if err := m.userStorage.SaveSession(session); err != nil {
		log.Printf("Warning: failed to save session: %v", err)
	}

	log.Printf("User logged in: %s (%s)", user.Email, user.ID)

	return token, user.Sanitize(), nil
}

// Register creates a new user account
func (m *AuthManager) Register(registration *UserRegistration) (*User, error) {
	// Validate email
	if err := ValidateEmail(registration.Email); err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	// Validate password
	if err := ValidatePassword(registration.Password); err != nil {
		return nil, fmt.Errorf("invalid password: %w", err)
	}

	// Check if user already exists
	if _, err := m.userStorage.GetUserByEmail(registration.Email); err == nil {
		return nil, fmt.Errorf("user with email %s already exists", registration.Email)
	}

	// Hash password
	hashedPassword, err := m.passwordHasher.HashPassword(registration.Password)
	if err != nil {
		return nil, fmt.Errorf("failed to hash password: %w", err)
	}

	// Determine role (first user is admin, others are members)
	role := string(RoleMember)
	users, _ := m.userStorage.ListUsers()
	if len(users) == 0 {
		role = string(RoleAdmin)
		log.Println("First user registration - assigning admin role")
	}

	// Create user
	user := &User{
		ID:        generateID("user"),
		Email:     registration.Email,
		Password:  hashedPassword,
		FirstName: registration.FirstName,
		LastName:  registration.LastName,
		Role:      role,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := m.userStorage.SaveUser(user); err != nil {
		return nil, fmt.Errorf("failed to save user: %w", err)
	}

	log.Printf("User registered: %s (%s) - role: %s", user.Email, user.ID, user.Role)

	return user.Sanitize(), nil
}

// ValidateToken validates a JWT token and returns the claims
func (m *AuthManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	claims, err := m.jwtManager.ValidateToken(tokenString)
	if err != nil {
		return nil, err
	}

	// Verify user still exists and is active
	user, err := m.userStorage.GetUser(claims.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if !user.Active {
		return nil, errors.New("user account is deactivated")
	}

	return claims, nil
}

// Logout invalidates a user session
func (m *AuthManager) Logout(token string) error {
	return m.userStorage.DeleteSession(token)
}

// GetCurrentUser returns the current user from a token
func (m *AuthManager) GetCurrentUser(token string) (*User, error) {
	claims, err := m.ValidateToken(token)
	if err != nil {
		return nil, err
	}

	user, err := m.userStorage.GetUser(claims.UserID)
	if err != nil {
		return nil, err
	}

	return user.Sanitize(), nil
}

// UpdateUser updates user information
func (m *AuthManager) UpdateUser(userID string, update *UserUpdate, actingUserID string) error {
	// Get acting user to check permissions
	actingUser, err := m.userStorage.GetUser(actingUserID)
	if err != nil {
		return errors.New("unauthorized")
	}

	// Check permissions
	if userID != actingUserID && actingUser.Role != string(RoleAdmin) {
		return errors.New("insufficient permissions")
	}

	// If changing password, hash it
	if update.Password != nil {
		if err := ValidatePassword(*update.Password); err != nil {
			return fmt.Errorf("invalid password: %w", err)
		}

		hashedPassword, err := m.passwordHasher.HashPassword(*update.Password)
		if err != nil {
			return fmt.Errorf("failed to hash password: %w", err)
		}
		update.Password = &hashedPassword
	}

	// Only admins can change roles
	if update.Role != nil && actingUser.Role != string(RoleAdmin) {
		return errors.New("only administrators can change user roles")
	}

	return m.userStorage.UpdateUser(userID, update)
}

// DeleteUser deletes a user account
func (m *AuthManager) DeleteUser(userID string, actingUserID string) error {
	// Only admins can delete users
	actingUser, err := m.userStorage.GetUser(actingUserID)
	if err != nil {
		return errors.New("unauthorized")
	}

	if actingUser.Role != string(RoleAdmin) {
		return errors.New("only administrators can delete users")
	}

	// Cannot delete self
	if userID == actingUserID {
		return errors.New("cannot delete your own account")
	}

	// Delete user sessions
	if err := m.userStorage.DeleteUserSessions(userID); err != nil {
		log.Printf("Warning: failed to delete user sessions: %v", err)
	}

	return m.userStorage.DeleteUser(userID)
}

// CreateAPIKey creates a new API key for a user
func (m *AuthManager) CreateAPIKey(userID, name string, expiresAt *time.Time) (*APIKey, string, error) {
	// Generate API key
	key, prefix, err := GenerateAPIKey()
	if err != nil {
		return nil, "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Hash the key for storage
	hashedKey, err := m.passwordHasher.HashPassword(key)
	if err != nil {
		return nil, "", fmt.Errorf("failed to hash API key: %w", err)
	}

	apiKey := &APIKey{
		ID:        generateID("apik"),
		UserID:    userID,
		Name:      name,
		Key:       hashedKey,
		Prefix:    prefix,
		ExpiresAt: expiresAt,
		Active:    true,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	if err := m.userStorage.SaveAPIKey(apiKey); err != nil {
		return nil, "", fmt.Errorf("failed to save API key: %w", err)
	}

	log.Printf("API key created for user %s: %s (%s)", userID, name, prefix)

	// Return the plain key (only shown once)
	return apiKey, key, nil
}

// ValidateAPIKey validates an API key and returns the user
func (m *AuthManager) ValidateAPIKey(key string) (*User, error) {
	// Get API key by prefix (first 8 chars)
	if len(key) < 8 {
		return nil, errors.New("invalid API key format")
	}

	prefix := key[:8]
	apiKey, err := m.userStorage.GetAPIKeyByPrefix(prefix)
	if err != nil {
		// Fallback to full key lookup
		apiKey, err = m.userStorage.GetAPIKey(key)
		if err != nil {
			return nil, errors.New("invalid API key")
		}
	}

	// Verify key
	if err := m.passwordHasher.VerifyPassword(key, apiKey.Key); err != nil {
		return nil, errors.New("invalid API key")
	}

	// Check if active
	if !apiKey.Active {
		return nil, errors.New("API key is inactive")
	}

	// Check expiration
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, errors.New("API key expired")
	}

	// Get user
	user, err := m.userStorage.GetUser(apiKey.UserID)
	if err != nil {
		return nil, errors.New("user not found")
	}

	if !user.Active {
		return nil, errors.New("user account is deactivated")
	}

	// Update last used timestamp
	now := time.Now()
	apiKey.LastUsed = &now
	m.userStorage.SaveAPIKey(apiKey)

	return user, nil
}

// InviteUser creates an invitation token for a new user
func (m *AuthManager) InviteUser(email, role, invitedBy string) (*InviteToken, error) {
	// Validate email
	if err := ValidateEmail(email); err != nil {
		return nil, fmt.Errorf("invalid email: %w", err)
	}

	// Validate role
	if !UserRole(role).IsValid() {
		return nil, fmt.Errorf("invalid role: %s", role)
	}

	// Check if user already exists
	if _, err := m.userStorage.GetUserByEmail(email); err == nil {
		return nil, fmt.Errorf("user with email %s already exists", email)
	}

	// Generate token
	token, err := GenerateRandomToken(32)
	if err != nil {
		return nil, fmt.Errorf("failed to generate invite token: %w", err)
	}

	invite := &InviteToken{
		ID:        generateID("inv"),
		Email:     email,
		Token:     token,
		Role:      role,
		InvitedBy: invitedBy,
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour), // 7 days
		CreatedAt: time.Now(),
	}

	if err := m.userStorage.SaveInviteToken(invite); err != nil {
		return nil, fmt.Errorf("failed to save invite token: %w", err)
	}

	log.Printf("User invited: %s (role: %s, by: %s)", email, role, invitedBy)

	return invite, nil
}

// AcceptInvite accepts an invitation and creates a user account
func (m *AuthManager) AcceptInvite(token, password string) (*User, error) {
	// Get invite
	invite, err := m.userStorage.GetInviteToken(token)
	if err != nil {
		return nil, errors.New("invalid or expired invitation")
	}

	// Create user
	registration := &UserRegistration{
		Email:    invite.Email,
		Password: password,
	}

	user, err := m.Register(registration)
	if err != nil {
		return nil, err
	}

	// Update user role from invite
	roleStr := invite.Role
	update := &UserUpdate{
		Role: &roleStr,
	}
	if err := m.userStorage.UpdateUser(user.ID, update); err != nil {
		log.Printf("Warning: failed to update user role: %v", err)
	}

	// Mark invite as used
	now := time.Now()
	invite.UsedAt = &now
	m.userStorage.SaveInviteToken(invite)

	return user, nil
}

// Helper function to generate IDs
func generateID(prefix string) string {
	return fmt.Sprintf("%s_%d", prefix, time.Now().UnixNano())
}
