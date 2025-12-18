// Package config provides configuration management for CDD CLI.
package config

import (
	"fmt"

	"github.com/charmbracelet/catwalk/pkg/catwalk"

	"github.com/guilhermegouw/cdd/internal/oauth"
)

const appName = "cdd"

// SelectedModelType represents a model capability tier.
type SelectedModelType string

const (
	// SelectedModelTypeLarge is for complex tasks requiring full reasoning.
	SelectedModelTypeLarge SelectedModelType = "large"
	// SelectedModelTypeSmall is for simpler, faster tasks.
	SelectedModelTypeSmall SelectedModelType = "small"
)

// SelectedModel defines which model to use for a tier.
//
//nolint:govet // Field order optimized for JSON readability over memory.
type SelectedModel struct {
	// ProviderOptions holds additional provider-specific options.
	ProviderOptions map[string]any `json:"provider_options,omitempty"`
	// Model is the model ID as used by the provider API.
	Model string `json:"model"`
	// Provider is the provider ID that matches a key in providers config.
	Provider string `json:"provider"`
	// ReasoningEffort is used by OpenAI models that support reasoning.
	ReasoningEffort string `json:"reasoning_effort,omitempty"`
	// Temperature controls sampling randomness (0-1).
	Temperature *float64 `json:"temperature,omitempty"`
	// TopP is the nucleus sampling parameter.
	TopP *float64 `json:"top_p,omitempty"`
	// FrequencyPenalty reduces repetition.
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
	// PresencePenalty increases topic diversity.
	PresencePenalty *float64 `json:"presence_penalty,omitempty"`
	// TopK is the top-k sampling parameter.
	TopK *int64 `json:"top_k,omitempty"`
	// MaxTokens overrides the default max tokens for responses.
	MaxTokens int64 `json:"max_tokens,omitempty"`
	// Think enables thinking mode for Anthropic models that support reasoning.
	Think bool `json:"think,omitempty"`
}

// ProviderConfig holds provider authentication and settings.
//
//nolint:govet // Field order optimized for JSON readability over memory.
type ProviderConfig struct {
	// ExtraHeaders are additional HTTP headers for requests.
	ExtraHeaders map[string]string `json:"extra_headers,omitempty"`
	// ProviderOptions holds additional provider-specific options.
	ProviderOptions map[string]any `json:"provider_options,omitempty"`
	// Models holds the available models from this provider.
	Models []catwalk.Model `json:"models,omitempty"`
	// OAuthToken for providers that use OAuth2 authentication.
	OAuthToken *oauth.Token `json:"oauth,omitempty"`
	// ID is the unique identifier for the provider.
	ID string `json:"id,omitempty"`
	// Name is the human-readable name for display.
	Name string `json:"name,omitempty"`
	// Type is the provider type (openai, anthropic, etc).
	Type catwalk.Type `json:"type,omitempty"`
	// BaseURL is the API endpoint URL.
	BaseURL string `json:"base_url,omitempty"`
	// APIKey is the authentication key.
	APIKey string `json:"api_key,omitempty"`
	// SystemPromptPrefix is prepended to system prompts for this provider.
	SystemPromptPrefix string `json:"-"`
	// Disable marks the provider as disabled.
	Disable bool `json:"disable,omitempty"`
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
	// OAuth requires specific beta headers for Claude Code subscription
	pc.ExtraHeaders["anthropic-beta"] = "oauth-2025-04-20,claude-code-20250219,interleaved-thinking-2025-05-14"
	pc.ExtraHeaders["User-Agent"] = "ai-sdk/anthropic"
}

// Config is the top-level configuration structure.
type Config struct {
	// Models maps tier types to selected models.
	Models map[SelectedModelType]SelectedModel `json:"models"`
	// Providers maps provider IDs to their configurations.
	Providers map[string]*ProviderConfig `json:"providers"`
	// Options holds application settings.
	Options *Options `json:"options,omitempty"`

	// knownProviders holds the catwalk provider metadata.
	knownProviders []catwalk.Provider
}

// Options holds application settings.
//
//nolint:govet // Field order optimized for JSON readability over memory.
type Options struct {
	// ContextPaths are files to load as context.
	ContextPaths []string `json:"context_paths,omitempty"`
	// DataDir is the directory for application data.
	DataDir string `json:"data_directory,omitempty"`
	// Debug enables debug mode.
	Debug bool `json:"debug,omitempty"`
}

// NewConfig creates a Config with initialized maps.
func NewConfig() *Config {
	return &Config{
		Models:    make(map[SelectedModelType]SelectedModel),
		Providers: make(map[string]*ProviderConfig),
		Options:   &Options{},
	}
}

// GetModel finds a model by ID within a provider's model list.
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

// KnownProviders returns the catwalk provider metadata.
func (c *Config) KnownProviders() []catwalk.Provider {
	return c.knownProviders
}

// SetKnownProviders sets the catwalk provider metadata.
func (c *Config) SetKnownProviders(providers []catwalk.Provider) {
	c.knownProviders = providers
}
