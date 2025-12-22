# Config Module

The config module provides configuration management for CDD CLI. It handles loading, merging, and saving configuration from multiple sources with environment variable resolution.

## Overview

| Aspect | Details |
|--------|---------|
| Location | `internal/config/` |
| Files | 6 source files (~900 lines total) |
| Purpose | Configuration management |
| Config Format | JSON |

## Package Structure

```
internal/config/
├── config.go      - Core types: Config, ProviderConfig, SelectedModel
├── load.go        - Load and merge config from files
├── save.go        - Save config to disk
├── firstrun.go    - Detect first run / needs setup
├── providers.go   - Fetch/cache provider metadata
└── resolve.go     - Resolve $ENV_VAR in config values
```

## Data Types

```mermaid
classDiagram
    class Config {
        +Models map~SelectedModelType~SelectedModel
        +Providers map~string~ProviderConfig
        +Options *Options
        -knownProviders []Provider
        +NewConfig() Config
        +GetModel(providerID, modelID) Model
        +KnownProviders() []Provider
        +SetKnownProviders(providers)
        +RefreshOAuthToken(ctx, providerID) error
    }

    class SelectedModelType {
        <<enumeration>>
        large
        small
    }

    class SelectedModel {
        +Model string
        +Provider string
        +Temperature *float64
        +MaxTokens int64
        +Think bool
        ...
    }

    class ProviderConfig {
        +ID string
        +Name string
        +Type string
        +APIKey string
        +BaseURL string
        +OAuthToken *Token
        +Models []Model
        +ExtraHeaders map
        +Disable bool
        +SetupClaudeCode()
    }

    class Options {
        +ContextPaths []string
        +DataDir string
        +Debug bool
    }

    Config --> SelectedModelType
    Config --> SelectedModel
    Config --> ProviderConfig
    Config --> Options
```

## Load Flow

The `Load()` function is the main entry point, called from `cmd/root.go`.

```mermaid
flowchart TD
    A[Load called] --> B[Create empty Config]
    B --> C[Load global config]
    C --> D{~/.config/cdd/cdd.json exists?}
    D -->|Yes| E[Parse JSON into Config]
    D -->|No| F[Continue with empty]
    E --> F
    F --> G[Find project config]
    G --> H{cdd.json or .cdd.json found?}
    H -->|Yes| I[Load and merge project config]
    H -->|No| J[Continue]
    I --> J
    J --> K[Apply defaults]
    K --> L[Load providers from catwalk]
    L --> M[Configure providers]
    M --> N[Configure default models]
    N --> O[Return Config]
```

## Config Sources

```mermaid
flowchart LR
    subgraph "Config Sources"
        G[Global Config<br>~/.config/cdd/cdd.json]
        P[Project Config<br>./cdd.json or ./.cdd.json]
        E[Environment Variables<br>$OPENAI_API_KEY, etc.]
        C[Catwalk API<br>Provider metadata]
    end

    subgraph "Merge Order"
        G --> M[Merged Config]
        P --> M
        E --> M
        C --> M
    end

    M --> F[Final Config]

    style P fill:#90EE90
    note1[Project config takes precedence]
```

## Project Config Search

The `findProjectConfig()` function searches upward from the current directory:

```mermaid
flowchart TD
    A[Start at cwd] --> B{cdd.json exists?}
    B -->|Yes| C[Return path]
    B -->|No| D{.cdd.json exists?}
    D -->|Yes| C
    D -->|No| E{At root?}
    E -->|Yes| F[Return empty]
    E -->|No| G[Move to parent dir]
    G --> B
```

## Provider Loading

```mermaid
sequenceDiagram
    participant L as Load()
    participant P as LoadProviders()
    participant API as Catwalk API
    participant Cache as Local Cache
    participant Embedded as Embedded Data

    L->>P: LoadProviders(cfg)
    P->>API: Fetch providers
    alt API Success
        API-->>P: Provider list
        P->>Cache: Save to cache
        P-->>L: Return providers
    else API Failure
        P->>Cache: Try load cache
        alt Cache valid (< 24h)
            Cache-->>P: Cached providers
            P-->>L: Return providers
        else Cache invalid/missing
            P->>Embedded: Get embedded
            Embedded-->>P: Fallback providers
            P-->>L: Return providers
        end
    end
```

## Environment Variable Resolution

The `Resolver` expands environment variables in config values:

```mermaid
flowchart LR
    A["$OPENAI_API_KEY"] --> R[Resolver]
    B["${ANTHROPIC_API_KEY}"] --> R
    C["https://api.example.com"] --> R

    R --> D["sk-abc123..."]
    R --> E["sk-ant-..."]
    R --> F["https://api.example.com"]

    style A fill:#FFE4B5
    style B fill:#FFE4B5
    style C fill:#98FB98
```

