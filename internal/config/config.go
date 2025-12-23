// Package config provides configuration management for CDD CLI.
package config

import (
	"context"
	"fmt"
	"os"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
	"github.com/tidwall/sjson"

	"github.com/guilhermegouw/cdd/internal/oauth"
	"github.com/guilhermegouw/cdd/internal/oauth/claude"
)

const appName = "cdd"

// SelectedModelType represents the tier of model (large or small).
type SelectedModelType string

// Model type constants.
const (
	SelectedModelTypeLarge SelectedModelType = "large"
	SelectedModelTypeSmall SelectedModelType = "small"
)

// SelectedModel represents a selected model configuration for a tier.
type SelectedModel struct {
	ProviderOptions  map[string]any `json:"provider_options,omitempty"`
	Temperature      *float64       `json:"temperature,omitempty"`
	TopP             *float64       `json:"top_p,omitempty"`
	FrequencyPenalty *float64       `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64       `json:"presence_penalty,omitempty"`
	TopK             *int64         `json:"top_k,omitempty"`
	Model            string         `json:"model"`
	Provider         string         `json:"provider"`
	ReasoningEffort  string         `json:"reasoning_effort,omitempty"`
	MaxTokens        int64          `json:"max_tokens,omitempty"`
	Think            bool           `json:"think,omitempty"`
}

// ProviderConfig holds provider authentication and settings.
//
//nolint:govet // Field order is intentional for JSON readability.
type ProviderConfig struct {
	ExtraHeaders       map[string]string `json:"extra_headers,omitempty"`
	ProviderOptions    map[string]any    `json:"provider_options,omitempty"`
	Models             []catwalk.Model   `json:"models,omitempty"`
	OAuthToken         *oauth.Token      `json:"oauth,omitempty"`
	ID                 string            `json:"id,omitempty"`
	Name               string            `json:"name,omitempty"`
	Type               catwalk.Type      `json:"type,omitempty"`
	BaseURL            string            `json:"base_url,omitempty"`
	APIKey             string            `json:"api_key,omitempty"`
	SystemPromptPrefix string            `json:"-"`
	Disable            bool              `json:"disable,omitempty"`
}

// SetupClaudeCode configures the provider for Claude Code OAuth authentication.
func (pc *ProviderConfig) SetupClaudeCode() {
	if pc.OAuthToken == nil {
		return
	}
	pc.APIKey = fmt.Sprintf("Bearer %s", pc.OAuthToken.AccessToken)
	pc.SystemPromptPrefix = "You are Claude Code, Anthropic's official CLI for Claude."
	if pc.ExtraHeaders == nil {
		pc.ExtraHeaders = make(map[string]string)
	}
	pc.ExtraHeaders["anthropic-version"] = "2023-06-01"
	pc.ExtraHeaders["anthropic-beta"] = "oauth-2025-04-20,claude-code-20250219,interleaved-thinking-2025-05-14"
	pc.ExtraHeaders["User-Agent"] = "ai-sdk/anthropic"
}

// Config is the top-level configuration structure.
type Config struct {
	Models         map[SelectedModelType]SelectedModel `json:"models"`
	Providers      map[string]*ProviderConfig          `json:"providers"`
	Options        *Options                            `json:"options,omitempty"`
	knownProviders []catwalk.Provider
}

// Options holds optional configuration settings.
//
//nolint:govet // Field order is intentional for JSON readability.
type Options struct {
	ContextPaths []string `json:"context_paths,omitempty"`
	DataDir      string   `json:"data_directory,omitempty"`
	Debug        bool     `json:"debug,omitempty"`
}

// NewConfig creates a new Config with initialized maps.
func NewConfig() *Config {
	return &Config{
		Models:    make(map[SelectedModelType]SelectedModel),
		Providers: make(map[string]*ProviderConfig),
		Options:   &Options{},
	}
}

// GetModel returns the model configuration for the given provider and model IDs.
func (c *Config) GetModel(providerID, modelID string) *catwalk.Model {
	provider, ok := c.Providers[providerID]
	if !ok {
		return nil
	}
	for i := range provider.Models {
		if provider.Models[i].ID == modelID {
			return &provider.Models[i]
		}
	}
	return nil
}

// KnownProviders returns the list of known providers from catwalk.
func (c *Config) KnownProviders() []catwalk.Provider {
	return c.knownProviders
}

// SetKnownProviders sets the list of known providers.
func (c *Config) SetKnownProviders(providers []catwalk.Provider) {
	c.knownProviders = providers
}

// RefreshOAuthToken refreshes the OAuth token for the given provider.
// It updates the provider config with the new token, persists to disk, and calls SetupClaudeCode.
func (c *Config) RefreshOAuthToken(ctx context.Context, providerID string) error {
	provider, ok := c.Providers[providerID]
	if !ok {
		return fmt.Errorf("provider %q not found", providerID)
	}
	if provider.OAuthToken == nil {
		return fmt.Errorf("provider %q has no OAuth token", providerID)
	}

	newToken, err := claude.RefreshToken(ctx, provider.OAuthToken.RefreshToken)
	if err != nil {
		return fmt.Errorf("refreshing token for %q: %w", providerID, err)
	}

	// Persist tokens to disk IMMEDIATELY before updating in-memory state.
	// This is critical because Anthropic uses token rotation - the old refresh token
	// is invalidated as soon as we receive the new one.
	if err := c.SetConfigField(fmt.Sprintf("providers.%s.oauth", providerID), newToken); err != nil {
		return fmt.Errorf("persisting refreshed oauth token: %w", err)
	}
	if err := c.SetConfigField(fmt.Sprintf("providers.%s.api_key", providerID), newToken.AccessToken); err != nil {
		return fmt.Errorf("persisting refreshed api_key: %w", err)
	}

	// Now safe to update in-memory state.
	provider.OAuthToken = newToken
	provider.SetupClaudeCode()
	return nil
}

// SetConfigField updates a single field in the config file using JSON path notation.
// This uses sjson for surgical updates - only the specified field is modified.
func (c *Config) SetConfigField(key string, value any) error {
	configPath := GlobalConfigPath()

	//nolint:gosec // G304: configPath is from trusted GlobalConfigPath(), not user input.
	data, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			data = []byte("{}")
		} else {
			return fmt.Errorf("reading config file: %w", err)
		}
	}

	newData, err := sjson.Set(string(data), key, value)
	if err != nil {
		return fmt.Errorf("setting config field %q: %w", key, err)
	}

	//nolint:gosec // 0o600 is intentionally restrictive for security.
	if err := os.WriteFile(configPath, []byte(newData), 0o600); err != nil {
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}
