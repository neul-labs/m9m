package auth

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/dipankar/m9m/internal/storage"
)

// UserStorage defines the interface for user persistence
type UserStorage interface {
	// User operations
	SaveUser(user *User) error
	GetUser(id string) (*User, error)
	GetUserByEmail(email string) (*User, error)
	ListUsers() ([]*User, error)
	UpdateUser(id string, update *UserUpdate) error
	DeleteUser(id string) error

	// Session operations
	SaveSession(session *Session) error
	GetSession(token string) (*Session, error)
	DeleteSession(token string) error
	DeleteUserSessions(userID string) error

	// API Key operations
	SaveAPIKey(apiKey *APIKey) error
	GetAPIKey(key string) (*APIKey, error)
	GetAPIKeyByPrefix(prefix string) (*APIKey, error)
	ListUserAPIKeys(userID string) ([]*APIKey, error)
	DeleteAPIKey(id string) error

	// Invite operations
	SaveInviteToken(invite *InviteToken) error
	GetInviteToken(token string) (*InviteToken, error)
	DeleteInviteToken(token string) error

	// Password reset operations
	SavePasswordResetToken(token *PasswordResetToken) error
	GetPasswordResetToken(token string) (*PasswordResetToken, error)
	DeletePasswordResetToken(token string) error
}

// MemoryUserStorage implements UserStorage using in-memory storage
type MemoryUserStorage struct {
	workflowStorage storage.WorkflowStorage
	users           map[string]*User
	emailIndex      map[string]*User
	sessions        map[string]*Session
	apiKeys         map[string]*APIKey
	apiKeysByPrefix map[string]*APIKey
	invites         map[string]*InviteToken
	resetTokens     map[string]*PasswordResetToken
	mu              sync.RWMutex
}

// NewMemoryUserStorage creates a new in-memory user storage
func NewMemoryUserStorage(workflowStorage storage.WorkflowStorage) *MemoryUserStorage {
	return &MemoryUserStorage{
		workflowStorage: workflowStorage,
		users:           make(map[string]*User),
		emailIndex:      make(map[string]*User),
		sessions:        make(map[string]*Session),
		apiKeys:         make(map[string]*APIKey),
		apiKeysByPrefix: make(map[string]*APIKey),
		invites:         make(map[string]*InviteToken),
		resetTokens:     make(map[string]*PasswordResetToken),
	}
}

// User operations

func (s *MemoryUserStorage) SaveUser(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if user.ID == "" {
		return errors.New("user ID cannot be empty")
	}

	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	// Check for duplicate email
	if existing, exists := s.emailIndex[user.Email]; exists && existing.ID != user.ID {
		return fmt.Errorf("user with email %s already exists", user.Email)
	}

	s.users[user.ID] = user
	s.emailIndex[user.Email] = user

	return nil
}

func (s *MemoryUserStorage) GetUser(id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, fmt.Errorf("user not found: %s", id)
	}

	return user, nil
}

func (s *MemoryUserStorage) GetUserByEmail(email string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.emailIndex[email]
	if !exists {
		return nil, fmt.Errorf("user not found with email: %s", email)
	}

	return user, nil
}

func (s *MemoryUserStorage) ListUsers() ([]*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*User, 0, len(s.users))
	for _, user := range s.users {
		users = append(users, user)
	}

	return users, nil
}

func (s *MemoryUserStorage) UpdateUser(id string, update *UserUpdate) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[id]
	if !exists {
		return fmt.Errorf("user not found: %s", id)
	}

	// Apply updates
	if update.FirstName != nil {
		user.FirstName = *update.FirstName
	}
	if update.LastName != nil {
		user.LastName = *update.LastName
	}
	if update.Email != nil && *update.Email != user.Email {
		// Check for duplicate email
		if _, exists := s.emailIndex[*update.Email]; exists {
			return fmt.Errorf("user with email %s already exists", *update.Email)
		}
		// Remove old email index
		delete(s.emailIndex, user.Email)
		user.Email = *update.Email
		s.emailIndex[user.Email] = user
	}
	if update.Password != nil {
		user.Password = *update.Password
	}
	if update.Role != nil {
		user.Role = *update.Role
	}
	if update.Active != nil {
		user.Active = *update.Active
	}
	if update.Settings != nil {
		user.Settings = *update.Settings
	}

	user.UpdatedAt = time.Now()

	return nil
}

