// Package config provides tests for custom provider management.
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

func TestCustomProviderManager_Load_Empty(t *testing.T) {
	// Create a temporary directory for testing.
	tmpDir := t.TempDir()
	manager := NewCustomProviderManager(tmpDir)

	providers, err := manager.Load()
	if err != nil {
		t.Fatalf("Load() error = %v", err)
	}

	if len(providers) != 0 {
		t.Errorf("Load() returned %d providers, want 0", len(providers))
	}
}

func TestCustomProviderManager_Add(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewCustomProviderManager(tmpDir)

	provider := CustomProvider{
		Name:        "Test Provider",
		ID:          "test-provider",
		Type:        catwalk.TypeOpenAICompat,
		APIEndpoint: "https://api.example.com/v1",
		Models: []catwalk.Model{
			{
				ID:            "model-1",
				Name:          "Model 1",
				ContextWindow: 128000,
			},
		},
	}

	if err := manager.Add(provider); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Verify the file was created.
	filePath := manager.GetFilePath()
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		t.Errorf("Add() did not create file at %s", filePath)
	}

	// Load and verify.
	providers, err := manager.Load()
	if err != nil {
		t.Fatalf("Load() after Add() error = %v", err)
	}

	if len(providers) != 1 {
		t.Fatalf("Load() returned %d providers, want 1", len(providers))
	}

	if providers[0].ID != provider.ID {
		t.Errorf("Load() returned provider with ID %q, want %q", providers[0].ID, provider.ID)
	}

	// Verify timestamps were set.
	if providers[0].CreatedAt.IsZero() {
		t.Error("CreatedAt was not set")
	}
	if providers[0].UpdatedAt.IsZero() {
		t.Error("UpdatedAt was not set")
	}
}

func TestCustomProviderManager_Add_Duplicate(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewCustomProviderManager(tmpDir)

	provider := CustomProvider{
		Name:        "Test Provider",
		ID:          "test-provider",
		Type:        catwalk.TypeOpenAICompat,
		APIEndpoint: "https://api.example.com/v1",
		Models:      []catwalk.Model{},
	}

	// Add the provider twice.
	if err := manager.Add(provider); err != nil {
		t.Fatalf("First Add() error = %v", err)
	}

	if err := manager.Add(provider); err == nil {
		t.Error("Second Add() should return error for duplicate ID")
	}
}

func TestCustomProviderManager_Update(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewCustomProviderManager(tmpDir)

	// Add initial provider.
	original := CustomProvider{
		Name:        "Original Name",
		ID:          "test-provider",
		Type:        catwalk.TypeOpenAICompat,
		APIEndpoint: "https://api.example.com/v1",
		Models:      []catwalk.Model{},
	}

	if err := manager.Add(original); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Wait a bit to ensure UpdatedAt timestamp changes.
	time.Sleep(10 * time.Millisecond)

	// Update the provider.
	updated := CustomProvider{
		Name:        "Updated Name",
		ID:          "test-provider",
		Type:        catwalk.TypeOpenAICompat,
		APIEndpoint: "https://api.example.com/v2",
		Models:      []catwalk.Model{},
	}

	if err := manager.Update("test-provider", updated); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	// Load and verify.
	providers, err := manager.Load()
	if err != nil {
		t.Fatalf("Load() after Update() error = %v", err)
	}

	if len(providers) != 1 {
		t.Fatalf("Load() returned %d providers, want 1", len(providers))
	}

	if providers[0].Name != "Updated Name" {
		t.Errorf("Name was not updated: got %q, want %q", providers[0].Name, "Updated Name")
	}

	if providers[0].APIEndpoint != "https://api.example.com/v2" {
		t.Errorf("APIEndpoint was not updated: got %q, want %q", providers[0].APIEndpoint, "https://api.example.com/v2")
	}

	// Verify CreatedAt was preserved.
	if providers[0].CreatedAt.IsZero() {
		t.Error("CreatedAt was not preserved")
	}

	// Verify UpdatedAt changed.
	if providers[0].UpdatedAt.Sub(providers[0].CreatedAt) < 0 {
		t.Error("UpdatedAt should be after CreatedAt")
	}
}

func TestCustomProviderManager_Update_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewCustomProviderManager(tmpDir)

	provider := CustomProvider{
		Name:   "Test Provider",
		ID:     "test-provider",
		Type:   catwalk.TypeOpenAICompat,
		Models: []catwalk.Model{},
	}

	if err := manager.Update("test-provider", provider); err == nil {
		t.Error("Update() should return error for non-existent provider")
	}
}

func TestCustomProviderManager_Remove(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewCustomProviderManager(tmpDir)

	// Add two providers.
	provider1 := CustomProvider{
		Name:   "Provider 1",
		ID:     "provider-1",
		Type:   catwalk.TypeOpenAICompat,
		Models: []catwalk.Model{},
	}

	provider2 := CustomProvider{
		Name:   "Provider 2",
		ID:     "provider-2",
		Type:   catwalk.TypeOpenAICompat,
		Models: []catwalk.Model{},
	}

	if err := manager.Add(provider1); err != nil {
		t.Fatalf("Add(provider1) error = %v", err)
	}
	if err := manager.Add(provider2); err != nil {
		t.Fatalf("Add(provider2) error = %v", err)
	}

	// Remove one provider.
	if err := manager.Remove("provider-1"); err != nil {
		t.Fatalf("Remove() error = %v", err)
	}

	// Verify only one remains.
	providers, err := manager.Load()
	if err != nil {
		t.Fatalf("Load() after Remove() error = %v", err)
	}

	if len(providers) != 1 {
		t.Errorf("Load() returned %d providers, want 1", len(providers))
	}

	if providers[0].ID != "provider-2" {
		t.Errorf("Remaining provider ID is %q, want %q", providers[0].ID, "provider-2")
	}
}

