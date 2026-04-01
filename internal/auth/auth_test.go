package auth

import (
	"testing"
	"time"

	"github.com/neul-labs/m9m/internal/storage"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// validPassword is a password that satisfies all validation rules.
const validPassword = "Str0ng!Pass"

// setupAuthManager creates a fresh AuthManager backed by MemoryUserStorage
// for use in tests. The JWT token duration defaults to 60 seconds so that
// tokens are long-lived enough for normal test execution but short enough
// to test expiry when needed.
func setupAuthManager() (*AuthManager, *MemoryUserStorage) {
	memStorage := storage.NewMemoryStorage()
	userStorage := NewMemoryUserStorage(memStorage)
	manager := NewAuthManager(userStorage, "test-secret-key-for-jwt", 60*time.Second)
	return manager, userStorage
}

// registerTestUser is a convenience helper that registers a user with the
// given email and returns the created user. It fails the test on error.
func registerTestUser(t *testing.T, manager *AuthManager, email string) *User {
	t.Helper()
	reg := &UserRegistration{
		Email:     email,
		Password:  validPassword,
		FirstName: "Test",
		LastName:  "User",
	}
	user, err := manager.Register(reg)
	require.NoError(t, err, "registerTestUser should succeed")
	require.NotNil(t, user)
	return user
}

// ---------------------------------------------------------------------------
// 1. TestValidatePassword
// ---------------------------------------------------------------------------

func TestValidatePassword(t *testing.T) {
	tests := []struct {
		name      string
		password  string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid password",
			password:  "Str0ng!Pass",
			expectErr: false,
		},
		{
			name:      "too short",
			password:  "Ab1!",
			expectErr: true,
			errMsg:    "at least 8 characters",
		},
		{
			name:      "missing uppercase",
			password:  "str0ng!pass",
			expectErr: true,
			errMsg:    "uppercase",
		},
		{
			name:      "missing lowercase",
			password:  "STR0NG!PASS",
			expectErr: true,
			errMsg:    "lowercase",
		},
		{
			name:      "missing number",
			password:  "Strong!Pass",
			expectErr: true,
			errMsg:    "number",
		},
		{
			name:      "missing special character",
			password:  "Str0ngPassw",
			expectErr: true,
			errMsg:    "special character",
		},
		{
			name:      "empty password",
			password:  "",
			expectErr: true,
			errMsg:    "at least 8 characters",
		},
		{
			name:      "exactly 8 chars with all requirements",
			password:  "Abcde1!x",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidatePassword(tt.password)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 2. TestValidateEmail
// ---------------------------------------------------------------------------

func TestValidateEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		expectErr bool
		errMsg    string
	}{
		{
			name:      "valid email",
			email:     "user@example.com",
			expectErr: false,
		},
		{
			name:      "valid email with subdomain",
			email:     "user@mail.example.com",
			expectErr: false,
		},
		{
			name:      "empty email",
			email:     "",
			expectErr: true,
			errMsg:    "cannot be empty",
		},
		{
			name:      "missing at symbol",
			email:     "userexample.com",
			expectErr: true,
			errMsg:    "@ symbol",
		},
		{
			name:      "missing domain extension",
			email:     "user@example",
			expectErr: true,
			errMsg:    "domain extension",
		},
		{
			name:      "starts with at",
			email:     "@example.com",
			expectErr: true,
			errMsg:    "cannot start with @",
		},
		{
			name:      "ends with at",
			email:     "user@",
			expectErr: true,
			errMsg:    "domain extension",
		},
		{
			name:      "multiple at symbols",
			email:     "user@@example.com",
			expectErr: true,
			errMsg:    "multiple @",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ValidateEmail(tt.email)
			if tt.expectErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.errMsg)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// 3. TestPasswordHasher_HashAndVerify
// ---------------------------------------------------------------------------

func TestPasswordHasher_HashAndVerify(t *testing.T) {
	hasher := NewPasswordHasher()

	t.Run("hash and verify correct password", func(t *testing.T) {
		hash, err := hasher.HashPassword(validPassword)
		require.NoError(t, err)
		assert.NotEmpty(t, hash)
		assert.NotEqual(t, validPassword, hash, "hash must differ from plaintext")

		err = hasher.VerifyPassword(validPassword, hash)
		assert.NoError(t, err)
	})

	t.Run("verify wrong password fails", func(t *testing.T) {
		hash, err := hasher.HashPassword(validPassword)
		require.NoError(t, err)

		err = hasher.VerifyPassword("Wr0ng!Password", hash)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid password")
	})

	t.Run("hash invalid password fails", func(t *testing.T) {
		_, err := hasher.HashPassword("short")
		assert.Error(t, err)
	})

	t.Run("different hashes for same password", func(t *testing.T) {
		hash1, err := hasher.HashPassword(validPassword)
		require.NoError(t, err)

		hash2, err := hasher.HashPassword(validPassword)
		require.NoError(t, err)

		assert.NotEqual(t, hash1, hash2, "bcrypt should produce different hashes due to salt")
	})
}

// ---------------------------------------------------------------------------
// 4. TestJWTManager_GenerateAndValidate
// ---------------------------------------------------------------------------

func TestJWTManager_GenerateAndValidate(t *testing.T) {
	jwtMgr := NewJWTManager("test-secret", 60*time.Second, "m9m-test")

	user := &User{
		ID:    "user_123",
		Email: "jwt@example.com",
		Role:  string(RoleAdmin),
	}

	t.Run("generate and validate token", func(t *testing.T) {
		token, err := jwtMgr.GenerateToken(user)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		claims, err := jwtMgr.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Role, claims.Role)
		assert.Equal(t, user.ID, claims.Subject)
		assert.Equal(t, "m9m-test", claims.Issuer)
	})

	t.Run("nil user returns error", func(t *testing.T) {
		_, err := jwtMgr.GenerateToken(nil)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "nil")
	})

	t.Run("invalid token string fails validation", func(t *testing.T) {
		_, err := jwtMgr.ValidateToken("this-is-not-a-jwt")
		assert.Error(t, err)
	})

	t.Run("token signed with different secret fails validation", func(t *testing.T) {
		otherMgr := NewJWTManager("other-secret", 60*time.Second, "m9m-test")
		token, err := otherMgr.GenerateToken(user)
		require.NoError(t, err)

		_, err = jwtMgr.ValidateToken(token)
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// 5. TestJWTManager_ExpiredToken
// ---------------------------------------------------------------------------

func TestJWTManager_ExpiredToken(t *testing.T) {
	// Use a 1-millisecond duration so the token expires almost immediately.
	jwtMgr := NewJWTManager("test-secret", 1*time.Millisecond, "m9m-test")

	user := &User{
		ID:    "user_expire",
		Email: "expire@example.com",
		Role:  string(RoleMember),
	}

	token, err := jwtMgr.GenerateToken(user)
	require.NoError(t, err)

	// Wait for the token to expire.
	time.Sleep(50 * time.Millisecond)

	_, err = jwtMgr.ValidateToken(token)
	assert.Error(t, err, "expired token should fail validation")
}

// ---------------------------------------------------------------------------
// 6. TestJWTManager_RefreshToken
// ---------------------------------------------------------------------------

func TestJWTManager_RefreshToken(t *testing.T) {
	jwtMgr := NewJWTManager("test-secret", 60*time.Second, "m9m-test")

	user := &User{
		ID:    "user_refresh",
		Email: "refresh@example.com",
		Role:  string(RoleMember),
	}

	t.Run("refresh valid token", func(t *testing.T) {
		originalToken, err := jwtMgr.GenerateToken(user)
		require.NoError(t, err)

		// Wait just over 1 second so the new token gets a different iat/exp.
		time.Sleep(1100 * time.Millisecond)

		newToken, err := jwtMgr.RefreshToken(originalToken)
		require.NoError(t, err)
		assert.NotEmpty(t, newToken)
		assert.NotEqual(t, originalToken, newToken)

		// Validate the refreshed token carries the same identity.
		claims, err := jwtMgr.ValidateToken(newToken)
		require.NoError(t, err)
		assert.Equal(t, user.ID, claims.UserID)
		assert.Equal(t, user.Email, claims.Email)
		assert.Equal(t, user.Role, claims.Role)
	})

	t.Run("refresh invalid token fails", func(t *testing.T) {
		_, err := jwtMgr.RefreshToken("garbage-token")
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// 7. TestAuthManager_Register
// ---------------------------------------------------------------------------

func TestAuthManager_Register(t *testing.T) {
	t.Run("first user becomes admin", func(t *testing.T) {
		manager, _ := setupAuthManager()

		reg := &UserRegistration{
			Email:     "first@example.com",
			Password:  validPassword,
			FirstName: "First",
			LastName:  "User",
		}

		user, err := manager.Register(reg)
		require.NoError(t, err)
		require.NotNil(t, user)
		assert.Equal(t, string(RoleAdmin), user.Role)
		assert.Equal(t, "first@example.com", user.Email)
		assert.Equal(t, "First", user.FirstName)
		assert.Equal(t, "User", user.LastName)
		assert.True(t, user.Active)
		assert.NotEmpty(t, user.ID)
		// Sanitize should have cleared the password.
		assert.Empty(t, user.Password, "password should be sanitized")
	})

	t.Run("second user becomes member", func(t *testing.T) {
		manager, _ := setupAuthManager()

		// Register first user (admin).
		registerTestUser(t, manager, "admin@example.com")

		// Register second user.
		reg := &UserRegistration{
			Email:    "member@example.com",
			Password: validPassword,
		}
		user, err := manager.Register(reg)
		require.NoError(t, err)
		assert.Equal(t, string(RoleMember), user.Role)
	})

	t.Run("invalid email rejected", func(t *testing.T) {
		manager, _ := setupAuthManager()

		reg := &UserRegistration{
			Email:    "not-an-email",
			Password: validPassword,
		}
		_, err := manager.Register(reg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email")
	})

	t.Run("invalid password rejected", func(t *testing.T) {
		manager, _ := setupAuthManager()

		reg := &UserRegistration{
			Email:    "valid@example.com",
			Password: "short",
		}
		_, err := manager.Register(reg)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid password")
	})
}

// ---------------------------------------------------------------------------
// 8. TestAuthManager_Register_DuplicateEmail
// ---------------------------------------------------------------------------

func TestAuthManager_Register_DuplicateEmail(t *testing.T) {
	manager, _ := setupAuthManager()

	registerTestUser(t, manager, "dup@example.com")

	reg := &UserRegistration{
		Email:    "dup@example.com",
		Password: validPassword,
	}
	_, err := manager.Register(reg)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

// ---------------------------------------------------------------------------
// 9. TestAuthManager_Login
// ---------------------------------------------------------------------------

func TestAuthManager_Login(t *testing.T) {
	manager, _ := setupAuthManager()
	registerTestUser(t, manager, "login@example.com")

	t.Run("login with correct credentials", func(t *testing.T) {
		creds := &UserCredentials{
			Email:    "login@example.com",
			Password: validPassword,
		}
		token, user, err := manager.Login(creds)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
		require.NotNil(t, user)
		assert.Equal(t, "login@example.com", user.Email)
		assert.Empty(t, user.Password, "returned user should be sanitized")
	})

	t.Run("login with wrong password", func(t *testing.T) {
		creds := &UserCredentials{
			Email:    "login@example.com",
			Password: "Wr0ng!Password",
		}
		_, _, err := manager.Login(creds)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email or password")
	})

	t.Run("login with unknown email", func(t *testing.T) {
		creds := &UserCredentials{
			Email:    "unknown@example.com",
			Password: validPassword,
		}
		_, _, err := manager.Login(creds)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email or password")
	})

	t.Run("login with empty credentials", func(t *testing.T) {
		creds := &UserCredentials{}
		_, _, err := manager.Login(creds)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "required")
	})
}

// ---------------------------------------------------------------------------
// 10. TestAuthManager_LoginInactiveUser
// ---------------------------------------------------------------------------

func TestAuthManager_LoginInactiveUser(t *testing.T) {
	manager, userStorage := setupAuthManager()
	user := registerTestUser(t, manager, "inactive@example.com")

	// Deactivate the user directly in storage.
	active := false
	err := userStorage.UpdateUser(user.ID, &UserUpdate{Active: &active})
	require.NoError(t, err)

	creds := &UserCredentials{
		Email:    "inactive@example.com",
		Password: validPassword,
	}
	_, _, err = manager.Login(creds)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deactivated")
}

// ---------------------------------------------------------------------------
// 11. TestAuthManager_ValidateToken
// ---------------------------------------------------------------------------

func TestAuthManager_ValidateToken(t *testing.T) {
	manager, _ := setupAuthManager()
	registerTestUser(t, manager, "validate@example.com")

	creds := &UserCredentials{
		Email:    "validate@example.com",
		Password: validPassword,
	}
	token, _, err := manager.Login(creds)
	require.NoError(t, err)

	t.Run("valid token", func(t *testing.T) {
		claims, err := manager.ValidateToken(token)
		require.NoError(t, err)
		assert.Equal(t, "validate@example.com", claims.Email)
	})

	t.Run("invalid token string", func(t *testing.T) {
		_, err := manager.ValidateToken("bogus")
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// 12. TestAuthManager_GetCurrentUser
// ---------------------------------------------------------------------------

func TestAuthManager_GetCurrentUser(t *testing.T) {
	manager, _ := setupAuthManager()
	registered := registerTestUser(t, manager, "current@example.com")

	creds := &UserCredentials{
		Email:    "current@example.com",
		Password: validPassword,
	}
	token, _, err := manager.Login(creds)
	require.NoError(t, err)

	t.Run("get current user from valid token", func(t *testing.T) {
		user, err := manager.GetCurrentUser(token)
		require.NoError(t, err)
		assert.Equal(t, registered.ID, user.ID)
		assert.Equal(t, "current@example.com", user.Email)
		assert.Empty(t, user.Password, "password should be sanitized")
	})

	t.Run("invalid token returns error", func(t *testing.T) {
		_, err := manager.GetCurrentUser("bad-token")
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// 13. TestAuthManager_Logout
// ---------------------------------------------------------------------------

func TestAuthManager_Logout(t *testing.T) {
	manager, userStorage := setupAuthManager()
	registerTestUser(t, manager, "logout@example.com")

	creds := &UserCredentials{
		Email:    "logout@example.com",
		Password: validPassword,
	}
	token, _, err := manager.Login(creds)
	require.NoError(t, err)

	// Session should exist.
	_, err = userStorage.GetSession(token)
	require.NoError(t, err)

	// Logout.
	err = manager.Logout(token)
	require.NoError(t, err)

	// Session should be gone.
	_, err = userStorage.GetSession(token)
	assert.Error(t, err)
}

// ---------------------------------------------------------------------------
// 14. TestAuthManager_CreateAPIKey
// ---------------------------------------------------------------------------

func TestAuthManager_CreateAPIKey(t *testing.T) {
	manager, _ := setupAuthManager()
	user := registerTestUser(t, manager, "apikey@example.com")

	t.Run("create and validate API key", func(t *testing.T) {
		apiKey, plainKey, err := manager.CreateAPIKey(user.ID, "test-key", nil)
		require.NoError(t, err)
		assert.NotNil(t, apiKey)
		assert.NotEmpty(t, plainKey)
		assert.Equal(t, "test-key", apiKey.Name)
		assert.Equal(t, user.ID, apiKey.UserID)
		assert.True(t, apiKey.Active)
		assert.NotEmpty(t, apiKey.Prefix)

		// Validate the key.
		validatedUser, err := manager.ValidateAPIKey(plainKey)
		require.NoError(t, err)
		assert.Equal(t, user.ID, validatedUser.ID)
	})

	t.Run("invalid API key fails validation", func(t *testing.T) {
		_, err := manager.ValidateAPIKey("totally-wrong-key-value")
		assert.Error(t, err)
	})

	t.Run("short API key format rejected", func(t *testing.T) {
		_, err := manager.ValidateAPIKey("short")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid API key")
	})

	t.Run("create API key with expiration", func(t *testing.T) {
		future := time.Now().Add(24 * time.Hour)
		apiKey, _, err := manager.CreateAPIKey(user.ID, "expiring-key", &future)
		require.NoError(t, err)
		assert.NotNil(t, apiKey.ExpiresAt)
	})
}

// ---------------------------------------------------------------------------
// 15. TestAuthManager_InviteAndAccept
// ---------------------------------------------------------------------------

func TestAuthManager_InviteAndAccept(t *testing.T) {
	manager, userStorage := setupAuthManager()
	admin := registerTestUser(t, manager, "inviter@example.com")

	t.Run("invite user and accept", func(t *testing.T) {
		invite, err := manager.InviteUser("invitee@example.com", string(RoleMember), admin.ID)
		require.NoError(t, err)
		assert.NotEmpty(t, invite.Token)
		assert.Equal(t, "invitee@example.com", invite.Email)
		assert.Equal(t, string(RoleMember), invite.Role)

		// Accept the invite.
		newUser, err := manager.AcceptInvite(invite.Token, validPassword)
		require.NoError(t, err)
		assert.Equal(t, "invitee@example.com", newUser.Email)

		// The user should now be retrievable from storage.
		storedUser, err := userStorage.GetUserByEmail("invitee@example.com")
		require.NoError(t, err)
		assert.Equal(t, string(RoleMember), storedUser.Role)
	})

	t.Run("invite with invalid email fails", func(t *testing.T) {
		_, err := manager.InviteUser("bad-email", string(RoleMember), admin.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid email")
	})

	t.Run("invite with invalid role fails", func(t *testing.T) {
		_, err := manager.InviteUser("valid2@example.com", "superuser", admin.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid role")
	})

	t.Run("invite existing email fails", func(t *testing.T) {
		_, err := manager.InviteUser("inviter@example.com", string(RoleMember), admin.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})

	t.Run("accept with invalid token fails", func(t *testing.T) {
		_, err := manager.AcceptInvite("nonexistent-token", validPassword)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "invalid or expired")
	})
}

// ---------------------------------------------------------------------------
// 16. TestMemoryUserStorage_UserCRUD
// ---------------------------------------------------------------------------

func TestMemoryUserStorage_UserCRUD(t *testing.T) {
	memStorage := storage.NewMemoryStorage()
	s := NewMemoryUserStorage(memStorage)

	user := &User{
		ID:        "user_crud_1",
		Email:     "crud@example.com",
		Password:  "hashed",
		FirstName: "CRUD",
		LastName:  "Test",
		Role:      string(RoleMember),
		Active:    true,
	}

	t.Run("save user", func(t *testing.T) {
		err := s.SaveUser(user)
		require.NoError(t, err)
	})

	t.Run("get user by ID", func(t *testing.T) {
		got, err := s.GetUser("user_crud_1")
		require.NoError(t, err)
		assert.Equal(t, "crud@example.com", got.Email)
	})

	t.Run("get user by email", func(t *testing.T) {
		got, err := s.GetUserByEmail("crud@example.com")
		require.NoError(t, err)
		assert.Equal(t, "user_crud_1", got.ID)
	})

	t.Run("list users", func(t *testing.T) {
		users, err := s.ListUsers()
		require.NoError(t, err)
		assert.Len(t, users, 1)
	})

	t.Run("update user", func(t *testing.T) {
		newFirst := "Updated"
		err := s.UpdateUser("user_crud_1", &UserUpdate{FirstName: &newFirst})
		require.NoError(t, err)

		got, err := s.GetUser("user_crud_1")
		require.NoError(t, err)
		assert.Equal(t, "Updated", got.FirstName)
	})

	t.Run("update user email updates index", func(t *testing.T) {
		newEmail := "new-crud@example.com"
		err := s.UpdateUser("user_crud_1", &UserUpdate{Email: &newEmail})
		require.NoError(t, err)

		// Old email should not find user.
		_, err = s.GetUserByEmail("crud@example.com")
		assert.Error(t, err)

		// New email should work.
		got, err := s.GetUserByEmail("new-crud@example.com")
		require.NoError(t, err)
		assert.Equal(t, "user_crud_1", got.ID)
	})

	t.Run("delete user", func(t *testing.T) {
		err := s.DeleteUser("user_crud_1")
		require.NoError(t, err)

		_, err = s.GetUser("user_crud_1")
		assert.Error(t, err)
	})

	t.Run("get nonexistent user fails", func(t *testing.T) {
		_, err := s.GetUser("nonexistent")
		assert.Error(t, err)
	})

	t.Run("save user with empty ID fails", func(t *testing.T) {
		err := s.SaveUser(&User{Email: "noId@example.com"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ID cannot be empty")
	})

	t.Run("save duplicate email fails", func(t *testing.T) {
		u1 := &User{ID: "u1", Email: "same@example.com", Active: true}
		u2 := &User{ID: "u2", Email: "same@example.com", Active: true}
		err := s.SaveUser(u1)
		require.NoError(t, err)

		err = s.SaveUser(u2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already exists")
	})
}

// ---------------------------------------------------------------------------
// 17. TestMemoryUserStorage_SessionCRUD
// ---------------------------------------------------------------------------

func TestMemoryUserStorage_SessionCRUD(t *testing.T) {
	memStorage := storage.NewMemoryStorage()
	s := NewMemoryUserStorage(memStorage)

	session := &Session{
		ID:        "sess_1",
		UserID:    "user_1",
		Token:     "token_abc",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	t.Run("save session", func(t *testing.T) {
		err := s.SaveSession(session)
		require.NoError(t, err)
	})

	t.Run("get session", func(t *testing.T) {
		got, err := s.GetSession("token_abc")
		require.NoError(t, err)
		assert.Equal(t, "user_1", got.UserID)
	})

	t.Run("get expired session fails", func(t *testing.T) {
		expiredSession := &Session{
			ID:        "sess_2",
			UserID:    "user_1",
			Token:     "token_expired",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		err := s.SaveSession(expiredSession)
		require.NoError(t, err)

		_, err = s.GetSession("token_expired")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("delete session", func(t *testing.T) {
		err := s.DeleteSession("token_abc")
		require.NoError(t, err)

		_, err = s.GetSession("token_abc")
		assert.Error(t, err)
	})

	t.Run("delete user sessions", func(t *testing.T) {
		s1 := &Session{ID: "s1", UserID: "user_x", Token: "t1", ExpiresAt: time.Now().Add(time.Hour)}
		s2 := &Session{ID: "s2", UserID: "user_x", Token: "t2", ExpiresAt: time.Now().Add(time.Hour)}
		s3 := &Session{ID: "s3", UserID: "user_y", Token: "t3", ExpiresAt: time.Now().Add(time.Hour)}
		require.NoError(t, s.SaveSession(s1))
		require.NoError(t, s.SaveSession(s2))
		require.NoError(t, s.SaveSession(s3))

		err := s.DeleteUserSessions("user_x")
		require.NoError(t, err)

		// Sessions for user_x should be gone.
		_, err = s.GetSession("t1")
		assert.Error(t, err)
		_, err = s.GetSession("t2")
		assert.Error(t, err)

		// Session for user_y should still exist.
		got, err := s.GetSession("t3")
		require.NoError(t, err)
		assert.Equal(t, "user_y", got.UserID)
	})

	t.Run("save session with empty token fails", func(t *testing.T) {
		err := s.SaveSession(&Session{ID: "empty", UserID: "u", Token: ""})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token cannot be empty")
	})

	t.Run("get nonexistent session fails", func(t *testing.T) {
		_, err := s.GetSession("nonexistent")
		assert.Error(t, err)
	})
}

// ---------------------------------------------------------------------------
// 18. TestUserRole_Permissions
// ---------------------------------------------------------------------------

func TestUserRole_Permissions(t *testing.T) {
	t.Run("IsValid", func(t *testing.T) {
		assert.True(t, RoleAdmin.IsValid())
		assert.True(t, RoleMember.IsValid())
		assert.True(t, RoleViewer.IsValid())
		assert.False(t, UserRole("superuser").IsValid())
		assert.False(t, UserRole("").IsValid())
	})

	t.Run("admin has all permissions", func(t *testing.T) {
		perms := []string{
			"workflow:read", "workflow:write", "workflow:execute",
			"execution:read", "credential:read", "credential:write",
			"user:delete", "anything:goes",
		}
		for _, p := range perms {
			assert.True(t, RoleAdmin.HasPermission(p), "admin should have permission: %s", p)
		}
	})

	t.Run("member has limited permissions", func(t *testing.T) {
		allowed := []string{
			"workflow:read", "workflow:write", "workflow:execute",
			"execution:read", "credential:read", "credential:write",
		}
		denied := []string{
			"user:delete", "admin:manage",
		}

		for _, p := range allowed {
			assert.True(t, RoleMember.HasPermission(p), "member should have permission: %s", p)
		}
		for _, p := range denied {
			assert.False(t, RoleMember.HasPermission(p), "member should NOT have permission: %s", p)
		}
	})

	t.Run("viewer has read-only permissions", func(t *testing.T) {
		allowed := []string{"workflow:read", "execution:read"}
		denied := []string{
			"workflow:write", "workflow:execute",
			"credential:read", "credential:write",
			"user:delete",
		}

		for _, p := range allowed {
			assert.True(t, RoleViewer.HasPermission(p), "viewer should have permission: %s", p)
		}
		for _, p := range denied {
			assert.False(t, RoleViewer.HasPermission(p), "viewer should NOT have permission: %s", p)
		}
	})

	t.Run("unknown role has no permissions", func(t *testing.T) {
		unknown := UserRole("unknown")
		assert.False(t, unknown.HasPermission("workflow:read"))
	})
}

// ---------------------------------------------------------------------------
// Additional edge-case tests for AuthManager operations
// ---------------------------------------------------------------------------

func TestAuthManager_UpdateUser(t *testing.T) {
	manager, _ := setupAuthManager()
	admin := registerTestUser(t, manager, "admin-update@example.com")
	member := registerTestUser(t, manager, "member-update@example.com")

	t.Run("user can update own name", func(t *testing.T) {
		newFirst := "NewFirst"
		err := manager.UpdateUser(member.ID, &UserUpdate{FirstName: &newFirst}, member.ID)
		assert.NoError(t, err)
	})

	t.Run("non-admin cannot update other user", func(t *testing.T) {
		newFirst := "Hacker"
		err := manager.UpdateUser(admin.ID, &UserUpdate{FirstName: &newFirst}, member.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "insufficient permissions")
	})

	t.Run("admin can update other user", func(t *testing.T) {
		newFirst := "AdminUpdated"
		err := manager.UpdateUser(member.ID, &UserUpdate{FirstName: &newFirst}, admin.ID)
		assert.NoError(t, err)
	})

	t.Run("non-admin cannot change roles", func(t *testing.T) {
		newRole := string(RoleAdmin)
		err := manager.UpdateUser(member.ID, &UserUpdate{Role: &newRole}, member.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only administrators")
	})

	t.Run("admin can change roles", func(t *testing.T) {
		newRole := string(RoleViewer)
		err := manager.UpdateUser(member.ID, &UserUpdate{Role: &newRole}, admin.ID)
		assert.NoError(t, err)
	})
}

func TestAuthManager_DeleteUser(t *testing.T) {
	manager, _ := setupAuthManager()
	admin := registerTestUser(t, manager, "admin-del@example.com")
	member := registerTestUser(t, manager, "member-del@example.com")

	t.Run("non-admin cannot delete user", func(t *testing.T) {
		err := manager.DeleteUser(admin.ID, member.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "only administrators")
	})

	t.Run("admin cannot delete self", func(t *testing.T) {
		err := manager.DeleteUser(admin.ID, admin.ID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "cannot delete your own account")
	})

	t.Run("admin can delete other user", func(t *testing.T) {
		err := manager.DeleteUser(member.ID, admin.ID)
		assert.NoError(t, err)
	})
}

func TestAuthManager_ValidateToken_InactiveUser(t *testing.T) {
	manager, userStorage := setupAuthManager()
	user := registerTestUser(t, manager, "deactivate@example.com")

	// Login to get a token.
	creds := &UserCredentials{
		Email:    "deactivate@example.com",
		Password: validPassword,
	}
	token, _, err := manager.Login(creds)
	require.NoError(t, err)

	// Deactivate the user after obtaining the token.
	active := false
	err = userStorage.UpdateUser(user.ID, &UserUpdate{Active: &active})
	require.NoError(t, err)

	// ValidateToken should now fail because the user is inactive.
	_, err = manager.ValidateToken(token)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "deactivated")
}

func TestMemoryUserStorage_APIKeyCRUD(t *testing.T) {
	memStorage := storage.NewMemoryStorage()
	s := NewMemoryUserStorage(memStorage)

	apiKey := &APIKey{
		ID:     "apik_1",
		UserID: "user_1",
		Name:   "my-key",
		Key:    "hashed-key-value",
		Prefix: "hashed-k",
		Active: true,
	}

	t.Run("save API key", func(t *testing.T) {
		err := s.SaveAPIKey(apiKey)
		require.NoError(t, err)
	})

	t.Run("get API key by key", func(t *testing.T) {
		got, err := s.GetAPIKey("hashed-key-value")
		require.NoError(t, err)
		assert.Equal(t, "apik_1", got.ID)
	})

	t.Run("get API key by prefix", func(t *testing.T) {
		got, err := s.GetAPIKeyByPrefix("hashed-k")
		require.NoError(t, err)
		assert.Equal(t, "apik_1", got.ID)
	})

	t.Run("list user API keys", func(t *testing.T) {
		keys, err := s.ListUserAPIKeys("user_1")
		require.NoError(t, err)
		assert.Len(t, keys, 1)
	})

	t.Run("delete API key", func(t *testing.T) {
		err := s.DeleteAPIKey("apik_1")
		require.NoError(t, err)

		_, err = s.GetAPIKey("hashed-key-value")
		assert.Error(t, err)
	})

	t.Run("save API key with empty ID fails", func(t *testing.T) {
		err := s.SaveAPIKey(&APIKey{Key: "k", Prefix: "p"})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "ID cannot be empty")
	})

	t.Run("delete nonexistent API key fails", func(t *testing.T) {
		err := s.DeleteAPIKey("nonexistent")
		assert.Error(t, err)
	})
}

func TestMemoryUserStorage_InviteCRUD(t *testing.T) {
	memStorage := storage.NewMemoryStorage()
	s := NewMemoryUserStorage(memStorage)

	invite := &InviteToken{
		ID:        "inv_1",
		Email:     "invite@example.com",
		Token:     "invite-token-123",
		Role:      string(RoleMember),
		InvitedBy: "admin_1",
		ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
	}

	t.Run("save invite token", func(t *testing.T) {
		err := s.SaveInviteToken(invite)
		require.NoError(t, err)
	})

	t.Run("get invite token", func(t *testing.T) {
		got, err := s.GetInviteToken("invite-token-123")
		require.NoError(t, err)
		assert.Equal(t, "invite@example.com", got.Email)
	})

	t.Run("get expired invite token fails", func(t *testing.T) {
		expired := &InviteToken{
			ID:        "inv_2",
			Email:     "expired@example.com",
			Token:     "expired-token",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		err := s.SaveInviteToken(expired)
		require.NoError(t, err)

		_, err = s.GetInviteToken("expired-token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("get used invite token fails", func(t *testing.T) {
		now := time.Now()
		used := &InviteToken{
			ID:        "inv_3",
			Email:     "used@example.com",
			Token:     "used-token",
			ExpiresAt: time.Now().Add(1 * time.Hour),
			UsedAt:    &now,
		}
		err := s.SaveInviteToken(used)
		require.NoError(t, err)

		_, err = s.GetInviteToken("used-token")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already used")
	})

	t.Run("delete invite token", func(t *testing.T) {
		err := s.DeleteInviteToken("invite-token-123")
		require.NoError(t, err)

		_, err = s.GetInviteToken("invite-token-123")
		assert.Error(t, err)
	})

	t.Run("save invite with empty token fails", func(t *testing.T) {
		err := s.SaveInviteToken(&InviteToken{ID: "inv_4", Token: ""})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token cannot be empty")
	})
}

func TestMemoryUserStorage_PasswordResetCRUD(t *testing.T) {
	memStorage := storage.NewMemoryStorage()
	s := NewMemoryUserStorage(memStorage)

	reset := &PasswordResetToken{
		ID:        "rst_1",
		UserID:    "user_1",
		Token:     "reset-token-123",
		ExpiresAt: time.Now().Add(1 * time.Hour),
	}

	t.Run("save password reset token", func(t *testing.T) {
		err := s.SavePasswordResetToken(reset)
		require.NoError(t, err)
	})

	t.Run("get password reset token", func(t *testing.T) {
		got, err := s.GetPasswordResetToken("reset-token-123")
		require.NoError(t, err)
		assert.Equal(t, "user_1", got.UserID)
	})

	t.Run("get expired reset token fails", func(t *testing.T) {
		expired := &PasswordResetToken{
			ID:        "rst_2",
			UserID:    "user_1",
			Token:     "expired-reset",
			ExpiresAt: time.Now().Add(-1 * time.Hour),
		}
		err := s.SavePasswordResetToken(expired)
		require.NoError(t, err)

		_, err = s.GetPasswordResetToken("expired-reset")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "expired")
	})

	t.Run("get used reset token fails", func(t *testing.T) {
		now := time.Now()
		used := &PasswordResetToken{
			ID:        "rst_3",
			UserID:    "user_1",
			Token:     "used-reset",
			ExpiresAt: time.Now().Add(1 * time.Hour),
			UsedAt:    &now,
		}
		err := s.SavePasswordResetToken(used)
		require.NoError(t, err)

		_, err = s.GetPasswordResetToken("used-reset")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "already used")
	})

	t.Run("delete password reset token", func(t *testing.T) {
		err := s.DeletePasswordResetToken("reset-token-123")
		require.NoError(t, err)

		_, err = s.GetPasswordResetToken("reset-token-123")
		assert.Error(t, err)
	})

	t.Run("save reset token with empty token fails", func(t *testing.T) {
		err := s.SavePasswordResetToken(&PasswordResetToken{ID: "rst_4", Token: ""})
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "token cannot be empty")
	})
}

func TestGenerateRandomToken(t *testing.T) {
	t.Run("generates non-empty token", func(t *testing.T) {
		token, err := GenerateRandomToken(32)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("minimum length enforced", func(t *testing.T) {
		token, err := GenerateRandomToken(4)
		require.NoError(t, err)
		// Should use minimum of 16 bytes, base64 encoded.
		assert.NotEmpty(t, token)
	})

	t.Run("tokens are unique", func(t *testing.T) {
		t1, _ := GenerateRandomToken(32)
		t2, _ := GenerateRandomToken(32)
		assert.NotEqual(t, t1, t2)
	})
}

func TestGenerateAPIKey(t *testing.T) {
	t.Run("generates key and prefix", func(t *testing.T) {
		key, prefix, err := GenerateAPIKey()
		require.NoError(t, err)
		assert.NotEmpty(t, key)
		assert.Len(t, prefix, 8)
		assert.Equal(t, key[:8], prefix)
	})

	t.Run("keys are unique", func(t *testing.T) {
		k1, _, _ := GenerateAPIKey()
		k2, _, _ := GenerateAPIKey()
		assert.NotEqual(t, k1, k2)
	})
}

func TestUser_Sanitize(t *testing.T) {
	user := &User{
		ID:       "u1",
		Email:    "test@example.com",
		Password: "secret-hash",
		Role:     string(RoleAdmin),
		Active:   true,
	}

	sanitized := user.Sanitize()
	assert.Empty(t, sanitized.Password, "sanitized user should have empty password")
	assert.Equal(t, user.ID, sanitized.ID)
	assert.Equal(t, user.Email, sanitized.Email)
	assert.Equal(t, user.Role, sanitized.Role)

	// Original should be unchanged.
	assert.Equal(t, "secret-hash", user.Password)
}
