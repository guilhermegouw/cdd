package config

import (
	"os"
)

// IsFirstRun checks if this is the first time running CDD.
// Returns true if no config file exists or if no providers have API keys.
func IsFirstRun() bool {
	// Check if global config file exists.
	configPath := GlobalConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return true
	}

	// Try to load config and check for valid providers.
	cfg, err := Load()
	if err != nil {
		// If config fails to load (e.g., no valid API keys), it's effectively first run.
		return true
	}

	// Check if any providers have API keys configured.
	return !hasConfiguredProviders(cfg)
}

// hasConfiguredProviders checks if any providers have API keys set.
func hasConfiguredProviders(cfg *Config) bool {
	for _, provider := range cfg.Providers {
		if provider.APIKey != "" && !provider.Disable {
			return true
		}
	}
	return false
}

// NeedsSetup checks if the application needs initial setup.
// This is similar to IsFirstRun but can be used after partial setup.
func NeedsSetup() bool {
	cfg, err := Load()
	if err != nil {
		return true
	}

	// Check if we have at least one configured model.
	if len(cfg.Models) == 0 {
		return true
	}

	// Check if the configured models reference valid providers.
	for _, model := range cfg.Models {
		provider, ok := cfg.Providers[model.Provider]
		if !ok || provider.APIKey == "" || provider.Disable {
			return true
		}
	}

	return false
}
