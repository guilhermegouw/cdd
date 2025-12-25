package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/guilhermegouw/cdd/internal/config"
	"github.com/guilhermegouw/cdd/internal/oauth"
)

func newStatusCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "status",
		Short: "Show current session info, model, and configuration",
		Long: `Display the current CDD status including:
  - Current working directory
  - Configured provider and model
  - OAuth token status (if applicable)
  - Session information`,
		RunE: runStatus,
	}
}

func runStatus(cmd *cobra.Command, _ []string) error {
	// Check if first run
	if config.IsFirstRun() {
		fmt.Println("Status: Not configured")
		fmt.Println("")
		fmt.Println("Run 'cdd' to start the setup wizard.")
		return nil
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Get working directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("getting working directory: %w", err)
	}

	// Print header
	fmt.Println("CDD Status")
	fmt.Println(strings.Repeat("─", 40))
	fmt.Println()

	// Working directory
	fmt.Printf("Working Directory: %s\n", cwd)
	fmt.Println()

	// Model configuration
	fmt.Println("Model Configuration:")
	printModelConfig(cfg, config.SelectedModelTypeLarge, "  Large")
	printModelConfig(cfg, config.SelectedModelTypeSmall, "  Small")
	fmt.Println()

	// Provider status
	fmt.Println("Providers:")
	if len(cfg.Providers) == 0 {
		fmt.Println("  No providers configured")
	} else {
		for id, provider := range cfg.Providers {
			printProviderStatus(id, provider)
		}
	}
	fmt.Println()

	// Config file location
	fmt.Printf("Config File: %s\n", config.GlobalConfigPath())

	return nil
}

func printModelConfig(cfg *config.Config, tier config.SelectedModelType, label string) {
	model, ok := cfg.Models[tier]
	if !ok {
		fmt.Printf("%s: (not configured)\n", label)
		return
	}
	fmt.Printf("%s: %s (%s)\n", label, model.Model, model.Provider)
}

func printProviderStatus(id string, provider *config.ProviderConfig) {
	name := provider.Name
	if name == "" {
		name = id
	}

	status := "API Key"
	if provider.OAuthToken != nil {
		status = getOAuthStatus(provider.OAuthToken)
	} else if provider.APIKey == "" {
		status = "Not configured"
	}

	if provider.Disable {
		status = "Disabled"
	}

	fmt.Printf("  %s: %s\n", name, status)
}

func getOAuthStatus(token *oauth.Token) string {
	if token == nil {
		return "Not authenticated"
	}

	if token.ExpiresAt == 0 {
		return "OAuth (no expiry)"
	}

	expiresAt := time.Unix(token.ExpiresAt, 0)
	remaining := time.Until(expiresAt)
	if remaining <= 0 {
		return "OAuth (expired)"
	}

	if remaining < time.Hour {
		return fmt.Sprintf("OAuth (expires in %d minutes)", int(remaining.Minutes()))
	}

	return fmt.Sprintf("OAuth (expires in %s)", formatDuration(remaining))
}

func formatDuration(d time.Duration) string {
	if d < time.Hour {
		return fmt.Sprintf("%d minutes", int(d.Minutes()))
	}
	hours := int(d.Hours())
	if hours < 24 {
		return fmt.Sprintf("%d hours", hours)
	}
	days := hours / 24
	return fmt.Sprintf("%d days", days)
}
