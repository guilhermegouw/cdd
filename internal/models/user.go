// Package models provides domain models for CDD.
//
// This package contains the core domain types used throughout the CDD application.
// Each model includes validation methods to ensure data integrity.
//
// # User Model
//
// The User model represents an application user with email and name fields.
// Users are created with auto-generated UUIDs and timestamps.
//
// Example usage:
//
//	user := models.NewUser("john@example.com", "John Doe")
//	if err := user.Validate(); err != nil {
//	    log.Printf("Invalid user: %v", err)
//	}
package models

import (
	"errors"
	"net/mail"
	"strings"
	"time"
	"unicode"

	"github.com/google/uuid"
)

// User represents a user of the CDD application.
//
// A User has a unique ID (UUID), email address, display name, and timestamps
// for creation and last update. All fields are validated before persistence.
//
// The Email field must be a valid email address format (RFC 5322).
// The Name field must be 2-100 characters and contain only letters,
// spaces, hyphens, and apostrophes.
type User struct {
	// ID is the unique identifier for the user (UUID format).
	ID string `json:"id"`

	// Email is the user's email address (must be valid RFC 5322 format).
	Email string `json:"email"`

	// Name is the user's display name (2-100 characters, letters/spaces/hyphens/apostrophes only).
	Name string `json:"name"`

	// CreatedAt is when the user was created.
	CreatedAt time.Time `json:"created_at"`

	// UpdatedAt is when the user was last modified.
	UpdatedAt time.Time `json:"updated_at"`
}

// Validation errors for User.
//
// These errors are returned by the validation methods when user data
// does not meet the required constraints.
var (
	// ErrEmptyEmail is returned when the email field is empty or whitespace-only.
	ErrEmptyEmail = errors.New("email cannot be empty")

	// ErrInvalidEmail is returned when the email does not match RFC 5322 format.
	ErrInvalidEmail = errors.New("email format is invalid")

	// ErrEmptyName is returned when the name field is empty or whitespace-only.
	ErrEmptyName = errors.New("name cannot be empty")

	// ErrNameTooShort is returned when the name is less than MinNameLength characters.
	ErrNameTooShort = errors.New("name must be at least 2 characters")

	// ErrNameTooLong is returned when the name exceeds MaxNameLength characters.
	ErrNameTooLong = errors.New("name cannot exceed 100 characters")

	// ErrInvalidName is returned when the name contains invalid characters.
	ErrInvalidName = errors.New("name contains invalid characters")
)

// User validation constants define the constraints for user fields.
const (
	// MinNameLength is the minimum allowed length for a user's name.
	MinNameLength = 2

	// MaxNameLength is the maximum allowed length for a user's name.
	MaxNameLength = 100
)

// NewUser creates a new User with the given email and name.
// It generates a UUID for the ID and sets timestamps.
func NewUser(email, name string) *User {
	now := time.Now()
	return &User{
		ID:        uuid.New().String(),
		Email:     strings.TrimSpace(email),
		Name:      strings.TrimSpace(name),
		CreatedAt: now,
		UpdatedAt: now,
	}
}

// Validate checks if the User has valid data.
// Returns nil if valid, or the first validation error encountered.
func (u *User) Validate() error {
	if err := u.ValidateEmail(); err != nil {
		return err
	}
	return u.ValidateName()
}

// ValidateEmail validates the user's email address.
func (u *User) ValidateEmail() error {
	email := strings.TrimSpace(u.Email)
	if email == "" {
		return ErrEmptyEmail
	}

	_, err := mail.ParseAddress(email)
	if err != nil {
		return ErrInvalidEmail
	}

	return nil
}

// ValidateName validates the user's name.
func (u *User) ValidateName() error {
	name := strings.TrimSpace(u.Name)
	if name == "" {
		return ErrEmptyName
	}

	if len(name) < MinNameLength {
		return ErrNameTooShort
	}

	if len(name) > MaxNameLength {
		return ErrNameTooLong
	}

	// Check for valid characters (letters, spaces, hyphens, apostrophes)
	for _, r := range name {
		if !unicode.IsLetter(r) && !unicode.IsSpace(r) && r != '-' && r != '\'' {
			return ErrInvalidName
		}
	}

	return nil
}

// SetEmail updates the user's email and the UpdatedAt timestamp.
func (u *User) SetEmail(email string) {
	u.Email = strings.TrimSpace(email)
	u.UpdatedAt = time.Now()
}

// SetName updates the user's name and the UpdatedAt timestamp.
func (u *User) SetName(name string) {
	u.Name = strings.TrimSpace(name)
	u.UpdatedAt = time.Now()
}

// IsValid returns true if the User passes all validations.
func (u *User) IsValid() bool {
	return u.Validate() == nil
}
