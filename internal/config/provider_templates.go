// Package config provides provider templates for common LLM services.
package config

import (
	"github.com/charmbracelet/catwalk/pkg/catwalk"
)

// ProviderTemplate defines a pre-built provider configuration template.
type ProviderTemplate struct {
	Name                string
	ID                  string
	Type                catwalk.Type
	APIEndpoint         string
	DefaultHeaders      map[string]string
	DefaultLargeModelID string
	DefaultSmallModelID string
	DefaultModels       []catwalk.Model
	Description         string
	Variables           []TemplateVariable
}

// TemplateVariable defines a variable that can be customized in a template.
type TemplateVariable struct {
	Name         string
	Description  string
	DefaultValue string
	Placeholder  string
}

// ProviderTemplates returns all available provider templates.
func ProviderTemplates() map[string]ProviderTemplate {
	return map[string]ProviderTemplate{
		"ollama": {
			Name:                "Ollama",
			ID:                  "ollama",
			Type:                catwalk.TypeOpenAICompat,
			APIEndpoint:         "http://localhost:11434/v1",
			DefaultLargeModelID: "qwen2.5:32b",
			DefaultSmallModelID: "qwen2.5:7b",
			DefaultModels: []catwalk.Model{
				{
					ID:               "qwen2.5:32b",
					Name:             "Qwen 2.5 32B",
					ContextWindow:    32768,
					DefaultMaxTokens: 8192,
				},
				{
					ID:               "qwen2.5:7b",
					Name:             "Qwen 2.5 7B",
					ContextWindow:    32768,
					DefaultMaxTokens: 4096,
				},
				{
					ID:               "llama3.3:70b",
					Name:             "Llama 3.3 70B",
					ContextWindow:    128000,
					DefaultMaxTokens: 8192,
				},
				{
					ID:               "llama3.3:8b",
					Name:             "Llama 3.3 8B",
					ContextWindow:    128000,
					DefaultMaxTokens: 4096,
				},
			},
			Description: "Local Ollama server for running open-source models",
			Variables: []TemplateVariable{
				{
					Name:         "base_url",
					Description:  "Ollama server URL",
					DefaultValue: "http://localhost:11434/v1",
					Placeholder:  "http://localhost:11434/v1",
				},
			},
		},
		"lmstudio": {
			Name:                "LM Studio",
			ID:                  "lmstudio",
			Type:                catwalk.TypeOpenAICompat,
			APIEndpoint:         "http://localhost:1234/v1",
			DefaultLargeModelID: "qwen2.5-32b-instruct",
			DefaultSmallModelID: "qwen2.5-7b-instruct",
			DefaultModels: []catwalk.Model{
				{
					ID:               "qwen2.5-32b-instruct",
					Name:             "Qwen 2.5 32B Instruct",
					ContextWindow:    32768,
					DefaultMaxTokens: 8192,
				},
				{
					ID:               "qwen2.5-7b-instruct",
					Name:             "Qwen 2.5 7B Instruct",
					ContextWindow:    32768,
					DefaultMaxTokens: 4096,
				},
			},
			Description: "LM Studio for running local LLMs with a GUI",
			Variables: []TemplateVariable{
				{
					Name:         "base_url",
					Description:  "LM Studio server URL",
					DefaultValue: "http://localhost:1234/v1",
					Placeholder:  "http://localhost:1234/v1",
				},
			},
		},
		"openrouter": {
			Name:        "OpenRouter",
			ID:          "openrouter-custom",
			Type:        catwalk.TypeOpenRouter,
			APIEndpoint: "https://openrouter.ai/api/v1",
			DefaultHeaders: map[string]string{
				"HTTP-Referer": "https://cdd.cli",
				"X-Title":      "CDD",
			},
			DefaultLargeModelID: "anthropic/claude-sonnet-4",
			DefaultSmallModelID: "anthropic/claude-3.5-haiku",
			Description:         "OpenRouter - API for accessing many LLMs",
			Variables: []TemplateVariable{
				{
					Name:         "api_key",
					Description:  "OpenRouter API key",
					DefaultValue: "$OPENROUTER_API_KEY",
					Placeholder:  "sk-or-v1-...",
				},
			},
		},
		"together": {
			Name:                "Together AI",
			ID:                  "together",
			Type:                catwalk.TypeOpenAICompat,
			APIEndpoint:         "https://api.together.xyz/v1",
			DefaultLargeModelID: "meta-llama/Meta-Llama-3.1-405B-Instruct-Turbo",
			DefaultSmallModelID: "meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo",
			DefaultModels: []catwalk.Model{
				{
					ID:               "meta-llama/Meta-Llama-3.1-405B-Instruct-Turbo",
					Name:             "Llama 3.1 405B Instruct Turbo",
					ContextWindow:    131072,
					DefaultMaxTokens: 4096,
					CostPer1MIn:      3.0,
					CostPer1MOut:     3.0,
				},
				{
					ID:               "meta-llama/Meta-Llama-3.1-8B-Instruct-Turbo",
					Name:             "Llama 3.1 8B Instruct Turbo",
					ContextWindow:    131072,
					DefaultMaxTokens: 4096,
					CostPer1MIn:      0.20,
					CostPer1MOut:     0.20,
				},
				{
					ID:               "mistralai/Mixtral-8x7B-Instruct-v0.1",
					Name:             "Mixtral 8x7B Instruct",
					ContextWindow:    32768,
					DefaultMaxTokens: 4096,
					CostPer1MIn:      0.30,
					CostPer1MOut:     0.30,
				},
			},
			Description: "Together AI - Fast inference for open-source models",
			Variables: []TemplateVariable{
				{
					Name:         "api_key",
					Description:  "Together AI API key",
					DefaultValue: "$TOGETHER_API_KEY",
					Placeholder:  "your-api-key-here",
				},
			},
		},
		"deepseek": {
			Name:                "DeepSeek",
			ID:                  "deepseek-custom",
			Type:                catwalk.TypeOpenAICompat,
			APIEndpoint:         "https://api.deepseek.com/v1",
			DefaultLargeModelID: "deepseek-reasoner",
			DefaultSmallModelID: "deepseek-chat",
			DefaultModels: []catwalk.Model{
				{
					ID:                 "deepseek-reasoner",
					Name:               "DeepSeek Reasoner",
					ContextWindow:      64000,
					DefaultMaxTokens:   8192,
					CostPer1MIn:        0.57,
					CostPer1MOut:       0.57,
					CostPer1MInCached:  0.14,
					CostPer1MOutCached: 0.57,
				},
				{
					ID:                 "deepseek-chat",
					Name:               "DeepSeek Chat",
					ContextWindow:      64000,
					DefaultMaxTokens:   8192,
					CostPer1MIn:        0.14,
					CostPer1MOut:       0.28,
					CostPer1MInCached:  0.014,
					CostPer1MOutCached: 0.28,
				},
			},
			Description: "DeepSeek - Advanced reasoning models",
			Variables: []TemplateVariable{
				{
					Name:         "api_key",
					Description:  "DeepSeek API key",
					DefaultValue: "$DEEPSEEK_API_KEY",
					Placeholder:  "sk-...",
				},
			},
		},
		"groq": {
			Name:                "Groq",
			ID:                  "groq",
			Type:                catwalk.TypeOpenAICompat,
			APIEndpoint:         "https://api.groq.com/openai/v1",
			DefaultLargeModelID: "llama-3.3-70b-versatile",
			DefaultSmallModelID: "llama-3.3-8b-instant",
			DefaultModels: []catwalk.Model{
				{
					ID:               "llama-3.3-70b-versatile",
					Name:             "Llama 3.3 70B Versatile",
					ContextWindow:    131072,
					DefaultMaxTokens: 8192,
				},
				{
					ID:               "llama-3.3-8b-instant",
					Name:             "Llama 3.3 8B Instant",
					ContextWindow:    131072,
					DefaultMaxTokens: 4096,
				},
				{
					ID:               "gemma2-9b-it",
					Name:             "Gemma 2 9B IT",
					ContextWindow:    8192,
					DefaultMaxTokens: 4096,
				},
			},
			Description: "Groq - Extremely fast inference",
			Variables: []TemplateVariable{
				{
					Name:         "api_key",
					Description:  "Groq API key",
					DefaultValue: "$GROQ_API_KEY",
					Placeholder:  "gsk_...",
				},
			},
		},
		"anthropic-compatible": {
			Name:        "Anthropic Compatible",
			ID:          "anthropic-compatible",
			Type:        catwalk.TypeAnthropic,
			APIEndpoint: "https://api.anthropic.com",
			DefaultHeaders: map[string]string{
				"anthropic-version": "2023-06-01",
			},
			DefaultLargeModelID: "claude-sonnet-4-5-20250929",
			DefaultSmallModelID: "claude-3-5-haiku-20241022",
			Description:         "Generic Anthropic-compatible API (e.g., via proxy)",
			Variables: []TemplateVariable{
				{
					Name:         "base_url",
					Description:  "API base URL",
					DefaultValue: "https://api.anthropic.com",
					Placeholder:  "https://api.anthropic.com",
				},
				{
					Name:         "api_key",
					Description:  "API key",
					DefaultValue: "$ANTHROPIC_API_KEY",
					Placeholder:  "sk-ant-...",
				},
			},
		},
		"azure-openai": {
			Name:                "Azure OpenAI",
			ID:                  "azure-openai",
			Type:                catwalk.TypeAzure,
			APIEndpoint:         "", // User must provide this
			DefaultLargeModelID: "gpt-4o",
			DefaultSmallModelID: "gpt-4o-mini",
			DefaultModels: []catwalk.Model{
				{
					ID:               "gpt-4o",
					Name:             "GPT-4o",
					ContextWindow:    128000,
					DefaultMaxTokens: 4096,
				},
				{
					ID:               "gpt-4o-mini",
					Name:             "GPT-4o Mini",
					ContextWindow:    128000,
					DefaultMaxTokens: 4096,
				},
			},
			Description: "Azure OpenAI Service",
			Variables: []TemplateVariable{
				{
					Name:         "base_url",
					Description:  "Azure OpenAI endpoint (e.g., https://your-resource.openai.azure.com/)",
					DefaultValue: "",
					Placeholder:  "https://your-resource.openai.azure.com/",
				},
				{
					Name:         "api_key",
					Description:  "Azure OpenAI API key",
					DefaultValue: "$AZURE_OPENAI_API_KEY",
					Placeholder:  "your-api-key",
				},
				{
					Name:         "api_version",
					Description:  "API version",
					DefaultValue: "2024-02-01",
					Placeholder:  "2024-02-01",
				},
			},
		},
		"vertexai": {
			Name:                "Google Vertex AI",
			ID:                  "vertexai",
			Type:                catwalk.TypeVertexAI,
			APIEndpoint:         "", // Configured via environment
			DefaultLargeModelID: "gemini-2.5-pro",
			DefaultSmallModelID: "gemini-2.5-flash",
			DefaultModels: []catwalk.Model{
				{
					ID:               "gemini-2.5-pro",
					Name:             "Gemini 2.5 Pro",
					ContextWindow:    1000000,
					DefaultMaxTokens: 8192,
				},
				{
					ID:               "gemini-2.5-flash",
					Name:             "Gemini 2.5 Flash",
					ContextWindow:    1000000,
					DefaultMaxTokens: 8192,
				},
			},
			Description: "Google Vertex AI - Requires GCloud authentication setup",
			Variables: []TemplateVariable{
				{
					Name:         "project",
					Description:  "Google Cloud project ID",
					DefaultValue: "$VERTEXAI_PROJECT",
					Placeholder:  "your-project-id",
				},
				{
					Name:         "location",
					Description:  "Google Cloud region",
					DefaultValue: "$VERTEXAI_LOCATION",
					Placeholder:  "us-central1",
				},
			},
		},
	}
}

