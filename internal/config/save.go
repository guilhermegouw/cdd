package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/guilhermegouw/cdd/internal/oauth"
)

// SaveConfig contains only the fields we want to save to disk.
// This excludes runtime-only fields like knownProviders and resolved API keys.
type SaveConfig struct {
	Models    map[SelectedModelType]SelectedModel `json:"models,omitempty"`
	Providers map[string]*SaveProviderConfig      `json:"providers,omitempty"`
	Options   *Options                            `json:"options,omitempty"`
}

// SaveProviderConfig is a minimal provider config for saving.
// It stores the API key template (e.g., "$OPENAI_API_KEY") rather than resolved values.
type SaveProviderConfig struct {
	OAuthToken *oauth.Token `json:"oauth,omitempty"`
	APIKey     string       `json:"api_key,omitempty"`
}

// Save writes the configuration to the global config file.
func Save(cfg *Config) error {
	return SaveToFile(cfg, GlobalConfigPath())
}

// SaveToFile writes the configuration to a specific file path.
func SaveToFile(cfg *Config, path string) error {
	// Ensure the directory exists.
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating config directory: %w", err)
	}

	// Create a minimal save config.
	saveCfg := &SaveConfig{
		Models:    cfg.Models,
		Providers: make(map[string]*SaveProviderConfig),
		Options:   cfg.Options,
	}

	// Only save provider API key templates and OAuth tokens.
	for id, p := range cfg.Providers {
		if p.APIKey != "" || p.OAuthToken != nil {
			saveCfg.Providers[id] = &SaveProviderConfig{
				APIKey:     p.APIKey,
				OAuthToken: p.OAuthToken,
			}
		}
	}

	data, err := json.MarshalIndent(saveCfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling config: %w", err)
	}

	if err := os.WriteFile(path, data, 0o644); err != nil { //nolint:gosec // Config file permissions are intentional.
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// SaveWizardResult saves the result of the setup wizard with API key authentication.
func SaveWizardResult(providerID, apiKey, largeModel, smallModel string) error {
	cfg := NewConfig()

	// Set provider with API key (could be actual key or env var reference).
	cfg.Providers[providerID] = &ProviderConfig{
		ID:     providerID,
		APIKey: apiKey,
	}

	// Set model selections.
	cfg.Models[SelectedModelTypeLarge] = SelectedModel{
		Model:    largeModel,
		Provider: providerID,
	}
	cfg.Models[SelectedModelTypeSmall] = SelectedModel{
		Model:    smallModel,
		Provider: providerID,
	}

	return Save(cfg)
}

// SaveWizardResultWithOAuth saves the result of the setup wizard with OAuth authentication.
func SaveWizardResultWithOAuth(providerID string, token *oauth.Token, largeModel, smallModel string) error {
	cfg := NewConfig()

	// Set provider with OAuth token.
	cfg.Providers[providerID] = &ProviderConfig{
		ID:         providerID,
		OAuthToken: token,
		APIKey:     token.AccessToken, // Store access token as API key for immediate use.
	}

	// Set model selections.
	cfg.Models[SelectedModelTypeLarge] = SelectedModel{
		Model:    largeModel,
		Provider: providerID,
	}
	cfg.Models[SelectedModelTypeSmall] = SelectedModel{
		Model:    smallModel,
		Provider: providerID,
	}

	return Save(cfg)
}
