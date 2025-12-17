//nolint:goconst // Test file uses repeated string literals for clarity.
package config

import (
	"testing"
)

// Note: IsFirstRun() and NeedsSetup() use xdg.ConfigHome which is cached at init time.
// We test the helper function hasConfiguredProviders directly since it contains
// the core logic.

func TestHasConfiguredProviders(t *testing.T) {
	//nolint:govet // Field order optimized for test readability.
	tests := []struct {
		name      string
		providers map[string]*ProviderConfig
		want      bool
	}{
		{
			name:      "nil providers",
			providers: nil,
			want:      false,
		},
		{
			name:      "empty providers",
			providers: map[string]*ProviderConfig{},
			want:      false,
		},
		{
			name: "provider without API key",
			providers: map[string]*ProviderConfig{
				"test": {ID: "test", APIKey: ""},
			},
			want: false,
		},
		{
			name: "provider with API key",
			providers: map[string]*ProviderConfig{
				"test": {ID: "test", APIKey: "key"},
			},
			want: true,
		},
		{
			name: "disabled provider with API key",
			providers: map[string]*ProviderConfig{
				"test": {ID: "test", APIKey: "key", Disable: true},
			},
			want: false,
		},
		{
			name: "mixed providers - one enabled with key",
			providers: map[string]*ProviderConfig{
				"disabled": {ID: "disabled", APIKey: "key", Disable: true},
				"enabled":  {ID: "enabled", APIKey: "key", Disable: false},
			},
			want: true,
		},
		{
			name: "multiple enabled providers",
			providers: map[string]*ProviderConfig{
				"first":  {ID: "first", APIKey: "key1", Disable: false},
				"second": {ID: "second", APIKey: "key2", Disable: false},
			},
			want: true,
		},
		{
			name: "provider with only whitespace API key",
			providers: map[string]*ProviderConfig{
				"test": {ID: "test", APIKey: "   "},
			},
			want: true, // Whitespace is considered a value.
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig()
			cfg.Providers = tt.providers

			got := hasConfiguredProviders(cfg)
			if got != tt.want {
				t.Errorf("hasConfiguredProviders() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHasConfiguredProviders_WithOAuthToken(t *testing.T) {
	// Test that OAuth tokens count as configured (since APIKey is set from AccessToken).
	cfg := NewConfig()
	cfg.Providers = map[string]*ProviderConfig{
		"anthropic": {
			ID:     "anthropic",
			APIKey: "oauth-access-token", // Set from OAuth flow.
		},
	}

	if !hasConfiguredProviders(cfg) {
		t.Error("hasConfiguredProviders() = false, want true when OAuth token provides API key")
	}
}
