// Package password provides secure password hashing and validation
package password

import (
	"fmt"
	"regexp"
	"unicode"

	"golang.org/x/crypto/bcrypt"
)

const (
	// MinPasswordLength минимальная длина пароля
	MinPasswordLength = 8

	// MaxPasswordLength максимальная длина пароля
	MaxPasswordLength = 128

	// BcryptCost стоимость bcrypt (баланс между безопасностью и производительностью)
	// 12 = ~250ms на хеширование
	BcryptCost = 12
)

// ValidationError ошибка валидации пароля
type ValidationError struct {
	Field   string
	Message string
}

func (e ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Hash хеширует пароль используя bcrypt
func Hash(password string) (string, error) {
	if password == "" {
		return "", ValidationError{Field: "password", Message: "password is required"}
	}

	hashedBytes, err := bcrypt.GenerateFromPassword([]byte(password), BcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hashedBytes), nil
}

// Verify проверяет соответствие пароля его хешу
func Verify(hashedPassword, password string) error {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	if err != nil {
		if err == bcrypt.ErrMismatchedHashAndPassword {
			return ValidationError{Field: "password", Message: "incorrect password"}
		}
		return fmt.Errorf("failed to verify password: %w", err)
	}
	return nil
}

// Validate проверяет пароль на соответствие требованиям безопасности
func Validate(password string) error {
	if len(password) < MinPasswordLength {
		return ValidationError{
			Field:   "password",
			Message: fmt.Sprintf("password must be at least %d characters long", MinPasswordLength),
		}
	}

	if len(password) > MaxPasswordLength {
		return ValidationError{
			Field:   "password",
			Message: fmt.Sprintf("password must be at most %d characters long", MaxPasswordLength),
		}
	}

	// Check for at least one uppercase letter
	hasUpper := false
	// Check for at least one lowercase letter
	hasLower := false
	// Check for at least one digit
	hasDigit := false
	// Check for at least one special character
	hasSpecial := false

	for _, char := range password {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsDigit(char):
			hasDigit = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}

	if !hasUpper {
		return ValidationError{
			Field:   "password",
			Message: "password must contain at least one uppercase letter",
		}
	}

	if !hasLower {
		return ValidationError{
			Field:   "password",
			Message: "password must contain at least one lowercase letter",
		}
	}

	if !hasDigit {
		return ValidationError{
			Field:   "password",
			Message: "password must contain at least one digit",
		}
	}

	if !hasSpecial {
		return ValidationError{
			Field:   "password",
			Message: "password must contain at least one special character",
		}
	}

	return nil
}

// ValidateUsername проверяет username на соответствие требованиям
func ValidateUsername(username string) error {
	if len(username) < 3 {
		return ValidationError{
			Field:   "username",
			Message: "username must be at least 3 characters long",
		}
	}

	if len(username) > 32 {
		return ValidationError{
			Field:   "username",
			Message: "username must be at most 32 characters long",
		}
	}

	// Username должен содержать только alphanumeric + underscore + hyphen
	matched, err := regexp.MatchString(`^[a-zA-Z0-9_-]+$`, username)
	if err != nil {
		return fmt.Errorf("failed to validate username: %w", err)
	}

	if !matched {
		return ValidationError{
			Field:   "username",
			Message: "username can only contain letters, numbers, underscores, and hyphens",
		}
	}

	// Username не должен начинаться или заканчиваться на - или _
	if username[0] == '-' || username[0] == '_' || username[len(username)-1] == '-' || username[len(username)-1] == '_' {
		return ValidationError{
			Field:   "username",
			Message: "username cannot start or end with hyphen or underscore",
		}
	}

	return nil
}

// ValidateEmail проверяет email на корректность
func ValidateEmail(email string) error {
	if email == "" {
		return ValidationError{
			Field:   "email",
			Message: "email is required",
		}
	}

	// Simple email validation (не RFC-compliant, но достаточно для большинства случаев)
	emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !emailRegex.MatchString(email) {
		return ValidationError{
			Field:   "email",
			Message: "invalid email format",
		}
	}

	if len(email) > 255 {
		return ValidationError{
			Field:   "email",
			Message: "email must be at most 255 characters long",
		}
	}

	return nil
}

