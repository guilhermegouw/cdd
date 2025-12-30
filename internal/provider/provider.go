// Package provider handles LLM provider instantiation and management.
package provider

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"maps"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/catwalk/pkg/catwalk"

	"charm.land/fantasy"
	"charm.land/fantasy/providers/anthropic"
	"charm.land/fantasy/providers/openai"

	"github.com/guilhermegouw/cdd/internal/config"
	"github.com/guilhermegouw/cdd/internal/debug"
)

// Model wraps a fantasy language model with its metadata.
type Model struct {
	// Model is the fantasy language model interface.
	Model fantasy.LanguageModel
	// CatwalkCfg holds the model metadata from catwalk.
	CatwalkCfg catwalk.Model
	// ModelCfg holds the user's selected configuration.
	ModelCfg config.SelectedModel
}

// Builder creates fantasy providers from configuration.
type Builder struct {
	cfg   *config.Config
	cache map[string]fantasy.Provider
	debug bool
}

// NewBuilder creates a new provider Builder.
func NewBuilder(cfg *config.Config) *Builder {
	return &Builder{
		cfg:   cfg,
		cache: make(map[string]fantasy.Provider),
		debug: cfg.Options != nil && cfg.Options.Debug,
	}
}

// BuildModels creates the large and small models from configuration.
func (b *Builder) BuildModels(ctx context.Context) (large, small Model, err error) {
	// Refresh any expired OAuth tokens before building models.
	if refreshErr := b.refreshExpiredTokens(ctx); refreshErr != nil {
		return Model{}, Model{}, fmt.Errorf("refreshing tokens: %w", refreshErr)
	}

	// Build large model.
	largeCfg, ok := b.cfg.Models[config.SelectedModelTypeLarge]
	if !ok {
		return Model{}, Model{}, fmt.Errorf("large model not configured")
	}
	large, err = b.buildModel(ctx, largeCfg)
	if err != nil {
		return Model{}, Model{}, fmt.Errorf("building large model: %w", err)
	}

	// Build small model.
	smallCfg, ok := b.cfg.Models[config.SelectedModelTypeSmall]
	if !ok {
		// Fall back to large model if small not configured.
		small = large
	} else {
		small, err = b.buildModel(ctx, smallCfg)
		if err != nil {
			return Model{}, Model{}, fmt.Errorf("building small model: %w", err)
		}
	}

	return large, small, nil
}

// refreshExpiredTokens checks all providers for expired OAuth tokens and refreshes them.
func (b *Builder) refreshExpiredTokens(ctx context.Context) error {
	for providerID, providerCfg := range b.cfg.Providers {
		if providerCfg.OAuthToken == nil {
			continue
		}

		if !providerCfg.OAuthToken.IsExpired() {
			expiresAt := time.Unix(providerCfg.OAuthToken.ExpiresAt, 0)
			expiresIn := time.Until(expiresAt)
			debug.Token("valid", fmt.Sprintf("provider=%s expires_in=%s", providerID, expiresIn.Round(time.Second)))
			continue
		}

		expiresAt := time.Unix(providerCfg.OAuthToken.ExpiresAt, 0)
		debug.Token("expired", fmt.Sprintf("provider=%s expiry=%s", providerID, expiresAt.Format(time.RFC3339)))

		// Token is expired, try to refresh it.
		// RefreshOAuthToken persists to disk immediately (critical for token rotation).
		if err := b.cfg.RefreshOAuthToken(ctx, providerID); err != nil {
			debug.Token("refresh_failed", fmt.Sprintf("provider=%s error=%v", providerID, err))
			return fmt.Errorf("refreshing token for provider %q: %w (re-authenticate with: rm ~/.config/cdd/cdd.json && cdd)", providerID, err)
		}

		newExpiresAt := time.Unix(providerCfg.OAuthToken.ExpiresAt, 0)
		debug.Token("refreshed", fmt.Sprintf("provider=%s new_expiry=%s", providerID, newExpiresAt.Format(time.RFC3339)))

		// Clear cached provider so it's rebuilt with new token.
		delete(b.cache, providerID)
	}

	return nil
}