func (s *MemoryUserStorage) DeleteUser(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	user, exists := s.users[id]
	if !exists {
		return fmt.Errorf("user not found: %s", id)
	}

	delete(s.users, id)
	delete(s.emailIndex, user.Email)

	return nil
}

// Session operations

func (s *MemoryUserStorage) SaveSession(session *Session) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session.Token == "" {
		return errors.New("session token cannot be empty")
	}

	if session.CreatedAt.IsZero() {
		session.CreatedAt = time.Now()
	}
	session.LastUsed = time.Now()

	s.sessions[session.Token] = session

	return nil
}

func (s *MemoryUserStorage) GetSession(token string) (*Session, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, exists := s.sessions[token]
	if !exists {
		return nil, fmt.Errorf("session not found")
	}

	// Check expiration
	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return session, nil
}

func (s *MemoryUserStorage) DeleteSession(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.sessions, token)
	return nil
}

func (s *MemoryUserStorage) DeleteUserSessions(userID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for token, session := range s.sessions {
		if session.UserID == userID {
			delete(s.sessions, token)
		}
	}

	return nil
}

// API Key operations

func (s *MemoryUserStorage) SaveAPIKey(apiKey *APIKey) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if apiKey.ID == "" {
		return errors.New("API key ID cannot be empty")
	}

	now := time.Now()
	if apiKey.CreatedAt.IsZero() {
		apiKey.CreatedAt = now
	}
	apiKey.UpdatedAt = now

	s.apiKeys[apiKey.Key] = apiKey
	if apiKey.Prefix != "" {
		s.apiKeysByPrefix[apiKey.Prefix] = apiKey
	}

	return nil
}

func (s *MemoryUserStorage) GetAPIKey(key string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	apiKey, exists := s.apiKeys[key]
	if !exists {
		return nil, fmt.Errorf("API key not found")
	}

	// Check expiration
	if apiKey.ExpiresAt != nil && time.Now().After(*apiKey.ExpiresAt) {
		return nil, fmt.Errorf("API key expired")
	}

	return apiKey, nil
}

func (s *MemoryUserStorage) GetAPIKeyByPrefix(prefix string) (*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	apiKey, exists := s.apiKeysByPrefix[prefix]
	if !exists {
		return nil, fmt.Errorf("API key not found with prefix: %s", prefix)
	}

	return apiKey, nil
}

func (s *MemoryUserStorage) ListUserAPIKeys(userID string) ([]*APIKey, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var keys []*APIKey
	for _, apiKey := range s.apiKeys {
		if apiKey.UserID == userID {
			keys = append(keys, apiKey)
		}
	}

	return keys, nil
}

func (s *MemoryUserStorage) DeleteAPIKey(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for key, apiKey := range s.apiKeys {
		if apiKey.ID == id {
			delete(s.apiKeys, key)
			if apiKey.Prefix != "" {
				delete(s.apiKeysByPrefix, apiKey.Prefix)
			}
			return nil
		}
	}

	return fmt.Errorf("API key not found: %s", id)
}

// Invite operations

func (s *MemoryUserStorage) SaveInviteToken(invite *InviteToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if invite.Token == "" {
		return errors.New("invite token cannot be empty")
	}

	if invite.CreatedAt.IsZero() {
		invite.CreatedAt = time.Now()
	}

	s.invites[invite.Token] = invite

	return nil
}

func (s *MemoryUserStorage) GetInviteToken(token string) (*InviteToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	invite, exists := s.invites[token]
	if !exists {
		return nil, fmt.Errorf("invite token not found")
	}

	// Check expiration
	if time.Now().After(invite.ExpiresAt) {
		return nil, fmt.Errorf("invite token expired")
	}

	// Check if already used
	if invite.UsedAt != nil {
		return nil, fmt.Errorf("invite token already used")
	}

	return invite, nil
}

func (s *MemoryUserStorage) DeleteInviteToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.invites, token)
	return nil
}

// Password reset operations

func (s *MemoryUserStorage) SavePasswordResetToken(token *PasswordResetToken) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if token.Token == "" {
		return errors.New("password reset token cannot be empty")
	}

	if token.CreatedAt.IsZero() {
		token.CreatedAt = time.Now()
	}

	s.resetTokens[token.Token] = token

	return nil
}

