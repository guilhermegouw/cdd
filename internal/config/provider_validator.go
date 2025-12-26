// Package config provides validation for custom provider configurations.
package config

import (
	"fmt"
	"net/url"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

// ValidationError represents a single validation error.
type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (ve ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", ve.Field, ve.Message)
}

// ValidationWarning represents a validation warning (non-fatal).
type ValidationWarning struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

func (vw ValidationWarning) String() string {
	return fmt.Sprintf("%s: %s", vw.Field, vw.Message)
}

// ValidationResult holds the result of validating a custom provider.
type ValidationResult struct {
	IsValid  bool                `json:"is_valid"`
	Errors   []ValidationError   `json:"errors,omitempty"`
	Warnings []ValidationWarning `json:"warnings,omitempty"`
}

// ValidateCustomProvider validates a custom provider configuration.
func ValidateCustomProvider(p *CustomProvider, existingProviderIDs []string) *ValidationResult {
	result := &ValidationResult{
		IsValid:  true,
		Errors:   []ValidationError{},
		Warnings: []ValidationWarning{},
	}

	// Validate required fields.
	if p.ID == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "id",
			Message: "provider ID is required",
		})
		result.IsValid = false
	}

	if p.Name == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "name",
			Message: "provider name is required",
		})
		result.IsValid = false
	}

	if p.Type == "" {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "type",
			Message: "provider type is required",
		})
		result.IsValid = false
	}

	// Validate provider type is supported.
	if p.Type != "" && !isValidProviderType(p.Type) {
		result.Errors = append(result.Errors, ValidationError{
			Field:   "type",
			Message: fmt.Sprintf("unsupported provider type %q, must be one of: anthropic, openai, openai-compat, google, azure, bedrock, vertexai, openrouter",
				p.Type),
		})
		result.IsValid = false
	}

	// Validate API endpoint / Base URL.
	apiEndpoint := p.APIEndpoint
	if apiEndpoint == "" {
		apiEndpoint = p.BaseURL
	}

	if apiEndpoint != "" {
		if err := validateURL(apiEndpoint); err != nil {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "api_endpoint",
				Message: err.Error(),
			})
			result.IsValid = false
		}
	}

	// Validate models.
	if len(p.Models) == 0 {
		result.Warnings = append(result.Warnings, ValidationWarning{
			Field:   "models",
			Message: "no models defined, provider will have no available models",
		})
	} else {
		modelIDs := make(map[string]bool)
		for i := range p.Models {
			if p.Models[i].ID == "" {
				result.Errors = append(result.Errors, ValidationError{
					Field:   fmt.Sprintf("models[%d].id", i),
					Message: "model ID is required",
				})
				result.IsValid = false
			}

			// Check for duplicate model IDs.
			if modelIDs[p.Models[i].ID] {
				result.Errors = append(result.Errors, ValidationError{
					Field:   fmt.Sprintf("models[%d].id", i),
					Message: fmt.Sprintf("duplicate model ID %q", p.Models[i].ID),
				})
				result.IsValid = false
			}
			modelIDs[p.Models[i].ID] = true

			// Warn if model name is missing.
			if p.Models[i].Name == "" {
				result.Warnings = append(result.Warnings, ValidationWarning{
					Field:   fmt.Sprintf("models[%d].name", i),
					Message: "model name is empty",
				})
			}
		}
	}

	// Validate provider ID uniqueness.
	for _, existingID := range existingProviderIDs {
		if p.ID == existingID {
			result.Errors = append(result.Errors, ValidationError{
				Field:   "id",
				Message: fmt.Sprintf("provider ID %q conflicts with existing provider", p.ID),
			})
			result.IsValid = false
			break
		}
	}

	// Validate default model IDs exist.
	if p.DefaultLargeModelID != "" {
		found := false
		for i := range p.Models {
			if p.Models[i].ID == p.DefaultLargeModelID {
				found = true
				break
			}
		}
		if !found {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "default_large_model_id",
				Message: fmt.Sprintf("default large model %q not found in models list", p.DefaultLargeModelID),
			})
		}
	}

	if p.DefaultSmallModelID != "" {
		found := false
		for i := range p.Models {
			if p.Models[i].ID == p.DefaultSmallModelID {
				found = true
				break
			}
		}
		if !found {
			result.Warnings = append(result.Warnings, ValidationWarning{
				Field:   "default_small_model_id",
				Message: fmt.Sprintf("default small model %q not found in models list", p.DefaultSmallModelID),
			})
		}
	}

	return result
}

// isValidProviderType checks if the provider type is supported.
func isValidProviderType(providerType catwalk.Type) bool {
	switch providerType {
	case catwalk.TypeAnthropic, catwalk.TypeOpenAI, catwalk.TypeOpenAICompat,
		catwalk.TypeGoogle, catwalk.TypeAzure, catwalk.TypeBedrock,
		catwalk.TypeVertexAI, catwalk.TypeOpenRouter:
		return true
	default:
		return false
	}
}

// validateURL validates that a string is a valid URL.
func validateURL(endpoint string) error {
	u, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	if u.Scheme == "" {
		return fmt.Errorf("URL must include a scheme (http:// or https://)")
	}

	if u.Scheme != "http" && u.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https, got %q", u.Scheme)
	}

	if u.Host == "" {
		return fmt.Errorf("URL must include a host")
	}

	return nil
}

// Error returns a combined error message from all validation errors.
func (vr *ValidationResult) Error() error {
	if len(vr.Errors) == 0 {
		return nil
	}

	msg := "validation failed:"
	for _, err := range vr.Errors {
		msg += "\n  - " + err.Error()
	}
	return fmt.Errorf("%s", msg)
}

// WarningStrings returns all warnings as strings.
func (vr *ValidationResult) WarningStrings() []string {
	warnings := make([]string, len(vr.Warnings))
	for i, w := range vr.Warnings {
		warnings[i] = w.String()
	}
	return warnings
}
