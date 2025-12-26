// Package config provides tests for provider loading.
package config

import (
	"os"
	"testing"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

func TestProviderLoader_LoadAllProviders(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewProviderLoader(tmpDir)

	cfg := NewConfig()
	cfg.Options = &Options{DataDir: tmpDir}

	// Load all providers (should include catwalk providers).
	providers, err := loader.LoadAllProviders(cfg)
	if err != nil {
		t.Fatalf("LoadAllProviders() error = %v", err)
	}

	// Should have at least some catwalk providers.
	if len(providers) == 0 {
		t.Error("LoadAllProviders() returned no providers")
	}
}

func TestProviderLoader_LoadAllProviders_WithCustom(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewProviderLoader(tmpDir)

	// Add a custom provider.
	customManager := loader.GetCustomProviderManager()
	customProvider := CustomProvider{
		Name:        "Custom Provider",
		ID:          "custom-provider",
		Type:        catwalk.TypeOpenAICompat,
		APIEndpoint: "https://custom.example.com/v1",
		Models: []catwalk.Model{
			{
				ID:            "custom-model",
				Name:          "Custom Model",
				ContextWindow: 128000,
			},
		},
	}

	if err := customManager.Add(customProvider); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	cfg := NewConfig()
	cfg.Options = &Options{DataDir: tmpDir}

	// Load all providers.
	providers, err := loader.LoadAllProviders(cfg)
	if err != nil {
		t.Fatalf("LoadAllProviders() error = %v", err)
	}

	// Should include both catwalk and custom providers.
	if len(providers) == 0 {
		t.Fatal("LoadAllProviders() returned no providers")
	}

	// Find the custom provider.
	found := false
	for _, p := range providers {
		if p.ID == catwalk.InferenceProvider("custom-provider") {
			found = true
			if p.Name != "Custom Provider" {
				t.Errorf("Custom provider name = %q, want %q", p.Name, "Custom Provider")
			}
			break
		}
	}

	if !found {
		t.Error("Custom provider not found in loaded providers")
	}
}

func TestProviderLoader_GetCustomProviderManager(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewProviderLoader(tmpDir)

	manager := loader.GetCustomProviderManager()
	if manager == nil {
		t.Fatal("GetCustomProviderManager() returned nil")
	}

	if manager.GetFilePath() == "" {
		t.Error("GetCustomProviderManager() returned manager with empty file path")
	}
}

func TestProviderLoader_SetCatwalkURL(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewProviderLoader(tmpDir)

	// Set custom URL - this is a coverage test.
	customURL := "https://custom.catwalk.example.com"
	loader.SetCatwalkURL(customURL)
	_ = loader
}

func TestProviderLoader_DisableAutoUpdates(t *testing.T) {
	tmpDir := t.TempDir()
	loader := NewProviderLoader(tmpDir)

	// Disable auto updates - this is a coverage test.
	loader.DisableAutoUpdates()
	_ = loader
}

func TestProviderLoader_SaveProvidersCache(t *testing.T) {
	tmpDir := t.TempDir()

	cfg := NewConfig()
	cfg.Options = &Options{DataDir: tmpDir}

	// Create some test providers.
	providers := []catwalk.Provider{
		{
			ID:   catwalk.InferenceProvider("test"),
			Name: "Test Provider",
			Type: catwalk.TypeOpenAICompat,
		},
	}

	cachePath := getProvidersCachePath(cfg.DataDir())

	if err := saveProvidersCache(cachePath, providers); err != nil {
		t.Fatalf("saveProvidersCache() error = %v", err)
	}

	// Verify file exists.
	if _, err := os.Stat(cachePath); os.IsNotExist(err) {
		t.Errorf("saveProvidersCache() did not create file at %s", cachePath)
	}
}
