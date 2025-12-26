package provider

import (
	"fmt"

	"github.com/guilhermegouw/cdd/internal/config"
)

// GetModelForTier returns the model configuration for a given tier.
func GetModelForTier(cfg *config.Config, tier config.SelectedModelType) (*config.SelectedModel, error) {
	model, ok := cfg.Models[tier]
	if !ok {
		return nil, fmt.Errorf("tier %q not configured", tier)
	}
	return &model, nil
}

// GetProviderForModel returns the provider configuration for a selected model.
func GetProviderForModel(cfg *config.Config, model *config.SelectedModel) (*config.ProviderConfig, error) {
	if model == nil {
		return nil, fmt.Errorf("model is nil")
	}

	provider, ok := cfg.Providers[model.Provider]
	if !ok {
		return nil, fmt.Errorf("provider %q not configured", model.Provider)
	}

	if provider.Disable {
		return nil, fmt.Errorf("provider %q is disabled", model.Provider)
	}

	return provider, nil
}

// ValidateConfig checks that all configured tiers have valid providers.
func ValidateConfig(cfg *config.Config) error {
	for tier := range cfg.Models {
		if _, ok := cfg.Providers[cfg.Models[tier].Provider]; !ok {
			return fmt.Errorf("tier %s references unknown provider %q", tier, cfg.Models[tier].Provider)
		}
	}
	return nil
}

// AllTiers returns all available tier types.
func AllTiers() []config.SelectedModelType {
	return []config.SelectedModelType{
		config.SelectedModelTypeLarge,
		config.SelectedModelTypeSmall,
	}
}
