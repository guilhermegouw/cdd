// Package config provides tests for provider validation.
package config

import (
	"testing"

	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

func TestValidateCustomProvider_Valid(t *testing.T) {
	provider := &CustomProvider{
		Name:        "Valid Provider",
		ID:          "valid-provider",
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

	result := ValidateCustomProvider(provider, []string{})

	if !result.IsValid {
		t.Errorf("ValidateCustomProvider() returned IsValid=false, errors: %v", result.Errors)
	}

	if len(result.Errors) > 0 {
		t.Errorf("ValidateCustomProvider() returned errors: %v", result.Errors)
	}
}

func TestValidateCustomProvider_MissingName(t *testing.T) {
	provider := &CustomProvider{
		ID:   "test-provider",
		Type: catwalk.TypeOpenAICompat,
	}

	result := ValidateCustomProvider(provider, []string{})

	if result.IsValid {
		t.Error("ValidateCustomProvider() returned IsValid=true for provider without name")
	}

	if len(result.Errors) != 1 {
		t.Fatalf("ValidateCustomProvider() returned %d errors, want 1", len(result.Errors))
	}

	if result.Errors[0].Field != "name" {
		t.Errorf("Error field = %q, want %q", result.Errors[0].Field, "name")
	}
}

func TestValidateCustomProvider_MissingID(t *testing.T) {
	provider := &CustomProvider{
		Name: "Test Provider",
		Type: catwalk.TypeOpenAICompat,
	}

	result := ValidateCustomProvider(provider, []string{})

	if result.IsValid {
		t.Error("ValidateCustomProvider() returned IsValid=true for provider without ID")
	}

	if len(result.Errors) != 1 {
		t.Fatalf("ValidateCustomProvider() returned %d errors, want 1", len(result.Errors))
	}

	if result.Errors[0].Field != "id" {
		t.Errorf("Error field = %q, want %q", result.Errors[0].Field, "id")
	}
}

func TestValidateCustomProvider_MissingType(t *testing.T) {
	provider := &CustomProvider{
		Name: "Test Provider",
		ID:   "test-provider",
	}

	result := ValidateCustomProvider(provider, []string{})

	if result.IsValid {
		t.Error("ValidateCustomProvider() returned IsValid=true for provider without type")
	}

	if len(result.Errors) != 1 {
		t.Fatalf("ValidateCustomProvider() returned %d errors, want 1", len(result.Errors))
	}

	if result.Errors[0].Field != "type" {
		t.Errorf("Error field = %q, want %q", result.Errors[0].Field, "type")
	}
}

func TestValidateCustomProvider_InvalidType(t *testing.T) {
	provider := &CustomProvider{
		Name: "Test Provider",
		ID:   "test-provider",
		Type: "invalid-type",
	}

	result := ValidateCustomProvider(provider, []string{})

	if result.IsValid {
		t.Error("ValidateCustomProvider() returned IsValid=true for provider with invalid type")
	}

	if len(result.Errors) != 1 {
		t.Fatalf("ValidateCustomProvider() returned %d errors, want 1", len(result.Errors))
	}

	if result.Errors[0].Field != "type" {
		t.Errorf("Error field = %q, want %q", result.Errors[0].Field, "type")
	}
}

func TestValidateCustomProvider_InvalidURL(t *testing.T) {
	tests := []struct {
		name        string
		apiEndpoint string
		wantError   bool
	}{
		{"Valid HTTPS URL", "https://api.example.com/v1", false},
		{"Valid HTTP URL", "http://api.example.com/v1", false},
		{"Missing scheme", "api.example.com/v1", true},
		{"Invalid scheme", "ftp://api.example.com/v1", true},
		{"Missing host", "https://", true},
		{"Empty string", "", false}, // Empty is allowed (optional field)
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			provider := &CustomProvider{
				Name:        "Test Provider",
				ID:          "test-provider",
				Type:        catwalk.TypeOpenAICompat,
				APIEndpoint: tt.apiEndpoint,
			}

			result := ValidateCustomProvider(provider, []string{})

			if tt.wantError && result.IsValid {
				t.Errorf("ValidateCustomProvider() returned IsValid=true for invalid URL %q", tt.apiEndpoint)
			}

			if !tt.wantError && tt.apiEndpoint != "" && !result.IsValid {
				t.Errorf("ValidateCustomProvider() returned IsValid=false for valid URL %q, errors: %v", tt.apiEndpoint, result.Errors)
			}
		})
	}
}

func TestValidateCustomProvider_DuplicateID(t *testing.T) {
	provider := &CustomProvider{
		Name: "Test Provider",
		ID:   "existing-id",
		Type: catwalk.TypeOpenAICompat,
	}

	existingIDs := []string{"existing-id"}

	result := ValidateCustomProvider(provider, existingIDs)

	if result.IsValid {
		t.Error("ValidateCustomProvider() returned IsValid=true for duplicate ID")
	}

	if len(result.Errors) != 1 {
		t.Fatalf("ValidateCustomProvider() returned %d errors, want 1", len(result.Errors))
	}

	if result.Errors[0].Field != "id" {
		t.Errorf("Error field = %q, want %q", result.Errors[0].Field, "id")
	}
}

