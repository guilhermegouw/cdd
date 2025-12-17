package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/adrg/xdg"
)

const configFileName = "cdd.json"

// Load finds and loads configuration from standard locations.
// It merges global config with project config (project takes precedence),
// then configures providers using catwalk metadata.
func Load() (*Config, error) {
	cfg := NewConfig()
	resolver := NewResolver()

	// Load global config.
	globalPath := filepath.Join(xdg.ConfigHome, appName, configFileName)
	if err := loadFile(globalPath, cfg); err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("loading global config: %w", err)
	}

	// Load project config (searches upward from cwd).
	projectPath := findProjectConfig()
	if projectPath != "" {
		projectCfg := NewConfig()
		if err := loadFile(projectPath, projectCfg); err != nil {
			return nil, fmt.Errorf("loading project config: %w", err)
		}
		mergeConfig(cfg, projectCfg)
	}

	// Apply defaults before loading providers.
	applyDefaults(cfg)

	// Load known providers from catwalk.
	providers, err := LoadProviders(cfg)
	if err != nil {
		return nil, fmt.Errorf("loading providers: %w", err)
	}
	cfg.SetKnownProviders(providers)

	// Configure providers (merge user config with catwalk metadata).
	configureProviders(cfg, resolver)

	// Configure default model selections if not set.
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

	providers, err := LoadProviders(cfg)
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

// loadFile reads and unmarshals a JSON config file.
func loadFile(path string, cfg *Config) error {
	data, err := os.ReadFile(path) //nolint:gosec // Config file paths are trusted.
	if err != nil {
		return err
	}
	return json.Unmarshal(data, cfg)
}

// findProjectConfig searches for config file in current and parent directories.
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

// mergeConfig merges src into dst (src takes precedence).
func mergeConfig(dst, src *Config) {
	// Merge models.
	for tier, model := range src.Models {
		dst.Models[tier] = model
	}

	// Merge providers.
	for name, provider := range src.Providers {
		dst.Providers[name] = provider
	}

	// Merge options.
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

// configureProviders merges user config with catwalk provider metadata.
func configureProviders(cfg *Config, resolver *Resolver) {
	knownProviders := cfg.KnownProviders()
	for i := range knownProviders {
		p := &knownProviders[i]
		userConfig, hasUserConfig := cfg.Providers[string(p.ID)]

		// Skip providers not in user config that require API keys.
		if !hasUserConfig {
			continue
		}

		// Resolve API key from environment.
		if userConfig.APIKey != "" {
			resolved, err := resolver.Resolve(userConfig.APIKey)
			if err != nil {
				// Skip provider if API key can't be resolved.
				delete(cfg.Providers, string(p.ID))
				continue
			}
			userConfig.APIKey = resolved
		}

		// Resolve base URL from environment.
		if userConfig.BaseURL != "" {
			resolved, err := resolver.Resolve(userConfig.BaseURL)
			if err == nil {
				userConfig.BaseURL = resolved
			}
		} else {
			// Use catwalk default endpoint.
			userConfig.BaseURL = p.APIEndpoint
		}

		// Set provider metadata from catwalk.
		userConfig.ID = string(p.ID)
		if userConfig.Name == "" {
			userConfig.Name = p.Name
		}
		if userConfig.Type == "" {
			userConfig.Type = p.Type
		}

		// Merge models: user models take precedence, then catwalk defaults.
		if len(userConfig.Models) == 0 {
			userConfig.Models = p.Models
		} else {
			// Keep user models, add any catwalk models not already present.
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

		// Initialize extra headers map if needed.
		if userConfig.ExtraHeaders == nil {
			userConfig.ExtraHeaders = make(map[string]string)
		}
	}
}

// configureDefaultModels sets default model selections if not configured.
func configureDefaultModels(cfg *Config) error {
	// If models are already configured, validate them.
	if len(cfg.Models) > 0 {
		return validateModels(cfg)
	}

	// Find first available provider with default models.
	knownProviders := cfg.KnownProviders()
	for i := range knownProviders {
		p := &knownProviders[i]
		providerCfg, ok := cfg.Providers[string(p.ID)]
		if !ok || providerCfg.Disable {
			continue
		}

		// Check if provider has API key configured.
		if providerCfg.APIKey == "" {
			continue
		}

		// Set default large model.
		if p.DefaultLargeModelID != "" {
			cfg.Models[SelectedModelTypeLarge] = SelectedModel{
				Model:    p.DefaultLargeModelID,
				Provider: string(p.ID),
			}
		}

		// Set default small model.
		if p.DefaultSmallModelID != "" {
			cfg.Models[SelectedModelTypeSmall] = SelectedModel{
				Model:    p.DefaultSmallModelID,
				Provider: string(p.ID),
			}
		}

		// Found a valid provider, stop searching.
		if len(cfg.Models) > 0 {
			break
		}
	}

	if len(cfg.Models) == 0 {
		return fmt.Errorf("no providers configured with valid API keys")
	}

	return nil
}

// validateModels checks that selected models reference valid providers.
func validateModels(cfg *Config) error {
	for tier, model := range cfg.Models {
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

// applyDefaults sets default values for unset configuration.
func applyDefaults(cfg *Config) {
	if cfg.Options == nil {
		cfg.Options = &Options{}
	}

	// Set default data directory.
	if cfg.Options.DataDir == "" {
		cfg.Options.DataDir = filepath.Join(xdg.DataHome, appName)
	}
}

// GlobalConfigPath returns the path to the global config file.
func GlobalConfigPath() string {
	return filepath.Join(xdg.ConfigHome, appName, configFileName)
}

// DataDir returns the data directory path from config or default.
func (c *Config) DataDir() string {
	if c.Options != nil && c.Options.DataDir != "" {
		return c.Options.DataDir
	}
	return filepath.Join(xdg.DataHome, appName)
}

// Resolve resolves environment variables in a value.
func (c *Config) Resolve(value string) (string, error) {
	resolver := NewResolver()
	return resolver.Resolve(value)
}