// GetTemplate returns a provider template by name.
func GetTemplate(name string) (ProviderTemplate, bool) {
	templates := ProviderTemplates()
	template, ok := templates[name]
	return template, ok
}

// ListTemplateNames returns all available template names.
func ListTemplateNames() []string {
	templates := ProviderTemplates()
	names := make([]string, 0, len(templates))
	for name := range templates {
		names = append(names, name)
	}
	return names
}

// ToCustomProvider converts a template to a CustomProvider with variable substitutions.
func (pt *ProviderTemplate) ToCustomProvider(vars map[string]string, customID, customName string) CustomProvider {
	apiEndpoint := pt.APIEndpoint
	if val, ok := vars["base_url"]; ok && val != "" {
		apiEndpoint = val
	}

	// Build headers from template defaults
	headers := make(map[string]string)
	for k, v := range pt.DefaultHeaders {
		headers[k] = v
	}

	providerID := pt.ID
	if customID != "" {
		providerID = customID
	}

	providerName := pt.Name
	if customName != "" {
		providerName = customName
	}

	return CustomProvider{
		Name:                providerName,
		ID:                  providerID,
		Type:                pt.Type,
		APIEndpoint:         apiEndpoint,
		DefaultHeaders:      headers,
		DefaultLargeModelID: pt.DefaultLargeModelID,
		DefaultSmallModelID: pt.DefaultSmallModelID,
		Models:              pt.DefaultModels,
	}
}
