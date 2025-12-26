# Provider System Documentation

This document provides comprehensive documentation for the provider system, including the configuration system, authentication methods, quality tooling, and development workflow.

## Table of Contents

- [Overview](#overview)
- [Provider System Architecture](#provider-system-architecture)
  - [Two-Tier Model System](#two-tier-model-system)
  - [Provider Builder](#provider-builder)
  - [Catwalk Integration](#catwalk-integration)
  - [Fantasy Integration](#fantasy-integration)
- [Authentication](#authentication)
  - [API Key Authentication](#api-key-authentication)
  - [OAuth2 Authentication (Claude Account)](#oauth2-authentication-claude-account)
  - [Token Management](#token-management)
- [Configuration System](#configuration-system)
  - [Configuration Loading](#configuration-loading)
  - [Configuration Structure](#configuration-structure)
  - [Environment Variable Resolution](#environment-variable-resolution)
  - [Provider Configuration](#provider-configuration)
  - [First-Run Detection](#first-run-detection)
  - [Configuration Persistence](#configuration-persistence)
- [Custom Providers](#custom-providers)
  - [Overview](#overview-1)
  - [Provider Templates](#provider-templates)
  - [CLI Commands](#cli-commands)
  - [Custom Provider Structure](#custom-provider-structure)
  - [Provider Fields](#provider-fields)
  - [Supported Provider Types](#supported-provider-types)
  - [Model Fields](#model-fields)
  - [Validation](#validation)
  - [Provider Merging](#provider-merging)
  - [Wizard Integration](#wizard-integration)
  - [Example: Adding an Ollama Provider](#example-adding-an-ollama-provider)
  - [File Storage](#file-storage)
- [Quality Tooling](#quality-tooling)
  - [GitHub Actions CI](#github-actions-ci)
  - [golangci-lint Configuration](#golangci-lint-configuration)
  - [Taskfile](#taskfile)
- [Test Coverage](#test-coverage)

---

## Overview

The provider system enables CDD to interact with multiple LLM providers (OpenAI, Anthropic, and OpenAI-compatible APIs). It follows a two-tier model architecture (large/small) for optimizing cost and performance across different task complexities.

Key features:
- **Multi-provider support**: OpenAI, Anthropic, and OpenAI-compatible providers
- **Two-tier model system**: Large models for complex tasks, small models for simpler tasks
- **Catwalk integration**: Provider metadata and model information from Charm's catwalk service
- **Fantasy integration**: LLM orchestration through Charm's fantasy library
- **OAuth2 with PKCE**: Secure authentication for Claude Account subscriptions
- **Environment variable resolution**: Secure configuration with `$VAR` and `${VAR}` syntax
- **Provider caching**: Performance optimization with embedded fallback
- **Token refresh**: Automatic OAuth token renewal support

---

## Provider System Architecture

### Two-Tier Model System

The system uses two model tiers following the pattern established by Crush CLI:

| Tier | Purpose | Use Case |
|------|---------|----------|
| `large` | Complex tasks requiring full reasoning | Code generation, analysis, complex refactoring |
| `small` | Simpler, faster tasks | Quick queries, simple transformations, summarization |

**Implementation**: `internal/config/config.go`

```go
// SelectedModelType represents the tier of model (large or small).
type SelectedModelType string

// Model type constants.
const (
    SelectedModelTypeLarge SelectedModelType = "large"
    SelectedModelTypeSmall SelectedModelType = "small"
)
```

If only a large model is configured, it automatically falls back to use the large model for small tasks as well.

### Provider Builder

The `Builder` struct (`internal/provider/provider.go`) creates and caches fantasy providers:

```go
type Builder struct {
    cfg   *config.Config
    cache map[string]fantasy.Provider
    debug bool
}
```

**Key methods**:
- `NewBuilder(cfg)`: Creates a new builder from configuration
- `BuildModels(ctx)`: Creates large and small models from configuration
- `buildModel(ctx, modelCfg)`: Builds a single model with provider and catwalk metadata
- `getOrBuildProvider(providerCfg, modelCfg)`: Returns cached provider or builds new one

**Provider caching**: Providers are cached by ID to avoid redundant instantiation when the same provider is used for both tiers.

### Catwalk Integration

Catwalk provides provider metadata including:
- Available models per provider
- Default large/small model IDs
- API endpoints
- Model capabilities and pricing

**Loading hierarchy** (`internal/config/providers.go`):
1. Fetch from Catwalk API (`https://catwalk.charm.sh`)
2. Fall back to local cache (24-hour TTL)
3. Fall back to embedded provider data

**Cache location**: `$XDG_DATA_HOME/cdd/providers.json`

**Manual update**:
```go
UpdateProviders(cfg, source) // source: "embedded", URL, or file path
```

### Fantasy Integration

Fantasy is Charm's LLM orchestration library providing a unified interface across providers.

**Supported providers**:
- `anthropic`: Native Anthropic API
- `openai`: Native OpenAI API
- `openai-compat`: OpenAI-compatible APIs (Ollama, vLLM, etc.)

**Special handling**:
- **Anthropic thinking mode**: Automatically adds `anthropic-beta: interleaved-thinking-2025-05-14` header when `think: true`
- **OAuth tokens**: Detects `Bearer ` prefix and handles authorization header correctly

---

## Authentication

CDD supports two authentication methods for LLM providers.

### API Key Authentication

The traditional method using provider-issued API keys.

**Configuration**:
```json
{
  "providers": {
    "anthropic": {
      "api_key": "$ANTHROPIC_API_KEY"
    },
    "openai": {
      "api_key": "sk-..."
    }
  }
}
```

**Features**:
- Supports environment variable references (`$VAR`, `${VAR}`)
- Keys are validated during configuration loading
- Providers with invalid/missing keys are automatically skipped

### OAuth2 Authentication (Claude Account)

For Anthropic Claude, CDD supports OAuth2 authentication with PKCE (Proof Key for Code Exchange), enabling users with Claude Account subscriptions to authenticate without managing API keys.

**Implementation**: `internal/oauth/claude/`

#### OAuth2 Flow

```
┌─────────────┐     ┌─────────────┐     ┌─────────────┐
│   CDD    │     │   Browser   │     │  Claude.ai  │
│     CLI     │     │             │     │             │
└──────┬──────┘     └──────┬──────┘     └──────┬──────┘
       │                   │                   │
       │ 1. Generate PKCE  │                   │
       │    verifier +     │                   │
       │    challenge      │                   │
       │                   │                   │
       │ 2. Open auth URL ─┼──────────────────►│
       │                   │                   │
       │                   │ 3. User logs in   │
       │                   │    and authorizes │
       │                   │◄──────────────────│
       │                   │                   │
       │ 4. User pastes    │                   │
       │◄── auth code ─────│                   │
       │                   │                   │
       │ 5. Exchange code ─┼──────────────────►│
       │    + verifier     │                   │
       │                   │                   │
       │◄── 6. Tokens ─────┼──────────────────│
       │                   │                   │
```

#### PKCE Challenge Generation

**Implementation**: `internal/oauth/claude/challenge.go`

```go
func GetChallenge() (verifier, challenge string, err error) {
    bytes := make([]byte, 32)
    rand.Read(bytes)
    verifier = encodeBase64(bytes)
    hash := sha256.Sum256([]byte(verifier))
    challenge = encodeBase64(hash[:])
    return verifier, challenge, nil
}
```

- Generates 32-byte random verifier
- Creates SHA-256 hash for challenge
- Uses URL-safe Base64 encoding (no padding)

#### Authorization URL

**Implementation**: `internal/oauth/claude/oauth.go`

| Parameter | Value |
|-----------|-------|
| Endpoint | `https://claude.ai/oauth/authorize` |
| Client ID | `9d1c250a-e61b-44d9-88ed-5944d1962f5e` |
| Redirect URI | `https://console.anthropic.com/oauth/code/callback` |
| Response Type | `code` |
| Scopes | `org:create_api_key`, `user:profile`, `user:inference` |
| Code Challenge Method | `S256` |

#### Token Exchange

**Implementation**: `internal/oauth/claude/oauth.go`

After the user authorizes and receives a code:

```go
func ExchangeToken(ctx context.Context, code, verifier string) (*oauth.Token, error)
```

- Endpoint: `https://console.anthropic.com/v1/oauth/token`
- Grant type: `authorization_code`
- Returns access token, refresh token, and expiration

#### Token Refresh

**Implementation**: `internal/oauth/claude/oauth.go`

```go
func RefreshToken(ctx context.Context, refreshToken string) (*oauth.Token, error)
```

- Endpoint: `https://console.anthropic.com/v1/oauth/token`
- Grant type: `refresh_token`
- Used when access token expires

### Token Management

**Implementation**: `internal/oauth/token.go`

```go
type Token struct {
    AccessToken  string `json:"access_token"`
    RefreshToken string `json:"refresh_token"`
    ExpiresIn    int    `json:"expires_in"`
    ExpiresAt    int64  `json:"expires_at"`
}
```

**Key methods**:
- `SetExpiresAt()`: Calculates absolute expiration time from `ExpiresIn`
- `IsExpired()`: Returns true if token is expired or within 10% of expiration

**Expiration buffer**: Tokens are considered expired when within 10% of their lifetime remaining, allowing proactive refresh before actual expiration.

**Storage**: OAuth tokens are stored in the config file:
```json
{
  "providers": {
    "anthropic": {
      "oauth": {
        "access_token": "...",
        "refresh_token": "...",
        "expires_in": 3600,
        "expires_at": 1702828800
      }
    }
  }
}
```

---

## Configuration System

### Configuration Loading

Configuration is loaded from multiple sources with cascading precedence (`internal/config/load.go`):

1. **Global config**: `$XDG_CONFIG_HOME/cdd/cdd.json`
2. **Project config**: `cdd.json` or `.cdd.json` (searched upward from cwd)

Project configuration takes precedence over global configuration.

**Loading process**:
1. Load global config (if exists)
2. Find and load project config (searching parent directories)
3. Merge configs (project overrides global)
4. Apply defaults
5. Load provider metadata from catwalk
6. Configure providers (merge user config with catwalk metadata)
7. Configure default model selections

### Configuration Structure

**Top-level config** (`internal/config/config.go`):

```json
{
  "models": {
    "large": { ... },
    "small": { ... }
  },
  "providers": {
    "anthropic": { ... },
    "openai": { ... }
  },
  "options": {
    "debug": false,
    "data_directory": "",
    "context_paths": []
  }
}
```

**Model selection** (`SelectedModel`):

| Field | Type | Description |
|-------|------|-------------|
| `model` | string | Model ID (e.g., "claude-sonnet-4-20250514") |
| `provider` | string | Provider ID matching a key in providers |
| `think` | bool | Enable thinking mode (Anthropic) |
| `reasoning_effort` | string | Reasoning effort (OpenAI) |
| `temperature` | float64 | Sampling temperature (0-1) |
| `top_p` | float64 | Nucleus sampling parameter |
| `top_k` | int64 | Top-k sampling parameter |
| `max_tokens` | int64 | Maximum response tokens |
| `frequency_penalty` | float64 | Reduces repetition |
| `presence_penalty` | float64 | Increases topic diversity |
| `provider_options` | map | Additional provider-specific options |

**Provider configuration** (`ProviderConfig`):

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique provider identifier |
| `name` | string | Human-readable display name |
| `type` | string | Provider type (openai, anthropic, openai-compat) |
| `api_key` | string | Authentication key (supports env vars) |
| `base_url` | string | Custom API endpoint |
| `disable` | bool | Disable this provider |
| `extra_headers` | map | Additional HTTP headers |
| `models` | array | Available models (from catwalk or user) |
| `provider_options` | map | Additional provider-specific options |

### Environment Variable Resolution

The resolver (`internal/config/resolve.go`) expands environment variables in configuration values:

**Supported syntax**:
- `$VAR` - Simple variable reference
- `${VAR}` - Braced variable reference

**Example configuration**:
```json
{
  "providers": {
    "anthropic": {
      "api_key": "$ANTHROPIC_API_KEY"
    },
    "openai": {
      "api_key": "${OPENAI_API_KEY}",
      "base_url": "${OPENAI_BASE_URL}"
    }
  }
}
```

**Behavior**:
- Returns error if referenced variable is not set
- Providers with unresolvable API keys are skipped (not fatal)
- Base URLs fall back to catwalk defaults if not set

### First-Run Detection

**Implementation**: `internal/config/firstrun.go`

CDD detects first-run scenarios to trigger the setup wizard.

```go
func IsFirstRun() bool
func NeedsSetup() bool
```

**`IsFirstRun()` returns true when**:
- No global config file exists at `$XDG_CONFIG_HOME/cdd/cdd.json`
- Config fails to load (e.g., invalid JSON)
- No providers have API keys configured

**`NeedsSetup()` returns true when**:
- Config fails to load
- No models are configured
- Configured models reference invalid/disabled providers

### Configuration Persistence

**Implementation**: `internal/config/save.go`

The save system uses minimal structs to avoid persisting runtime-only data.

**SaveConfig structure**:
```go
type SaveConfig struct {
    Models    map[SelectedModelType]SelectedModel
    Providers map[string]*SaveProviderConfig
    Options   *Options
}

type SaveProviderConfig struct {
    APIKey     string       `json:"api_key,omitempty"`
    OAuthToken *oauth.Token `json:"oauth,omitempty"`
}
```

**Key functions**:

| Function | Purpose |
|----------|---------|
| `Save(cfg)` | Saves to global config path |
| `SaveToFile(cfg, path)` | Saves to specific file |
| `SaveWizardResult(...)` | Saves API key wizard result |
| `SaveWizardResultWithOAuth(...)` | Saves OAuth wizard result |

**What gets saved**:
- Model selections (large/small)
- Provider API keys or OAuth tokens
- Application options

**What doesn't get saved**:
- Catwalk provider metadata
- Resolved API key values
- Runtime state

### Provider Configuration

Example complete configuration:

```json
{
  "models": {
    "large": {
      "model": "claude-sonnet-4-20250514",
      "provider": "anthropic",
      "think": true,
      "max_tokens": 8192
    },
    "small": {
      "model": "claude-3-5-haiku-latest",
      "provider": "anthropic"
    }
  },
  "providers": {
    "anthropic": {
      "api_key": "$ANTHROPIC_API_KEY"
    },
    "openai": {
      "api_key": "$OPENAI_API_KEY"
    }
  },
  "options": {
    "debug": false
  }
}
```

---

## Custom Providers

CDD supports custom provider configurations that extend beyond the built-in catwalk providers. Custom providers allow you to add support for additional LLM services, self-hosted models, or specialized APIs.

### Overview

Custom providers are stored in `$XDG_DATA_HOME/cdd/custom-providers.json` and are merged with catwalk providers during configuration loading. Custom providers can override catwalk providers if they share the same ID.

**Key features**:
- Pre-built templates for common providers (Ollama, LM Studio, OpenRouter, etc.)
- Manual provider definition with full customization
- File-based import/export for sharing configurations
- URL-based import for remote configurations
- Full validation support

**Implementation**: `internal/config/custom_provider.go`, `internal/config/provider_loader.go`, `internal/config/provider_validator.go`

### Provider Templates

CDD includes pre-built templates for popular LLM services:

| Template | Description | Type |
|----------|-------------|------|
| `ollama` | Local Ollama server | openai-compat |
| `lmstudio` | LM Studio GUI | openai-compat |
| `openrouter` | OpenRouter API | openrouter |
| `together` | Together AI | openai-compat |
| `deepseek` | DeepSeek API | openai-compat |
| `groq` | Groq fast inference | openai-compat |
| `anthropic-compatible` | Generic Anthropic API | anthropic |
| `azure-openai` | Azure OpenAI Service | azure |
| `vertexai` | Google Vertex AI | google-vertex |

**Implementation**: `internal/config/provider_templates.go`

### CLI Commands

#### List Providers

```bash
cdd providers list                # List all providers
cdd providers list --custom-only  # List only custom providers
cdd providers list --catwalk-only # List only catwalk providers
```

#### Show Provider Details

```bash
cdd providers show <provider-id>  # Show detailed provider information
```

#### Add Provider from Template

```bash
cdd providers add-template <template-name> [id] [name]

# Examples:
cdd providers add-template ollama
cdd providers add-template ollama my-ollama "My Ollama"
cdd providers add-template openrouter
```

#### Add Provider Manually (Interactive)

```bash
cdd providers add
```

This launches an interactive wizard for creating custom providers.

#### Add Provider from File

```bash
cdd providers add-file <path-to-json>
```

#### Add Provider from URL

```bash
cdd providers add-url <url>
```

#### Remove Provider

```bash
cdd providers remove <provider-id>
```

#### Export Provider Configuration

```bash
cdd providers export <provider-id>          # Print to stdout
cdd providers export <provider-id> -o file  # Write to file
```

#### Validate Provider

```bash
cdd providers validate <provider-id>
```

#### List Templates

```bash
cdd providers templates  # List all available templates
```

### Custom Provider Structure

A custom provider configuration:

```json
{
  "name": "My Custom Provider",
  "id": "my-custom-provider",
  "type": "openai-compat",
  "api_endpoint": "https://api.example.com/v1",
  "default_headers": {
    "X-API-Key": "$MY_API_KEY",
    "X-Custom-Header": "value"
  },
  "default_large_model_id": "model-large",
  "default_small_model_id": "model-small",
  "models": [
    {
      "id": "model-large",
      "name": "Large Model",
      "context_window": 128000,
      "default_max_tokens": 8192,
      "cost_per_1m_in": 1.0,
      "cost_per_1m_out": 2.0
    }
  ],
  "created_at": "2025-12-25T10:00:00Z",
  "updated_at": "2025-12-25T10:00:00Z"
}
```

### Provider Fields

| Field | Type | Required | Description |
|-------|------|----------|-------------|
| `name` | string | Yes | Human-readable display name |
| `id` | string | Yes | Unique provider identifier |
| `type` | string | Yes | Provider type |
| `api_endpoint` | string | No | API endpoint URL |
| `base_url` | string | No | Fallback base URL (deprecated) |
| `default_headers` | map | No | Default HTTP headers |
| `default_large_model_id` | string | No | Default large model ID |
| `default_small_model_id` | string | No | Default small model ID |
| `models` | array | No | Available models |
| `created_at` | string | Auto | Creation timestamp |
| `updated_at` | string | Auto | Last update timestamp |

### Supported Provider Types

- `anthropic` - Native Anthropic API
- `openai` - Native OpenAI API
- `openai-compat` - OpenAI-compatible APIs
- `google` - Google AI API
- `azure` - Azure OpenAI Service
- `bedrock` - AWS Bedrock
- `google-vertex` - Google Vertex AI
- `openrouter` - OpenRouter API

### Model Fields

| Field | Type | Description |
|-------|------|-------------|
| `id` | string | Unique model identifier |
| `name` | string | Human-readable model name |
| `context_window` | int | Maximum context length |
| `default_max_tokens` | int | Default max tokens |
| `cost_per_1m_in` | float | Input cost per 1M tokens |
| `cost_per_1m_out` | float | Output cost per 1M tokens |
| `cost_per_1m_in_cached` | float | Cached input cost per 1M tokens |
| `cost_per_1m_out_cached` | float | Cached output cost per 1M tokens |

### Validation

Custom providers are validated before being added:

**Required fields**: name, id, type
**Valid types**: Must match supported provider types
**URL validation**: api_endpoint must be a valid HTTP/HTTPS URL if provided
**Duplicate IDs**: Provider IDs must be unique
**Duplicate model IDs**: Model IDs within a provider must be unique

**Implementation**: `internal/config/provider_validator.go`

### Provider Merging

Custom providers are merged with catwalk providers during configuration loading:

1. Catwalk providers are loaded first
2. Custom providers are loaded from `custom-providers.json`
3. Custom providers override catwalk providers with matching IDs
4. Final provider list is returned to the application

**Implementation**: `internal/config/provider_loader.go`

### Wizard Integration

Custom providers can be added through the TUI wizard:

1. Run `cdd` or `cdd config`
2. Select "➕ Add Custom Provider" from the provider list
3. Choose import method:
   - **Manual**: Define provider step-by-step
   - **From URL**: Import from a URL
   - **From File**: Import from a local file
4. Configure provider details
5. Add models (optional)
6. Confirm and save

**Implementation**: `internal/tui/components/wizard/custom_provider_*.go`

### Example: Adding an Ollama Provider

**Using CLI**:
```bash
cdd providers add-template ollama
```

**Using Wizard**:
```bash
cdd
# Select "➕ Add Custom Provider"
# Select "Manual"
# Enter provider details or use template
```

**Manual Configuration**:
```json
{
  "name": "Ollama",
  "id": "ollama",
  "type": "openai-compat",
  "api_endpoint": "http://localhost:11434/v1",
  "default_large_model_id": "qwen2.5:32b",
  "default_small_model_id": "qwen2.5:7b",
  "models": [
    {
      "id": "qwen2.5:32b",
      "name": "Qwen 2.5 32B",
      "context_window": 32768
    },
    {
      "id": "qwen2.5:7b",
      "name": "Qwen 2.5 7B",
      "context_window": 32768
    }
  ]
}
```

### File Storage

Custom providers are stored in:
```
~/.local/share/cdd/custom-providers.json
```

The file contains an array of custom provider configurations:

```json
[
  {
    "name": "Ollama",
    "id": "ollama",
    ...
  },
  {
    "name": "LM Studio",
    "id": "lmstudio",
    ...
  }
]
```

---

## Quality Tooling

### GitHub Actions CI

The CI workflow (`.github/workflows/ci.yml`) runs on push/PR to main:

| Job | Description |
|-----|-------------|
| **lint** | Runs golangci-lint with 5-minute timeout |
| **test** | Runs tests with race detection and coverage |
| **build** | Builds all packages |
| **security** | Runs govulncheck for vulnerability scanning |

**Features**:
- Go 1.25 support
- Race condition detection
- Code coverage with Codecov integration
- Security vulnerability scanning

### golangci-lint Configuration

The `.golangci.yml` enables 26 linters organized by category:

**Default linters**:
- `errcheck`: Unchecked errors
- `govet`: Suspicious constructs
- `ineffassign`: Unused assignments
- `staticcheck`: Static analysis
- `unused`: Unused code detection

**Additional linters**:
- `bodyclose`: HTTP response body closure
- `gosec`: Security issues
- `gocritic`: Opinionated checks
- `gocyclo`: Cyclomatic complexity (max 15)
- `dupl`: Code duplication (threshold 100)
- `errorlint`: Error wrapping issues
- `exhaustive`: Enum switch exhaustiveness
- And more...

**Exclusions**:
- Test files: Relaxed rules for dupl, gocyclo, gosec, errcheck
- Main files: Allow init functions

### Taskfile

The `Taskfile.yaml` provides common development commands:

**Development**:
```bash
task build          # Build binary with version info
task run            # Run application
task install        # Install to $GOPATH/bin
```

**Quality**:
```bash
task fmt            # Format code (gofmt + goimports)
task lint           # Run golangci-lint
task lint:fix       # Run linters with auto-fix
task vet            # Run go vet
```

**Testing**:
```bash
task test           # Run tests with race detection
task test:coverage  # Run tests with coverage report
task test:short     # Run short tests only
```

**Other**:
```bash
task security       # Run govulncheck
task deps           # Download and tidy dependencies
task deps:update    # Update all dependencies
task clean          # Remove build artifacts
task check          # Run all checks (fmt, lint, vet, test)
task ci             # Run CI checks locally
```

**Build information**: Version, commit, and build date are injected via ldflags:
```go
-X github.com/guilhermegouw/cdd/cmd.Version={{.VERSION}}
-X github.com/guilhermegouw/cdd/cmd.Commit={{.COMMIT}}
-X github.com/guilhermegouw/cdd/cmd.BuildDate={{.BUILD_DATE}}
```

---

## Test Coverage

The provider system has comprehensive test coverage:

| Package | Coverage | Key Test Areas |
|---------|----------|----------------|
| `internal/config` | 88.4% | Config loading, merging, resolution, provider configuration |
| `internal/provider` | 98.8% | Builder, model creation, provider caching, tier selection |

**Test files**:
- `internal/config/config_test.go`: Config struct methods
- `internal/config/load_test.go`: Configuration loading and merging
- `internal/config/providers_test.go`: Provider metadata loading and caching
- `internal/config/resolve_test.go`: Environment variable resolution
- `internal/config/custom_provider_test.go`: Custom provider management tests
- `internal/config/provider_loader_test.go`: Provider loading and merging tests
- `internal/config/provider_validator_test.go`: Provider validation tests
- `internal/provider/provider_test.go`: Provider building and model creation
- `internal/provider/tier_test.go`: Tier selection and validation

**Running tests**:
```bash
task test           # All tests
task test:coverage  # With coverage report
go test -v ./internal/config/...    # Config package only
go test -v ./internal/provider/...  # Provider package only
```

---

## CLI Commands

### Root Command

```bash
cdd              # Shows help
cdd --help       # Detailed help
```

### Version Command

```bash
cdd version
# Output:
# cdd v0.1.0
#   commit: abc1234
#   built:  2025-12-17T12:00:00Z
```

---

## File Structure

```
cdd/
├── cmd/
│   ├── root.go           # Root cobra command
│   ├── version.go        # Version command with build info
│   └── providers.go      # Provider management commands
├── internal/
│   ├── config/
│   │   ├── config.go         # Config structures and types
│   │   ├── firstrun.go       # First-run detection
│   │   ├── load.go           # Configuration loading logic
│   │   ├── providers.go      # Catwalk provider integration
│   │   ├── resolve.go        # Environment variable resolver
│   │   ├── save.go           # Configuration persistence
│   │   ├── custom_provider.go      # Custom provider management
│   │   ├── provider_loader.go      # Provider loading and merging
│   │   ├── provider_validator.go   # Provider validation
│   │   ├── provider_templates.go   # Pre-built provider templates
│   │   └── *_test.go         # Test files
│   ├── oauth/
│   │   ├── token.go      # OAuth token struct
│   │   └── claude/
│   │       ├── challenge.go  # PKCE verifier/challenge
│   │       └── oauth.go      # Claude OAuth2 implementation
│   ├── provider/
│   │   ├── provider.go   # Provider builder and model creation
│   │   └── tier.go       # Tier selection utilities
│   └── tui/              # Terminal UI (see tui-wizard.md)
│       ├── tui.go
│       ├── keys.go
│       ├── page/
│       ├── util/
│       ├── styles/
│       └── components/
│           └── wizard/
│               ├── custom_provider_method.go    # Import method selection
│               ├── custom_provider_define.go    # Manual provider definition
│               └── custom_provider_models.go    # Model configuration
├── .github/
│   └── workflows/
│       └── ci.yml        # GitHub Actions workflow
├── .golangci.yml         # Linter configuration
├── Taskfile.yaml         # Development tasks
└── main.go               # Application entry point
```

---

## Dependencies

Key external dependencies:

| Package | Purpose |
|---------|---------|
| `github.com/spf13/cobra` | CLI framework |
| `github.com/adrg/xdg` | XDG Base Directory support |
| `github.com/charmbracelet/catwalk` | Provider metadata |
| `charm.land/fantasy` | LLM orchestration |
| `charm.land/fantasy/providers/anthropic` | Anthropic provider |
| `charm.land/fantasy/providers/openai` | OpenAI provider |
| `charm.land/bubbletea/v2` | Terminal UI framework |
| `charm.land/bubbles/v2` | TUI components |
| `charm.land/lipgloss/v2` | Terminal styling |

---

## Related Documentation

- [TUI and Setup Wizard](./tui-wizard.md) - Terminal UI and first-run wizard documentation
