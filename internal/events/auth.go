package events

import "time"

// AuthEventType represents auth-specific event types.
type AuthEventType string

// Auth event type constants.
//
//nolint:gosec // G101 false positive - these are event type names, not credentials
const (
	AuthEventTokenRefreshed AuthEventType = "token_refreshed"
	AuthEventTokenExpiring  AuthEventType = "token_expiring"
	AuthEventTokenExpired   AuthEventType = "token_expired"
	AuthEventRefreshFailed  AuthEventType = "refresh_failed"
)

// AuthEvent represents an authentication event.
type AuthEvent struct { //nolint:govet // fieldalignment: preserving logical field order
	ProviderID string
	Type       AuthEventType
	Timestamp  time.Time

	// Optional fields
	ExpiresAt time.Time // For TokenRefreshed, TokenExpiring
	Error     error     // For RefreshFailed
}

// NewTokenRefreshedEvent creates a token refreshed event.
func NewTokenRefreshedEvent(providerID string, expiresAt time.Time) AuthEvent {
	return AuthEvent{
		ProviderID: providerID,
		Type:       AuthEventTokenRefreshed,
		ExpiresAt:  expiresAt,
		Timestamp:  time.Now(),
	}
}

// NewTokenExpiringEvent creates a token expiring warning event.
func NewTokenExpiringEvent(providerID string, expiresAt time.Time) AuthEvent {
	return AuthEvent{
		ProviderID: providerID,
		Type:       AuthEventTokenExpiring,
		ExpiresAt:  expiresAt,
		Timestamp:  time.Now(),
	}
}

// NewTokenExpiredEvent creates a token expired event.
func NewTokenExpiredEvent(providerID string) AuthEvent {
	return AuthEvent{
		ProviderID: providerID,
		Type:       AuthEventTokenExpired,
		Timestamp:  time.Now(),
	}
}

// NewRefreshFailedEvent creates a refresh failed event.
func NewRefreshFailedEvent(providerID string, err error) AuthEvent {
	return AuthEvent{
		ProviderID: providerID,
		Type:       AuthEventRefreshFailed,
		Error:      err,
		Timestamp:  time.Now(),
	}
}