// buildModel creates a Model from a selected model configuration.
func (b *Builder) buildModel(ctx context.Context, modelCfg config.SelectedModel) (Model, error) {
	// Determine which provider to use.
	providerID := modelCfg.Provider

	// If ConnectionID is set, get the provider from the connection.
	var conn *config.Connection
	if modelCfg.ConnectionID != "" {
		connManager := config.NewConnectionManager(b.cfg)
		conn = connManager.Get(modelCfg.ConnectionID)
		if conn != nil {
			providerID = conn.ProviderID
		}
	}

	providerCfg, ok := b.cfg.Providers[providerID]
	if !ok {
		return Model{}, fmt.Errorf("provider %q not configured", providerID)
	}

	// If we have a connection, use its credentials instead of the provider's.
	if conn != nil {
		providerCfg = applyConnectionCredentials(providerCfg, conn)
	}

	// Build or get cached fantasy provider.
	provider, err := b.getOrBuildProvider(providerCfg, modelCfg)
	if err != nil {
		return Model{}, err
	}

	// Get language model from provider.
	lm, err := provider.LanguageModel(ctx, modelCfg.Model)
	if err != nil {
		return Model{}, fmt.Errorf("getting language model %q: %w", modelCfg.Model, err)
	}

	// Find catwalk model metadata.
	var catwalkModel catwalk.Model
	if m := b.cfg.GetModel(modelCfg.Provider, modelCfg.Model); m != nil {
		catwalkModel = *m
	}

	return Model{
		Model:      lm,
		CatwalkCfg: catwalkModel,
		ModelCfg:   modelCfg,
	}, nil
}

// applyConnectionCredentials creates a copy of the provider config with connection credentials applied.
// Environment variables in API key and base URL are resolved.
func applyConnectionCredentials(providerCfg *config.ProviderConfig, conn *config.Connection) *config.ProviderConfig {
	providerCfgCopy := *providerCfg
	resolver := config.NewResolver()

	// Resolve API key (may contain $ENV_VAR references).
	if conn.APIKey != "" {
		if resolved, err := resolver.Resolve(conn.APIKey); err == nil {
			providerCfgCopy.APIKey = resolved
		} else {
			providerCfgCopy.APIKey = conn.APIKey // Use as-is if resolution fails
		}
	}
	if conn.OAuthToken != nil {
		providerCfgCopy.OAuthToken = conn.OAuthToken
		providerCfgCopy.SetupClaudeCode()
	}
	// Resolve base URL (may contain $ENV_VAR references).
	if conn.BaseURL != "" {
		if resolved, err := resolver.Resolve(conn.BaseURL); err == nil {
			providerCfgCopy.BaseURL = resolved
		} else {
			providerCfgCopy.BaseURL = conn.BaseURL // Use as-is if resolution fails
		}
	}
	if len(conn.ExtraHeaders) > 0 {
		if providerCfgCopy.ExtraHeaders == nil {
			providerCfgCopy.ExtraHeaders = make(map[string]string)
		}
		for k, v := range conn.ExtraHeaders {
			providerCfgCopy.ExtraHeaders[k] = v
		}
	}
	return &providerCfgCopy
}

// getOrBuildProvider returns a cached provider or builds a new one.
func (b *Builder) getOrBuildProvider(providerCfg *config.ProviderConfig, modelCfg config.SelectedModel) (fantasy.Provider, error) {
	if p, ok := b.cache[providerCfg.ID]; ok {
		return p, nil
	}

	p, err := b.buildProvider(providerCfg, modelCfg)
	if err != nil {
		return nil, err
	}

	b.cache[providerCfg.ID] = p
	return p, nil
}

// buildProvider creates a fantasy provider from configuration.
func (b *Builder) buildProvider(providerCfg *config.ProviderConfig, modelCfg config.SelectedModel) (fantasy.Provider, error) {
	headers := maps.Clone(providerCfg.ExtraHeaders)
	if headers == nil {
		headers = make(map[string]string)
	}

	// Handle special headers for anthropic thinking mode.
	if providerCfg.Type == anthropic.Name && modelCfg.Think {
		if v, ok := headers["anthropic-beta"]; ok {
			headers["anthropic-beta"] = v + ",interleaved-thinking-2025-05-14"
		} else {
			headers["anthropic-beta"] = "interleaved-thinking-2025-05-14"
		}
	}

	apiKey := providerCfg.APIKey
	baseURL := providerCfg.BaseURL

	//nolint:exhaustive // Only openai and anthropic are supported initially.
	switch providerCfg.Type {
	case openai.Name, catwalk.TypeOpenAICompat:
		return b.buildOpenAIProvider(baseURL, apiKey, headers)
	case anthropic.Name:
		return b.buildAnthropicProvider(baseURL, apiKey, headers)
	default:
		return nil, fmt.Errorf("unsupported provider type: %q", providerCfg.Type)
	}
}

