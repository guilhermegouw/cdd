// Package cmd provides the CLI commands for CDD.
package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/charmbracelet/catwalk/pkg/catwalk"
	"github.com/guilhermegouw/cdd/internal/config"
)

// newProvidersCmd creates the providers command group.
func newProvidersCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "providers",
		Short: "Manage custom AI providers",
		Long: `Manage custom AI providers for CDD.

Custom providers allow you to use your own AI models or
enterprise LLMs that are not available through the default catwalk service.

Examples:
  cdd providers list              List all providers (catwalk + custom)
  cdd providers show <provider-id> Show provider details
  cdd providers add               Add a custom provider interactively
  cdd providers add-template ollama  Add from a pre-built template
  cdd providers add-file providers.json  Import from file
  cdd providers add-url <url>      Import from URL
  cdd providers remove my-provider  Remove a custom provider
  cdd providers export providers.json  Export custom providers to file
  cdd providers validate           Validate custom provider configurations
  cdd providers templates           List available templates`,
	}

	cmd.AddCommand(newProvidersListCmd())
	cmd.AddCommand(newProvidersShowCmd())
	cmd.AddCommand(newProvidersAddCmd())
	cmd.AddCommand(newProvidersAddTemplateCmd())
	cmd.AddCommand(newProvidersAddFileCmd())
	cmd.AddCommand(newProvidersAddURLCmd())
	cmd.AddCommand(newProvidersRemoveCmd())
	cmd.AddCommand(newProvidersExportCmd())
	cmd.AddCommand(newProvidersValidateCmd())
	cmd.AddCommand(newProvidersTemplatesCmd())

	return cmd
}

// newProvidersListCmd lists all providers (catwalk + custom).
func newProvidersListCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "list",
		Short: "List all providers",
		Long:  `List all available providers including both catwalk providers and custom providers.`,
		RunE:  runProvidersList,
	}

	cmd.Flags().Bool("catwalk-only", false, "Show only catwalk providers")
	cmd.Flags().Bool("custom-only", false, "Show only custom providers")

	return cmd
}

// runProvidersList executes the providers list command.
func runProvidersList(cmd *cobra.Command, args []string) error {
	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	catwalkOnly, _ := cmd.Flags().GetBool("catwalk-only")
	customOnly, _ := cmd.Flags().GetBool("custom-only")

	// Get catwalk providers.
	loader := config.NewProviderLoader(cfg.DataDir())
	allProviders, err := loader.LoadAllProviders(cfg)
	if err != nil {
		return fmt.Errorf("loading providers: %w", err)
	}

	// Get custom providers.
	customManager := loader.GetCustomProviderManager()
	customProviders, err := customManager.Load()
	if err != nil {
		return fmt.Errorf("loading custom providers: %w", err)
	}

	// Print header.
	fmt.Println("Available Providers:")
	fmt.Println()

	if !customOnly {
		// Print catwalk providers.
		catwalkCount := 0
		for _, p := range allProviders {
			isCustom := false
			for _, cp := range customProviders {
				if cp.ID == string(p.ID) {
					isCustom = true
					break
				}
			}
			if isCustom {
				continue
			}
			catwalkCount++
			fmt.Printf("  %s (%s)\n", p.Name, p.ID)
			if p.DefaultLargeModelID != "" {
				fmt.Printf("    Large: %s\n", p.DefaultLargeModelID)
			}
			if p.DefaultSmallModelID != "" {
				fmt.Printf("    Small: %s\n", p.DefaultSmallModelID)
			}
		}
		if catwalkCount > 0 {
			fmt.Printf("\nCatwalk providers: %d\n", catwalkCount)
		}
	}

	if !catwalkOnly {
		if len(customProviders) > 0 {
			if !customOnly {
				fmt.Println()
			}
			for _, cp := range customProviders {
				fmt.Printf("  %s (%s) [Custom]\n", cp.Name, cp.ID)
				if cp.DefaultLargeModelID != "" {
					fmt.Printf("    Large: %s\n", cp.DefaultLargeModelID)
				}
				if cp.DefaultSmallModelID != "" {
					fmt.Printf("    Small: %s\n", cp.DefaultSmallModelID)
				}
				fmt.Printf("    API: %s\n", cp.APIEndpoint)
				fmt.Printf("    Models: %d\n", len(cp.Models))
			}
			fmt.Printf("\nCustom providers: %d\n", len(customProviders))
		} else {
			fmt.Println("\nNo custom providers configured.")
			fmt.Println("Run 'cdd providers add' or 'cdd providers add-template <template>' to add a custom provider.")
		}
	}

	return nil
}

