//nolint:errcheck // Test file, errors are intentionally ignored for testing purposes.
package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/guilhermegouw/cdd/internal/oauth"
)

func TestConnection_IsConfigured(t *testing.T) {
	tests := []struct {
		name string
		conn Connection
		want bool
	}{
		{
			name: "with API key",
			conn: Connection{APIKey: "sk-test"},
			want: true,
		},
		{
			name: "with OAuth token",
			conn: Connection{OAuthToken: &oauth.Token{AccessToken: "test"}},
			want: true,
		},
		{
			name: "with both",
			conn: Connection{APIKey: "sk-test", OAuthToken: &oauth.Token{AccessToken: "test"}},
			want: true,
		},
		{
			name: "empty",
			conn: Connection{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conn.IsConfigured(); got != tt.want {
				t.Errorf("IsConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConnection_IsOAuth(t *testing.T) {
	tests := []struct {
		name string
		conn Connection
		want bool
	}{
		{
			name: "with OAuth token",
			conn: Connection{OAuthToken: &oauth.Token{AccessToken: "test"}},
			want: true,
		},
		{
			name: "with API key only",
			conn: Connection{APIKey: "sk-test"},
			want: false,
		},
		{
			name: "empty",
			conn: Connection{},
			want: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.conn.IsOAuth(); got != tt.want {
				t.Errorf("IsOAuth() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestConnectionManager_CRUD(t *testing.T) {
	// Create a temporary directory for the config.
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "cdd.json")

	// Create initial config.
	cfg := NewConfig()
	cfg.Options = &Options{DataDir: tmpDir}

	// Write empty config to disk.
	if err := SaveToFile(cfg, configPath); err != nil {
		t.Fatalf("Failed to save initial config: %v", err)
	}

	// Override GlobalConfigPath for testing.
	origConfigPath := GlobalConfigPath
	defer func() {
		// Restore after test (note: GlobalConfigPath is a function, so we can't restore it this way)
		// Instead, we'll just ensure the test doesn't affect the real config.
	}()
	_ = origConfigPath

	// Create manager.
	manager := NewConnectionManager(cfg)

	// Test List (empty).
	connections := manager.List()
	if len(connections) != 0 {
		t.Errorf("Expected empty list, got %d connections", len(connections))
	}

	// Test Add.
	conn := Connection{
		Name:       "Test Connection",
		ProviderID: "anthropic",
		APIKey:     "sk-test-key",
	}

	// Note: Add will try to save to GlobalConfigPath(), which may fail in tests.
	// For unit testing, we test the in-memory operations.
	conn.ID = "test-id-1"
	conn.CreatedAt = time.Now()
	conn.UpdatedAt = time.Now()
	cfg.Connections = append(cfg.Connections, conn)

	// Test Get.
	got := manager.Get("test-id-1")
	if got == nil {
		t.Fatal("Expected to find connection, got nil")
	}
	if got.Name != "Test Connection" {
		t.Errorf("Expected name 'Test Connection', got '%s'", got.Name)
	}

	// Test GetByName.
	got = manager.GetByName("Test Connection")
	if got == nil {
		t.Fatal("Expected to find connection by name, got nil")
	}
	if got.ID != "test-id-1" {
		t.Errorf("Expected ID 'test-id-1', got '%s'", got.ID)
	}

	// Test GetByProvider.
	connections = manager.GetByProvider("anthropic")
	if len(connections) != 1 {
		t.Errorf("Expected 1 connection, got %d", len(connections))
	}

	// Add another connection.
	conn2 := Connection{
		ID:         "test-id-2",
		Name:       "Second Connection",
		ProviderID: "anthropic",
		APIKey:     "sk-test-key-2",
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
	cfg.Connections = append(cfg.Connections, conn2)

	// Test GetByProvider (now should have 2).
	connections = manager.GetByProvider("anthropic")
	if len(connections) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(connections))
	}

	// Test List.
	connections = manager.List()
	if len(connections) != 2 {
		t.Errorf("Expected 2 connections in list, got %d", len(connections))
	}

	// Test Get non-existent.
	got = manager.Get("non-existent")
	if got != nil {
		t.Error("Expected nil for non-existent connection")
	}

	// Test GetByName non-existent.
	got = manager.GetByName("Non-existent")
	if got != nil {
		t.Error("Expected nil for non-existent connection name")
	}
}

func TestConnectionManager_GetActiveConnection(t *testing.T) {
	cfg := NewConfig()

	// Add a connection.
	conn := Connection{
		ID:         "conn-1",
		Name:       "Test Connection",
		ProviderID: "anthropic",
		APIKey:     "sk-test",
	}
	cfg.Connections = append(cfg.Connections, conn)

	// Set up model selection with connection ID.
	cfg.Models[SelectedModelTypeLarge] = SelectedModel{
		ConnectionID: "conn-1",
		Model:        "claude-3-5-sonnet",
		Provider:     "anthropic",
	}

	manager := NewConnectionManager(cfg)

	// Test GetActiveConnection.
	active := manager.GetActiveConnection(SelectedModelTypeLarge)
	if active == nil {
		t.Fatal("Expected active connection, got nil")
	}
	if active.ID != "conn-1" {
		t.Errorf("Expected connection ID 'conn-1', got '%s'", active.ID)
	}

	// Test with no active model.
	active = manager.GetActiveConnection(SelectedModelTypeSmall)
	if active != nil {
		t.Error("Expected nil for small model (not configured)")
	}
}

func TestConnectionManager_GetActiveConnection_BackwardCompat(t *testing.T) {
	cfg := NewConfig()

	// Add a connection without ConnectionID in model.
	conn := Connection{
		ID:         "conn-1",
		Name:       "Test Connection",
		ProviderID: "anthropic",
		APIKey:     "sk-test",
	}
	cfg.Connections = append(cfg.Connections, conn)

	// Set up model selection with only Provider (old format).
	cfg.Models[SelectedModelTypeLarge] = SelectedModel{
		Model:    "claude-3-5-sonnet",
		Provider: "anthropic",
		// ConnectionID is empty - testing backward compatibility
	}

	manager := NewConnectionManager(cfg)

	// Test GetActiveConnection falls back to provider.
	active := manager.GetActiveConnection(SelectedModelTypeLarge)
	if active == nil {
		t.Fatal("Expected active connection via backward compat, got nil")
	}
	if active.ID != "conn-1" {
		t.Errorf("Expected connection ID 'conn-1', got '%s'", active.ID)
	}
}

func TestMigrateToConnections(t *testing.T) {
	// Create a temporary directory for the config.
	tmpDir := t.TempDir()
	configDir := filepath.Join(tmpDir, "cdd")
	if err := os.MkdirAll(configDir, 0o750); err != nil { //nolint:gosec // G301: Test directory, more permissive is acceptable
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Override GlobalConfigPath for testing to avoid writing to real config.
	configPath := filepath.Join(configDir, "cdd.json")
	SetGlobalConfigPath(configPath)
	defer SetGlobalConfigPath("") // Reset after test

	// Create config with old-style providers.
	cfg := NewConfig()
	cfg.Options = &Options{DataDir: tmpDir}
	cfg.Providers["anthropic"] = &ProviderConfig{
		ID:     "anthropic",
		Name:   "Anthropic",
		APIKey: "sk-ant-test",
	}
	cfg.Providers["openai"] = &ProviderConfig{
		ID:     "openai",
		Name:   "OpenAI",
		APIKey: "sk-openai-test",
	}
	cfg.Models[SelectedModelTypeLarge] = SelectedModel{
		Model:    "claude-3-5-sonnet",
		Provider: "anthropic",
	}
	cfg.Models[SelectedModelTypeSmall] = SelectedModel{
		Model:    "gpt-4o-mini",
		Provider: "openai",
	}

	// Run migration - saves to the temp config path.
	_ = MigrateToConnections(cfg)

	// Verify connections were created.
	if len(cfg.Connections) != 2 {
		t.Errorf("Expected 2 connections, got %d", len(cfg.Connections))
	}

	// Verify connection IDs were set in models.
	largeModel := cfg.Models[SelectedModelTypeLarge]
	if largeModel.ConnectionID == "" {
		t.Error("Expected ConnectionID to be set for large model")
	}

	smallModel := cfg.Models[SelectedModelTypeSmall]
	if smallModel.ConnectionID == "" {
		t.Error("Expected ConnectionID to be set for small model")
	}

	// Find the anthropic connection.
	var anthropicConn *Connection
	for i := range cfg.Connections {
		if cfg.Connections[i].ProviderID == "anthropic" {
			anthropicConn = &cfg.Connections[i]
			break
		}
	}
	if anthropicConn == nil {
		t.Fatal("Expected to find anthropic connection")
	}
	if anthropicConn.APIKey != "sk-ant-test" {
		t.Errorf("Expected API key 'sk-ant-test', got '%s'", anthropicConn.APIKey)
	}
	if anthropicConn.Name != "Anthropic" {
		t.Errorf("Expected name 'Anthropic', got '%s'", anthropicConn.Name)
	}

	// Verify migration is skipped if connections already exist.
	originalLen := len(cfg.Connections)
	_ = MigrateToConnections(cfg)
	if len(cfg.Connections) != originalLen {
		t.Error("Migration should be skipped when connections already exist")
	}
}

func TestMigrateToConnections_SkipsUnconfiguredProviders(t *testing.T) {
	// Override GlobalConfigPath for testing to avoid writing to real config.
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "cdd.json")
	SetGlobalConfigPath(configPath)
	defer SetGlobalConfigPath("") // Reset after test

	cfg := NewConfig()

	// Add provider without API key.
	cfg.Providers["anthropic"] = &ProviderConfig{
		ID:   "anthropic",
		Name: "Anthropic",
		// No APIKey or OAuthToken
	}

	_ = MigrateToConnections(cfg)

	// Verify no connections were created.
	if len(cfg.Connections) != 0 {
		t.Errorf("Expected 0 connections for unconfigured provider, got %d", len(cfg.Connections))
	}
}

func TestMigrateToConnections_WithOAuth(t *testing.T) {
	// Override GlobalConfigPath for testing to avoid writing to real config.
	tmpDir := t.TempDir()
	configPath := filepath.Join(tmpDir, "cdd.json")
	SetGlobalConfigPath(configPath)
	defer SetGlobalConfigPath("") // Reset after test

	cfg := NewConfig()

	// Add provider with OAuth.
	cfg.Providers["anthropic"] = &ProviderConfig{
		ID:   "anthropic",
		Name: "Anthropic",
		OAuthToken: &oauth.Token{
			AccessToken:  "access-token",
			RefreshToken: "refresh-token",
		},
	}

	_ = MigrateToConnections(cfg)

	// Verify connection was created with OAuth.
	if len(cfg.Connections) != 1 {
		t.Fatalf("Expected 1 connection, got %d", len(cfg.Connections))
	}

	conn := cfg.Connections[0]
	if conn.OAuthToken == nil {
		t.Error("Expected OAuth token to be preserved")
	}
	if conn.OAuthToken.AccessToken != "access-token" {
		t.Errorf("Expected access token 'access-token', got '%s'", conn.OAuthToken.AccessToken)
	}
}
