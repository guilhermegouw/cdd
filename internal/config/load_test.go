package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

const (
	testModelGPT4o = "gpt-4o"
	testProviderID = "openai"
)

func TestLoadFile(t *testing.T) {
	// Create a temporary config file.
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "test.json")

	configContent := `{
		"models": {
			"large": {"model": "gpt-4o", "provider": "openai"},
			"small": {"model": "gpt-4o-mini", "provider": "openai"}
		},
		"providers": {
			"openai": {
				"api_key": "$OPENAI_API_KEY",
				"type": "openai"
			}
		},
		"options": {
			"debug": true
		}
	}`

	//nolint:gosec // Test file, permissions not critical.
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg := NewConfig()
	err := loadFile(configPath, cfg)
	if err != nil {
		t.Fatalf("loadFile() error = %v", err)
	}

	// Verify models.
	if cfg.Models[SelectedModelTypeLarge].Model != testModelGPT4o {
		t.Errorf("Large model = %q, want %q", cfg.Models[SelectedModelTypeLarge].Model, testModelGPT4o)
	}
	if cfg.Models[SelectedModelTypeSmall].Model != "gpt-4o-mini" {
		t.Errorf("Small model = %q, want %q", cfg.Models[SelectedModelTypeSmall].Model, "gpt-4o-mini")
	}

	// Verify provider.
	if cfg.Providers[testProviderID] == nil {
		t.Fatal("Provider 'openai' not loaded")
	}
	if cfg.Providers[testProviderID].APIKey != "$OPENAI_API_KEY" {
		t.Errorf("APIKey = %q, want %q", cfg.Providers[testProviderID].APIKey, "$OPENAI_API_KEY")
	}

	// Verify options.
	if !cfg.Options.Debug {
		t.Error("Debug = false, want true")
	}
}

func TestLoadFile_NonExistent(t *testing.T) {
	cfg := NewConfig()
	err := loadFile("/non/existent/path.json", cfg)
	if err == nil {
		t.Error("loadFile() expected error for non-existent file")
	}
}

func TestLoadFile_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.json")

	//nolint:gosec // Test file, permissions not critical.
	if err := os.WriteFile(configPath, []byte("not valid json"), 0o644); err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	cfg := NewConfig()
	err := loadFile(configPath, cfg)
	if err == nil {
		t.Error("loadFile() expected error for invalid JSON")
	}
}

func TestMergeConfig(t *testing.T) {
	dst := NewConfig()
	dst.Models[SelectedModelTypeLarge] = SelectedModel{Model: "dst-large", Provider: "openai"}
	dst.Models[SelectedModelTypeSmall] = SelectedModel{Model: "dst-small", Provider: "openai"}
	dst.Providers["openai"] = &ProviderConfig{ID: "openai", APIKey: "dst-key"}
	dst.Options = &Options{
		ContextPaths: []string{"DST.md"},
		DataDir:      "/dst/data",
		Debug:        false,
	}

	src := NewConfig()
	src.Models[SelectedModelTypeLarge] = SelectedModel{Model: "src-large", Provider: "anthropic"}
	src.Providers["anthropic"] = &ProviderConfig{ID: "anthropic", APIKey: "src-key"}
	src.Options = &Options{
		ContextPaths: []string{"SRC.md"},
		Debug:        true,
	}

	mergeConfig(dst, src)

	// Large model should be overwritten.
	if dst.Models[SelectedModelTypeLarge].Model != "src-large" {
		t.Errorf("Large model = %q, want %q", dst.Models[SelectedModelTypeLarge].Model, "src-large")
	}

	// Small model should remain.
	if dst.Models[SelectedModelTypeSmall].Model != "dst-small" {
		t.Errorf("Small model = %q, want %q", dst.Models[SelectedModelTypeSmall].Model, "dst-small")
	}

	// Both providers should exist.
	if dst.Providers["openai"] == nil {
		t.Error("Provider 'openai' missing after merge")
	}
	if dst.Providers["anthropic"] == nil {
		t.Error("Provider 'anthropic' missing after merge")
	}

	// Context paths should be from src.
	if len(dst.Options.ContextPaths) != 1 || dst.Options.ContextPaths[0] != "SRC.md" {
		t.Errorf("ContextPaths = %v, want [SRC.md]", dst.Options.ContextPaths)
	}

	// DataDir should remain from dst (src was empty).
	if dst.Options.DataDir != "/dst/data" {
		t.Errorf("DataDir = %q, want %q", dst.Options.DataDir, "/dst/data")
	}

	// Debug should be true (from src).
	if !dst.Options.Debug {
		t.Error("Debug = false, want true")
	}
}