// newProvidersShowCmd shows details of a specific provider.
func newProvidersShowCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "show <provider-id>",
		Short: "Show provider details",
		Long:  `Show detailed information about a specific provider including its models.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runProvidersShow,
	}

	return cmd
}

// runProvidersShow executes the providers show command.
func runProvidersShow(cmd *cobra.Command, args []string) error {
	providerID := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	loader := config.NewProviderLoader(cfg.DataDir())
	allProviders, err := loader.LoadAllProviders(cfg)
	if err != nil {
		return fmt.Errorf("loading providers: %w", err)
	}

	// Find the provider.
	var foundProvider *catwalk.Provider
	for i := range allProviders {
		if string(allProviders[i].ID) == providerID {
			foundProvider = &allProviders[i]
			break
		}
	}

	if foundProvider == nil {
		return fmt.Errorf("provider %q not found", providerID)
	}

	// Display provider details.
	fmt.Printf("Provider: %s\n", foundProvider.Name)
	fmt.Printf("ID: %s\n", foundProvider.ID)
	fmt.Printf("Type: %s\n", foundProvider.Type)
	fmt.Printf("API Endpoint: %s\n", foundProvider.APIEndpoint)

	// Check if it's a custom provider.
	manager := loader.GetCustomProviderManager()
	customProvider, err := manager.Get(providerID)
	isCustom := err == nil

	if isCustom {
		fmt.Println("[Custom Provider]")
		if len(customProvider.DefaultHeaders) > 0 {
			fmt.Println("\nDefault Headers:")
			for k, v := range customProvider.DefaultHeaders {
				fmt.Printf("  %s: %s\n", k, v)
			}
		}
	}

	// Display models.
	fmt.Printf("\nModels (%d):\n", len(foundProvider.Models))
	for _, model := range foundProvider.Models {
		fmt.Printf("  %s\n", model.Name)
		fmt.Printf("    ID: %s\n", model.ID)
		fmt.Printf("    Context: %d tokens\n", model.ContextWindow)
		if model.DefaultMaxTokens > 0 {
			fmt.Printf("    Max Tokens: %d\n", model.DefaultMaxTokens)
		}
		if model.CostPer1MIn > 0 || model.CostPer1MOut > 0 {
			fmt.Printf("    Cost: $%.2f / 1M in, $%.2f / 1M out\n", model.CostPer1MIn, model.CostPer1MOut)
		}
	}

	// Display default models.
	if foundProvider.DefaultLargeModelID != "" {
		fmt.Printf("\nDefault Large Model: %s\n", foundProvider.DefaultLargeModelID)
	}
	if foundProvider.DefaultSmallModelID != "" {
		fmt.Printf("Default Small Model: %s\n", foundProvider.DefaultSmallModelID)
	}

	return nil
}

// newProvidersAddCmd adds a new custom provider interactively.
func newProvidersAddCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add",
		Short: "Add a custom provider",
		Long:  `Add a new custom provider through an interactive wizard.`,
		RunE:  runProvidersAdd,
	}

	return cmd
}

// runProvidersAdd executes the providers add command.
func runProvidersAdd(cmd *cobra.Command, args []string) error {
	fmt.Println("To add a custom provider, use one of the following methods:")
	fmt.Println()
	fmt.Println("1. Use a pre-built template:")
	fmt.Println("   cdd providers templates")
	fmt.Println("   cdd providers add-template ollama")
	fmt.Println()
	fmt.Println("2. Import from a JSON file:")
	fmt.Println("   cdd providers add-file ./my-providers.json")
	fmt.Println()
	fmt.Println("3. Import from a URL:")
	fmt.Println("   cdd providers add-url https://example.com/providers.json")
	fmt.Println()
	fmt.Println("4. Use the interactive setup wizard:")
	fmt.Println("   cdd")
	fmt.Println("   (select '➕ Add Custom Provider')")

	return nil
}

// newProvidersAddTemplateCmd adds a provider from a template.
func newProvidersAddTemplateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-template <template-name>",
		Short: "Add a provider from a template",
		Long: `Add a custom provider from a pre-built template.

