package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

const (
	configFileName = "cdd.json"

	// Default API endpoints for providers.
	defaultAnthropicEndpoint = "https://api.anthropic.com"
	defaultOpenAIEndpoint    = "https://api.openai.com/v1"
)

// Load finds and loads configuration from standard locations.
// It merges global config with project config (project takes precedence),
// then configures providers using catwalk metadata and custom providers.
func Load() (*Config, error) {
	cfg := NewConfig()
	resolver := NewResolver()
	globalPath := filepath.Join(xdg.ConfigHome, appName, configFileName)
	if err := loadFile(globalPath, cfg); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading global config: %w", err)
	}

	projectPath := findProjectConfig()
	if projectPath != "" {
		projectCfg := NewConfig()
		if err := loadFile(projectPath, projectCfg); err != nil {
			return nil, fmt.Errorf("loading project config: %w", err)
		}
		mergeConfig(cfg, projectCfg)
	}

	applyDefaults(cfg)

	// Migrate existing providers to connections (backward compatibility).
	if err := MigrateToConnections(cfg); err != nil {
		return nil, fmt.Errorf("migrating to connections: %w", err)
	}

	// Load providers using ProviderLoader (catwalk + custom).
	loader := NewProviderLoader(cfg.DataDir())
	providers, err := loader.LoadAllProviders(cfg)
	if err != nil {
		return nil, fmt.Errorf("loading providers: %w", err)
	}
	cfg.SetKnownProviders(providers)
	configureProviders(cfg, resolver)
	if err := configureDefaultModels(cfg); err != nil {
		return nil, fmt.Errorf("configuring models: %w", err)
	}

	return cfg, nil
}

// LoadFromFile loads configuration from a specific file path.
func LoadFromFile(path string) (*Config, error) {
	cfg := NewConfig()
	resolver := NewResolver()

	if err := loadFile(path, cfg); err != nil {
		return nil, err
	}

	applyDefaults(cfg)

	// Migrate existing providers to connections (backward compatibility).
	if err := MigrateToConnections(cfg); err != nil {
		return nil, fmt.Errorf("migrating to connections: %w", err)
	}

	// Load providers using ProviderLoader (catwalk + custom).
	loader := NewProviderLoader(cfg.DataDir())
	providers, err := loader.LoadAllProviders(cfg)
	if err != nil {
		return nil, fmt.Errorf("loading providers: %w", err)
	}
	cfg.SetKnownProviders(providers)

	configureProviders(cfg, resolver)

	if err := configureDefaultModels(cfg); err != nil {
		return nil, fmt.Errorf("configuring models: %w", err)
	}

	return cfg, nil
}

func loadFile(path string, cfg *Config) error {
	//nolint:gosec // G304: Path is from trusted config locations, not user input.
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}

func findProjectConfig() string {
	cwd, err := os.Getwd()
	if err != nil {
		return ""
	}

	dir := cwd
	for {
		// Check for cdd.json.
		path := filepath.Join(dir, configFileName)
		if _, err := os.Stat(path); err == nil {
			return path
		}

		// Check for .cdd.json (hidden).
		hiddenPath := filepath.Join(dir, "."+configFileName)
		if _, err := os.Stat(hiddenPath); err == nil {
			return hiddenPath
		}

		// Move to parent directory.
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}

	return ""
}

func mergeConfig(dst, src *Config) {
	for tier := range src.Models {
		dst.Models[tier] = src.Models[tier]
	}

	for name := range src.Providers {
		dst.Providers[name] = src.Providers[name]
	}

	// Merge connections (project connections override global by ID).
	if len(src.Connections) > 0 {
		connMap := make(map[string]Connection)
		for i := range dst.Connections {
			connMap[dst.Connections[i].ID] = dst.Connections[i]
		}
		for i := range src.Connections {
			connMap[src.Connections[i].ID] = src.Connections[i]
		}
		dst.Connections = make([]Connection, 0, len(connMap))
		for id := range connMap {
			dst.Connections = append(dst.Connections, connMap[id])
		}
	}

	if src.Options != nil {
		if dst.Options == nil {
			dst.Options = &Options{}
		}
		if len(src.Options.ContextPaths) > 0 {
			dst.Options.ContextPaths = src.Options.ContextPaths
		}
		if src.Options.DataDir != "" {
			dst.Options.DataDir = src.Options.DataDir
		}
		if src.Options.Debug {
			dst.Options.Debug = true
		}
	}
}

