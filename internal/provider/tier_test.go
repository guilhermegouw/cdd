package provider

import (
	"testing"

	"github.com/guilhermegouw/cdd/internal/config"
)

func TestGetModelForTier(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Models[config.SelectedModelTypeLarge] = config.SelectedModel{
		Model:    "gpt-4o",
		Provider: "openai",
	}
	cfg.Models[config.SelectedModelTypeSmall] = config.SelectedModel{
		Model:    "gpt-4o-mini",
		Provider: "openai",
	}

	tests := []struct {
		name      string
		tier      config.SelectedModelType
		wantModel string
		wantErr   bool
	}{
		{
			name:      "large tier",
			tier:      config.SelectedModelTypeLarge,
			wantModel: "gpt-4o",
			wantErr:   false,
		},
		{
			name:      "small tier",
			tier:      config.SelectedModelTypeSmall,
			wantModel: "gpt-4o-mini",
			wantErr:   false,
		},
		{
			name:      "unknown tier",
			tier:      "unknown",
			wantModel: "",
			wantErr:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetModelForTier(cfg, tt.tier)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetModelForTier() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.Model != tt.wantModel {
				t.Errorf("GetModelForTier().Model = %q, want %q", got.Model, tt.wantModel)
			}
		})
	}
}

func TestGetModelForTier_EmptyConfig(t *testing.T) {
	cfg := config.NewConfig()

	_, err := GetModelForTier(cfg, config.SelectedModelTypeLarge)
	if err == nil {
		t.Error("GetModelForTier() expected error for empty config")
	}
}

func TestGetProviderForModel(t *testing.T) {
	cfg := config.NewConfig()
	cfg.Providers["openai"] = &config.ProviderConfig{
		ID:     "openai",
		APIKey: "sk-test",
	}
	cfg.Providers["disabled"] = &config.ProviderConfig{
		ID:      "disabled",
		Disable: true,
	}

	tests := []struct {
		name    string
		model   *config.SelectedModel
		wantID  string
		wantErr bool
	}{
		{
			name: "existing provider",
			model: &config.SelectedModel{
				Model:    "gpt-4o",
				Provider: "openai",
			},
			wantID:  "openai",
			wantErr: false,
		},
		{
			name:    "nil model",
			model:   nil,
			wantID:  "",
			wantErr: true,
		},
		{
			name: "non-existent provider",
			model: &config.SelectedModel{
				Model:    "model",
				Provider: "nonexistent",
			},
			wantID:  "",
			wantErr: true,
		},
		{
			name: "disabled provider",
			model: &config.SelectedModel{
				Model:    "model",
				Provider: "disabled",
			},
			wantID:  "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := GetProviderForModel(cfg, tt.model)
			if (err != nil) != tt.wantErr {
				t.Errorf("GetProviderForModel() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr && got.ID != tt.wantID {
				t.Errorf("GetProviderForModel().ID = %q, want %q", got.ID, tt.wantID)
			}
		})
	}
}

func TestValidateConfig(t *testing.T) {
	//nolint:govet // Test struct field order optimized for readability.
	tests := []struct {
		name    string
		setup   func(*config.Config)
		wantErr bool
	}{
		{
			name: "valid config",
			setup: func(cfg *config.Config) {
				cfg.Providers["openai"] = &config.ProviderConfig{ID: "openai"}
				cfg.Models[config.SelectedModelTypeLarge] = config.SelectedModel{
					Model:    "gpt-4o",
					Provider: "openai",
				}
			},
			wantErr: false,
		},
		{
			name: "empty config",
			setup: func(_ *config.Config) {
				// No setup - empty config is valid.
			},
			wantErr: false,
		},
		{
			name: "model references unknown provider",
			setup: func(cfg *config.Config) {
				cfg.Models[config.SelectedModelTypeLarge] = config.SelectedModel{
					Model:    "model",
					Provider: "unknown",
				}
			},
			wantErr: true,
		},
		{
			name: "multiple tiers all valid",
			setup: func(cfg *config.Config) {
				cfg.Providers["openai"] = &config.ProviderConfig{ID: "openai"}
				cfg.Providers["anthropic"] = &config.ProviderConfig{ID: "anthropic"}
				cfg.Models[config.SelectedModelTypeLarge] = config.SelectedModel{
					Model:    "gpt-4o",
					Provider: "openai",
				}
				cfg.Models[config.SelectedModelTypeSmall] = config.SelectedModel{
					Model:    "claude-haiku",
					Provider: "anthropic",
				}
			},
			wantErr: false,
		},
		{
			name: "one tier invalid",
			setup: func(cfg *config.Config) {
				cfg.Providers["openai"] = &config.ProviderConfig{ID: "openai"}
				cfg.Models[config.SelectedModelTypeLarge] = config.SelectedModel{
					Model:    "gpt-4o",
					Provider: "openai",
				}
				cfg.Models[config.SelectedModelTypeSmall] = config.SelectedModel{
					Model:    "model",
					Provider: "missing",
				}
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := config.NewConfig()
			tt.setup(cfg)

			err := ValidateConfig(cfg)
			if (err != nil) != tt.wantErr {
				t.Errorf("ValidateConfig() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestAllTiers(t *testing.T) {
	tiers := AllTiers()

	if len(tiers) != 2 {
		t.Errorf("AllTiers() returned %d tiers, want 2", len(tiers))
	}

	// Check both tiers are present.
	foundLarge := false
	foundSmall := false
	for _, tier := range tiers {
		if tier == config.SelectedModelTypeLarge {
			foundLarge = true
		}
		if tier == config.SelectedModelTypeSmall {
			foundSmall = true
		}
	}

	if !foundLarge {
		t.Error("AllTiers() missing large tier")
	}
	if !foundSmall {
		t.Error("AllTiers() missing small tier")
	}
}
