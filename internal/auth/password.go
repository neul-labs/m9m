package auth

import (
	"errors"
	"fmt"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordLength is the minimum password length
	MinPasswordLength = 8

	// BcryptCost is the cost factor for bcrypt hashing
	BcryptCost = 12
)

// PasswordHasher handles password hashing and verification
type PasswordHasher struct {
	cost int
}

// NewPasswordHasher creates a new password hasher
func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{
		cost: BcryptCost,
	}
}

// HashPassword hashes a password using bcrypt
func (h *PasswordHasher) HashPassword(password string) (string, error) {
	if err := ValidatePassword(password); err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), h.cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword verifies a password against a hash
func (h *PasswordHasher) VerifyPassword(password, hash string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	if err != nil {
		if errors.Is(err, bcrypt.ErrMismatchedHashAndPassword) {
			return errors.New("invalid password")
		}
		return fmt.Errorf("failed to verify password: %w", err)
	}

	return nil
}

// ValidatePassword validates password strength
func ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters", MinPasswordLength)
	}

	var (
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return errors.New("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return errors.New("password must contain at least one lowercase letter")
	}
	if !hasNumber {
		return errors.New("password must contain at least one number")
	}
	if !hasSpecial {
		return errors.New("password must contain at least one special character")
	}

	return nil
}

// ValidateEmail validates email format
func ValidateEmail(email string) error {
	if len(email) == 0 {
		return errors.New("email cannot be empty")
	}

	// Basic email validation
	// For production, consider using a proper email validation library
	hasAt := false
	hasDot := false
	atIndex := -1

	for i, char := range email {
		if char == '@' {
			if hasAt {
				return errors.New("email cannot contain multiple @ symbols")
			}
			hasAt = true
			atIndex = i
		}
		if char == '.' && hasAt && i > atIndex {
			hasDot = true
		}
	}

	if !hasAt {
		return errors.New("email must contain @ symbol")
	}
	if !hasDot {
		return errors.New("email must contain domain extension")
	}
	if atIndex == 0 {
		return errors.New("email cannot start with @")
	}
	if atIndex == len(email)-1 {
		return errors.New("email cannot end with @")
	}

	return nil
}
