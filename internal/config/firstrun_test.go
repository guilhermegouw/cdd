//nolint:goconst // Test file uses repeated string literals for clarity.
package config

import (
	"testing"

	"github.com/guilhermegouw/cdd/internal/oauth"
)

// Note: IsFirstRun() and NeedsSetup() use xdg.ConfigHome which is cached at init time.
// We test the helper function hasConfiguredConnections directly since it contains
// the core logic.

func TestHasConfiguredConnections(t *testing.T) {
	//nolint:govet // Field order optimized for test readability.
	tests := []struct {
		name        string
		connections []Connection
		want        bool
	}{
		{
			name:        "nil connections",
			connections: nil,
			want:        false,
		},
		{
			name:        "empty connections",
			connections: []Connection{},
			want:        false,
		},
		{
			name: "connection without authentication",
			connections: []Connection{
				{ID: "test", Name: "Test", ProviderID: "anthropic"},
			},
			want: false,
		},
		{
			name: "connection with API key",
			connections: []Connection{
				{ID: "test", Name: "Test", ProviderID: "anthropic", APIKey: "key"},
			},
			want: true,
		},
		{
			name: "connection with OAuth token",
			connections: []Connection{
				{ID: "test", Name: "Test", ProviderID: "anthropic", OAuthToken: &oauth.Token{AccessToken: "token"}},
			},
			want: true,
		},
		{
			name: "multiple connections - one configured",
			connections: []Connection{
				{ID: "empty", Name: "Empty", ProviderID: "openai"},
				{ID: "configured", Name: "Configured", ProviderID: "anthropic", APIKey: "key"},
			},
			want: true,
		},
		{
			name: "multiple configured connections",
			connections: []Connection{
				{ID: "first", Name: "First", ProviderID: "anthropic", APIKey: "key1"},
				{ID: "second", Name: "Second", ProviderID: "openai", APIKey: "key2"},
			},
			want: true,
		},
		{
			name: "connection with only whitespace API key",
			connections: []Connection{
				{ID: "test", Name: "Test", ProviderID: "anthropic", APIKey: "   "},
			},
			want: true, // Whitespace is considered a value (IsConfigured checks for non-empty string).
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig()
			cfg.Connections = tt.connections

			got := hasConfiguredConnections(cfg)
			if got != tt.want {
				t.Errorf("hasConfiguredConnections() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasConfiguredConnections_WithOAuthToken(t *testing.T) {
	// Test that OAuth tokens count as configured.
	cfg := NewConfig()
	cfg.Connections = []Connection{
		{
			ID:         "anthropic-oauth",
			Name:       "Anthropic (OAuth)",
			ProviderID: "anthropic",
			OAuthToken: &oauth.Token{
				AccessToken: "oauth-access-token",
			},
		},
	}

	if !hasConfiguredConnections(cfg) {
		t.Error("hasConfiguredConnections() = false, want true when OAuth token is configured")
	}
}