func TestMergeConfig_NilOptions(t *testing.T) {
	dst := NewConfig()
	dst.Options = nil

	src := NewConfig()
	src.Options = &Options{Debug: true}

	mergeConfig(dst, src)

	if dst.Options == nil {
		t.Fatal("Options is nil after merge")
	}
	if !dst.Options.Debug {
		t.Error("Debug = false, want true")
	}
}

func TestMergeConfig_SrcNilOptions(t *testing.T) {
	dst := NewConfig()
	dst.Options = &Options{Debug: true}

	src := NewConfig()
	src.Options = nil

	mergeConfig(dst, src)

	// Should remain unchanged.
	if !dst.Options.Debug {
		t.Error("Debug = false, want true")
	}
}

func TestConfigureProviders(t *testing.T) {
	t.Setenv("TEST_API_KEY", "resolved-key")

	cfg := NewConfig()
	cfg.Providers[testProviderID] = &ProviderConfig{
		APIKey: "$TEST_API_KEY",
	}

	providers := []catwalk.Provider{
		{
			ID:          testProviderID,
			Name:        "OpenAI",
			Type:        catwalk.TypeOpenAI,
			APIEndpoint: "https://api.openai.com/v1",
			Models: []catwalk.Model{
				{ID: testModelGPT4o},
			},
		},
	}
	cfg.SetKnownProviders(providers)

	resolver := NewResolver()
	configureProviders(cfg, resolver)

	provider := cfg.Providers[testProviderID]
	if provider == nil {
		t.Fatal("Provider 'openai' is nil")
	}

	// API key should be resolved.
	if provider.APIKey != "resolved-key" {
		t.Errorf("APIKey = %q, want %q", provider.APIKey, "resolved-key")
	}

	// ID should be set from catwalk.
	if provider.ID != testProviderID {
		t.Errorf("ID = %q, want %q", provider.ID, testProviderID)
	}

	// Name should be set from catwalk.
	if provider.Name != "OpenAI" {
		t.Errorf("Name = %q, want %q", provider.Name, "OpenAI")
	}

	// Type should be set from catwalk.
	if provider.Type != catwalk.TypeOpenAI {
		t.Errorf("Type = %q, want %q", provider.Type, catwalk.TypeOpenAI)
	}

	// BaseURL should be set from catwalk.
	if provider.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("BaseURL = %q, want %q", provider.BaseURL, "https://api.openai.com/v1")
	}

	// Models should be set from catwalk.
	if len(provider.Models) != 1 {
		t.Errorf("Models length = %d, want 1", len(provider.Models))
	}

	// ExtraHeaders should be initialized.
	if provider.ExtraHeaders == nil {
		t.Error("ExtraHeaders is nil")
	}
}

func TestConfigureProviders_UnresolvedAPIKey(t *testing.T) {
	cfg := NewConfig()
	cfg.Providers[testProviderID] = &ProviderConfig{
		APIKey: "$UNDEFINED_API_KEY",
	}

	providers := []catwalk.Provider{
		{ID: testProviderID, Name: "OpenAI"},
	}
	cfg.SetKnownProviders(providers)

	resolver := NewResolver()
	configureProviders(cfg, resolver)

	// Provider should be removed due to unresolved API key.
	if cfg.Providers[testProviderID] != nil {
		t.Error("Provider 'openai' should be removed due to unresolved API key")
	}
}

func TestConfigureProviders_CustomBaseURL(t *testing.T) {
	t.Setenv("TEST_KEY", "key")
	t.Setenv("CUSTOM_URL", "https://custom.api.com")

	cfg := NewConfig()
	cfg.Providers[testProviderID] = &ProviderConfig{
		APIKey:  "$TEST_KEY",
		BaseURL: "$CUSTOM_URL",
	}

	providers := []catwalk.Provider{
		{ID: testProviderID, APIEndpoint: "https://default.api.com"},
	}
	cfg.SetKnownProviders(providers)

	resolver := NewResolver()
	configureProviders(cfg, resolver)

	// Custom URL should be preserved.
	if cfg.Providers[testProviderID].BaseURL != "https://custom.api.com" {
		t.Errorf("BaseURL = %q, want %q", cfg.Providers[testProviderID].BaseURL, "https://custom.api.com")
	}
}

