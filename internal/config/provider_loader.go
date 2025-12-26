// Package config provides provider loading and merging functionality.
//
//nolint:gocritic // rangeValCopy is acceptable for catwalk types.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

// ProviderLoader combines multiple provider sources.
type ProviderLoader struct {
	catwalkURL     string
	customManager  *CustomProviderManager
	disableUpdates bool
}

// NewProviderLoader creates a new ProviderLoader.
func NewProviderLoader(dataDir string) *ProviderLoader {
	return &ProviderLoader{
		catwalkURL:     defaultCatwalkURL,
		customManager:  NewCustomProviderManager(dataDir),
		disableUpdates: os.Getenv("CDD_DISABLE_PROVIDER_AUTO_UPDATE") == "1",
	}
}

// SetCatwalkURL sets a custom catwalk URL.
func (pl *ProviderLoader) SetCatwalkURL(url string) {
	pl.catwalkURL = url
}

// DisableAutoUpdates disables automatic provider updates.
func (pl *ProviderLoader) DisableAutoUpdates() {
	pl.disableUpdates = true
}

// LoadAllProviders returns merged provider list from catwalk and custom providers.
// Merge strategy:
// 1. Load catwalk providers (cached or fresh)
// 2. Load custom providers from storage
// 3. Merge by provider ID (custom providers override catwalk)
// 4. Validate all providers
// 5. Return combined list.
func (pl *ProviderLoader) LoadAllProviders(cfg *Config) ([]catwalk.Provider, error) {
	// Load catwalk providers.
	catwalkProviders, err := pl.loadCatwalkProviders(cfg)
	if err != nil {
		return nil, fmt.Errorf("loading catwalk providers: %w", err)
	}

	// Load custom providers.
	customProviders, err := pl.customManager.Load()
	if err != nil {
		return nil, fmt.Errorf("loading custom providers: %w", err)
	}

	// Merge providers.
	return pl.mergeProviders(catwalkProviders, customProviders), nil
}

// loadCatwalkProviders loads providers from catwalk API, cache, or embedded fallback.
func (pl *ProviderLoader) loadCatwalkProviders(cfg *Config) ([]catwalk.Provider, error) {
	// If auto-updates are disabled, only use cache or embedded.
	if pl.disableUpdates {
		return pl.loadCachedOrEmbedded(cfg)
	}

	// Try to fetch from catwalk API.
	client := catwalk.NewWithURL(pl.catwalkURL)
	providers, err := client.GetProviders()
	if err == nil {
		// Successfully fetched, update cache.
		dataDir := cfg.DataDir()
		cachePath := getProvidersCachePath(dataDir)
		// Cache write failure is non-fatal, ignore error.
		_ = saveProvidersCache(cachePath, providers) //nolint:errcheck // intentionally ignoring cache write error
		return providers, nil
	}

	// Fetch failed, try cache or embedded.
	return pl.loadCachedOrEmbedded(cfg)
}

// loadCachedOrEmbedded loads providers from cache or falls back to embedded.
func (pl *ProviderLoader) loadCachedOrEmbedded(cfg *Config) ([]catwalk.Provider, error) {
	return LoadProviders(cfg)
}

// mergeProviders merges catwalk providers with custom providers.
// Custom providers override catwalk providers with the same ID.
func (pl *ProviderLoader) mergeProviders(catwalkProviders []catwalk.Provider, customProviders []CustomProvider) []catwalk.Provider {
	// Build a map of catwalk providers for easy lookup.
	catwalkMap := make(map[string]catwalk.Provider)
	for _, p := range catwalkProviders {
		catwalkMap[string(p.ID)] = p
	}

	// Override with custom providers.
	for i := range customProviders {
		catwalkMap[customProviders[i].ID] = customProviders[i].ToCatwalkProvider()
	}

	// Convert map back to slice.
	result := make([]catwalk.Provider, 0, len(catwalkMap))
	for _, p := range catwalkMap {
		result = append(result, p)
	}

	return result
}

// UpdateProviders fetches and caches provider metadata from the given source.
// Source can be "embedded", an HTTP URL, or a local file path.
func (pl *ProviderLoader) UpdateProviders(cfg *Config, source string) error {
	return UpdateProviders(cfg, source)
}

// GetCustomProviderManager returns the custom provider manager.
func (pl *ProviderLoader) GetCustomProviderManager() *CustomProviderManager {
	return pl.customManager
}

// getProvidersCachePath returns the path to the providers cache file.
func getProvidersCachePath(dataDir string) string {
	return filepath.Join(dataDir, "providers.json")
}