Available templates: ollama, lmstudio, openrouter, together, deepseek, groq, anthropic-compatible, azure-openai, vertexai`,
		Args: cobra.ExactArgs(1),
		RunE: runProvidersAddTemplate,
	}

	cmd.Flags().String("id", "", "Custom provider ID (defaults to template ID)")
	cmd.Flags().String("name", "", "Custom provider name (defaults to template name)")
	cmd.Flags().StringSlice("var", []string{}, "Template variables (format: key=value)")

	return cmd
}

// runProvidersAddTemplate executes the add-template command.
func runProvidersAddTemplate(cmd *cobra.Command, args []string) error {
	templateName := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Get the template.
	template, ok := config.GetTemplate(templateName)
	if !ok {
		return fmt.Errorf("unknown template %q", templateName)
	}

	// Parse flags.
	customID, _ := cmd.Flags().GetString("id")
	customName, _ := cmd.Flags().GetString("name")
	varValues, _ := cmd.Flags().GetStringSlice("var")

	// Parse variable values.
	vars := make(map[string]string)
	for _, v := range varValues {
		parts := strings.SplitN(v, "=", 2)
		if len(parts) == 2 {
			vars[parts[0]] = parts[1]
		}
	}

	// Apply default values for unset variables.
	for _, tv := range template.Variables {
		if _, ok := vars[tv.Name]; !ok && tv.DefaultValue != "" {
			vars[tv.Name] = tv.DefaultValue
		}
	}

	// Create provider from template.
	customProvider := template.ToCustomProvider(vars, customID, customName)

	// Validate the provider.
	existingProviders := getExistingProviderIDs(cfg)
	result := config.ValidateCustomProvider(&customProvider, existingProviders)

	if !result.IsValid {
		fmt.Printf("Validation failed:\n")
		for _, e := range result.Errors {
			fmt.Printf("  - %s\n", e)
		}
		return fmt.Errorf("provider validation failed")
	}

	// Add the provider.
	loader := config.NewProviderLoader(cfg.DataDir())
	manager := loader.GetCustomProviderManager()

	if err := manager.Add(customProvider); err != nil {
		return fmt.Errorf("adding provider: %w", err)
	}

	fmt.Printf("Added provider: %s (%s)\n", customProvider.Name, customProvider.ID)
	fmt.Printf("\nNext steps:\n")
	fmt.Printf("  1. Set your API key in cdd.json:\n")
	fmt.Printf("     cdd providers show %s\n", customProvider.ID)
	fmt.Printf("  2. Or set environment variable and run cdd\n")

	return nil
}

// newProvidersAddFileCmd adds providers from a JSON file.
func newProvidersAddFileCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-file <file-path>",
		Short: "Add providers from a JSON file",
		Long:  `Import custom providers from a JSON file. The file should match the custom-providers.json format.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runProvidersAddFile,
	}

	return cmd
}

// runProvidersAddFile executes the add-file command.
func runProvidersAddFile(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	// Read the file.
	data, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("reading file: %w", err)
	}

	// Parse the JSON.
	var file config.CustomProvidersFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	if len(file.Providers) == 0 {
		return fmt.Errorf("no providers found in file")
	}

	// Get existing provider IDs for validation.
	existingProviders := getExistingProviderIDs(cfg)
	loader := config.NewProviderLoader(cfg.DataDir())
	manager := loader.GetCustomProviderManager()

	addedCount := 0
	for _, provider := range file.Providers {
		// Validate.
		result := config.ValidateCustomProvider(&provider, existingProviders)
		if !result.IsValid {
			fmt.Printf("Skipping %s: validation failed\n", provider.ID)
			for _, e := range result.Errors {
				fmt.Printf("  - %s\n", e)
			}
			continue
		}

		// Add if not duplicate.
		if manager.Exists(provider.ID) {
			fmt.Printf("Skipping %s: already exists\n", provider.ID)
			continue
		}

		if err := manager.Add(provider); err != nil {
			fmt.Printf("Error adding %s: %v\n", provider.ID, err)
			continue
		}

		existingProviders = append(existingProviders, provider.ID)
		addedCount++
		fmt.Printf("Added: %s (%s)\n", provider.Name, provider.ID)
	}

	fmt.Printf("\nAdded %d provider(s) from %s\n", addedCount, filePath)

	return nil
}

