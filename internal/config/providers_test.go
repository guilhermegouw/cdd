package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

func TestProvidersCache_Struct(t *testing.T) {
	now := time.Now()
	cache := ProvidersCache{
		UpdatedAt: now,
		Providers: []catwalk.Provider{
			{ID: "openai", Name: "OpenAI"},
		},
	}

	if cache.UpdatedAt != now {
		t.Errorf("UpdatedAt = %v, want %v", cache.UpdatedAt, now)
	}
	if len(cache.Providers) != 1 {
		t.Errorf("Providers length = %d, want 1", len(cache.Providers))
	}
	if cache.Providers[0].ID != "openai" {
		t.Errorf("Providers[0].ID = %q, want %q", cache.Providers[0].ID, "openai")
	}
}

func TestLoadProvidersCache(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "providers.json")

	// Create cache file.
	cache := ProvidersCache{
		UpdatedAt: time.Now(),
		Providers: []catwalk.Provider{
			{ID: "openai", Name: "OpenAI"},
			{ID: "anthropic", Name: "Anthropic"},
		},
	}
	data, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("Failed to marshal cache: %v", err)
	}
	//nolint:gosec // Test file, permissions not critical.
	if writeErr := os.WriteFile(cachePath, data, 0o644); writeErr != nil {
		t.Fatalf("Failed to write cache: %v", writeErr)
	}

	loaded, err := loadProvidersCache(cachePath)
	if err != nil {
		t.Fatalf("loadProvidersCache() error = %v", err)
	}

	if len(loaded.Providers) != 2 {
		t.Errorf("Providers length = %d, want 2", len(loaded.Providers))
	}
	if loaded.Providers[0].ID != "openai" {
		t.Errorf("Providers[0].ID = %q, want %q", loaded.Providers[0].ID, "openai")
	}
}

func TestLoadProvidersCache_NonExistent(t *testing.T) {
	_, err := loadProvidersCache("/non/existent/path.json")
	if err == nil {
		t.Error("loadProvidersCache() expected error for non-existent file")
	}
}

func TestLoadProvidersCache_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "providers.json")

	//nolint:gosec // Test file, permissions not critical.
	if err := os.WriteFile(cachePath, []byte("invalid json"), 0o644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	_, err := loadProvidersCache(cachePath)
	if err == nil {
		t.Error("loadProvidersCache() expected error for invalid JSON")
	}
}

func TestSaveProvidersCache(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "subdir", "providers.json")

	providers := []catwalk.Provider{
		{ID: "openai", Name: "OpenAI"},
	}

	err := saveProvidersCache(cachePath, providers)
	if err != nil {
		t.Fatalf("saveProvidersCache() error = %v", err)
	}

	// Verify file was created.
	if _, statErr := os.Stat(cachePath); os.IsNotExist(statErr) {
		t.Error("Cache file was not created")
	}

	// Verify content.
	loaded, err := loadProvidersCache(cachePath)
	if err != nil {
		t.Fatalf("Failed to load saved cache: %v", err)
	}
	if len(loaded.Providers) != 1 {
		t.Errorf("Providers length = %d, want 1", len(loaded.Providers))
	}
	if loaded.Providers[0].ID != "openai" {
		t.Errorf("Providers[0].ID = %q, want %q", loaded.Providers[0].ID, "openai")
	}
	if loaded.UpdatedAt.IsZero() {
		t.Error("UpdatedAt is zero")
	}
}

func TestSaveProvidersCache_CreateDirectory(t *testing.T) {
	tempDir := t.TempDir()
	cachePath := filepath.Join(tempDir, "deep", "nested", "dir", "providers.json")

	providers := []catwalk.Provider{{ID: "test"}}

	err := saveProvidersCache(cachePath, providers)
	if err != nil {
		t.Fatalf("saveProvidersCache() error = %v", err)
	}

	if _, statErr := os.Stat(cachePath); os.IsNotExist(statErr) {
		t.Error("Cache file was not created in nested directory")
	}
}

func TestDefaultDataDir(t *testing.T) {
	dir := DefaultDataDir()
	if dir == "" {
		t.Error("DefaultDataDir() returned empty string")
	}
	if !filepath.IsAbs(dir) {
		t.Errorf("DefaultDataDir() = %q, expected absolute path", dir)
	}
}

func TestUpdateProviders_Embedded(t *testing.T) {
	tempDir := t.TempDir()
	cfg := NewConfig()
	cfg.Options = &Options{DataDir: tempDir}

	err := UpdateProviders(cfg, "embedded")
	if err != nil {
		t.Fatalf("UpdateProviders() error = %v", err)
	}

	// Verify cache file was created.
	cachePath := filepath.Join(tempDir, "providers.json")
	if _, statErr := os.Stat(cachePath); os.IsNotExist(statErr) {
		t.Error("Cache file was not created")
	}

	// Verify it contains providers.
	loaded, err := loadProvidersCache(cachePath)
	if err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}
	if len(loaded.Providers) == 0 {
		t.Error("No providers in cache")
	}
}

