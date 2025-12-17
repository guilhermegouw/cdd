//nolint:goconst // Test file uses repeated string literals for clarity.
package provider

import (
	"context"
	"testing"

	"github.com/charmbracelet/catwalk/pkg/catwalk"

	"github.com/guilhermegouw/cdd/internal/config"
)

func TestModel_Struct(t *testing.T) {
	model := Model{
		CatwalkCfg: catwalk.Model{
			ID:   "gpt-4o",
			Name: "GPT-4o",
		},
		ModelCfg: config.SelectedModel{
			Model:    "gpt-4o",
			Provider: "openai",
		},
	}

	if model.CatwalkCfg.ID != "gpt-4o" {
		t.Errorf("CatwalkCfg.ID = %q, want %q", model.CatwalkCfg.ID, "gpt-4o")
	}
	if model.ModelCfg.Model != "gpt-4o" {
		t.Errorf("ModelCfg.Model = %q, want %q", model.ModelCfg.Model, "gpt-4o")
	}
	if model.ModelCfg.Provider != "openai" {
		t.Errorf("ModelCfg.Provider = %q, want %q", model.ModelCfg.Provider, "openai")
	}
}

func TestNewBuilder(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	if builder == nil {
		t.Fatal("NewBuilder returned nil")
	}
	if builder.cfg != cfg {
		t.Error("Builder.cfg not set correctly")
	}
	if builder.cache == nil {
		t.Error("Builder.cache not initialized")
	}
	if builder.debug {
		t.Error("Builder.debug should be false when Options is nil-ish")
	}
}

func TestNewBuilder_WithDebug(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Options = &config.Options{Debug: true}

	builder := NewBuilder(cfg)

	if !builder.debug {
		t.Error("Builder.debug should be true when Options.Debug is true")
	}
}

func TestNewBuilder_NilOptions(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Options = nil

	builder := NewBuilder(cfg)

	if builder.debug {
		t.Error("Builder.debug should be false when Options is nil")
	}
}

func TestBuilder_BuildModels_MissingLargeModel(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	_, _, err := builder.BuildModels(context.Background())
	if err == nil {
		t.Error("BuildModels() expected error for missing large model")
	}
}

func TestBuilder_BuildModels_MissingProvider(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Models[config.SelectedModelTypeLarge] = config.SelectedModel{
		Model:    "gpt-4o",
		Provider: "openai",
	}
	// Provider "openai" is not configured.
	builder := NewBuilder(cfg)

	_, _, err := builder.BuildModels(context.Background())
	if err == nil {
		t.Error("BuildModels() expected error for missing provider")
	}
}

func TestBuilder_BuildModels_UnsupportedProviderType(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Models[config.SelectedModelTypeLarge] = config.SelectedModel{
		Model:    "model",
		Provider: "custom",
	}
	cfg.Providers["custom"] = &config.ProviderConfig{
		ID:   "custom",
		Type: "unsupported-type",
	}
	builder := NewBuilder(cfg)

	_, _, err := builder.BuildModels(context.Background())
	if err == nil {
		t.Error("BuildModels() expected error for unsupported provider type")
	}
}

func TestBuilder_buildProvider_OpenAI(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	providerCfg := &config.ProviderConfig{
		ID:      "openai",
		Type:    catwalk.TypeOpenAI,
		APIKey:  "sk-test",
		BaseURL: "https://api.openai.com/v1",
	}
	modelCfg := config.SelectedModel{
		Model:    "gpt-4o",
		Provider: "openai",
	}

	provider, err := builder.buildProvider(providerCfg, modelCfg)
	if err != nil {
		t.Fatalf("buildProvider() error = %v", err)
	}
	if provider == nil {
		t.Error("buildProvider() returned nil provider")
	}
}