// newProvidersAddURLCmd adds providers from a URL.
func newProvidersAddURLCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "add-url <url>",
		Short: "Add providers from a URL",
		Long:  `Import custom providers from a URL that serves a providers.json file.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runProvidersAddURL,
	}

	return cmd
}

// runProvidersAddURL executes the add-url command.
func runProvidersAddURL(cmd *cobra.Command, args []string) error {
	url := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	fmt.Printf("Fetching providers from %s...\n", url)

	// Fetch the URL.
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("fetching URL: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %d: %s", resp.StatusCode, resp.Status)
	}

	// Read and parse the response.
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("reading response: %w", err)
	}

	var file config.CustomProvidersFile
	if err := json.Unmarshal(data, &file); err != nil {
		return fmt.Errorf("parsing JSON: %w", err)
	}

	if len(file.Providers) == 0 {
		return fmt.Errorf("no providers found at URL")
	}

	// Get existing provider IDs for validation.
	existingProviders := getExistingProviderIDs(cfg)
	loader := config.NewProviderLoader(cfg.DataDir())
	manager := loader.GetCustomProviderManager()

	addedCount := 0
	for _, provider := range file.Providers {
		// Validate.
		result := config.ValidateCustomProvider(&provider, existingProviders)
		if !result.IsValid {
			fmt.Printf("Skipping %s: validation failed\n", provider.ID)
			for _, e := range result.Errors {
				fmt.Printf("  - %s\n", e)
			}
			continue
		}

		// Add if not duplicate.
		if manager.Exists(provider.ID) {
			fmt.Printf("Skipping %s: already exists\n", provider.ID)
			continue
		}

		if err := manager.Add(provider); err != nil {
			fmt.Printf("Error adding %s: %v\n", provider.ID, err)
			continue
		}

		existingProviders = append(existingProviders, provider.ID)
		addedCount++
		fmt.Printf("Added: %s (%s)\n", provider.Name, provider.ID)
	}

	fmt.Printf("\nAdded %d provider(s) from URL\n", addedCount)

	return nil
}

// newProvidersRemoveCmd removes a custom provider.
func newProvidersRemoveCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "remove <provider-id>",
		Short: "Remove a custom provider",
		Long:  `Remove a custom provider by its ID.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runProvidersRemove,
	}

	return cmd
}

// runProvidersRemove executes the providers remove command.
func runProvidersRemove(cmd *cobra.Command, args []string) error {
	providerID := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	loader := config.NewProviderLoader(cfg.DataDir())
	manager := loader.GetCustomProviderManager()

	// Check if provider exists.
	if !manager.Exists(providerID) {
		return fmt.Errorf("custom provider %q not found", providerID)
	}

	// Remove the provider.
	if err := manager.Remove(providerID); err != nil {
		return fmt.Errorf("removing provider: %w", err)
	}

	fmt.Printf("Removed custom provider: %s\n", providerID)
	fmt.Println()
	fmt.Println("Note: Provider configuration may still exist in cdd.json.")
	fmt.Println("      Run 'cdd providers list' to see remaining providers.")

	return nil
}

// newProvidersExportCmd exports custom providers to a file.
func newProvidersExportCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "export [output-file]",
		Short: "Export custom providers to a file",
		Long:  `Export custom providers to a JSON file that can be shared or imported elsewhere.`,
		Args:  cobra.ExactArgs(1),
		RunE:  runProvidersExport,
	}

	return cmd
}