func TestUpdateProviders_LocalFile(t *testing.T) {
	tempDir := t.TempDir()

	// Create a local providers file.
	localPath := filepath.Join(tempDir, "local-providers.json")
	providers := []catwalk.Provider{
		{ID: "custom", Name: "Custom Provider"},
	}
	data, err := json.Marshal(providers)
	if err != nil {
		t.Fatalf("Failed to marshal providers: %v", err)
	}
	//nolint:gosec // Test file, permissions not critical.
	if writeErr := os.WriteFile(localPath, data, 0o644); writeErr != nil {
		t.Fatalf("Failed to write local file: %v", writeErr)
	}

	cfg := NewConfig()
	cfg.Options = &Options{DataDir: tempDir}

	err = UpdateProviders(cfg, localPath)
	if err != nil {
		t.Fatalf("UpdateProviders() error = %v", err)
	}

	// Verify cache was updated.
	cachePath := filepath.Join(tempDir, "providers.json")
	loaded, err := loadProvidersCache(cachePath)
	if err != nil {
		t.Fatalf("Failed to load cache: %v", err)
	}
	if len(loaded.Providers) != 1 {
		t.Errorf("Providers length = %d, want 1", len(loaded.Providers))
	}
	if loaded.Providers[0].ID != "custom" {
		t.Errorf("Providers[0].ID = %q, want %q", loaded.Providers[0].ID, "custom")
	}
}

func TestUpdateProviders_LocalFile_NonExistent(t *testing.T) {
	tempDir := t.TempDir()
	cfg := NewConfig()
	cfg.Options = &Options{DataDir: tempDir}

	err := UpdateProviders(cfg, "/non/existent/file.json")
	if err == nil {
		t.Error("UpdateProviders() expected error for non-existent file")
	}
}

func TestUpdateProviders_LocalFile_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()

	localPath := filepath.Join(tempDir, "invalid.json")
	//nolint:gosec // Test file, permissions not critical.
	if err := os.WriteFile(localPath, []byte("not json"), 0o644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	cfg := NewConfig()
	cfg.Options = &Options{DataDir: tempDir}

	err := UpdateProviders(cfg, localPath)
	if err == nil {
		t.Error("UpdateProviders() expected error for invalid JSON")
	}
}

func TestLoadProviders_FromCache(t *testing.T) {
	tempDir := t.TempDir()

	// Create a fresh cache.
	cachePath := filepath.Join(tempDir, "providers.json")
	cache := ProvidersCache{
		UpdatedAt: time.Now(), // Recent.
		Providers: []catwalk.Provider{
			{ID: "cached-provider", Name: "Cached"},
		},
	}
	data, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("Failed to marshal cache: %v", err)
	}
	//nolint:gosec // Test file, permissions not critical.
	if writeErr := os.WriteFile(cachePath, data, 0o644); writeErr != nil {
		t.Fatalf("Failed to write cache: %v", writeErr)
	}

	// Set CATWALK_URL to invalid URL to force cache usage.
	t.Setenv("CATWALK_URL", "http://invalid.invalid.invalid")

	cfg := NewConfig()
	cfg.Options = &Options{DataDir: tempDir}

	providers, err := LoadProviders(cfg)
	if err != nil {
		t.Fatalf("LoadProviders() error = %v", err)
	}

	// Should have loaded from cache since URL is invalid.
	// Note: might fall back to embedded if cache is stale.
	if len(providers) == 0 {
		t.Error("No providers loaded")
	}
}

func TestLoadProviders_FallbackToEmbedded(t *testing.T) {
	tempDir := t.TempDir()

	// Set CATWALK_URL to invalid URL.
	t.Setenv("CATWALK_URL", "http://invalid.invalid.invalid")

	cfg := NewConfig()
	cfg.Options = &Options{DataDir: tempDir}

	// No cache file exists, should fall back to embedded.
	providers, err := LoadProviders(cfg)
	if err != nil {
		t.Fatalf("LoadProviders() error = %v", err)
	}

	if len(providers) == 0 {
		t.Error("No providers loaded from embedded fallback")
	}
}

