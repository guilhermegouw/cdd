package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/catwalk/pkg/catwalk"
	"github.com/charmbracelet/catwalk/pkg/embedded"
)

const (
	providersCacheFile = "providers.json"
	defaultCatwalkURL  = "https://catwalk.charm.sh"
	cacheMaxAge        = 24 * time.Hour
)

// ProvidersCache holds cached provider metadata from catwalk.
type ProvidersCache struct {
	UpdatedAt time.Time          `json:"updated_at"`
	Providers []catwalk.Provider `json:"providers"`
}

// LoadProviders loads provider metadata from catwalk.
// It tries: 1) fetch from URL, 2) cached data, 3) embedded fallback.
func LoadProviders(cfg *Config) ([]catwalk.Provider, error) {
	dataDir := cfg.DataDir()
	cachePath := filepath.Join(dataDir, providersCacheFile)

	// Try to fetch from catwalk API.
	catwalkURL := os.Getenv("CATWALK_URL")
	if catwalkURL == "" {
		catwalkURL = defaultCatwalkURL
	}

	client := catwalk.NewWithURL(catwalkURL)
	providers, err := client.GetProviders()
	if err == nil {
		// Successfully fetched, update cache (ignore cache write errors).
		if cacheErr := saveProvidersCache(cachePath, providers); cacheErr != nil {
			// Cache write failure is non-fatal, continue with fetched data.
			_ = cacheErr
		}
		return providers, nil
	}

	// Fetch failed, try cache.
	if cache, err := loadProvidersCache(cachePath); err == nil {
		if time.Since(cache.UpdatedAt) < cacheMaxAge {
			return cache.Providers, nil
		}
	}

	// Fall back to embedded providers.
	return embedded.GetAll(), nil
}

// UpdateProviders fetches and caches provider metadata from the given source.
// Source can be "embedded", an HTTP URL, or a local file path.
func UpdateProviders(cfg *Config, source string) error {
	var providers []catwalk.Provider
	var err error

	switch {
	case source == "embedded":
		providers = embedded.GetAll()
	case len(source) > 4 && source[:4] == "http":
		client := catwalk.NewWithURL(source)
		providers, err = client.GetProviders()
		if err != nil {
			return err
		}
	default:
		// Load from local file.
		data, err := os.ReadFile(source) //nolint:gosec // User-provided file path is trusted.
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &providers); err != nil {
			return err
		}
	}

	dataDir := cfg.DataDir()
	cachePath := filepath.Join(dataDir, providersCacheFile)
	return saveProvidersCache(cachePath, providers)
}

// loadProvidersCache reads cached provider data.
func loadProvidersCache(path string) (*ProvidersCache, error) {
	data, err := os.ReadFile(path) //nolint:gosec // Cache file path is derived from XDG.
	if err != nil {
		return nil, err
	}

	var cache ProvidersCache
	if err := json.Unmarshal(data, &cache); err != nil {
		return nil, err
	}

	return &cache, nil
}

// saveProvidersCache writes provider data to cache.
func saveProvidersCache(path string, providers []catwalk.Provider) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return err
	}

	cache := ProvidersCache{
		Providers: providers,
		UpdatedAt: time.Now(),
	}

	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(path, data, 0o600)
}

// DefaultDataDir returns the default data directory path.
func DefaultDataDir() string {
	return filepath.Join(xdg.DataHome, appName)
}