func TestConfigureProviders_MergeModels(t *testing.T) {
	t.Setenv("TEST_KEY", "key")

	cfg := NewConfig()
	cfg.Providers[testProviderID] = &ProviderConfig{
		APIKey: "$TEST_KEY",
		Models: []catwalk.Model{
			{ID: "custom-model", Name: "Custom"},
		},
	}

	providers := []catwalk.Provider{
		{
			ID: testProviderID,
			Models: []catwalk.Model{
				{ID: testModelGPT4o, Name: "GPT-4o"},
				{ID: "custom-model", Name: "Default Custom"}, // Same ID, should not duplicate.
			},
		},
	}
	cfg.SetKnownProviders(providers)

	resolver := NewResolver()
	configureProviders(cfg, resolver)

	// Should have 2 models (custom + gpt-4o, no duplicate).
	if len(cfg.Providers[testProviderID].Models) != 2 {
		t.Errorf("Models length = %d, want 2", len(cfg.Providers[testProviderID].Models))
	}

	// Custom model should keep user's name.
	found := false
	for _, m := range cfg.Providers[testProviderID].Models {
		if m.ID == "custom-model" && m.Name == "Custom" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Custom model not preserved with user's name")
	}
}

func TestConfigureDefaultModels(t *testing.T) {
	cfg := NewConfig()
	cfg.Providers["anthropic"] = &ProviderConfig{
		ID:     "anthropic",
		APIKey: "test-key",
	}

	providers := []catwalk.Provider{
		{
			ID:                  "anthropic",
			DefaultLargeModelID: "claude-opus-4",
			DefaultSmallModelID: "claude-haiku",
		},
	}
	cfg.SetKnownProviders(providers)

	err := configureDefaultModels(cfg)
	if err != nil {
		t.Fatalf("configureDefaultModels() error = %v", err)
	}

	// Should have default models set.
	if cfg.Models[SelectedModelTypeLarge].Model != "claude-opus-4" {
		t.Errorf("Large model = %q, want %q", cfg.Models[SelectedModelTypeLarge].Model, "claude-opus-4")
	}
	if cfg.Models[SelectedModelTypeSmall].Model != "claude-haiku" {
		t.Errorf("Small model = %q, want %q", cfg.Models[SelectedModelTypeSmall].Model, "claude-haiku")
	}
}

func TestConfigureDefaultModels_WithExistingModels(t *testing.T) {
	cfg := NewConfig()
	cfg.Models[SelectedModelTypeLarge] = SelectedModel{Model: "existing", Provider: "test"}
	cfg.Providers["test"] = &ProviderConfig{ID: "test"}

	providers := []catwalk.Provider{
		{ID: "test", DefaultLargeModelID: "default"},
	}
	cfg.SetKnownProviders(providers)

	err := configureDefaultModels(cfg)
	if err != nil {
		t.Fatalf("configureDefaultModels() error = %v", err)
	}

	// Should keep existing model.
	if cfg.Models[SelectedModelTypeLarge].Model != "existing" {
		t.Errorf("Large model = %q, want %q", cfg.Models[SelectedModelTypeLarge].Model, "existing")
	}
}

func TestConfigureDefaultModels_NoValidProviders(t *testing.T) {
	cfg := NewConfig()
	// No providers configured.

	providers := []catwalk.Provider{
		{ID: testProviderID, DefaultLargeModelID: testModelGPT4o},
	}
	cfg.SetKnownProviders(providers)

	err := configureDefaultModels(cfg)
	if err == nil {
		t.Error("configureDefaultModels() expected error when no valid providers")
	}
}

func TestConfigureDefaultModels_DisabledProvider(t *testing.T) {
	cfg := NewConfig()
	cfg.Providers[testProviderID] = &ProviderConfig{
		ID:      testProviderID,
		APIKey:  "key",
		Disable: true,
	}

	providers := []catwalk.Provider{
		{ID: testProviderID, DefaultLargeModelID: testModelGPT4o},
	}
	cfg.SetKnownProviders(providers)

	err := configureDefaultModels(cfg)
	if err == nil {
		t.Error("configureDefaultModels() expected error when provider is disabled")
	}
}

func TestConfigureDefaultModels_NoAPIKey(t *testing.T) {
	cfg := NewConfig()
	cfg.Providers[testProviderID] = &ProviderConfig{
		ID:     testProviderID,
		APIKey: "", // No API key.
	}

	providers := []catwalk.Provider{
		{ID: testProviderID, DefaultLargeModelID: testModelGPT4o},
	}
	cfg.SetKnownProviders(providers)

	err := configureDefaultModels(cfg)
	if err == nil {
		t.Error("configureDefaultModels() expected error when no API key")
	}
}