func TestLoadProviders_StaleCache(t *testing.T) {
	tempDir := t.TempDir()

	// Create a stale cache.
	cachePath := filepath.Join(tempDir, "providers.json")
	cache := ProvidersCache{
		UpdatedAt: time.Now().Add(-48 * time.Hour), // 48 hours ago, past 24h max age.
		Providers: []catwalk.Provider{
			{ID: "stale-provider"},
		},
	}
	data, err := json.Marshal(cache)
	if err != nil {
		t.Fatalf("Failed to marshal cache: %v", err)
	}
	//nolint:gosec // Test file, permissions not critical.
	if writeErr := os.WriteFile(cachePath, data, 0o644); writeErr != nil {
		t.Fatalf("Failed to write cache: %v", writeErr)
	}

	// Set CATWALK_URL to invalid URL.
	t.Setenv("CATWALK_URL", "http://invalid.invalid.invalid")

	cfg := NewConfig()
	cfg.Options = &Options{DataDir: tempDir}

	providers, err := LoadProviders(cfg)
	if err != nil {
		t.Fatalf("LoadProviders() error = %v", err)
	}

	// Stale cache should be skipped, falling back to embedded.
	// Embedded providers should have standard providers like openai, anthropic.
	hasStandardProvider := false
	for _, p := range providers {
		if p.ID == "openai" || p.ID == "anthropic" {
			hasStandardProvider = true
			break
		}
	}
	if !hasStandardProvider && len(providers) > 0 {
		// If we got providers but none are standard, that's fine - embedded might have different ones.
		// Just ensure we didn't get the stale "stale-provider".
		for _, p := range providers {
			if p.ID == "stale-provider" {
				t.Error("Loaded stale provider instead of falling back to embedded")
			}
		}
	}
}

func TestProvidersCache_JSONMarshaling(t *testing.T) {
	original := ProvidersCache{
		UpdatedAt: time.Date(2024, 1, 15, 10, 30, 0, 0, time.UTC),
		Providers: []catwalk.Provider{
			{ID: "openai", Name: "OpenAI", Type: catwalk.TypeOpenAI},
		},
	}

	data, err := json.Marshal(original)
	if err != nil {
		t.Fatalf("json.Marshal() error = %v", err)
	}

	var decoded ProvidersCache
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("json.Unmarshal() error = %v", err)
	}

	if decoded.Providers[0].ID != original.Providers[0].ID {
		t.Errorf("Provider ID = %q, want %q", decoded.Providers[0].ID, original.Providers[0].ID)
	}
}

func TestUpdateProviders_HTTPUrl_Fails(t *testing.T) {
	tempDir := t.TempDir()
	cfg := NewConfig()
	cfg.Options = &Options{DataDir: tempDir}

	// Use an invalid HTTP URL to test the HTTP path (will fail to fetch).
	err := UpdateProviders(cfg, "http://invalid.invalid.invalid/providers.json")
	if err == nil {
		t.Error("UpdateProviders() expected error for invalid HTTP URL")
	}
}

func TestLoadProviders_CacheWriteFailure(t *testing.T) {
	// Create a read-only directory to simulate cache write failure.
	tempDir := t.TempDir()

	// Use a path within a non-existent deeply nested directory that can't be created
	// Actually, MkdirAll will create it, so let's use a file as a "directory" to cause failure.
	blockingFile := filepath.Join(tempDir, "blocking")
	//nolint:gosec // Test file.
	if err := os.WriteFile(blockingFile, []byte("block"), 0o644); err != nil {
		t.Fatalf("Failed to create blocking file: %v", err)
	}

	// Create config that points to a path where the file exists (not a directory).
	cfg := NewConfig()
	cfg.Options = &Options{DataDir: blockingFile}

	// Set CATWALK_URL to invalid to use embedded fallback.
	t.Setenv("CATWALK_URL", "http://invalid.invalid.invalid")

	// Should still work because cache failures are non-fatal.
	providers, err := LoadProviders(cfg)
	if err != nil {
		t.Fatalf("LoadProviders() error = %v", err)
	}
	if len(providers) == 0 {
		t.Error("No providers loaded")
	}
}

func TestSaveProvidersCache_MkdirAllError(t *testing.T) {
	// Try to save to a path where the parent is a file, not a directory.
	tempDir := t.TempDir()
	blockingFile := filepath.Join(tempDir, "notadir")
	//nolint:gosec // Test file.
	if err := os.WriteFile(blockingFile, []byte("block"), 0o644); err != nil {
		t.Fatalf("Failed to create blocking file: %v", err)
	}

	cachePath := filepath.Join(blockingFile, "subdir", "cache.json")
	providers := []catwalk.Provider{{ID: "test"}}

	err := saveProvidersCache(cachePath, providers)
	if err == nil {
		t.Error("saveProvidersCache() expected error when parent is a file")
	}
}

func TestLoadProviders_DefaultURL(t *testing.T) {
	tempDir := t.TempDir()
	cfg := NewConfig()
	cfg.Options = &Options{DataDir: tempDir}

	// Unset CATWALK_URL to use default (will fail but cover the path).
	t.Setenv("CATWALK_URL", "")

	// Should use embedded fallback when default URL fails.
	providers, err := LoadProviders(cfg)
	if err != nil {
		t.Fatalf("LoadProviders() error = %v", err)
	}
	if len(providers) == 0 {
		t.Error("No providers loaded")
	}
}