func TestBuilder_buildProvider_OpenAICompat(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	providerCfg := &config.ProviderConfig{
		ID:      "local",
		Type:    catwalk.TypeOpenAICompat,
		BaseURL: "http://localhost:8080",
	}
	modelCfg := config.SelectedModel{
		Model:    "local-model",
		Provider: "local",
	}

	provider, err := builder.buildProvider(providerCfg, modelCfg)
	if err != nil {
		t.Fatalf("buildProvider() error = %v", err)
	}
	if provider == nil {
		t.Error("buildProvider() returned nil provider")
	}
}

func TestBuilder_buildProvider_Anthropic(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	providerCfg := &config.ProviderConfig{
		ID:     "anthropic",
		Type:   catwalk.TypeAnthropic,
		APIKey: "sk-ant-test",
	}
	modelCfg := config.SelectedModel{
		Model:    "claude-3-opus",
		Provider: "anthropic",
	}

	provider, err := builder.buildProvider(providerCfg, modelCfg)
	if err != nil {
		t.Fatalf("buildProvider() error = %v", err)
	}
	if provider == nil {
		t.Error("buildProvider() returned nil provider")
	}
}

func TestBuilder_buildProvider_AnthropicWithThink(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	providerCfg := &config.ProviderConfig{
		ID:     "anthropic",
		Type:   catwalk.TypeAnthropic,
		APIKey: "sk-ant-test",
	}
	modelCfg := config.SelectedModel{
		Model:    "claude-3-opus",
		Provider: "anthropic",
		Think:    true,
	}

	provider, err := builder.buildProvider(providerCfg, modelCfg)
	if err != nil {
		t.Fatalf("buildProvider() error = %v", err)
	}
	if provider == nil {
		t.Error("buildProvider() returned nil provider")
	}
}

func TestBuilder_buildProvider_AnthropicWithExistingBetaHeader(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	providerCfg := &config.ProviderConfig{
		ID:     "anthropic",
		Type:   catwalk.TypeAnthropic,
		APIKey: "sk-ant-test",
		ExtraHeaders: map[string]string{
			"anthropic-beta": "existing-feature",
		},
	}
	modelCfg := config.SelectedModel{
		Model:    "claude-3-opus",
		Provider: "anthropic",
		Think:    true,
	}

	provider, err := builder.buildProvider(providerCfg, modelCfg)
	if err != nil {
		t.Fatalf("buildProvider() error = %v", err)
	}
	if provider == nil {
		t.Error("buildProvider() returned nil provider")
	}
}

func TestBuilder_buildProvider_AnthropicWithBearerToken(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	providerCfg := &config.ProviderConfig{
		ID:     "anthropic",
		Type:   catwalk.TypeAnthropic,
		APIKey: "Bearer oauth-token",
	}
	modelCfg := config.SelectedModel{
		Model:    "claude-3-opus",
		Provider: "anthropic",
	}

	provider, err := builder.buildProvider(providerCfg, modelCfg)
	if err != nil {
		t.Fatalf("buildProvider() error = %v", err)
	}
	if provider == nil {
		t.Error("buildProvider() returned nil provider")
	}
}

func TestBuilder_buildProvider_UnsupportedType(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	providerCfg := &config.ProviderConfig{
		ID:   "custom",
		Type: "unknown-type",
	}
	modelCfg := config.SelectedModel{
		Model:    "model",
		Provider: "custom",
	}

	_, err := builder.buildProvider(providerCfg, modelCfg)
	if err == nil {
		t.Error("buildProvider() expected error for unsupported type")
	}
}

func TestBuilder_buildProvider_WithExtraHeaders(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	providerCfg := &config.ProviderConfig{
		ID:     "openai",
		Type:   catwalk.TypeOpenAI,
		APIKey: "sk-test",
		ExtraHeaders: map[string]string{
			"X-Custom-Header": "custom-value",
		},
	}
	modelCfg := config.SelectedModel{
		Model:    "gpt-4o",
		Provider: "openai",
	}

	provider, err := builder.buildProvider(providerCfg, modelCfg)
	if err != nil {
		t.Fatalf("buildProvider() error = %v", err)
	}
	if provider == nil {
		t.Error("buildProvider() returned nil provider")
	}
}

