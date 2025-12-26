//nolint:dupl,errorlint // Test files use direct error comparison.
package events

import (
	"errors"
	"testing"
	"time"
)

func TestAuthEventTypes(t *testing.T) {
	// Verify all event types are distinct
	types := []AuthEventType{
		AuthEventTokenRefreshed,
		AuthEventTokenExpiring,
		AuthEventTokenExpired,
		AuthEventRefreshFailed,
	}

	seen := make(map[AuthEventType]bool)
	for _, typ := range types {
		if seen[typ] {
			t.Errorf("duplicate event type: %s", typ)
		}
		seen[typ] = true

		// Verify non-empty string value
		if string(typ) == "" {
			t.Error("event type should have non-empty string value")
		}
	}
}

func TestNewTokenRefreshedEvent(t *testing.T) {
	t.Run("creates token refreshed event with correct fields", func(t *testing.T) {
		expiresAt := time.Now().Add(1 * time.Hour)

		before := time.Now()
		event := NewTokenRefreshedEvent("anthropic", expiresAt)
		after := time.Now()

		if event.ProviderID != "anthropic" {
			t.Errorf("expected ProviderID 'anthropic', got %q", event.ProviderID)
		}
		if event.Type != AuthEventTokenRefreshed {
			t.Errorf("expected Type AuthEventTokenRefreshed, got %q", event.Type)
		}
		if event.ExpiresAt != expiresAt {
			t.Errorf("expected ExpiresAt %v, got %v", expiresAt, event.ExpiresAt)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}

		// Error should be nil
		if event.Error != nil {
			t.Error("Error should be nil")
		}
	})

	t.Run("handles zero expiration time", func(t *testing.T) {
		event := NewTokenRefreshedEvent("provider", time.Time{})
		if !event.ExpiresAt.IsZero() {
			t.Error("expected zero ExpiresAt")
		}
	})
}

func TestNewTokenExpiringEvent(t *testing.T) {
	t.Run("creates token expiring event with correct fields", func(t *testing.T) {
		expiresAt := time.Now().Add(5 * time.Minute)

		before := time.Now()
		event := NewTokenExpiringEvent("openai", expiresAt)
		after := time.Now()

		if event.ProviderID != "openai" {
			t.Errorf("expected ProviderID 'openai', got %q", event.ProviderID)
		}
		if event.Type != AuthEventTokenExpiring {
			t.Errorf("expected Type AuthEventTokenExpiring, got %q", event.Type)
		}
		if event.ExpiresAt != expiresAt {
			t.Errorf("expected ExpiresAt %v, got %v", expiresAt, event.ExpiresAt)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}

		// Error should be nil
		if event.Error != nil {
			t.Error("Error should be nil")
		}
	})
}

func TestNewTokenExpiredEvent(t *testing.T) {
	t.Run("creates token expired event with correct fields", func(t *testing.T) {
		before := time.Now()
		event := NewTokenExpiredEvent("google")
		after := time.Now()

		if event.ProviderID != "google" {
			t.Errorf("expected ProviderID 'google', got %q", event.ProviderID)
		}
		if event.Type != AuthEventTokenExpired {
			t.Errorf("expected Type AuthEventTokenExpired, got %q", event.Type)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}

		// ExpiresAt and Error should be zero
		if !event.ExpiresAt.IsZero() {
			t.Error("ExpiresAt should be zero")
		}
		if event.Error != nil {
			t.Error("Error should be nil")
		}
	})
}

func TestNewRefreshFailedEvent(t *testing.T) {
	t.Run("creates refresh failed event with correct fields", func(t *testing.T) {
		testErr := errors.New("token refresh failed: invalid_grant")

		before := time.Now()
		event := NewRefreshFailedEvent("anthropic", testErr)
		after := time.Now()

		if event.ProviderID != "anthropic" {
			t.Errorf("expected ProviderID 'anthropic', got %q", event.ProviderID)
		}
		if event.Type != AuthEventRefreshFailed {
			t.Errorf("expected Type AuthEventRefreshFailed, got %q", event.Type)
		}
		if event.Error != testErr {
			t.Errorf("expected Error to be testErr, got %v", event.Error)
		}
		if event.Timestamp.Before(before) || event.Timestamp.After(after) {
			t.Error("timestamp should be within test bounds")
		}

		// ExpiresAt should be zero
		if !event.ExpiresAt.IsZero() {
			t.Error("ExpiresAt should be zero")
		}
	})

	t.Run("handles nil error", func(t *testing.T) {
		event := NewRefreshFailedEvent("provider", nil)

		if event.Error != nil {
			t.Error("Error should be nil")
		}
		if event.Type != AuthEventRefreshFailed {
			t.Error("Type should still be AuthEventRefreshFailed")
		}
	})
}

func TestAuthEventStruct(t *testing.T) {
	t.Run("all fields accessible", func(t *testing.T) {
		now := time.Now()
		expiresAt := now.Add(1 * time.Hour)
		testErr := errors.New("test error")

		event := AuthEvent{
			ProviderID: "provider-1",
			Type:       AuthEventTokenRefreshed,
			Timestamp:  now,
			ExpiresAt:  expiresAt,
			Error:      testErr,
		}

		if event.ProviderID != "provider-1" {
			t.Error("ProviderID mismatch")
		}
		if event.Type != AuthEventTokenRefreshed {
			t.Error("Type mismatch")
		}
		if event.Timestamp != now {
			t.Error("Timestamp mismatch")
		}
		if event.ExpiresAt != expiresAt {
			t.Error("ExpiresAt mismatch")
		}
		if event.Error != testErr {
			t.Error("Error mismatch")
		}
	})
}

func TestAuthEventProviderIDs(t *testing.T) {
	t.Run("handles various provider ID formats", func(t *testing.T) {
		providerIDs := []string{
			"anthropic",
			"openai",
			"claude-oauth",
			"provider_123",
			"Provider.Name",
			"",
		}

		for _, id := range providerIDs {
			event := NewTokenRefreshedEvent(id, time.Now())
			if event.ProviderID != id {
				t.Errorf("expected ProviderID %q, got %q", id, event.ProviderID)
			}
		}
	})
}
