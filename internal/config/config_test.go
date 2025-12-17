//nolint:goconst // Test file uses repeated string literals for clarity.
package config

import (
	"testing"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

func TestSelectedModelType_Constants(t *testing.T) {
	if SelectedModelTypeLarge != "large" {
		t.Errorf("SelectedModelTypeLarge = %q, want %q", SelectedModelTypeLarge, "large")
	}
	if SelectedModelTypeSmall != "small" {
		t.Errorf("SelectedModelTypeSmall = %q, want %q", SelectedModelTypeSmall, "small")
	}
}

func TestNewConfig(t *testing.T) {
	cfg := NewConfig()

	if cfg == nil {
		t.Fatal("NewConfig returned nil")
	}

	if cfg.Models == nil {
		t.Error("Models map is nil")
	}

	if cfg.Providers == nil {
		t.Error("Providers map is nil")
	}

	if cfg.Options == nil {
		t.Error("Options is nil")
	}

	// Verify maps are initialized and usable.
	cfg.Models[SelectedModelTypeLarge] = SelectedModel{Model: "test"}
	if cfg.Models[SelectedModelTypeLarge].Model != "test" {
		t.Error("Models map not usable")
	}

	cfg.Providers["test"] = &ProviderConfig{ID: "test"}
	if cfg.Providers["test"].ID != "test" {
		t.Error("Providers map not usable")
	}
}

func TestConfig_GetModel(t *testing.T) {
	cfg := NewConfig()

	// Setup provider with models.
	cfg.Providers["openai"] = &ProviderConfig{
		ID: "openai",
		Models: []catwalk.Model{
			{ID: "gpt-4o", Name: "GPT-4o"},
			{ID: "gpt-4o-mini", Name: "GPT-4o Mini"},
		},
	}
	cfg.Providers["anthropic"] = &ProviderConfig{
		ID: "anthropic",
		Models: []catwalk.Model{
			{ID: "claude-3-opus", Name: "Claude 3 Opus"},
		},
	}

	//nolint:govet // Test struct field order optimized for readability.
	tests := []struct {
		name       string
		providerID string
		modelID    string
		wantNil    bool
		wantName   string
	}{
		{
			name:       "existing model",
			providerID: "openai",
			modelID:    "gpt-4o",
			wantNil:    false,
			wantName:   "GPT-4o",
		},
		{
			name:       "another existing model",
			providerID: "openai",
			modelID:    "gpt-4o-mini",
			wantNil:    false,
			wantName:   "GPT-4o Mini",
		},
		{
			name:       "model from different provider",
			providerID: "anthropic",
			modelID:    "claude-3-opus",
			wantNil:    false,
			wantName:   "Claude 3 Opus",
		},
		{
			name:       "non-existent model",
			providerID: "openai",
			modelID:    "non-existent",
			wantNil:    true,
		},
		{
			name:       "non-existent provider",
			providerID: "non-existent",
			modelID:    "gpt-4o",
			wantNil:    true,
		},
		{
			name:       "empty provider ID",
			providerID: "",
			modelID:    "gpt-4o",
			wantNil:    true,
		},
		{
			name:       "empty model ID",
			providerID: "openai",
			modelID:    "",
			wantNil:    true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := cfg.GetModel(tt.providerID, tt.modelID)
			if tt.wantNil {
				if got != nil {
					t.Errorf("GetModel() = %v, want nil", got)
				}
			} else {
				if got == nil {
					t.Error("GetModel() = nil, want non-nil")
				} else if got.Name != tt.wantName {
					t.Errorf("GetModel().Name = %q, want %q", got.Name, tt.wantName)
				}
			}
		})
	}
}

func TestConfig_KnownProviders(t *testing.T) {
	cfg := NewConfig()

	// Initially empty.
	if len(cfg.KnownProviders()) != 0 {
		t.Errorf("KnownProviders() initially has %d items, want 0", len(cfg.KnownProviders()))
	}

	// Set providers.
	providers := []catwalk.Provider{
		{ID: "openai", Name: "OpenAI"},
		{ID: "anthropic", Name: "Anthropic"},
	}
	cfg.SetKnownProviders(providers)

	// Verify.
	got := cfg.KnownProviders()
	if len(got) != 2 {
		t.Errorf("KnownProviders() has %d items, want 2", len(got))
	}
	if got[0].ID != "openai" {
		t.Errorf("KnownProviders()[0].ID = %q, want %q", got[0].ID, "openai")
	}
	if got[1].ID != "anthropic" {
		t.Errorf("KnownProviders()[1].ID = %q, want %q", got[1].ID, "anthropic")
	}
}