func TestBuilder_getOrBuildProvider_Caching(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	providerCfg := &config.ProviderConfig{
		ID:     "openai",
		Type:   catwalk.TypeOpenAI,
		APIKey: "sk-test",
	}
	modelCfg := config.SelectedModel{
		Model:    "gpt-4o",
		Provider: "openai",
	}

	// First call should build.
	p1, err := builder.getOrBuildProvider(providerCfg, modelCfg)
	if err != nil {
		t.Fatalf("getOrBuildProvider() first call error = %v", err)
	}

	// Second call should return cached.
	p2, err := builder.getOrBuildProvider(providerCfg, modelCfg)
	if err != nil {
		t.Fatalf("getOrBuildProvider() second call error = %v", err)
	}

	// Should be the same instance.
	if p1 != p2 {
		t.Error("getOrBuildProvider() did not return cached provider")
	}
}

func TestBuilder_buildOpenAIProvider_MinimalConfig(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	// Test with minimal config (no API key, no base URL, no headers).
	provider, err := builder.buildOpenAIProvider("", "", nil)
	if err != nil {
		t.Fatalf("buildOpenAIProvider() error = %v", err)
	}
	if provider == nil {
		t.Error("buildOpenAIProvider() returned nil provider")
	}
}

func TestBuilder_buildAnthropicProvider_MinimalConfig(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	// Test with minimal config.
	provider, err := builder.buildAnthropicProvider("", "", nil)
	if err != nil {
		t.Fatalf("buildAnthropicProvider() error = %v", err)
	}
	if provider == nil {
		t.Error("buildAnthropicProvider() returned nil provider")
	}
}

func TestBuilder_buildAnthropicProvider_WithBaseURL(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	provider, err := builder.buildAnthropicProvider("https://custom.api.com", "sk-ant-test", nil)
	if err != nil {
		t.Fatalf("buildAnthropicProvider() error = %v", err)
	}
	if provider == nil {
		t.Error("buildAnthropicProvider() returned nil provider")
	}
}

func TestBuilder_buildOpenAIProvider_WithAllOptions(t *testing.T) {
	cfg := config.NewConfig()
	builder := NewBuilder(cfg)

	headers := map[string]string{
		"X-Custom": "value",
	}
	provider, err := builder.buildOpenAIProvider("https://api.openai.com/v1", "sk-test", headers)
	if err != nil {
		t.Fatalf("buildOpenAIProvider() error = %v", err)
	}
	if provider == nil {
		t.Error("buildOpenAIProvider() returned nil provider")
	}
}

func TestBuilder_BuildModels_Success(t *testing.T) {
	cfg := config.NewConfig()

	// Configure provider.
	cfg.Providers["openai"] = &config.ProviderConfig{
		ID:     "openai",
		Type:   catwalk.TypeOpenAI,
		APIKey: "sk-test",
		Models: []catwalk.Model{
			{ID: "gpt-4o", Name: "GPT-4o"},
			{ID: "gpt-4o-mini", Name: "GPT-4o Mini"},
		},
	}

	// Configure models.
	cfg.Models[config.SelectedModelTypeLarge] = config.SelectedModel{
		Model:    "gpt-4o",
		Provider: "openai",
	}
	cfg.Models[config.SelectedModelTypeSmall] = config.SelectedModel{
		Model:    "gpt-4o-mini",
		Provider: "openai",
	}

	builder := NewBuilder(cfg)
	large, small, err := builder.BuildModels(context.Background())
	if err != nil {
		t.Fatalf("BuildModels() error = %v", err)
	}

	if large.Model == nil {
		t.Error("large.Model is nil")
	}
	if small.Model == nil {
		t.Error("small.Model is nil")
	}
	if large.ModelCfg.Model != "gpt-4o" {
		t.Errorf("large.ModelCfg.Model = %q, want %q", large.ModelCfg.Model, "gpt-4o")
	}
	if small.ModelCfg.Model != "gpt-4o-mini" {
		t.Errorf("small.ModelCfg.Model = %q, want %q", small.ModelCfg.Model, "gpt-4o-mini")
	}
}

