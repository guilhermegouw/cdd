//nolint:goconst,gosec // Test file uses repeated string literals and reads test files.
package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/guilhermegouw/cdd/internal/oauth"
)

func TestSaveToFile(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := NewConfig()
	cfg.Providers["anthropic"] = &ProviderConfig{
		ID:     "anthropic",
		APIKey: "$ANTHROPIC_API_KEY",
	}
	cfg.Models[SelectedModelTypeLarge] = SelectedModel{
		Model:    "claude-3-opus",
		Provider: "anthropic",
	}
	cfg.Models[SelectedModelTypeSmall] = SelectedModel{
		Model:    "claude-3-haiku",
		Provider: "anthropic",
	}

	err := SaveToFile(cfg, configPath)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	// Verify file exists.
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		t.Fatal("SaveToFile() did not create config file")
	}

	// Read and parse the file.
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var saved SaveConfig
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("Failed to parse config file: %v", err)
	}

	// Verify contents.
	if saved.Providers["anthropic"] == nil {
		t.Error("Provider 'anthropic' not saved")
	} else if saved.Providers["anthropic"].APIKey != "$ANTHROPIC_API_KEY" {
		t.Errorf("APIKey = %q, want %q", saved.Providers["anthropic"].APIKey, "$ANTHROPIC_API_KEY")
	}

	if saved.Models[SelectedModelTypeLarge].Model != "claude-3-opus" {
		t.Errorf("Large model = %q, want %q", saved.Models[SelectedModelTypeLarge].Model, "claude-3-opus")
	}
	if saved.Models[SelectedModelTypeSmall].Model != "claude-3-haiku" {
		t.Errorf("Small model = %q, want %q", saved.Models[SelectedModelTypeSmall].Model, "claude-3-haiku")
	}
}

func TestSaveToFile_CreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	nestedPath := filepath.Join(tmpDir, "nested", "dir", "config.json")

	cfg := NewConfig()
	err := SaveToFile(cfg, nestedPath)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	// Verify nested directory was created.
	if _, err := os.Stat(filepath.Dir(nestedPath)); os.IsNotExist(err) {
		t.Error("SaveToFile() did not create nested directories")
	}
}

func TestSaveToFile_OnlySavesProvidersWithAPIKeyOrOAuth(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := NewConfig()
	cfg.Providers["with-key"] = &ProviderConfig{
		ID:     "with-key",
		APIKey: "test-key",
	}
	cfg.Providers["with-oauth"] = &ProviderConfig{
		ID:         "with-oauth",
		OAuthToken: &oauth.Token{AccessToken: "oauth-token"},
	}
	cfg.Providers["empty"] = &ProviderConfig{
		ID: "empty",
	}

	err := SaveToFile(cfg, configPath)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var saved SaveConfig
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("Failed to parse config file: %v", err)
	}

	if saved.Providers["with-key"] == nil {
		t.Error("Provider 'with-key' should be saved")
	}
	if saved.Providers["with-oauth"] == nil {
		t.Error("Provider 'with-oauth' should be saved")
	}
	if saved.Providers["empty"] != nil {
		t.Error("Provider 'empty' should not be saved")
	}
}

func TestSaveWizardResult(t *testing.T) {
	// Override global config path for testing.
	tmpDir := t.TempDir()
	originalConfigDir := os.Getenv("XDG_CONFIG_HOME")
	t.Setenv("XDG_CONFIG_HOME", tmpDir)
	defer func() {
		if originalConfigDir != "" {
			_ = os.Setenv("XDG_CONFIG_HOME", originalConfigDir) //nolint:errcheck // Test cleanup.
		}
	}()

	err := SaveWizardResult("openai", "$OPENAI_API_KEY", "gpt-4o", "gpt-4o-mini")
	if err != nil {
		t.Fatalf("SaveWizardResult() error = %v", err)
	}

	// Verify file was created.
	configPath := GlobalConfigPath()
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		t.Fatal("SaveWizardResult() did not create config file")
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var saved SaveConfig
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("Failed to parse config file: %v", err)
	}

	// Verify connection was created.
	if len(saved.Connections) == 0 {
		t.Fatal("No connections saved")
	}
	conn := saved.Connections[0]
	if conn.ProviderID != "openai" {
		t.Errorf("Connection.ProviderID = %q, want %q", conn.ProviderID, "openai")
	}
	if conn.APIKey != "$OPENAI_API_KEY" {
		t.Errorf("Connection.APIKey = %q, want %q", conn.APIKey, "$OPENAI_API_KEY")
	}

	// Verify models have ConnectionID set.
	if saved.Models[SelectedModelTypeLarge].Model != "gpt-4o" {
		t.Errorf("Large model = %q, want %q", saved.Models[SelectedModelTypeLarge].Model, "gpt-4o")
	}
	if saved.Models[SelectedModelTypeLarge].ConnectionID == "" {
		t.Error("Large model should have ConnectionID set")
	}
	if saved.Models[SelectedModelTypeLarge].Provider != "openai" {
		t.Errorf("Large model provider = %q, want %q", saved.Models[SelectedModelTypeLarge].Provider, "openai")
	}
	if saved.Models[SelectedModelTypeSmall].Model != "gpt-4o-mini" {
		t.Errorf("Small model = %q, want %q", saved.Models[SelectedModelTypeSmall].Model, "gpt-4o-mini")
	}
	if saved.Models[SelectedModelTypeSmall].ConnectionID == "" {
		t.Error("Small model should have ConnectionID set")
	}
}

