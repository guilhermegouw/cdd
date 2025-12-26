//nolint:dupl,errorlint // Test files use direct error comparison.
package models

import (
	"strings"
	"testing"
)

func TestNewUser(t *testing.T) {
	t.Run("creates user with valid data", func(t *testing.T) {
		user := NewUser("test@example.com", "John Doe")

		if user.ID == "" {
			t.Error("Expected user ID to be set")
		}
		if user.Email != "test@example.com" {
			t.Errorf("Expected email 'test@example.com', got %q", user.Email)
		}
		if user.Name != "John Doe" {
			t.Errorf("Expected name 'John Doe', got %q", user.Name)
		}
		if user.CreatedAt.IsZero() {
			t.Error("Expected CreatedAt to be set")
		}
		if user.UpdatedAt.IsZero() {
			t.Error("Expected UpdatedAt to be set")
		}
	})

	t.Run("trims whitespace from email and name", func(t *testing.T) {
		user := NewUser("  test@example.com  ", "  John Doe  ")

		if user.Email != "test@example.com" {
			t.Errorf("Expected trimmed email, got %q", user.Email)
		}
		if user.Name != "John Doe" {
			t.Errorf("Expected trimmed name, got %q", user.Name)
		}
	})

	t.Run("generates unique IDs", func(t *testing.T) {
		user1 := NewUser("test1@example.com", "User One")
		user2 := NewUser("test2@example.com", "User Two")

		if user1.ID == user2.ID {
			t.Error("Expected unique IDs for different users")
		}
	})
}

func TestUserValidate(t *testing.T) {
	t.Run("valid user passes validation", func(t *testing.T) {
		user := NewUser("test@example.com", "John Doe")

		if err := user.Validate(); err != nil {
			t.Errorf("Expected valid user, got error: %v", err)
		}
	})

	t.Run("returns email error first", func(t *testing.T) {
		user := NewUser("", "")

		err := user.Validate()
		if err != ErrEmptyEmail {
			t.Errorf("Expected ErrEmptyEmail, got %v", err)
		}
	})

	t.Run("returns name error if email is valid", func(t *testing.T) {
		user := NewUser("test@example.com", "")

		err := user.Validate()
		if err != ErrEmptyName {
			t.Errorf("Expected ErrEmptyName, got %v", err)
		}
	})
}

func TestUserValidateEmail(t *testing.T) {
	tests := []struct {
		name    string
		email   string
		wantErr error
	}{
		{
			name:    "valid email",
			email:   "test@example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with subdomain",
			email:   "test@mail.example.com",
			wantErr: nil,
		},
		{
			name:    "valid email with plus",
			email:   "test+tag@example.com",
			wantErr: nil,
		},
		{
			name:    "empty email",
			email:   "",
			wantErr: ErrEmptyEmail,
		},
		{
			name:    "whitespace only email",
			email:   "   ",
			wantErr: ErrEmptyEmail,
		},
		{
			name:    "invalid email no at",
			email:   "testexample.com",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "invalid email no domain",
			email:   "test@",
			wantErr: ErrInvalidEmail,
		},
		{
			name:    "invalid email no local part",
			email:   "@example.com",
			wantErr: ErrInvalidEmail,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			user := &User{Email: tc.email, Name: "Valid Name"}
			err := user.ValidateEmail()

			if err != tc.wantErr {
				t.Errorf("ValidateEmail() = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestUserValidateName(t *testing.T) {
	tests := []struct {
		name    string
		input   string
		wantErr error
	}{
		{
			name:    "valid name",
			input:   "John Doe",
			wantErr: nil,
		},
		{
			name:    "valid name with hyphen",
			input:   "Mary-Jane Watson",
			wantErr: nil,
		},
		{
			name:    "valid name with apostrophe",
			input:   "O'Connor",
			wantErr: nil,
		},
		{
			name:    "valid minimum length",
			input:   "Jo",
			wantErr: nil,
		},
		{
			name:    "empty name",
			input:   "",
			wantErr: ErrEmptyName,
		},
		{
			name:    "whitespace only name",
			input:   "   ",
			wantErr: ErrEmptyName,
		},
		{
			name:    "name too short",
			input:   "J",
			wantErr: ErrNameTooShort,
		},
		{
			name:    "name too long",
			input:   strings.Repeat("a", MaxNameLength+1),
			wantErr: ErrNameTooLong,
		},
		{
			name:    "name with numbers",
			input:   "John123",
			wantErr: ErrInvalidName,
		},
		{
			name:    "name with special characters",
			input:   "John@Doe",
			wantErr: ErrInvalidName,
		},
		{
			name:    "name with underscores",
			input:   "John_Doe",
			wantErr: ErrInvalidName,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			user := &User{Email: "test@example.com", Name: tc.input}
			err := user.ValidateName()

			if err != tc.wantErr {
				t.Errorf("ValidateName() = %v, want %v", err, tc.wantErr)
			}
		})
	}
}

func TestUserSetEmail(t *testing.T) {
	t.Run("updates email and timestamp", func(t *testing.T) {
		user := NewUser("old@example.com", "Test User")
		originalUpdatedAt := user.UpdatedAt

		user.SetEmail("new@example.com")

		if user.Email != "new@example.com" {
			t.Errorf("Expected email 'new@example.com', got %q", user.Email)
		}
		if !user.UpdatedAt.After(originalUpdatedAt) {
			t.Error("Expected UpdatedAt to be updated")
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		user := NewUser("old@example.com", "Test User")

		user.SetEmail("  new@example.com  ")

		if user.Email != "new@example.com" {
			t.Errorf("Expected trimmed email, got %q", user.Email)
		}
	})
}

func TestUserSetName(t *testing.T) {
	t.Run("updates name and timestamp", func(t *testing.T) {
		user := NewUser("test@example.com", "Old Name")
		originalUpdatedAt := user.UpdatedAt

		user.SetName("New Name")

		if user.Name != "New Name" {
			t.Errorf("Expected name 'New Name', got %q", user.Name)
		}
		if !user.UpdatedAt.After(originalUpdatedAt) {
			t.Error("Expected UpdatedAt to be updated")
		}
	})

	t.Run("trims whitespace", func(t *testing.T) {
		user := NewUser("test@example.com", "Old Name")

		user.SetName("  New Name  ")

		if user.Name != "New Name" {
			t.Errorf("Expected trimmed name, got %q", user.Name)
		}
	})
}

func TestUserIsValid(t *testing.T) {
	t.Run("returns true for valid user", func(t *testing.T) {
		user := NewUser("test@example.com", "John Doe")

		if !user.IsValid() {
			t.Error("Expected IsValid() to return true")
		}
	})

	t.Run("returns false for invalid email", func(t *testing.T) {
		user := NewUser("invalid-email", "John Doe")

		if user.IsValid() {
			t.Error("Expected IsValid() to return false for invalid email")
		}
	})

	t.Run("returns false for invalid name", func(t *testing.T) {
		user := NewUser("test@example.com", "J")

		if user.IsValid() {
			t.Error("Expected IsValid() to return false for invalid name")
		}
	})
}
