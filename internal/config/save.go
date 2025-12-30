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
	Models      map[SelectedModelType]SelectedModel `json:"models,omitempty"`
	Providers   map[string]*SaveProviderConfig      `json:"providers,omitempty"`
	Connections []Connection                        `json:"connections,omitempty"`
	Options     *Options                            `json:"options,omitempty"`
}

// SaveProviderConfig is a minimal provider config for saving.
// It stores the API key template (e.g., "$OPENAI_API_KEY") rather than resolved values.
// Custom providers are stored separately in custom-providers.json via CustomProviderManager.
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
		Models:      cfg.Models,
		Providers:   make(map[string]*SaveProviderConfig),
		Connections: cfg.Connections,
		Options:     cfg.Options,
	}

	// Save provider configs (only API key/OAuth for standard providers).
	// Custom providers are stored separately via CustomProviderManager.
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

	if err := os.WriteFile(path, data, 0o600); err != nil { //nolint:gosec // Restrictive permissions for security.
		return fmt.Errorf("writing config file: %w", err)
	}

	return nil
}

// SaveWizardResult saves the result of the setup wizard with API key authentication.
// It saves to the Connections system (not legacy Providers).
func SaveWizardResult(providerID, apiKey, largeModel, smallModel string) error {
	conn := Connection{
		Name:       providerID,
		ProviderID: providerID,
		APIKey:     apiKey,
	}
	return saveWizardConnection(conn, providerID, largeModel, smallModel)
}

// SaveWizardResultWithOAuth saves the result of the setup wizard with OAuth authentication.
// It saves to the Connections system (not legacy Providers).
func SaveWizardResultWithOAuth(providerID string, token *oauth.Token, largeModel, smallModel string) error {
	conn := Connection{
		Name:       providerID,
		ProviderID: providerID,
		OAuthToken: token,
	}
	return saveWizardConnection(conn, providerID, largeModel, smallModel)
}

// loadExistingConfig loads the existing config file without full provider configuration.
// This is used to preserve connections when adding new ones via wizard.
func loadExistingConfig() (*Config, error) {
	cfg := NewConfig()
	configPath := GlobalConfigPath()

	//nolint:gosec // G304: configPath is from trusted GlobalConfigPath(), not user input.
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	return cfg, nil
}

// saveWizardConnection is a helper that saves a connection and sets model selections.
// It loads existing config to preserve other connections, then adds the new connection.
func saveWizardConnection(conn Connection, providerID, largeModel, smallModel string) error {
	// Try to load existing config to preserve other connections.
	cfg, err := loadExistingConfig()
	if err != nil {
		// If loading fails, start fresh (first run scenario).
		cfg = NewConfig()
	}

	connManager := NewConnectionManager(cfg)

	// Check if a connection with the same name already exists.
	// If so, remove it first to allow the new one to be added.
	if existing := connManager.GetByName(conn.Name); existing != nil {
		_ = connManager.Delete(existing.ID) //nolint:errcheck // Best effort cleanup.
	}

	if err := connManager.Add(conn); err != nil {
		return fmt.Errorf("adding connection: %w", err)
	}

	// Get the connection ID (Add generates it).
	addedConn := connManager.GetByName(conn.Name)
	if addedConn == nil {
		return fmt.Errorf("failed to retrieve added connection")
	}

	// Set model selections with ConnectionID.
	cfg.Models[SelectedModelTypeLarge] = SelectedModel{
		Model:        largeModel,
		Provider:     providerID,
		ConnectionID: addedConn.ID,
	}
	cfg.Models[SelectedModelTypeSmall] = SelectedModel{
		Model:        smallModel,
		Provider:     providerID,
		ConnectionID: addedConn.ID,
	}

	return Save(cfg)
}