func TestConfig_DataDir(t *testing.T) {
	tests := []struct {
		name    string
		options *Options
		want    string
	}{
		{
			name:    "nil options uses default",
			options: nil,
			want:    DefaultDataDir(),
		},
		{
			name:    "empty data dir uses default",
			options: &Options{DataDir: ""},
			want:    DefaultDataDir(),
		},
		{
			name:    "custom data dir",
			options: &Options{DataDir: "/custom/path"},
			want:    "/custom/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := NewConfig()
			cfg.Options = tt.options

			got := cfg.DataDir()
			if got != tt.want {
				t.Errorf("DataDir() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestConfig_Resolve(t *testing.T) {
	t.Setenv("TEST_API_KEY", "secret123")

	cfg := NewConfig()

	// Test successful resolution.
	got, err := cfg.Resolve("$TEST_API_KEY")
	if err != nil {
		t.Errorf("Resolve() error = %v", err)
	}
	if got != "secret123" {
		t.Errorf("Resolve() = %q, want %q", got, "secret123")
	}

	// Test failed resolution.
	_, err = cfg.Resolve("$UNDEFINED_VAR")
	if err == nil {
		t.Error("Resolve() expected error for undefined variable")
	}
}

func TestSelectedModel_Fields(t *testing.T) {
	temp := 0.7
	topP := 0.9
	topK := int64(40)

	model := SelectedModel{
		Model:           "gpt-4o",
		Provider:        "openai",
		ReasoningEffort: "high",
		Think:           true,
		MaxTokens:       4096,
		Temperature:     &temp,
		TopP:            &topP,
		TopK:            &topK,
		ProviderOptions: map[string]any{"custom": "value"},
	}

	if model.Model != "gpt-4o" {
		t.Errorf("Model = %q, want %q", model.Model, "gpt-4o")
	}
	if model.Provider != "openai" {
		t.Errorf("Provider = %q, want %q", model.Provider, "openai")
	}
	if model.ReasoningEffort != "high" {
		t.Errorf("ReasoningEffort = %q, want %q", model.ReasoningEffort, "high")
	}
	if !model.Think {
		t.Error("Think = false, want true")
	}
	if model.MaxTokens != 4096 {
		t.Errorf("MaxTokens = %d, want %d", model.MaxTokens, 4096)
	}
	if *model.Temperature != 0.7 {
		t.Errorf("Temperature = %f, want %f", *model.Temperature, 0.7)
	}
	if *model.TopP != 0.9 {
		t.Errorf("TopP = %f, want %f", *model.TopP, 0.9)
	}
	if *model.TopK != 40 {
		t.Errorf("TopK = %d, want %d", *model.TopK, 40)
	}
	if model.ProviderOptions["custom"] != "value" {
		t.Error("ProviderOptions not set correctly")
	}
}

func TestProviderConfig_Fields(t *testing.T) {
	provider := ProviderConfig{
		ID:              "openai",
		Name:            "OpenAI",
		Type:            catwalk.TypeOpenAI,
		BaseURL:         "https://api.openai.com/v1",
		APIKey:          "sk-test",
		Disable:         false,
		ExtraHeaders:    map[string]string{"X-Custom": "header"},
		ProviderOptions: map[string]any{"option": "value"},
		Models: []catwalk.Model{
			{ID: "gpt-4o"},
		},
	}

	if provider.ID != "openai" {
		t.Errorf("ID = %q, want %q", provider.ID, "openai")
	}
	if provider.Name != "OpenAI" {
		t.Errorf("Name = %q, want %q", provider.Name, "OpenAI")
	}
	if provider.Type != catwalk.TypeOpenAI {
		t.Errorf("Type = %q, want %q", provider.Type, catwalk.TypeOpenAI)
	}
	if provider.BaseURL != "https://api.openai.com/v1" {
		t.Errorf("BaseURL = %q, want %q", provider.BaseURL, "https://api.openai.com/v1")
	}
	if provider.APIKey != "sk-test" {
		t.Errorf("APIKey = %q, want %q", provider.APIKey, "sk-test")
	}
	if provider.Disable {
		t.Error("Disable = true, want false")
	}
	if provider.ExtraHeaders["X-Custom"] != "header" {
		t.Error("ExtraHeaders not set correctly")
	}
	if provider.ProviderOptions["option"] != "value" {
		t.Error("ProviderOptions not set correctly")
	}
	if len(provider.Models) != 1 || provider.Models[0].ID != "gpt-4o" {
		t.Error("Models not set correctly")
	}
}

func TestOptions_Fields(t *testing.T) {
	options := Options{
		ContextPaths: []string{"CONTEXT.md", "README.md"},
		DataDir:      "/data/dir",
		Debug:        true,
	}

	if len(options.ContextPaths) != 2 {
		t.Errorf("ContextPaths length = %d, want 2", len(options.ContextPaths))
	}
	if options.ContextPaths[0] != "CONTEXT.md" {
		t.Errorf("ContextPaths[0] = %q, want %q", options.ContextPaths[0], "CONTEXT.md")
	}
	if options.DataDir != "/data/dir" {
		t.Errorf("DataDir = %q, want %q", options.DataDir, "/data/dir")
	}
	if !options.Debug {
		t.Error("Debug = false, want true")
	}
}