func TestBuilder_BuildModels_FallbackSmallToLarge(t *testing.T) {
	cfg := config.NewConfig()

	// Configure provider.
	cfg.Providers["openai"] = &config.ProviderConfig{
		ID:     "openai",
		Type:   catwalk.TypeOpenAI,
		APIKey: "sk-test",
		Models: []catwalk.Model{
			{ID: "gpt-4o", Name: "GPT-4o"},
		},
	}

	// Only configure large model, no small.
	cfg.Models[config.SelectedModelTypeLarge] = config.SelectedModel{
		Model:    "gpt-4o",
		Provider: "openai",
	}

	builder := NewBuilder(cfg)
	large, small, err := builder.BuildModels(context.Background())
	if err != nil {
		t.Fatalf("BuildModels() error = %v", err)
	}

	// Small should fall back to large.
	if large.ModelCfg.Model != small.ModelCfg.Model {
		t.Error("small should fall back to large when not configured")
	}
}

func TestBuilder_BuildModels_SmallModelError(t *testing.T) {
	cfg := config.NewConfig()

	// Configure only one provider for large.
	cfg.Providers["openai"] = &config.ProviderConfig{
		ID:     "openai",
		Type:   catwalk.TypeOpenAI,
		APIKey: "sk-test",
	}

	cfg.Models[config.SelectedModelTypeLarge] = config.SelectedModel{
		Model:    "gpt-4o",
		Provider: "openai",
	}
	// Small model with missing provider.
	cfg.Models[config.SelectedModelTypeSmall] = config.SelectedModel{
		Model:    "model",
		Provider: "missing-provider",
	}

	builder := NewBuilder(cfg)
	_, _, err := builder.BuildModels(context.Background())
	if err == nil {
		t.Error("BuildModels() expected error for missing small model provider")
	}
}

func TestBuilder_buildModel_WithCatwalkMetadata(t *testing.T) {
	cfg := config.NewConfig()

	// Configure provider with models.
	cfg.Providers["openai"] = &config.ProviderConfig{
		ID:     "openai",
		Type:   catwalk.TypeOpenAI,
		APIKey: "sk-test",
		Models: []catwalk.Model{
			{ID: "gpt-4o", Name: "GPT-4o", ContextWindow: 128000},
		},
	}

	cfg.Models[config.SelectedModelTypeLarge] = config.SelectedModel{
		Model:    "gpt-4o",
		Provider: "openai",
	}

	builder := NewBuilder(cfg)
	large, _, err := builder.BuildModels(context.Background())
	if err != nil {
		t.Fatalf("BuildModels() error = %v", err)
	}

	// Check that catwalk metadata was populated.
	if large.CatwalkCfg.ID != "gpt-4o" {
		t.Errorf("CatwalkCfg.ID = %q, want %q", large.CatwalkCfg.ID, "gpt-4o")
	}
	if large.CatwalkCfg.ContextWindow != 128000 {
		t.Errorf("CatwalkCfg.ContextWindow = %d, want %d", large.CatwalkCfg.ContextWindow, 128000)
	}
}

func TestBuilder_BuildModels_Anthropic(t *testing.T) {
	cfg := config.NewConfig()

	// Configure Anthropic provider.
	cfg.Providers["anthropic"] = &config.ProviderConfig{
		ID:     "anthropic",
		Type:   catwalk.TypeAnthropic,
		APIKey: "sk-ant-test",
		Models: []catwalk.Model{
			{ID: "claude-3-opus", Name: "Claude 3 Opus"},
		},
	}

	cfg.Models[config.SelectedModelTypeLarge] = config.SelectedModel{
		Model:    "claude-3-opus",
		Provider: "anthropic",
	}

	builder := NewBuilder(cfg)
	large, _, err := builder.BuildModels(context.Background())
	if err != nil {
		t.Fatalf("BuildModels() error = %v", err)
	}

	if large.Model == nil {
		t.Error("large.Model is nil")
	}
}