// buildOpenAIProvider creates an OpenAI fantasy provider.
func (b *Builder) buildOpenAIProvider(baseURL, apiKey string, headers map[string]string) (fantasy.Provider, error) {
	var opts []openai.Option

	if apiKey != "" {
		opts = append(opts, openai.WithAPIKey(apiKey))
	}
	if len(headers) > 0 {
		opts = append(opts, openai.WithHeaders(headers))
	}
	if baseURL != "" {
		opts = append(opts, openai.WithBaseURL(baseURL))
	}

	return openai.New(opts...)
}

// buildAnthropicProvider creates an Anthropic fantasy provider.
func (b *Builder) buildAnthropicProvider(baseURL, apiKey string, headers map[string]string) (fantasy.Provider, error) {
	var opts []anthropic.Option
	isOAuth := strings.HasPrefix(apiKey, "Bearer ")

	// Handle OAuth token format.
	if isOAuth {
		// Prevent the SDK from picking up the API key from env.
		// This avoids conflict between OAuth Bearer token and x-api-key header.
		_ = os.Setenv("ANTHROPIC_API_KEY", "") //nolint:errcheck // Error extremely unlikely, safe to ignore

		headers["Authorization"] = apiKey

		// Use custom HTTP client to strip x-stainless-* headers for OAuth
		httpClient := &http.Client{
			Transport: &oauthTransport{
				base:    http.DefaultTransport,
				headers: headers,
			},
		}
		opts = append(opts, anthropic.WithHTTPClient(httpClient))
	} else if apiKey != "" {
		opts = append(opts, anthropic.WithAPIKey(apiKey))
	}

	// Only add headers via WithHeaders if not using OAuth (OAuth uses custom transport)
	if !isOAuth && len(headers) > 0 {
		opts = append(opts, anthropic.WithHeaders(headers))
	}
	if baseURL != "" {
		opts = append(opts, anthropic.WithBaseURL(baseURL))
	}

	return anthropic.New(opts...)
}

// oauthTransport is a custom HTTP transport for OAuth that removes
// x-api-key and x-stainless-* headers and adds OAuth headers.
type oauthTransport struct {
	base    http.RoundTripper
	headers map[string]string
}

func (t *oauthTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	// Clone the request to avoid mutating the original
	reqCopy := req.Clone(req.Context())

	// Remove x-api-key header (SDK might add it)
	reqCopy.Header.Del("x-api-key")
	reqCopy.Header.Del("X-Api-Key")

	// Remove x-stainless-* headers (SDK adds these, OAuth may reject them)
	for key := range reqCopy.Header {
		if strings.HasPrefix(strings.ToLower(key), "x-stainless") {
			reqCopy.Header.Del(key)
		}
	}

	// Add our OAuth headers
	for key, value := range t.headers {
		reqCopy.Header.Set(key, value)
	}

	// Log the API request
	debug.API(reqCopy.Method, reqCopy.URL.Path, 0, "sending request")

	resp, err := t.base.RoundTrip(reqCopy)
	if err != nil {
		debug.API(reqCopy.Method, reqCopy.URL.Path, 0, fmt.Sprintf("transport error: %v", err))
		return nil, err
	}

	// Log the API response
	debug.API(reqCopy.Method, reqCopy.URL.Path, resp.StatusCode, resp.Status)

	// Log error response body for debugging
	if resp.StatusCode >= 400 {
		bodyBytes, readErr := io.ReadAll(resp.Body)
		if readErr == nil {
			debug.API(reqCopy.Method, reqCopy.URL.Path, resp.StatusCode, fmt.Sprintf("error body: %s", string(bodyBytes)))
			// Restore the body for the SDK to read
			resp.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}
	}

	return resp, nil
}
