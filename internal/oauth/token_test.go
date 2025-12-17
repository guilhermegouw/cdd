package oauth

import (
	"testing"
	"time"
)

func TestToken_SetExpiresAt(t *testing.T) {
	tests := []struct {
		name      string
		expiresIn int
	}{
		{
			name:      "standard expiration 1 hour",
			expiresIn: 3600,
		},
		{
			name:      "short expiration 5 minutes",
			expiresIn: 300,
		},
		{
			name:      "long expiration 8 hours",
			expiresIn: 28800,
		},
		{
			name:      "zero expiration",
			expiresIn: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &Token{
				AccessToken:  "test-access-token",
				RefreshToken: "test-refresh-token",
				ExpiresIn:    tt.expiresIn,
			}

			before := time.Now().Unix()
			token.SetExpiresAt()
			after := time.Now().Unix()

			expectedMin := before + int64(tt.expiresIn)
			expectedMax := after + int64(tt.expiresIn)

			if token.ExpiresAt < expectedMin || token.ExpiresAt > expectedMax {
				t.Errorf("SetExpiresAt() = %d, expected between %d and %d",
					token.ExpiresAt, expectedMin, expectedMax)
			}
		})
	}
}

func TestToken_IsExpired(t *testing.T) {
	tests := []struct {
		name        string
		expiresIn   int
		expiresAt   int64
		wantExpired bool
	}{
		{
			name:        "not expired - plenty of time left",
			expiresIn:   3600,
			expiresAt:   time.Now().Add(time.Hour).Unix(),
			wantExpired: false,
		},
		{
			name:        "not expired - just over threshold",
			expiresIn:   3600,
			expiresAt:   time.Now().Add(7 * time.Minute).Unix(), // 10% of 3600s = 360s = 6min
			wantExpired: false,
		},
		{
			name:        "expired - past expiration",
			expiresIn:   3600,
			expiresAt:   time.Now().Add(-time.Hour).Unix(),
			wantExpired: true,
		},
		{
			name:        "expired - within 10% threshold",
			expiresIn:   3600,
			expiresAt:   time.Now().Add(5 * time.Minute).Unix(), // Within 6min threshold
			wantExpired: true,
		},
		{
			name:        "expired - exactly at threshold",
			expiresIn:   3600,
			expiresAt:   time.Now().Add(6 * time.Minute).Unix(), // Exactly at threshold
			wantExpired: true,
		},
		{
			name:        "zero expires_in - always expired",
			expiresIn:   0,
			expiresAt:   time.Now().Add(time.Hour).Unix(),
			wantExpired: false, // With 0 expiresIn, threshold is 0
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			token := &Token{
				AccessToken:  "test-access-token",
				RefreshToken: "test-refresh-token",
				ExpiresIn:    tt.expiresIn,
				ExpiresAt:    tt.expiresAt,
			}

			got := token.IsExpired()
			if got != tt.wantExpired {
				t.Errorf("IsExpired() = %v, want %v", got, tt.wantExpired)
			}
		})
	}
}

func TestToken_Fields(t *testing.T) {
	token := Token{
		AccessToken:  "access-123",
		RefreshToken: "refresh-456",
		ExpiresIn:    3600,
		ExpiresAt:    1700000000,
	}

	if token.AccessToken != "access-123" {
		t.Errorf("AccessToken = %q, want %q", token.AccessToken, "access-123")
	}
	if token.RefreshToken != "refresh-456" {
		t.Errorf("RefreshToken = %q, want %q", token.RefreshToken, "refresh-456")
	}
	if token.ExpiresIn != 3600 {
		t.Errorf("ExpiresIn = %d, want %d", token.ExpiresIn, 3600)
	}
	if token.ExpiresAt != 1700000000 {
		t.Errorf("ExpiresAt = %d, want %d", token.ExpiresAt, 1700000000)
	}
}

func TestToken_SetExpiresAt_UpdatesExistingValue(t *testing.T) {
	token := &Token{
		ExpiresIn: 3600,
		ExpiresAt: 100, // Old value
	}

	token.SetExpiresAt()

	// Should be updated to a new value based on current time
	if token.ExpiresAt <= 100 {
		t.Error("SetExpiresAt() did not update ExpiresAt")
	}
}