func configureProviders(cfg *Config, resolver *Resolver) {
	knownProviders := cfg.KnownProviders()

	// First, ensure all providers referenced by connections have entries in cfg.Providers.
	// This handles the case where a connection was added via /models without a legacy provider entry.
	ensureProvidersFromConnections(cfg, knownProviders)

	for i := range knownProviders {
		p := &knownProviders[i]
		userConfig, hasUserConfig := cfg.Providers[string(p.ID)]
		if !hasUserConfig {
			continue
		}
		if !configureProviderAuth(cfg, userConfig, p, resolver) {
			continue
		}
		configureProviderMetadata(userConfig, p)
		configureProviderModels(userConfig, p)
	}
}

// ensureProvidersFromConnections creates provider entries for any providers
// referenced by connections that don't already exist in cfg.Providers.
func ensureProvidersFromConnections(cfg *Config, knownProviders []catwalk.Provider) {
	// Build a map of known providers for quick lookup.
	knownByID := make(map[string]*catwalk.Provider)
	for i := range knownProviders {
		knownByID[string(knownProviders[i].ID)] = &knownProviders[i]
	}

	for i := range cfg.Connections {
		if cfg.Connections[i].ProviderID == "" {
			continue
		}

		// Skip if provider already exists in cfg.Providers.
		if _, exists := cfg.Providers[cfg.Connections[i].ProviderID]; exists {
			continue
		}

		// Look up the provider in knownProviders.
		known, ok := knownByID[cfg.Connections[i].ProviderID]
		if !ok {
			// Provider not in known list - might be a custom provider.
			// Create a minimal entry.
			cfg.Providers[cfg.Connections[i].ProviderID] = &ProviderConfig{
				ID: cfg.Connections[i].ProviderID,
			}
			continue
		}

		// Create a provider config from the known provider metadata.
		cfg.Providers[cfg.Connections[i].ProviderID] = &ProviderConfig{
			ID:      string(known.ID),
			Name:    known.Name,
			Type:    known.Type,
			BaseURL: known.APIEndpoint,
			Models:  known.Models,
		}
	}
}

// configureProviderAuth resolves API key and base URL. Returns false if provider should be removed.
func configureProviderAuth(cfg *Config, userConfig *ProviderConfig, p *catwalk.Provider, resolver *Resolver) bool {
	if userConfig.APIKey != "" {
		resolved, err := resolver.Resolve(userConfig.APIKey)
		if err != nil {
			delete(cfg.Providers, string(p.ID))
			return false
		}
		userConfig.APIKey = resolved
	}
	resolveBaseURL(userConfig, p, resolver)
	return true
}

func resolveBaseURL(userConfig *ProviderConfig, p *catwalk.Provider, resolver *Resolver) {
	if userConfig.BaseURL != "" {
		if resolved, err := resolver.Resolve(userConfig.BaseURL); err == nil {
			userConfig.BaseURL = resolved
		}
		return
	}
	if resolved, err := resolver.Resolve(p.APIEndpoint); err == nil {
		userConfig.BaseURL = resolved
	} else {
		userConfig.BaseURL = getDefaultAPIEndpoint(p.Type)
	}
}

func configureProviderMetadata(userConfig *ProviderConfig, p *catwalk.Provider) {
	userConfig.ID = string(p.ID)
	if userConfig.Name == "" {
		userConfig.Name = p.Name
	}
	if userConfig.Type == "" {
		userConfig.Type = p.Type
	}
	if userConfig.ExtraHeaders == nil {
		userConfig.ExtraHeaders = make(map[string]string)
	}
	if userConfig.OAuthToken != nil {
		userConfig.SetupClaudeCode()
	}
}