func (s *MemoryUserStorage) GetPasswordResetToken(token string) (*PasswordResetToken, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	resetToken, exists := s.resetTokens[token]
	if !exists {
		return nil, fmt.Errorf("password reset token not found")
	}

	// Check expiration
	if time.Now().After(resetToken.ExpiresAt) {
		return nil, fmt.Errorf("password reset token expired")
	}

	// Check if already used
	if resetToken.UsedAt != nil {
		return nil, fmt.Errorf("password reset token already used")
	}

	return resetToken, nil
}

func (s *MemoryUserStorage) DeletePasswordResetToken(token string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	delete(s.resetTokens, token)
	return nil
}

// PersistentUserStorage implements UserStorage using WorkflowStorage backend
type PersistentUserStorage struct {
	workflowStorage storage.WorkflowStorage
	mu              sync.RWMutex
}

// NewPersistentUserStorage creates a persistent user storage
func NewPersistentUserStorage(workflowStorage storage.WorkflowStorage) *PersistentUserStorage {
	return &PersistentUserStorage{
		workflowStorage: workflowStorage,
	}
}

// User operations

func (s *PersistentUserStorage) SaveUser(user *User) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if user.ID == "" {
		return errors.New("user ID cannot be empty")
	}

	now := time.Now()
	if user.CreatedAt.IsZero() {
		user.CreatedAt = now
	}
	user.UpdatedAt = now

	data, err := json.Marshal(user)
	if err != nil {
		return fmt.Errorf("failed to marshal user: %w", err)
	}

	key := fmt.Sprintf("user:%s", user.ID)
	if err := s.workflowStorage.SaveRaw(key, data); err != nil {
		return fmt.Errorf("failed to save user: %w", err)
	}

	// Save email index
	emailKey := fmt.Sprintf("user_email:%s", user.Email)
	if err := s.workflowStorage.SaveRaw(emailKey, []byte(user.ID)); err != nil {
		return fmt.Errorf("failed to save email index: %w", err)
	}

	return nil
}

func (s *PersistentUserStorage) GetUser(id string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	key := fmt.Sprintf("user:%s", id)
	data, err := s.workflowStorage.GetRaw(key)
	if err != nil {
		return nil, fmt.Errorf("user not found: %s", id)
	}

	var user User
	if err := json.Unmarshal(data, &user); err != nil {
		return nil, fmt.Errorf("failed to unmarshal user: %w", err)
	}

	return &user, nil
}

func (s *PersistentUserStorage) GetUserByEmail(email string) (*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	emailKey := fmt.Sprintf("user_email:%s", email)
	userID, err := s.workflowStorage.GetRaw(emailKey)
	if err != nil {
		return nil, fmt.Errorf("user not found with email: %s", email)
	}

	return s.GetUser(string(userID))
}

func (s *PersistentUserStorage) ListUsers() ([]*User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	keys, err := s.workflowStorage.ListKeys("user:")
	if err != nil {
		return nil, err
	}

	var users []*User
	for _, key := range keys {
		data, err := s.workflowStorage.GetRaw(key)
		if err != nil {
			continue
		}

		var user User
		if err := json.Unmarshal(data, &user); err != nil {
			continue
		}

		users = append(users, &user)
	}

	return users, nil
}

func (s *PersistentUserStorage) UpdateUser(id string, update *UserUpdate) error {
	user, err := s.GetUser(id)
	if err != nil {
		return err
	}

	// Apply updates
	if update.FirstName != nil {
		user.FirstName = *update.FirstName
	}
	if update.LastName != nil {
		user.LastName = *update.LastName
	}
	if update.Email != nil && *update.Email != user.Email {
		// Remove old email index
		oldEmailKey := fmt.Sprintf("user_email:%s", user.Email)
		s.workflowStorage.DeleteRaw(oldEmailKey)

		user.Email = *update.Email
	}
	if update.Password != nil {
		user.Password = *update.Password
	}
	if update.Role != nil {
		user.Role = *update.Role
	}
	if update.Active != nil {
		user.Active = *update.Active
	}
	if update.Settings != nil {
		user.Settings = *update.Settings
	}

	return s.SaveUser(user)
}