func TestValidateCustomProvider_DuplicateModelID(t *testing.T) {
	provider := &CustomProvider{
		Name: "Test Provider",
		ID:   "test-provider",
		Type: catwalk.TypeOpenAICompat,
		Models: []catwalk.Model{
			{
				ID:   "duplicate-model",
				Name: "Model 1",
			},
			{
				ID:   "duplicate-model",
				Name: "Model 2",
			},
		},
	}

	result := ValidateCustomProvider(provider, []string{})

	if result.IsValid {
		t.Error("ValidateCustomProvider() returned IsValid=true for duplicate model IDs")
	}

	// Should have at least one error for duplicate model ID.
	hasDuplicateError := false
	for _, err := range result.Errors {
		if err.Field == "models[1].id" && err.Message == "duplicate model ID \"duplicate-model\"" {
			hasDuplicateError = true
			break
		}
	}

	if !hasDuplicateError {
		t.Error("ValidateCustomProvider() did not return error for duplicate model ID")
	}
}

func TestValidateCustomProvider_NoModels(t *testing.T) {
	provider := &CustomProvider{
		Name:        "Test Provider",
		ID:          "test-provider",
		Type:        catwalk.TypeOpenAICompat,
		APIEndpoint: "https://api.example.com/v1",
		Models:      []catwalk.Model{},
	}

	result := ValidateCustomProvider(provider, []string{})

	// No models is allowed, but should produce a warning.
	if !result.IsValid {
		t.Errorf("ValidateCustomProvider() returned IsValid=false, errors: %v", result.Errors)
	}

	if len(result.Warnings) == 0 {
		t.Error("ValidateCustomProvider() did not return warning for no models")
	}
}

func TestValidateCustomProvider_MissingModelName(t *testing.T) {
	provider := &CustomProvider{
		Name: "Test Provider",
		ID:   "test-provider",
		Type: catwalk.TypeOpenAICompat,
		Models: []catwalk.Model{
			{
				ID:   "model-1",
				Name: "", // Missing name
			},
		},
	}

	result := ValidateCustomProvider(provider, []string{})

	// Missing model name is a warning, not an error.
	if !result.IsValid {
		t.Errorf("ValidateCustomProvider() returned IsValid=false, errors: %v", result.Errors)
	}

	if len(result.Warnings) == 0 {
		t.Error("ValidateCustomProvider() did not return warning for missing model name")
	}
}

func TestValidateCustomProvider_DefaultModelNotFound(t *testing.T) {
	provider := &CustomProvider{
		Name:                "Test Provider",
		ID:                  "test-provider",
		Type:                catwalk.TypeOpenAICompat,
		DefaultLargeModelID: "non-existent-model",
		Models: []catwalk.Model{
			{
				ID:   "different-model",
				Name: "Different Model",
			},
		},
	}

	result := ValidateCustomProvider(provider, []string{})

	// This is a warning, not an error.
	if !result.IsValid {
		t.Errorf("ValidateCustomProvider() returned IsValid=false, errors: %v", result.Errors)
	}

	if len(result.Warnings) == 0 {
		t.Error("ValidateCustomProvider() did not return warning for default model not found")
	}
}

func TestValidateCustomProvider_AllValidTypes(t *testing.T) {
	validTypes := []catwalk.Type{
		catwalk.TypeAnthropic,
		catwalk.TypeOpenAI,
		catwalk.TypeOpenAICompat,
		catwalk.TypeGoogle,
		catwalk.TypeAzure,
		catwalk.TypeBedrock,
		catwalk.TypeVertexAI,
		catwalk.TypeOpenRouter,
	}

	for _, providerType := range validTypes {
		t.Run(string(providerType), func(t *testing.T) {
			provider := &CustomProvider{
				Name: "Test Provider",
				ID:   "test-provider-" + string(providerType),
				Type: providerType,
				Models: []catwalk.Model{
					{
						ID:   "model-1",
						Name: "Model 1",
					},
				},
			}

			result := ValidateCustomProvider(provider, []string{})

			if !result.IsValid {
				t.Errorf("ValidateCustomProvider() returned IsValid=false for type %s, errors: %v", providerType, result.Errors)
			}
		})
	}
}

func TestValidationResult_Error(t *testing.T) {
	result := &ValidationResult{
		IsValid: false,
		Errors: []ValidationError{
			{Field: "id", Message: "is required"},
		},
	}

	err := result.Error()
	if err == nil {
		t.Error("Error() returned nil for invalid result")
	}

	errStr := err.Error()
	if errStr == "" {
		t.Error("Error() returned empty string")
	}
}

func TestValidationResult_Error_Valid(t *testing.T) {
	result := &ValidationResult{
		IsValid: true,
		Errors:   []ValidationError{},
	}

	err := result.Error()
	if err != nil {
		t.Errorf("Error() returned error for valid result: %v", err)
	}
}

func TestValidationResult_WarningStrings(t *testing.T) {
	result := &ValidationResult{
		IsValid: true,
		Warnings: []ValidationWarning{
			{Field: "models", Message: "no models defined"},
		},
	}

	warnings := result.WarningStrings()
	if len(warnings) != 1 {
		t.Fatalf("WarningStrings() returned %d warnings, want 1", len(warnings))
	}

	if warnings[0] == "" {
		t.Error("WarningStrings() returned empty string")
	}
}
