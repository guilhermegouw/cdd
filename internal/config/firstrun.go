package config

import (
	"os"
)

// IsFirstRun checks if this is the first time running CDD.
// Returns true if no config file exists or if no connections are configured.
//
// The system uses Connections as the single source of truth for authentication.
// Legacy Providers data is automatically migrated to Connections on load.
func IsFirstRun() bool {
	// Check if global config file exists.
	configPath := GlobalConfigPath()
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return true
	}

	// Try to load config.
	cfg, err := Load()
	if err != nil {
		// If config fails to load, it's effectively first run.
		return true
	}

	// Check if any connections are configured with authentication.
	return !hasConfiguredConnections(cfg)
}

// hasConfiguredConnections checks if any connections have authentication configured.
func hasConfiguredConnections(cfg *Config) bool {
	for i := range cfg.Connections {
		if cfg.Connections[i].IsConfigured() {
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

	// Check if the configured models reference valid connections.
	connManager := NewConnectionManager(cfg)
	for tier := range cfg.Models {
		if cfg.Models[tier].ConnectionID == "" {
			// No connection ID means this model hasn't been properly configured.
			return true
		}
		conn := connManager.Get(cfg.Models[tier].ConnectionID)
		if conn == nil || !conn.IsConfigured() {
			return true
		}
	}

	return false
}
