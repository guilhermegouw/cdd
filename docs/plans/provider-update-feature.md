# Provider Update Feature for CDD

## Problem
CDD's provider list appears smaller than Crush's because:
- CDD prioritizes cached data (24-hour TTL)
- No manual update mechanism
- Less aggressive about fetching fresh data

## Solution: Add Provider Update Commands

### 1. Add CLI Command
**File**: `cmd/update_providers.go`

```go
var updateProvidersCmd = &cobra.Command{
	Use:   "update-providers [path-or-url]",
	Short: "Update providers from catwalk or custom source",
	Long: `Update the list of providers from catwalk.charm.sh or custom source.
	
Examples:
  cdd update-providers                    # Update from catwalk.charm.sh
  cdd update-providers https://custom.com/ # Update from custom URL
  cdd update-providers ./local-providers.json # Update from local file
  cdd update-providers embedded           # Reset to embedded providers
`,
	RunE: func(cmd *cobra.Command, args []string) error {
		var source string
		if len(args) > 0 {
			source = args[0]
		}
		return config.UpdateProviders(source)
	},
}
```

### 2. Enhanced Provider Loading
**File**: `internal/config/providers.go`

Add equivalent of Crush's `UpdateProviders` function:

```go
// UpdateProviders fetches and caches provider metadata from the given source.
// Source can be "embedded", an HTTP URL, or a local file path.
func UpdateProviders(source string) error {
	var providers []catwalk.Provider
	var err error

	switch {
	case source == "embedded":
		providers = embedded.GetAll()
	case len(source) > 4 && source[:4] == "http":
		client := catwalk.NewWithURL(source)
		providers, err = client.GetProviders()
		if err != nil {
			return err
		}
	default:
		// Load from local file.
		data, err := os.ReadFile(source)
		if err != nil {
			return err
		}
		if err := json.Unmarshal(data, &providers); err != nil {
			return err
		}
	}

	dataDir := DefaultDataDir()
	cachePath := filepath.Join(dataDir, providersCacheFile)
	return saveProvidersCache(cachePath, providers)
}
```

### 3. Add Command to Root
**File**: `cmd/root.go`

```go
cmd.AddCommand(newUpdateProvidersCmd())
```

### 4. Update Provider Loading Logic
**File**: `internal/config/providers.go`

Make the provider loading more aggressive (like Crush):

```go
func LoadProviders(cfg *Config) ([]catwalk.Provider, error) {
	dataDir := cfg.DataDir()
	cachePath := filepath.Join(dataDir, providersCacheFile)

	// Always try to fetch fresh data first (like Crush)
	catwalkURL := os.Getenv("CATWALK_URL")
	if catwalkURL == "" {
		catwalkURL = defaultCatwalkURL
	}

	client := catwalk.NewWithURL(catwalkURL)
	providers, err := client.GetProviders()
	if err == nil {
		// Success! Save to cache and return
		if cacheErr := saveProvidersCache(cachePath, providers); cacheErr != nil {
			_ = cacheErr // Non-fatal
		}
		return providers, nil
	}

	// Fetch failed, try cache
	if cache, err := loadProvidersCache(cachePath); err == nil {
		if time.Since(cache.UpdatedAt) < cacheMaxAge {
			return cache.Providers, nil
		}
	}

	// All else failed, use embedded
	return embedded.GetAll(), nil
}
```

## Benefits

1. **Fresh Data**: Users can manually update to get latest models
2. **Enterprise Ready**: Custom provider sources for organizations  
3. **Debugging**: Force refresh when troubleshooting provider issues
4. **Consistency**: Match Crush's provider management capabilities
5. **Transparency**: Clear indication when provider data is updated

## Usage Examples

```bash
# Get latest providers from catwalk
cdd update-providers

# Use custom provider catalog
cdd update-providers https://company.com/providers.json

# Reset to embedded providers
cdd update-providers embedded

# Check current provider status
cdd status
```

## Implementation Priority

**High Priority**:
- Add `cdd update-providers` command
- Match Crush's update behavior
- Test with latest catwalk data

**Medium Priority**:
- Add provider update status to `cdd status`
- Add last-update timestamp display
- Better error messages for update failures

This will bring CDD's provider management in line with Crush's capabilities.