**Supported syntax:**
- `$VAR` - Simple variable
- `${VAR}` - Braced variable

## Config Merge Strategy

When both global and project configs exist:

```mermaid
flowchart TD
    subgraph Global["Global Config"]
        GM[Models: large=gpt-4]
        GP[Providers: openai]
        GO[Options: debug=false]
    end

    subgraph Project["Project Config"]
        PM[Models: large=claude-opus]
        PP[Providers: anthropic]
        PO[Options: debug=true]
    end

    subgraph Merged["Merged Result"]
        MM[Models: large=claude-opus]
        MP[Providers: openai, anthropic]
        MO[Options: debug=true]
    end

    GM --> MM
    PM --> MM
    GP --> MP
    PP --> MP
    GO --> MO
    PO --> MO

    style PM fill:#90EE90
    style PP fill:#90EE90
    style PO fill:#90EE90
```

**Rule:** Project config values override global config values.

## Save Flow

```mermaid
flowchart TD
    A[Save called] --> B[Create SaveConfig]
    B --> C[Copy Models]
    C --> D[Copy minimal Provider info]
    D --> E[Copy Options]
    E --> F[Marshal to JSON]
    F --> G[Ensure directory exists]
    G --> H[Write to file]
    H --> I[Done]

    subgraph "What gets saved"
        J[API key templates: $OPENAI_API_KEY]
        K[OAuth tokens]
        L[Model selections]
        M[Options]
    end

    subgraph "What is NOT saved"
        N[Resolved API keys]
        O[Provider metadata]
        P[Computed headers]
    end
```

## First Run Detection

```mermaid
flowchart TD
    A[IsFirstRun called] --> B{Config file exists?}
    B -->|No| C[Return true]
    B -->|Yes| D[Load config]
    D --> E{Load successful?}
    E -->|No| F[Return true]
    E -->|Yes| G{Any providers with API keys?}
    G -->|No| H[Return true]
    G -->|Yes| I[Return false]
```

## File Locations

| File | Path | Purpose |
|------|------|---------|
| Global config | `~/.config/cdd/cdd.json` | User-wide settings |
| Project config | `./cdd.json` or `./.cdd.json` | Project-specific overrides |
| Provider cache | `~/.local/share/cdd/providers.json` | Cached catwalk data |
| Data directory | `~/.local/share/cdd/` | App data storage |

## Usage Examples

### Basic Config File

```json
{
  "providers": {
    "anthropic": {
      "api_key": "$ANTHROPIC_API_KEY"
    }
  },
  "models": {
    "large": {
      "provider": "anthropic",
      "model": "claude-sonnet-4-20250514"
    },
    "small": {
      "provider": "anthropic",
      "model": "claude-haiku-3-5-20241022"
    }
  }
}
```

### Multi-Provider Config

```json
{
  "providers": {
    "anthropic": {
      "api_key": "$ANTHROPIC_API_KEY"
    },
    "openai": {
      "api_key": "$OPENAI_API_KEY"
    }
  },
  "models": {
    "large": {
      "provider": "anthropic",
      "model": "claude-opus-4-5-20251101"
    },
    "small": {
      "provider": "openai",
      "model": "gpt-4o-mini"
    }
  }
}
```

## API Reference

### Load Functions

| Function | Purpose |
|----------|---------|
| `Load()` | Load config from standard locations |
| `LoadFromFile(path)` | Load config from specific file |
| `LoadProviders(cfg)` | Fetch provider metadata |

### Save Functions

| Function | Purpose |
|----------|---------|
| `Save(cfg)` | Save to global config path |
| `SaveToFile(cfg, path)` | Save to specific path |
| `SaveWizardResult(...)` | Save setup wizard result (API key) |
| `SaveWizardResultWithOAuth(...)` | Save setup wizard result (OAuth) |

### Check Functions

| Function | Purpose |
|----------|---------|
| `IsFirstRun()` | Check if this is first run |
| `NeedsSetup()` | Check if setup is incomplete |

### Utility Functions

| Function | Purpose |
|----------|---------|
| `GlobalConfigPath()` | Get global config file path |
| `DefaultDataDir()` | Get default data directory |
| `NewResolver()` | Create environment resolver |

## Design Decisions

1. **Two-tier model selection**: "large" for complex tasks, "small" for fast/cheap tasks
2. **Environment variable support**: API keys stored as `$VAR` templates, resolved at load time
3. **Project config override**: Allows per-project customization
4. **Fallback chain for providers**: API → Cache → Embedded ensures offline functionality
5. **Minimal save format**: Only saves what's needed, not runtime-computed values
6. **XDG compliance**: Uses standard Linux directory locations