func TestCustomProviderManager_Remove_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewCustomProviderManager(tmpDir)

	if err := manager.Remove("non-existent"); err == nil {
		t.Error("Remove() should return error for non-existent provider")
	}
}

func TestCustomProviderManager_Get(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewCustomProviderManager(tmpDir)

	provider := CustomProvider{
		Name:        "Test Provider",
		ID:          "test-provider",
		Type:        catwalk.TypeOpenAICompat,
		APIEndpoint: "https://api.example.com/v1",
		Models:      []catwalk.Model{},
	}

	if err := manager.Add(provider); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	// Get the provider.
	retrieved, err := manager.Get("test-provider")
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}

	if retrieved.Name != provider.Name {
		t.Errorf("Get() returned name %q, want %q", retrieved.Name, provider.Name)
	}
}

func TestCustomProviderManager_Get_NotFound(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewCustomProviderManager(tmpDir)

	if _, err := manager.Get("non-existent"); err == nil {
		t.Error("Get() should return error for non-existent provider")
	}
}

func TestCustomProviderManager_Exists(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewCustomProviderManager(tmpDir)

	if manager.Exists("test-provider") {
		t.Error("Exists() returned true for non-existent provider")
	}

	provider := CustomProvider{
		Name:   "Test Provider",
		ID:     "test-provider",
		Type:   catwalk.TypeOpenAICompat,
		Models: []catwalk.Model{},
	}

	if err := manager.Add(provider); err != nil {
		t.Fatalf("Add() error = %v", err)
	}

	if !manager.Exists("test-provider") {
		t.Error("Exists() returned false for existing provider")
	}
}

func TestCustomProviderManager_GetFilePath(t *testing.T) {
	tmpDir := t.TempDir()
	manager := NewCustomProviderManager(tmpDir)

	expectedPath := filepath.Join(tmpDir, "custom-providers.json")
	if manager.GetFilePath() != expectedPath {
		t.Errorf("GetFilePath() = %q, want %q", manager.GetFilePath(), expectedPath)
	}
}

func TestCustomProviderToCatwalkProvider(t *testing.T) {
	customProvider := CustomProvider{
		Name:        "Test Provider",
		ID:          "test-provider",
		Type:        catwalk.TypeOpenAICompat,
		APIEndpoint: "https://api.example.com/v1",
		BaseURL:     "https://api.example.com/v2",
		DefaultHeaders: map[string]string{
			"X-Custom-Header": "custom-value",
		},
		DefaultLargeModelID: "model-large",
		DefaultSmallModelID: "model-small",
		Models: []catwalk.Model{
			{
				ID:   "model-1",
				Name: "Model 1",
			},
		},
	}

	catwalkProvider := customProvider.ToCatwalkProvider()

	if catwalkProvider.ID != catwalk.InferenceProvider("test-provider") {
		t.Errorf("ID = %q, want %q", catwalkProvider.ID, "test-provider")
	}

	if catwalkProvider.Name != "Test Provider" {
		t.Errorf("Name = %q, want %q", catwalkProvider.Name, "Test Provider")
	}

	if catwalkProvider.Type != catwalk.TypeOpenAICompat {
		t.Errorf("Type = %q, want %q", catwalkProvider.Type, catwalk.TypeOpenAICompat)
	}

	// APIEndpoint should be used from APIEndpoint if set.
	if catwalkProvider.APIEndpoint != "https://api.example.com/v1" {
		t.Errorf("APIEndpoint = %q, want %q", catwalkProvider.APIEndpoint, "https://api.example.com/v1")
	}

	if catwalkProvider.DefaultLargeModelID != "model-large" {
		t.Errorf("DefaultLargeModelID = %q, want %q", catwalkProvider.DefaultLargeModelID, "model-large")
	}

	if catwalkProvider.DefaultSmallModelID != "model-small" {
		t.Errorf("DefaultSmallModelID = %q, want %q", catwalkProvider.DefaultSmallModelID, "model-small")
	}

	if len(catwalkProvider.Models) != 1 {
		t.Errorf("Models length = %d, want 1", len(catwalkProvider.Models))
	}
}

func TestCustomProviderToCatwalkProvider_BaseURLFallback(t *testing.T) {
	// Test that BaseURL is used when APIEndpoint is empty.
	customProvider := CustomProvider{
		Name:        "Test Provider",
		ID:          "test-provider",
		Type:        catwalk.TypeOpenAICompat,
		BaseURL:     "https://api.example.com/v1",
		Models:      []catwalk.Model{},
	}

	catwalkProvider := customProvider.ToCatwalkProvider()

	if catwalkProvider.APIEndpoint != "https://api.example.com/v1" {
		t.Errorf("APIEndpoint = %q, want %q (from BaseURL)", catwalkProvider.APIEndpoint, "https://api.example.com/v1")
	}
}
