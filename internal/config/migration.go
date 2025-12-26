package config

import (
	"time"

	"github.com/google/uuid"
)

// MigrateToConnections migrates existing provider configurations to the new
// connections-based model. This provides backward compatibility for users
// upgrading from older versions.
//
// The migration:
// 1. Checks if connections already exist (skip if so)
// 2. Creates a connection for each provider with API key or OAuth configured
// 3. Updates model selections to use the new connection IDs
// 4. Persists the changes to disk
func MigrateToConnections(cfg *Config) error {
	// Skip if already migrated (connections exist).
	if len(cfg.Connections) > 0 {
		return nil
	}

	// Skip if no providers configured.
	if len(cfg.Providers) == 0 {
		return nil
	}

	// Track which providers have been migrated and their new connection IDs.
	providerToConnectionID := make(map[string]string)
	var migratedAny bool

	for providerID, providerCfg := range cfg.Providers {
		// Skip providers without authentication.
		if providerCfg.APIKey == "" && providerCfg.OAuthToken == nil {
			continue
		}

		// Create a connection from this provider.
		connID := uuid.New().String()
		name := providerCfg.Name
		if name == "" {
			name = providerID
		}

		conn := Connection{
			ID:           connID,
			Name:         name,
			ProviderID:   providerID,
			APIKey:       providerCfg.APIKey,
			OAuthToken:   providerCfg.OAuthToken,
			BaseURL:      providerCfg.BaseURL,
			ExtraHeaders: providerCfg.ExtraHeaders,
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		}

		cfg.Connections = append(cfg.Connections, conn)
		providerToConnectionID[providerID] = connID
		migratedAny = true
	}

	// If no providers were migrated, nothing to do.
	if !migratedAny {
		return nil
	}

	// Update model selections to use connection IDs.
	for tier, model := range cfg.Models {
		if model.Provider != "" {
			if connID, ok := providerToConnectionID[model.Provider]; ok {
				model.ConnectionID = connID
				cfg.Models[tier] = model
			}
		}
	}

	// Persist the migration.
	return Save(cfg)
}