func configureProviderModels(userConfig *ProviderConfig, p *catwalk.Provider) {
	if len(userConfig.Models) == 0 {
		userConfig.Models = p.Models
		return
	}
	existingIDs := make(map[string]bool)
	for j := range userConfig.Models {
		existingIDs[userConfig.Models[j].ID] = true
	}
	for j := range p.Models {
		if !existingIDs[p.Models[j].ID] {
			userConfig.Models = append(userConfig.Models, p.Models[j])
		}
	}
}

func configureDefaultModels(cfg *Config) error {
	if len(cfg.Models) > 0 {
		return validateModels(cfg)
	}

	knownProviders := cfg.KnownProviders()
	for i := range knownProviders {
		p := &knownProviders[i]
		providerCfg, ok := cfg.Providers[string(p.ID)]
		if !ok || providerCfg.Disable {
			continue
		}
		if providerCfg.APIKey == "" {
			continue
		}
		if p.DefaultLargeModelID != "" {
			cfg.Models[SelectedModelTypeLarge] = SelectedModel{
				Model:    p.DefaultLargeModelID,
				Provider: string(p.ID),
			}
		}
		if p.DefaultSmallModelID != "" {
			cfg.Models[SelectedModelTypeSmall] = SelectedModel{
				Model:    p.DefaultSmallModelID,
				Provider: string(p.ID),
			}
		}
		if len(cfg.Models) > 0 {
			break
		}
	}

	if len(cfg.Models) == 0 {
		return fmt.Errorf("no providers configured with valid API keys")
	}

	return nil
}

func validateModels(cfg *Config) error {
	connManager := NewConnectionManager(cfg)

	for tier := range cfg.Models {
		model := cfg.Models[tier]
		// If ConnectionID is set, validate via connection (new system).
		if model.ConnectionID != "" {
			conn := connManager.Get(model.ConnectionID)
			if conn == nil {
				return fmt.Errorf("tier %s: connection %q not found", tier, model.ConnectionID)
			}
			if !conn.IsConfigured() {
				return fmt.Errorf("tier %s: connection %q has no authentication configured", tier, conn.Name)
			}
			continue
		}

		// Fall back to legacy provider validation.
		provider, ok := cfg.Providers[model.Provider]
		if !ok {
			return fmt.Errorf("tier %s: provider %q not configured", tier, model.Provider)
		}
		if provider.Disable {
			return fmt.Errorf("tier %s: provider %q is disabled", tier, model.Provider)
		}
	}
	return nil
}

func applyDefaults(cfg *Config) {
	if cfg.Options == nil {
		cfg.Options = &Options{}
	}
	if cfg.Options.DataDir == "" {
		cfg.Options.DataDir = filepath.Join(xdg.DataHome, appName)
	}
}

func getDefaultAPIEndpoint(providerType catwalk.Type) string {
	switch providerType {
	case catwalk.TypeAnthropic:
		return defaultAnthropicEndpoint
	case catwalk.TypeOpenAI, catwalk.TypeOpenAICompat, catwalk.TypeOpenRouter:
		return defaultOpenAIEndpoint
	case catwalk.TypeGoogle, catwalk.TypeAzure, catwalk.TypeBedrock, catwalk.TypeVertexAI:
		// These providers require user-configured endpoints.
		return ""
	default:
		return ""
	}
}

// GlobalConfigPath returns the path to the global configuration file.
func GlobalConfigPath() string {
	return filepath.Join(xdg.ConfigHome, appName, configFileName)
}

// DataDir returns the data directory path from configuration.
func (c *Config) DataDir() string {
	if c.Options != nil && c.Options.DataDir != "" {
		return c.Options.DataDir
	}
	return filepath.Join(xdg.DataHome, appName)
}

// Resolve resolves environment variables in a configuration value.
func (c *Config) Resolve(value string) (string, error) {
	resolver := NewResolver()
	return resolver.Resolve(value)
}