func TestValidateModels(t *testing.T) {
	cfg := NewConfig()
	cfg.Providers[testProviderID] = &ProviderConfig{ID: testProviderID}
	cfg.Models[SelectedModelTypeLarge] = SelectedModel{Model: testModelGPT4o, Provider: testProviderID}

	err := validateModels(cfg)
	if err != nil {
		t.Errorf("validateModels() error = %v", err)
	}
}

func TestValidateModels_UnknownProvider(t *testing.T) {
	cfg := NewConfig()
	cfg.Models[SelectedModelTypeLarge] = SelectedModel{Model: testModelGPT4o, Provider: "unknown"}

	err := validateModels(cfg)
	if err == nil {
		t.Error("validateModels() expected error for unknown provider")
	}
}

func TestValidateModels_DisabledProvider(t *testing.T) {
	cfg := NewConfig()
	cfg.Providers[testProviderID] = &ProviderConfig{ID: testProviderID, Disable: true}
	cfg.Models[SelectedModelTypeLarge] = SelectedModel{Model: testModelGPT4o, Provider: testProviderID}

	err := validateModels(cfg)
	if err == nil {
		t.Error("validateModels() expected error for disabled provider")
	}
}

func TestApplyDefaults(t *testing.T) {
	cfg := NewConfig()
	cfg.Options = nil

	applyDefaults(cfg)

	if cfg.Options == nil {
		t.Fatal("Options is nil after applyDefaults")
	}
	if cfg.Options.DataDir == "" {
		t.Error("DataDir is empty after applyDefaults")
	}
}

func TestApplyDefaults_PreservesExisting(t *testing.T) {
	cfg := NewConfig()
	cfg.Options = &Options{DataDir: "/custom/path"}

	applyDefaults(cfg)

	if cfg.Options.DataDir != "/custom/path" {
		t.Errorf("DataDir = %q, want %q", cfg.Options.DataDir, "/custom/path")
	}
}

func TestGlobalConfigPath(t *testing.T) {
	path := GlobalConfigPath()
	if path == "" {
		t.Error("GlobalConfigPath() returned empty string")
	}
	if filepath.Base(path) != "cdd.json" {
		t.Errorf("GlobalConfigPath() base = %q, want %q", filepath.Base(path), "cdd.json")
	}
}

func TestFindProjectConfig(t *testing.T) {
	// Create temp directory structure.
	tempDir := t.TempDir()
	subDir := filepath.Join(tempDir, "sub", "dir")
	//nolint:gosec // Test directory, permissions not critical.
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatalf("Failed to create subdirectory: %v", err)
	}

	// Create config file in parent.
	configPath := filepath.Join(tempDir, "cdd.json")
	//nolint:gosec // Test file, permissions not critical.
	if err := os.WriteFile(configPath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Change to subdirectory.
	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Warning: failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(subDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	// Should find config in parent.
	found := findProjectConfig()
	if found != configPath {
		t.Errorf("findProjectConfig() = %q, want %q", found, configPath)
	}
}

func TestFindProjectConfig_Hidden(t *testing.T) {
	tempDir := t.TempDir()

	// Create hidden config file.
	configPath := filepath.Join(tempDir, ".cdd.json")
	//nolint:gosec // Test file, permissions not critical.
	if err := os.WriteFile(configPath, []byte("{}"), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Warning: failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	found := findProjectConfig()
	if found != configPath {
		t.Errorf("findProjectConfig() = %q, want %q", found, configPath)
	}
}

func TestFindProjectConfig_NotFound(t *testing.T) {
	tempDir := t.TempDir()

	oldWd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Failed to get working directory: %v", err)
	}
	defer func() {
		if err := os.Chdir(oldWd); err != nil {
			t.Logf("Warning: failed to restore working directory: %v", err)
		}
	}()

	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Failed to change directory: %v", err)
	}

	found := findProjectConfig()
	if found != "" {
		t.Errorf("findProjectConfig() = %q, want empty string", found)
	}
}

func TestLoadFromFile(t *testing.T) {
	tempDir := t.TempDir()

	// Override GlobalConfigPath for testing to avoid writing to real config
	// when MigrateToConnections is called.
	globalConfigPath := filepath.Join(tempDir, "cdd.json")
	SetGlobalConfigPath(globalConfigPath)
	defer SetGlobalConfigPath("") // Reset after test

	// Set CATWALK_URL to invalid to use embedded fallback.
	t.Setenv("CATWALK_URL", "http://invalid.invalid.invalid")
	t.Setenv("TEST_API_KEY", "sk-test-key")

	configPath := filepath.Join(tempDir, "config.json")
	configContent := `{
		"providers": {
			"openai": {
				"api_key": "$TEST_API_KEY",
				"type": "openai"
			}
		},
		"models": {
			"large": {"model": "gpt-4o", "provider": "openai"},
			"small": {"model": "gpt-4o-mini", "provider": "openai"}
		},
		"options": {
			"data_directory": "` + tempDir + `"
		}
	}`
	//nolint:gosec // Test file, permissions not critical.
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	cfg, err := LoadFromFile(configPath)
	if err != nil {
		t.Fatalf("LoadFromFile() error = %v", err)
	}

	if cfg.Providers[testProviderID] == nil {
		t.Fatal("Provider 'openai' not loaded")
	}
	if cfg.Providers[testProviderID].APIKey != "sk-test-key" {
		t.Errorf("APIKey not resolved, got %q", cfg.Providers[testProviderID].APIKey)
	}
}

