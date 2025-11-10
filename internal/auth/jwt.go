package auth

import (
	"crypto/rand"
	"encoding/base64"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

// JWTManager handles JWT token generation and validation
type JWTManager struct {
	secretKey     []byte
	tokenDuration time.Duration
	issuer        string
}

// JWTClaims represents the JWT token claims
type JWTClaims struct {
	UserID string `json:"userId"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

// NewJWTManager creates a new JWT manager
func NewJWTManager(secretKey string, tokenDuration time.Duration, issuer string) *JWTManager {
	return &JWTManager{
		secretKey:     []byte(secretKey),
		tokenDuration: tokenDuration,
		issuer:        issuer,
	}
}

// GenerateToken generates a new JWT token for a user
func (m *JWTManager) GenerateToken(user *User) (string, error) {
	if user == nil {
		return "", errors.New("user cannot be nil")
	}

	now := time.Now()
	expiresAt := now.Add(m.tokenDuration)

	claims := &JWTClaims{
		UserID: user.ID,
		Email:  user.Email,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   user.ID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signedToken, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return signedToken, nil
}

// ValidateToken validates a JWT token and returns the claims
func (m *JWTManager) ValidateToken(tokenString string) (*JWTClaims, error) {
	token, err := jwt.ParseWithClaims(tokenString, &JWTClaims{}, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return m.secretKey, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("token is invalid")
	}

	claims, ok := token.Claims.(*JWTClaims)
	if !ok {
		return nil, errors.New("invalid token claims")
	}

	return claims, nil
}

// RefreshToken generates a new token for an existing valid token
func (m *JWTManager) RefreshToken(tokenString string) (string, error) {
	claims, err := m.ValidateToken(tokenString)
	if err != nil {
		return "", fmt.Errorf("invalid token for refresh: %w", err)
	}

	// Create new token with extended expiration
	now := time.Now()
	expiresAt := now.Add(m.tokenDuration)

	newClaims := &JWTClaims{
		UserID: claims.UserID,
		Email:  claims.Email,
		Role:   claims.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   claims.UserID,
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(expiresAt),
			NotBefore: jwt.NewNumericDate(now),
			Issuer:    m.issuer,
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, newClaims)
	signedToken, err := token.SignedString(m.secretKey)
	if err != nil {
		return "", fmt.Errorf("failed to sign refreshed token: %w", err)
	}

	return signedToken, nil
}

// GenerateRandomToken generates a random token for invites or password resets
func GenerateRandomToken(length int) (string, error) {
	if length < 16 {
		length = 16 // Minimum secure length
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random token: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// GenerateAPIKey generates a secure API key
func GenerateAPIKey() (key string, prefix string, err error) {
	// Generate 32 bytes (256 bits) of random data
	bytes := make([]byte, 32)
	if _, err := rand.Read(bytes); err != nil {
		return "", "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Encode as base64
	key = base64.URLEncoding.EncodeToString(bytes)

	// Prefix is first 8 characters for identification
	if len(key) >= 8 {
		prefix = key[:8]
	} else {
		prefix = key
	}

	return key, prefix, nil
}