func (s *PersistentUserStorage) DeleteUser(id string) error {
	user, err := s.GetUser(id)
	if err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	key := fmt.Sprintf("user:%s", id)
	if err := s.workflowStorage.DeleteRaw(key); err != nil {
		return err
	}

	// Delete email index
	emailKey := fmt.Sprintf("user_email:%s", user.Email)
	s.workflowStorage.DeleteRaw(emailKey)

	return nil
}

// Session, APIKey, Invite, and PasswordReset operations follow similar pattern
// Implementing simplified versions for brevity

func (s *PersistentUserStorage) SaveSession(session *Session) error {
	data, err := json.Marshal(session)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("session:%s", session.Token)
	return s.workflowStorage.SaveRaw(key, data)
}

func (s *PersistentUserStorage) GetSession(token string) (*Session, error) {
	key := fmt.Sprintf("session:%s", token)
	data, err := s.workflowStorage.GetRaw(key)
	if err != nil {
		return nil, fmt.Errorf("session not found")
	}

	var session Session
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, fmt.Errorf("session expired")
	}

	return &session, nil
}

func (s *PersistentUserStorage) DeleteSession(token string) error {
	key := fmt.Sprintf("session:%s", token)
	return s.workflowStorage.DeleteRaw(key)
}

func (s *PersistentUserStorage) DeleteUserSessions(userID string) error {
	// List all sessions and delete matching ones
	keys, err := s.workflowStorage.ListKeys("session:")
	if err != nil {
		return err
	}

	for _, key := range keys {
		data, err := s.workflowStorage.GetRaw(key)
		if err != nil {
			continue
		}

		var session Session
		if err := json.Unmarshal(data, &session); err != nil {
			continue
		}

		if session.UserID == userID {
			s.workflowStorage.DeleteRaw(key)
		}
	}

	return nil
}

// Stub implementations for other methods (similar pattern)
func (s *PersistentUserStorage) SaveAPIKey(apiKey *APIKey) error {
	data, _ := json.Marshal(apiKey)
	return s.workflowStorage.SaveRaw(fmt.Sprintf("apikey:%s", apiKey.Key), data)
}

func (s *PersistentUserStorage) GetAPIKey(key string) (*APIKey, error) {
	data, err := s.workflowStorage.GetRaw(fmt.Sprintf("apikey:%s", key))
	if err != nil {
		return nil, err
	}
	var apiKey APIKey
	json.Unmarshal(data, &apiKey)
	return &apiKey, nil
}

func (s *PersistentUserStorage) GetAPIKeyByPrefix(prefix string) (*APIKey, error) {
	return nil, errors.New("not implemented - use GetAPIKey with full key")
}

func (s *PersistentUserStorage) ListUserAPIKeys(userID string) ([]*APIKey, error) {
	return []*APIKey{}, nil // Simplified
}

func (s *PersistentUserStorage) DeleteAPIKey(id string) error {
	return s.workflowStorage.DeleteRaw(fmt.Sprintf("apikey:%s", id))
}

func (s *PersistentUserStorage) SaveInviteToken(invite *InviteToken) error {
	data, _ := json.Marshal(invite)
	return s.workflowStorage.SaveRaw(fmt.Sprintf("invite:%s", invite.Token), data)
}

func (s *PersistentUserStorage) GetInviteToken(token string) (*InviteToken, error) {
	data, err := s.workflowStorage.GetRaw(fmt.Sprintf("invite:%s", token))
	if err != nil {
		return nil, err
	}
	var invite InviteToken
	json.Unmarshal(data, &invite)
	return &invite, nil
}

func (s *PersistentUserStorage) DeleteInviteToken(token string) error {
	return s.workflowStorage.DeleteRaw(fmt.Sprintf("invite:%s", token))
}

func (s *PersistentUserStorage) SavePasswordResetToken(token *PasswordResetToken) error {
	data, _ := json.Marshal(token)
	return s.workflowStorage.SaveRaw(fmt.Sprintf("reset:%s", token.Token), data)
}

func (s *PersistentUserStorage) GetPasswordResetToken(token string) (*PasswordResetToken, error) {
	data, err := s.workflowStorage.GetRaw(fmt.Sprintf("reset:%s", token))
	if err != nil {
		return nil, err
	}
	var resetToken PasswordResetToken
	json.Unmarshal(data, &resetToken)
	return &resetToken, nil
}

func (s *PersistentUserStorage) DeletePasswordResetToken(token string) error {
	return s.workflowStorage.DeleteRaw(fmt.Sprintf("reset:%s", token))
}