func TestSaveWizardResultWithOAuth(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tmpDir)

	token := &oauth.Token{
		AccessToken:  "access-token-123",
		RefreshToken: "refresh-token-456",
		ExpiresIn:    3600,
		ExpiresAt:    1700000000,
	}

	err := SaveWizardResultWithOAuth("anthropic", token, "claude-opus-4", "claude-haiku-3")
	if err != nil {
		t.Fatalf("SaveWizardResultWithOAuth() error = %v", err)
	}

	configPath := GlobalConfigPath()
	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	var saved SaveConfig
	if err := json.Unmarshal(data, &saved); err != nil {
		t.Fatalf("Failed to parse config file: %v", err)
	}

	// Verify connection with OAuth token is saved.
	if len(saved.Connections) == 0 {
		t.Fatal("No connections saved")
	}
	conn := saved.Connections[0]
	if conn.ProviderID != "anthropic" {
		t.Errorf("Connection.ProviderID = %q, want %q", conn.ProviderID, "anthropic")
	}
	if conn.OAuthToken == nil {
		t.Fatal("Connection.OAuthToken not saved")
	}
	if conn.OAuthToken.AccessToken != "access-token-123" {
		t.Errorf("AccessToken = %q, want %q", conn.OAuthToken.AccessToken, "access-token-123")
	}
	if conn.OAuthToken.RefreshToken != "refresh-token-456" {
		t.Errorf("RefreshToken = %q, want %q", conn.OAuthToken.RefreshToken, "refresh-token-456")
	}

	// Verify models have ConnectionID set.
	if saved.Models[SelectedModelTypeLarge].Model != "claude-opus-4" {
		t.Errorf("Large model = %q, want %q", saved.Models[SelectedModelTypeLarge].Model, "claude-opus-4")
	}
	if saved.Models[SelectedModelTypeLarge].ConnectionID == "" {
		t.Error("Large model should have ConnectionID set")
	}
	if saved.Models[SelectedModelTypeSmall].Model != "claude-haiku-3" {
		t.Errorf("Small model = %q, want %q", saved.Models[SelectedModelTypeSmall].Model, "claude-haiku-3")
	}
	if saved.Models[SelectedModelTypeSmall].ConnectionID == "" {
		t.Error("Small model should have ConnectionID set")
	}
}

func TestSaveConfig_JSONFormat(t *testing.T) {
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "config.json")

	cfg := NewConfig()
	cfg.Providers["test"] = &ProviderConfig{
		ID:     "test",
		APIKey: "key",
	}

	err := SaveToFile(cfg, configPath)
	if err != nil {
		t.Fatalf("SaveToFile() error = %v", err)
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("Failed to read config file: %v", err)
	}

	// Verify it's properly indented JSON.
	if data[0] != '{' {
		t.Error("Config file should start with {")
	}

	// Check for indentation (should have spaces for pretty print).
	if len(data) > 10 && data[2] != ' ' {
		t.Error("Config file should be indented")
	}
}

func TestSaveProviderConfig_Fields(t *testing.T) {
	spc := SaveProviderConfig{
		APIKey: "test-key",
		OAuthToken: &oauth.Token{
			AccessToken: "oauth-access",
		},
	}

	if spc.APIKey != "test-key" {
		t.Errorf("APIKey = %q, want %q", spc.APIKey, "test-key")
	}
	if spc.OAuthToken.AccessToken != "oauth-access" {
		t.Errorf("OAuthToken.AccessToken = %q, want %q", spc.OAuthToken.AccessToken, "oauth-access")
	}
}

func TestSaveConfig_Fields(t *testing.T) {
	sc := SaveConfig{
		Models: map[SelectedModelType]SelectedModel{
			SelectedModelTypeLarge: {Model: "large-model"},
		},
		Providers: map[string]*SaveProviderConfig{
			"test": {APIKey: "key"},
		},
		Options: &Options{
			Debug: true,
		},
	}

	if sc.Models[SelectedModelTypeLarge].Model != "large-model" {
		t.Errorf("Models[large] = %q, want %q", sc.Models[SelectedModelTypeLarge].Model, "large-model")
	}
	if sc.Providers["test"].APIKey != "key" {
		t.Errorf("Providers[test].APIKey = %q, want %q", sc.Providers["test"].APIKey, "key")
	}
	if !sc.Options.Debug {
		t.Error("Options.Debug = false, want true")
	}
}