// runProvidersExport executes the providers export command.
func runProvidersExport(cmd *cobra.Command, args []string) error {
	outputPath := args[0]

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	loader := config.NewProviderLoader(cfg.DataDir())
	manager := loader.GetCustomProviderManager()

	customProviders, err := manager.Load()
	if err != nil {
		return fmt.Errorf("loading custom providers: %w", err)
	}

	if len(customProviders) == 0 {
		fmt.Println("No custom providers to export.")
		return nil
	}

	// Get the custom providers file path.
	customProvidersPath := manager.GetFilePath()

	// Read the current custom providers file.
	data, err := os.ReadFile(customProvidersPath)
	if err != nil {
		return fmt.Errorf("reading custom providers file: %w", err)
	}

	// Write to the output path.
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("writing output file: %w", err)
	}

	fmt.Printf("Exported %d custom provider(s) to: %s\n", len(customProviders), outputPath)

	return nil
}

// newProvidersValidateCmd validates custom provider configurations.
func newProvidersValidateCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate custom provider configurations",
		Long:  `Validate all custom provider configurations and report any errors or warnings.`,
		RunE:  runProvidersValidate,
	}

	cmd.Flags().Bool("verbose", false, "Show detailed validation results")

	return cmd
}

// runProvidersValidate executes the providers validate command.
func runProvidersValidate(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("loading config: %w", err)
	}

	loader := config.NewProviderLoader(cfg.DataDir())
	manager := loader.GetCustomProviderManager()

	customProviders, err := manager.Load()
	if err != nil {
		return fmt.Errorf("loading custom providers: %w", err)
	}

	if len(customProviders) == 0 {
		fmt.Println("No custom providers configured.")
		return nil
	}

	fmt.Printf("Validating %d custom provider(s)...\n\n", len(customProviders))

	allValid := true
	existingIDs := make([]string, 0, len(customProviders))

	for _, provider := range customProviders {
		// Collect existing IDs for uniqueness validation.
		existingIDs = append(existingIDs, provider.ID)

		// Validate the provider.
		result := config.ValidateCustomProvider(&provider, existingIDs)

		if result.IsValid {
			fmt.Printf("✓ %s (%s)\n", provider.Name, provider.ID)
		} else {
			fmt.Printf("✗ %s (%s)\n", provider.Name, provider.ID)
			allValid = false
		}

		if verbose || !result.IsValid {
			// Show errors.
			for _, e := range result.Errors {
				fmt.Printf("  Error: %s\n", e)
			}
			// Show warnings.
			for _, warning := range result.Warnings {
				fmt.Printf("  Warning: %s\n", warning)
			}
		}
	}

	fmt.Println()
	if allValid {
		fmt.Println("All custom providers are valid!")
	} else {
		fmt.Println("Some custom providers have validation errors.")
		return fmt.Errorf("validation failed")
	}

	return nil
}

// newProvidersTemplatesCmd lists available provider templates.
func newProvidersTemplatesCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "templates",
		Short: "List available provider templates",
		Long:  `List all available provider templates that can be used with add-template.`,
		RunE:  runProvidersTemplates,
	}

	return cmd
}

// runProvidersTemplates executes the templates command.
func runProvidersTemplates(cmd *cobra.Command, args []string) error {
	templates := config.ProviderTemplates()
	names := config.ListTemplateNames()

	fmt.Println("Available Provider Templates:")
	fmt.Println()

	for _, name := range names {
		t := templates[name]
		fmt.Printf("  %s\n", name)
		fmt.Printf("    %s\n", t.Description)
		if len(t.DefaultModels) > 0 {
			fmt.Printf("    Models: %d default\n", len(t.DefaultModels))
		}
		if len(t.Variables) > 0 {
			fmt.Printf("    Variables: ")
			for i, v := range t.Variables {
				if i > 0 {
					fmt.Printf(", ")
				}
				fmt.Printf("%s", v.Name)
			}
			fmt.Println()
		}
		fmt.Println()
	}

	return nil
}

// getExistingProviderIDs returns all existing provider IDs (both catwalk and custom).
func getExistingProviderIDs(cfg *config.Config) []string {
	loader := config.NewProviderLoader(cfg.DataDir())
	allProviders, _ := loader.LoadAllProviders(cfg)

	ids := make([]string, 0, len(allProviders))
	for _, p := range allProviders {
		ids = append(ids, string(p.ID))
	}
	return ids
}