func TestLoadFromFile_NonExistent(t *testing.T) {
	_, err := LoadFromFile("/non/existent/path.json")
	if err == nil {
		t.Error("LoadFromFile() expected error for non-existent file")
	}
}

func TestLoadFromFile_InvalidJSON(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, "invalid.json")

	//nolint:gosec // Test file, permissions not critical.
	if err := os.WriteFile(configPath, []byte("not json"), 0o644); err != nil {
		t.Fatalf("Failed to write file: %v", err)
	}

	_, err := LoadFromFile(configPath)
	if err == nil {
		t.Error("LoadFromFile() expected error for invalid JSON")
	}
}

func TestMergeConfig_NilDstOptions(t *testing.T) {
	dst := NewConfig()
	dst.Options = nil // Explicitly set to nil.

	src := NewConfig()
	src.Options = &Options{
		DataDir: "/custom/path",
		Debug:   true,
	}

	mergeConfig(dst, src)

	if dst.Options == nil {
		t.Fatal("dst.Options should be initialized after merge")
	}
	if dst.Options.DataDir != "/custom/path" {
		t.Errorf("DataDir = %q, want %q", dst.Options.DataDir, "/custom/path")
	}
	if !dst.Options.Debug {
		t.Error("Debug should be true")
	}
}

func TestConfigureProviders_BaseURLResolutionFails(t *testing.T) {
	cfg := NewConfig()

	// Provider with unresolvable base URL.
	cfg.Providers["test"] = &ProviderConfig{
		ID:      "test",
		APIKey:  "sk-test",
		BaseURL: "$UNDEFINED_BASE_URL",
		Type:    catwalk.TypeOpenAI,
	}

	// Set known providers with matching ID.
	cfg.SetKnownProviders([]catwalk.Provider{
		{ID: "test", Name: "Test Provider", APIEndpoint: "https://default.api.com"},
	})

	resolver := NewResolver()
	configureProviders(cfg, resolver)

	// Provider should still exist, base URL should fall back to default or be left as-is.
	if cfg.Providers["test"] == nil {
		t.Fatal("Provider should still exist")
	}
}

func TestConfigureProviders_WithUserModels(t *testing.T) {
	cfg := NewConfig()

	// Provider with user-defined models.
	cfg.Providers["test"] = &ProviderConfig{
		ID:     "test",
		APIKey: "sk-test",
		Type:   catwalk.TypeOpenAI,
		Models: []catwalk.Model{
			{ID: "custom-model", Name: "Custom Model"},
		},
	}

	// Set known providers with some default models.
	cfg.SetKnownProviders([]catwalk.Provider{
		{
			ID:   "test",
			Name: "Test Provider",
			Models: []catwalk.Model{
				{ID: "default-model", Name: "Default Model"},
				{ID: "custom-model", Name: "Default Custom"}, // Same ID as user model.
			},
		},
	})

	resolver := NewResolver()
	configureProviders(cfg, resolver)

	// Should have both custom-model and default-model, but not duplicate custom-model.
	if len(cfg.Providers["test"].Models) != 2 {
		t.Errorf("Expected 2 models, got %d", len(cfg.Providers["test"].Models))
	}
}

func TestConfigureDefaultModels_ProviderWithOnlyLargeModel(t *testing.T) {
	cfg := NewConfig()

	cfg.Providers["test"] = &ProviderConfig{
		ID:     "test",
		APIKey: "sk-test",
	}

	cfg.SetKnownProviders([]catwalk.Provider{
		{ID: "test", DefaultLargeModelID: "large-model", DefaultSmallModelID: ""},
	})

	err := configureDefaultModels(cfg)
	if err != nil {
		t.Fatalf("configureDefaultModels() error = %v", err)
	}

	if _, ok := cfg.Models[SelectedModelTypeLarge]; !ok {
		t.Error("Large model should be configured")
	}
}
