// Package config provides custom provider management for CDD CLI.
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/adrg/xdg"
	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

const customProvidersFile = "custom-providers.json"

// CustomProvider represents a user-defined provider.
type CustomProvider struct {
	Name                string            `json:"name"`
	ID                  string            `json:"id"`
	Type                catwalk.Type      `json:"type"`
	APIEndpoint         string            `json:"api_endpoint,omitempty"`
	BaseURL             string            `json:"base_url,omitempty"`
	DefaultHeaders      map[string]string `json:"default_headers,omitempty"`
	DefaultLargeModelID string            `json:"default_large_model_id,omitempty"`
	DefaultSmallModelID string            `json:"default_small_model_id,omitempty"`
	Models              []catwalk.Model   `json:"models"`
	CreatedAt           time.Time         `json:"created_at"`
	UpdatedAt           time.Time         `json:"updated_at"`
}

// ToCatwalkProvider converts a CustomProvider to a catwalk.Provider.
func (cp *CustomProvider) ToCatwalkProvider() catwalk.Provider {
	// Determine the API endpoint.
	apiEndpoint := cp.APIEndpoint
	if apiEndpoint == "" {
		apiEndpoint = cp.BaseURL
	}

	p := catwalk.Provider{
		ID:                  catwalk.InferenceProvider(cp.ID),
		Name:                cp.Name,
		Type:                cp.Type,
		APIEndpoint:         apiEndpoint,
		DefaultLargeModelID: cp.DefaultLargeModelID,
		DefaultSmallModelID: cp.DefaultSmallModelID,
		Models:              cp.Models,
	}

	// Set default headers if provided.
	if len(cp.DefaultHeaders) > 0 {
		p.DefaultHeaders = cp.DefaultHeaders
	}

	return p
}

// CustomProvidersFile holds the custom providers storage structure.
type CustomProvidersFile struct {
	Version   string           `json:"version"`
	Providers []CustomProvider `json:"providers"`
}

// CustomProviderManager handles custom provider lifecycle.
type CustomProviderManager struct {
	filePath string
}

// NewCustomProviderManager creates a new CustomProviderManager.
func NewCustomProviderManager(dataDir string) *CustomProviderManager {
	if dataDir == "" {
		dataDir = filepath.Join(xdg.DataHome, appName)
	}
	return &CustomProviderManager{
		filePath: filepath.Join(dataDir, customProvidersFile),
	}
}

// Load loads all custom providers from storage.
func (m *CustomProviderManager) Load() ([]CustomProvider, error) {
	data, err := os.ReadFile(m.filePath) //nolint:gosec // File path is derived from XDG.
	if err != nil {
		if os.IsNotExist(err) {
			return []CustomProvider{}, nil
		}
		return nil, fmt.Errorf("reading custom providers file: %w", err)
	}

	var file CustomProvidersFile
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, fmt.Errorf("parsing custom providers file: %w", err)
	}

	return file.Providers, nil
}

// Save saves all custom providers to storage.
func (m *CustomProviderManager) Save(providers []CustomProvider) error {
	dir := filepath.Dir(m.filePath)
	if err := os.MkdirAll(dir, 0o750); err != nil {
		return fmt.Errorf("creating custom providers directory: %w", err)
	}

	file := CustomProvidersFile{
		Version:   "1.0",
		Providers: providers,
	}

	data, err := json.MarshalIndent(file, "", "  ")
	if err != nil {
		return fmt.Errorf("marshaling custom providers: %w", err)
	}

	//nolint:gosec // 0o600 is intentionally restrictive for security.
	if err := os.WriteFile(m.filePath, data, 0o600); err != nil {
		return fmt.Errorf("writing custom providers file: %w", err)
	}

	return nil
}

// Add adds a new custom provider.
func (m *CustomProviderManager) Add(provider CustomProvider) error {
	providers, err := m.Load()
	if err != nil {
		return err
	}

	// Check for duplicate ID.
	for i := range providers {
		if providers[i].ID == provider.ID {
			return fmt.Errorf("provider with ID %q already exists", provider.ID)
		}
	}

	provider.CreatedAt = time.Now()
	provider.UpdatedAt = time.Now()

	providers = append(providers, provider)
	return m.Save(providers)
}

// Update updates an existing custom provider.
func (m *CustomProviderManager) Update(providerID string, updated CustomProvider) error {
	providers, err := m.Load()
	if err != nil {
		return err
	}

	found := false
	for i := range providers {
		if providers[i].ID != providerID {
			continue
		}
		// Preserve creation timestamp.
		updated.CreatedAt = providers[i].CreatedAt
		updated.UpdatedAt = time.Now()
		providers[i] = updated
		found = true
		break
	}

	if !found {
		return fmt.Errorf("provider with ID %q not found", providerID)
	}

	return m.Save(providers)
}

// Remove removes a custom provider by ID.
func (m *CustomProviderManager) Remove(providerID string) error {
	providers, err := m.Load()
	if err != nil {
		return err
	}

	found := false
	capacity := len(providers) - 1
	if capacity < 0 {
		capacity = 0
	}
	newProviders := make([]CustomProvider, 0, capacity)
	for i := range providers {
		if providers[i].ID != providerID {
			newProviders = append(newProviders, providers[i])
		} else {
			found = true
		}
	}

	if !found {
		return fmt.Errorf("provider with ID %q not found", providerID)
	}

	return m.Save(newProviders)
}

// Get retrieves a custom provider by ID.
func (m *CustomProviderManager) Get(providerID string) (*CustomProvider, error) {
	providers, err := m.Load()
	if err != nil {
		return nil, err
	}

	for i := range providers {
		if providers[i].ID == providerID {
			return &providers[i], nil
		}
	}

	return nil, fmt.Errorf("provider with ID %q not found", providerID)
}

// Exists checks if a custom provider with the given ID exists.
func (m *CustomProviderManager) Exists(providerID string) bool {
	providers, err := m.Load()
	if err != nil {
		return false
	}

	for i := range providers {
		if providers[i].ID == providerID {
			return true
		}
	}

	return false
}

// GetFilePath returns the file path where custom providers are stored.
func (m *CustomProviderManager) GetFilePath() string {
	return m.filePath
}
